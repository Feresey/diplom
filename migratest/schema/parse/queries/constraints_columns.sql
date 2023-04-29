-- constraints columns
SELECT
	ccu.table_schema,
	ccu.table_name,
	ccu.column_name,
	ccu.constraint_schema,
	ccu.constraint_name
FROM
	information_schema.constraint_column_usage ccu
WHERE
	ccu.table_schema || '.' || ccu.table_name = ANY($1)
	AND ccu.constraint_schema || '.' || ccu.constraint_name = ANY($2)
