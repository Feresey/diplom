SELECT
	ns_table.nspname AS schema_name,
	pc_table.relname AS table_name,
	a.attname AS column_name,
	ns_type.nspname AS column_type_schema,
	t.typname AS column_type_name,
	t.typtype::TEXT AS column_type_type,
	a.attnotnull AS is_nullable,
	a.atthasdef AS has_default,
	a.attgenerated = 's' AS is_generated,
	pg_get_expr(ad.adbin, ad.adrelid) AS default_expr,
	a.attndims AS array_dims,
	information_schema._pg_char_max_length(
		information_schema._pg_truetypid(a.*, t.*),
		information_schema._pg_truetypmod(a.*, t.*))::information_schema.cardinal_number AS characted_max_length,
	ns_composite.nspname AS composite_schema,
	t_composite.typname AS composite_type,
	t.typnotnull AS domain_is_not_nullable,
	t_domain.typname AS domain_type,
	information_schema._pg_char_max_length(
		information_schema._pg_truetypid(a.*, t_domain.*),
		information_schema._pg_truetypmod(a.*, t_domain.*))::information_schema.cardinal_number AS domain_characted_max_length,
	t.typndims AS domain_array_dims
FROM
	pg_attribute a
	JOIN pg_class pc_table ON a.attrelid = pc_table.oid
	JOIN pg_namespace ns_table ON pc_table.relnamespace = ns_table.oid
	JOIN pg_type t ON a.atttypid = t.oid
	JOIN pg_namespace ns_type ON t.typnamespace = ns_type.oid
	LEFT JOIN pg_type t_composite ON t.typrelid = t_composite.oid
	LEFT JOIN pg_namespace ns_composite ON t_composite.typnamespace = ns_composite.oid
	LEFT JOIN pg_type t_domain ON t.typbasetype = t_domain.oid
	LEFT JOIN pg_namespace ns_domain ON t_domain.typnamespace = ns_domain.oid
	LEFT JOIN pg_attrdef ad ON a.attrelid = ad.adrelid AND a.attnum = ad.adnum
WHERE
	attnum > 0
	AND attisdropped = False
	AND ns_table.nspname || '.' || pc_table.relname = ANY($1)
ORDER BY
	a.attnum;
