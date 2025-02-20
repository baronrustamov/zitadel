package projection

import (
	"context"

	"github.com/zitadel/zitadel/internal/errors"
	"github.com/zitadel/zitadel/internal/eventstore"
	"github.com/zitadel/zitadel/internal/eventstore/handler"
	"github.com/zitadel/zitadel/internal/eventstore/handler/crdb"
	"github.com/zitadel/zitadel/internal/repository/instance"
	"github.com/zitadel/zitadel/internal/repository/member"
	"github.com/zitadel/zitadel/internal/repository/org"
	"github.com/zitadel/zitadel/internal/repository/project"
	"github.com/zitadel/zitadel/internal/repository/user"
)

const (
	ProjectGrantMemberProjectionTable   = "projections.project_grant_members3"
	ProjectGrantMemberProjectIDCol      = "project_id"
	ProjectGrantMemberGrantIDCol        = "grant_id"
	ProjectGrantMemberGrantedOrg        = "granted_org"
	ProjectGrantMemberGrantedOrgRemoved = "granted_org_removed"
)

type projectGrantMemberProjection struct {
	crdb.StatementHandler
}

func newProjectGrantMemberProjection(ctx context.Context, config crdb.StatementHandlerConfig) *projectGrantMemberProjection {
	p := new(projectGrantMemberProjection)
	config.ProjectionName = ProjectGrantMemberProjectionTable
	config.Reducers = p.reducers()
	config.InitCheck = crdb.NewTableCheck(
		crdb.NewTable(
			append(memberColumns,
				crdb.NewColumn(ProjectGrantMemberProjectIDCol, crdb.ColumnTypeText),
				crdb.NewColumn(ProjectGrantMemberGrantIDCol, crdb.ColumnTypeText),
				crdb.NewColumn(ProjectGrantMemberGrantedOrg, crdb.ColumnTypeText),
				crdb.NewColumn(ProjectGrantMemberGrantedOrgRemoved, crdb.ColumnTypeBool, crdb.Default(false)),
			),
			crdb.NewPrimaryKey(MemberInstanceID, ProjectGrantMemberProjectIDCol, ProjectGrantMemberGrantIDCol, MemberUserIDCol),
			crdb.WithIndex(crdb.NewIndex("user_id", []string{MemberUserIDCol})),
			crdb.WithIndex(crdb.NewIndex("owner_removed", []string{MemberOwnerRemoved})),
			crdb.WithIndex(crdb.NewIndex("user_owner_removed", []string{MemberUserOwnerRemoved})),
			crdb.WithIndex(crdb.NewIndex("granted_org_removed", []string{ProjectGrantMemberGrantedOrgRemoved})),
		),
	)

	p.StatementHandler = crdb.NewStatementHandler(ctx, config)
	return p
}

func (p *projectGrantMemberProjection) reducers() []handler.AggregateReducer {
	return []handler.AggregateReducer{
		{
			Aggregate: project.AggregateType,
			EventRedusers: []handler.EventReducer{
				{
					Event:  project.GrantMemberAddedType,
					Reduce: p.reduceAdded,
				},
				{
					Event:  project.GrantMemberChangedType,
					Reduce: p.reduceChanged,
				},
				{
					Event:  project.GrantMemberCascadeRemovedType,
					Reduce: p.reduceCascadeRemoved,
				},
				{
					Event:  project.GrantMemberRemovedType,
					Reduce: p.reduceRemoved,
				},
				{
					Event:  project.ProjectRemovedType,
					Reduce: p.reduceProjectRemoved,
				},
				{
					Event:  project.GrantRemovedType,
					Reduce: p.reduceProjectGrantRemoved,
				},
			},
		},
		{
			Aggregate: user.AggregateType,
			EventRedusers: []handler.EventReducer{
				{
					Event:  user.UserRemovedType,
					Reduce: p.reduceUserRemoved,
				},
			},
		},
		{
			Aggregate: org.AggregateType,
			EventRedusers: []handler.EventReducer{
				{
					Event:  org.OrgRemovedEventType,
					Reduce: p.reduceOrgRemoved,
				},
			},
		},
		{
			Aggregate: instance.AggregateType,
			EventRedusers: []handler.EventReducer{
				{
					Event:  instance.InstanceRemovedEventType,
					Reduce: reduceInstanceRemovedHelper(MemberInstanceID),
				},
			},
		},
	}
}

func (p *projectGrantMemberProjection) reduceAdded(event eventstore.Event) (*handler.Statement, error) {
	e, ok := event.(*project.GrantMemberAddedEvent)
	if !ok {
		return nil, errors.ThrowInvalidArgumentf(nil, "HANDL-0EBQf", "reduce.wrong.event.type %s", project.GrantMemberAddedType)
	}
	ctx := setMemberContext(e.Aggregate())
	userOwner, err := getResourceOwnerOfUser(ctx, p.Eventstore, e.Aggregate().InstanceID, e.UserID)
	if err != nil {
		return nil, err
	}
	grantedOrg, err := getGrantedOrgOfGrantedProject(ctx, p.Eventstore, e.Aggregate().InstanceID, e.Aggregate().ID, e.GrantID)
	if err != nil {
		return nil, err
	}
	return reduceMemberAdded(
		*member.NewMemberAddedEvent(&e.BaseEvent, e.UserID, e.Roles...),
		userOwner,
		withMemberCol(ProjectGrantMemberProjectIDCol, e.Aggregate().ID),
		withMemberCol(ProjectGrantMemberGrantIDCol, e.GrantID),
		withMemberCol(ProjectGrantMemberGrantedOrg, grantedOrg),
		withMemberCol(ProjectGrantMemberGrantedOrgRemoved, false),
	)
}

func (p *projectGrantMemberProjection) reduceChanged(event eventstore.Event) (*handler.Statement, error) {
	e, ok := event.(*project.GrantMemberChangedEvent)
	if !ok {
		return nil, errors.ThrowInvalidArgumentf(nil, "HANDL-YX5Tk", "reduce.wrong.event.type %s", project.GrantMemberChangedType)
	}
	return reduceMemberChanged(
		*member.NewMemberChangedEvent(&e.BaseEvent, e.UserID, e.Roles...),
		withMemberCond(ProjectGrantMemberProjectIDCol, e.Aggregate().ID),
		withMemberCond(ProjectGrantMemberGrantIDCol, e.GrantID),
	)
}

func (p *projectGrantMemberProjection) reduceCascadeRemoved(event eventstore.Event) (*handler.Statement, error) {
	e, ok := event.(*project.GrantMemberCascadeRemovedEvent)
	if !ok {
		return nil, errors.ThrowInvalidArgumentf(nil, "HANDL-adnHG", "reduce.wrong.event.type %s", project.GrantMemberCascadeRemovedType)
	}
	return reduceMemberCascadeRemoved(
		*member.NewCascadeRemovedEvent(&e.BaseEvent, e.UserID),
		withMemberCond(ProjectGrantMemberProjectIDCol, e.Aggregate().ID),
		withMemberCond(ProjectGrantMemberGrantIDCol, e.GrantID),
	)
}

func (p *projectGrantMemberProjection) reduceRemoved(event eventstore.Event) (*handler.Statement, error) {
	e, ok := event.(*project.GrantMemberRemovedEvent)
	if !ok {
		return nil, errors.ThrowInvalidArgumentf(nil, "HANDL-MGNnA", "reduce.wrong.event.type %s", project.GrantMemberRemovedType)
	}
	return reduceMemberRemoved(e,
		withMemberCond(MemberUserIDCol, e.UserID),
		withMemberCond(ProjectGrantMemberProjectIDCol, e.Aggregate().ID),
		withMemberCond(ProjectGrantMemberGrantIDCol, e.GrantID),
	)
}

func (p *projectGrantMemberProjection) reduceUserRemoved(event eventstore.Event) (*handler.Statement, error) {
	e, ok := event.(*user.UserRemovedEvent)
	if !ok {
		return nil, errors.ThrowInvalidArgumentf(nil, "HANDL-rufJr", "reduce.wrong.event.type %s", user.UserRemovedType)
	}
	return reduceMemberRemoved(e, withMemberCond(MemberUserIDCol, e.Aggregate().ID))
}

func (p *projectGrantMemberProjection) reduceInstanceRemoved(event eventstore.Event) (*handler.Statement, error) {
	e, ok := event.(*instance.InstanceRemovedEvent)
	if !ok {
		return nil, errors.ThrowInvalidArgumentf(nil, "HANDL-Z2p6o", "reduce.wrong.event.type %s", instance.InstanceRemovedEventType)
	}
	return reduceMemberRemoved(e, withMemberCond(MemberInstanceID, e.Aggregate().ID))
}

func (p *projectGrantMemberProjection) reduceOrgRemoved(event eventstore.Event) (*handler.Statement, error) {
	e, ok := event.(*org.OrgRemovedEvent)
	if !ok {
		return nil, errors.ThrowInvalidArgumentf(nil, "HANDL-Zzp6o", "reduce.wrong.event.type %s", org.OrgRemovedEventType)
	}
	return crdb.NewMultiStatement(
		e,
		multiReduceMemberOwnerRemoved(e),
		multiReduceMemberUserOwnerRemoved(e),
		crdb.AddUpdateStatement(
			[]handler.Column{
				handler.NewCol(MemberChangeDate, e.CreationDate()),
				handler.NewCol(MemberSequence, e.Sequence()),
				handler.NewCol(ProjectGrantMemberGrantedOrgRemoved, true),
			},
			[]handler.Condition{
				handler.NewCond(ProjectGrantColumnInstanceID, e.Aggregate().InstanceID),
				handler.NewCond(ProjectGrantMemberGrantedOrg, e.Aggregate().ID),
			},
		),
	), nil
}

func (p *projectGrantMemberProjection) reduceProjectRemoved(event eventstore.Event) (*handler.Statement, error) {
	e, ok := event.(*project.ProjectRemovedEvent)
	if !ok {
		return nil, errors.ThrowInvalidArgumentf(nil, "HANDL-JLODy", "reduce.wrong.event.type %s", project.ProjectRemovedType)
	}
	return reduceMemberRemoved(e, withMemberCond(ProjectGrantMemberProjectIDCol, e.Aggregate().ID))
}

func (p *projectGrantMemberProjection) reduceProjectGrantRemoved(event eventstore.Event) (*handler.Statement, error) {
	e, ok := event.(*project.GrantRemovedEvent)
	if !ok {
		return nil, errors.ThrowInvalidArgumentf(nil, "HANDL-D1J9R", "reduce.wrong.event.type %s", project.GrantRemovedType)
	}
	return reduceMemberRemoved(e,
		withMemberCond(ProjectGrantMemberGrantIDCol, e.GrantID),
		withMemberCond(ProjectGrantMemberProjectIDCol, e.Aggregate().ID),
	)
}
