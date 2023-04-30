-- foreign keys
SELECT
	ns.nspname AS constraint_schema,
	c.conname AS constraint_name,
	nc.relname AS table_name,
	a.attname AS column_name
FROM pg_constraint c
	JOIN pg_class nc ON c.conrelid = nc.oid
	JOIN pg_attribute a ON a.attrelid = nc.oid
		AND a.attnum = ANY(c.conkey)
	JOIN pg_namespace ns ON ns.oid = c.connamespace
WHERE
	ns.nspname || '.' || nc.relname = ANY($1)
	AND ns.nspname || '.' || c.conname = ANY($2)