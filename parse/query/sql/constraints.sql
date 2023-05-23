-- constraints
SELECT
	-- constraint oid
	c.oid::INT AS constraint_oid,
	c.conname AS constraint_name,
	c.connamespace::regnamespace::TEXT AS constraint_schema,
	-- constraint info
	c.contype::TEXT AS constraint_type,
	COALESCE(
		(SELECT NOT pg_index.indnullsnotdistinct
			FROM pg_index
			WHERE pg_index.indexrelid = c.conindid
		), False
	) AS nulls_not_distinct,
	pg_get_constraintdef(c.oid) AS constraint_def,
	-- local table
	c.conrelid::INT AS table_oid,
	COALESCE(c.conkey, '{}'::INT[]) AS table_colnums,
	-- foreign table
	c.confrelid::INT AS fc_table_oid,
	COALESCE(c.confkey, '{}'::INT[]) AS fc_colnums
FROM
	pg_constraint c
WHERE
	c.conrelid = ANY($1);
