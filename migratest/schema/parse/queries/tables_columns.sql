-- tables columns
SELECT
	table_name,
	table_schema,
	column_name,
	data_type,
	udt_schema,
	udt_name,
	character_maximum_length,
	domain_schema,
	domain_name,
	is_nullable = 'YES' AS is_nullable,
	column_default,
	is_generated = 'ALWAYS' AS is_generated,
	generation_expression
FROM
	information_schema.columns
WHERE
	table_schema || '.' || table_name = ANY($1)
