-- tables constraints
SELECT
	nr.nspname::TEXT AS table_schema,
	r.relname::TEXT  AS table_name,
	nr.nspname::TEXT AS constraint_schema,
	c.conname::TEXT  AS constraint_name,
	c.contype::TEXT  AS constraint_type,
	COALESCE(
		(SELECT NOT pg_index.indnullsnotdistinct
			FROM pg_index
			WHERE pg_index.indexrelid = c.conindid
		), False
	) AS nulls_not_distinct,
	pg_get_constraintdef(c.oid) as constraint_def
FROM
	pg_constraint c
	JOIN pg_class r ON c.conrelid = r.oid
	JOIN pg_namespace nr ON nr.oid = r.relnamespace AND c.conrelid = r.oid
WHERE
	nr.nspname || '.' || r.relname = ANY($1)
