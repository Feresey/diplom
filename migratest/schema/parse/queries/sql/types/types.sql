SELECT
	ns.nspname AS schema_name,
	t.typname  AS type_name,
	t.typtype::TEXT AS type_type,
	t.typcategory = 'A' AS is_array,
	ns_e.nspname AS element_type_shema,
	et.typname   AS element_type_name,
	t.typnotnull AS domain_is_not_nullable,
	nd.nspname   AS domain_schema,
	dt.typname   AS domain_type,
	information_schema._pg_char_max_length(dt.oid, t.typtypmod)::INT AS domain_character_max_length,
	t.typndims AS domain_array_dims
FROM
	pg_type t
	JOIN pg_namespace ns ON t.typnamespace = ns.oid
	LEFT JOIN pg_type et ON t.typelem = et.oid
	LEFT JOIN pg_namespace ns_e ON et.typnamespace = ns_e.oid
	LEFT JOIN pg_type dt ON t.typbasetype = dt.oid
	LEFT JOIN pg_namespace nd ON dt.typnamespace = nd.oid
WHERE
	ns.nspname || '.' || t.typname = ANY($1)
