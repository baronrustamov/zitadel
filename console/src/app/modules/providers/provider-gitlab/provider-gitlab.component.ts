import { COMMA, ENTER, SPACE } from '@angular/cdk/keycodes';
import { Location } from '@angular/common';
import { Component, Injector, Type } from '@angular/core';
import { AbstractControl, FormControl, FormGroup } from '@angular/forms';
import { MatLegacyChipInputEvent as MatChipInputEvent } from '@angular/material/legacy-chips';
import { ActivatedRoute } from '@angular/router';
import { take } from 'rxjs';
import {
  AddGitLabProviderRequest as AdminAddGitLabProviderRequest,
  GetProviderByIDRequest as AdminGetProviderByIDRequest,
  UpdateGitLabProviderRequest as AdminUpdateGitLabProviderRequest,
} from 'src/app/proto/generated/zitadel/admin_pb';
import { Options, Provider } from 'src/app/proto/generated/zitadel/idp_pb';
import {
  AddGitLabProviderRequest as MgmtAddGitLabProviderRequest,
  GetProviderByIDRequest as MgmtGetProviderByIDRequest,
  UpdateGitLabProviderRequest as MgmtUpdateGitLabProviderRequest,
} from 'src/app/proto/generated/zitadel/management_pb';
import { AdminService } from 'src/app/services/admin.service';
import { Breadcrumb, BreadcrumbService, BreadcrumbType } from 'src/app/services/breadcrumb.service';
import { ManagementService } from 'src/app/services/mgmt.service';
import { ToastService } from 'src/app/services/toast.service';
import { requiredValidator } from '../../form-field/validators/validators';

import { PolicyComponentServiceType } from '../../policies/policy-component-types.enum';

@Component({
  selector: 'cnsl-provider-gitlab',
  templateUrl: './provider-gitlab.component.html',
})
export class ProviderGitlabComponent {
  public showOptional: boolean = false;
  public options: Options = new Options();
  public id: string | null = '';
  public serviceType: PolicyComponentServiceType = PolicyComponentServiceType.MGMT;
  private service!: ManagementService | AdminService;

  public readonly separatorKeysCodes: number[] = [ENTER, COMMA, SPACE];

  public form!: FormGroup;

  public loading: boolean = false;

  public provider?: Provider.AsObject;
  public updateClientSecret: boolean = false;

  constructor(
    private route: ActivatedRoute,
    private toast: ToastService,
    private injector: Injector,
    private _location: Location,
    private breadcrumbService: BreadcrumbService,
  ) {
    this.form = new FormGroup({
      name: new FormControl('', []),
      clientId: new FormControl('', [requiredValidator]),
      clientSecret: new FormControl('', [requiredValidator]),
      scopesList: new FormControl(['openid', 'profile', 'email'], []),
    });

    this.route.data.pipe(take(1)).subscribe((data) => {
      this.serviceType = data.serviceType;

      switch (this.serviceType) {
        case PolicyComponentServiceType.MGMT:
          this.service = this.injector.get(ManagementService as Type<ManagementService>);

          const bread: Breadcrumb = {
            type: BreadcrumbType.ORG,
            routerLink: ['/org'],
          };

          this.breadcrumbService.setBreadcrumb([bread]);
          break;
        case PolicyComponentServiceType.ADMIN:
          this.service = this.injector.get(AdminService as Type<AdminService>);

          const iamBread = new Breadcrumb({
            type: BreadcrumbType.ORG,
            name: 'Instance',
            routerLink: ['/instance'],
          });
          this.breadcrumbService.setBreadcrumb([iamBread]);
          break;
      }

      this.id = this.route.snapshot.paramMap.get('id');
      if (this.id) {
        this.clientSecret?.setValidators([]);
        this.getData(this.id);
      }
    });
  }

  private getData(id: string): void {
    const req =
      this.serviceType === PolicyComponentServiceType.ADMIN
        ? new AdminGetProviderByIDRequest()
        : new MgmtGetProviderByIDRequest();
    req.setId(id);
    this.service
      .getProviderByID(req)
      .then((resp) => {
        this.provider = resp.idp;
        this.loading = false;
        if (this.provider?.config?.gitlab) {
          this.form.patchValue(this.provider.config.gitlab);
          this.name?.setValue(this.provider.name);
        }
      })
      .catch((error) => {
        this.toast.showError(error);
        this.loading = false;
      });
  }

  public submitForm(): void {
    this.provider ? this.updateGitlabProvider() : this.addGitlabProvider();
  }

  public addGitlabProvider(): void {
    if (this.serviceType === PolicyComponentServiceType.MGMT) {
      const req = new MgmtAddGitLabProviderRequest();

      req.setName(this.name?.value);
      req.setClientId(this.clientId?.value);
      req.setClientSecret(this.clientSecret?.value);
      req.setScopesList(this.scopesList?.value);
      req.setProviderOptions(this.options);

      this.loading = true;
      (this.service as ManagementService)
        .addGitLabProvider(req)
        .then((idp) => {
          setTimeout(() => {
            this.loading = false;
            this.close();
          }, 2000);
        })
        .catch((error) => {
          this.toast.showError(error);
          this.loading = false;
        });
    } else if (PolicyComponentServiceType.ADMIN) {
      const req = new AdminAddGitLabProviderRequest();
      req.setName(this.name?.value);
      req.setClientId(this.clientId?.value);
      req.setClientSecret(this.clientSecret?.value);
      req.setScopesList(this.scopesList?.value);
      req.setProviderOptions(this.options);

      this.loading = true;
      (this.service as AdminService)
        .addGitLabProvider(req)
        .then((idp) => {
          setTimeout(() => {
            this.loading = false;
            this.close();
          }, 2000);
        })
        .catch((error) => {
          this.loading = false;
          this.toast.showError(error);
        });
    }
  }

  public updateGitlabProvider(): void {
    if (this.provider) {
      if (this.serviceType === PolicyComponentServiceType.MGMT) {
        const req = new MgmtUpdateGitLabProviderRequest();
        req.setId(this.provider.id);
        req.setName(this.name?.value);
        req.setClientId(this.clientId?.value);
        req.setScopesList(this.scopesList?.value);
        req.setProviderOptions(this.options);

        if (this.updateClientSecret) {
          req.setClientSecret(this.clientSecret?.value);
        }

        this.loading = true;
        (this.service as ManagementService)
          .updateGitLabProvider(req)
          .then((idp) => {
            setTimeout(() => {
              this.loading = false;
              this.close();
            }, 2000);
          })
          .catch((error) => {
            this.toast.showError(error);
            this.loading = false;
          });
      } else if (PolicyComponentServiceType.ADMIN) {
        const req = new AdminUpdateGitLabProviderRequest();
        req.setId(this.provider.id);
        req.setName(this.name?.value);
        req.setClientId(this.clientId?.value);
        req.setScopesList(this.scopesList?.value);
        req.setProviderOptions(this.options);

        if (this.updateClientSecret) {
          req.setClientSecret(this.clientSecret?.value);
        }

        this.loading = true;
        (this.service as AdminService)
          .updateGoogleProvider(req)
          .then((idp) => {
            setTimeout(() => {
              this.loading = false;
              this.close();
            }, 2000);
          })
          .catch((error) => {
            this.loading = false;
            this.toast.showError(error);
          });
      }
    }
  }

  public close(): void {
    this._location.back();
  }

  public addScope(event: MatChipInputEvent): void {
    const input = event.chipInput?.inputElement;
    const value = event.value.trim();

    if (value !== '') {
      if (this.scopesList?.value) {
        this.scopesList.value.push(value);
        if (input) {
          input.value = '';
        }
      }
    }
  }

  public removeScope(uri: string): void {
    if (this.scopesList?.value) {
      const index = this.scopesList.value.indexOf(uri);

      if (index !== undefined && index >= 0) {
        this.scopesList.value.splice(index, 1);
      }
    }
  }

  public get name(): AbstractControl | null {
    return this.form.get('name');
  }

  public get clientId(): AbstractControl | null {
    return this.form.get('clientId');
  }

  public get clientSecret(): AbstractControl | null {
    return this.form.get('clientSecret');
  }

  public get scopesList(): AbstractControl | null {
    return this.form.get('scopesList');
  }
}
