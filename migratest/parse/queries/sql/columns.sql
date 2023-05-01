SELECT
	-- column
	ns_table.nspname AS schema_name,
	pc_table.relname AS table_name,
	a.attname        AS column_name,
	-- type
	ns_type.nspname  AS type_schema,
	t.typname        AS type_name,
	-- attributes
	a.attnotnull     AS is_nullable,
	a.atthasdef      AS has_default,
	a.attndims       AS array_dims,
	a.attgenerated = 's' AS is_generated,
	pg_get_expr(ad.adbin, ad.adrelid) AS default_expr,
	COALESCE(
		information_schema._pg_char_max_length(
			information_schema._pg_truetypid(a.*, t.*),
			information_schema._pg_truetypmod(a.*, t.*)
		),
		information_schema._pg_char_max_length(
			elem_t.oid,
			a.atttypmod
		)
	)::INT AS character_max_length
FROM
	pg_attribute a
	JOIN pg_class pc_table ON a.attrelid = pc_table.oid
	JOIN pg_namespace ns_table ON pc_table.relnamespace = ns_table.oid
	JOIN pg_type t ON a.atttypid = t.oid
	JOIN pg_namespace ns_type ON t.typnamespace = ns_type.oid
	LEFT JOIN pg_type elem_t ON elem_t.oid = t.typelem
	LEFT JOIN pg_attrdef ad ON a.attrelid = ad.adrelid AND a.attnum = ad.adnum
WHERE
	attnum > 0
	AND attisdropped = False
	AND ns_table.nspname || '.' || pc_table.relname = ANY($1)
ORDER BY
	a.attnum;
