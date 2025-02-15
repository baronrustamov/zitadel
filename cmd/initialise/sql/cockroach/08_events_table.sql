SET experimental_enable_hash_sharded_indexes = on;

CREATE TABLE IF NOT EXISTS eventstore.events (
	id UUID DEFAULT gen_random_uuid()
	, event_type TEXT NOT NULL
	, aggregate_type TEXT NOT NULL
	, aggregate_id TEXT NOT NULL
	, aggregate_version TEXT NOT NULL
	, event_sequence BIGINT NOT NULL
	, previous_aggregate_sequence BIGINT
	, previous_aggregate_type_sequence INT8
	, creation_date TIMESTAMPTZ NOT NULL DEFAULT now()
	, event_data JSONB
	, editor_user TEXT NOT NULL 
	, editor_service TEXT NOT NULL
	, resource_owner TEXT NOT NULL
	, instance_id TEXT NOT NULL

	, PRIMARY KEY (event_sequence DESC, instance_id) USING HASH WITH BUCKET_COUNT = 10
	, INDEX agg_type_agg_id (aggregate_type, aggregate_id, instance_id)
	, INDEX agg_type (aggregate_type, instance_id)
	, INDEX agg_type_seq (aggregate_type, event_sequence DESC, instance_id)
		STORING (id, event_type, aggregate_id, aggregate_version, previous_aggregate_sequence, creation_date, event_data, editor_user, editor_service, resource_owner, previous_aggregate_type_sequence)
	, INDEX max_sequence (aggregate_type, aggregate_id, event_sequence DESC, instance_id)
	, CONSTRAINT previous_sequence_unique UNIQUE (previous_aggregate_sequence DESC, instance_id)
	, CONSTRAINT prev_agg_type_seq_unique UNIQUE(previous_aggregate_type_sequence, instance_id)
);