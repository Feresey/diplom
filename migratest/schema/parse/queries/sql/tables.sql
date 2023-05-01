-- list tables
SELECT
	c.oid::INT AS table_oid,
	ns.nspname AS schema_name,
	c.relname AS table_name
FROM
	pg_class c
	JOIN pg_namespace ns ON ns.oid = c.relnamespace
WHERE
