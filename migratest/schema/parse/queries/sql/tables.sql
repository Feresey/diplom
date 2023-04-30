-- list tables
SELECT
	schemaname,
	tablename
FROM
	pg_tables
WHERE
	schemaname = ANY($1)
