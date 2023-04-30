SELECT
	ns.nspname AS schema_name,
	t.typname AS type_name,
	t.typtype::TEXT AS type_type,
	t.typnotnull AS domain_is_not_nullable,
	dt.typname AS domain_type,
	information_schema._pg_char_max_length(dt.oid, t.typtypmod)::INT AS domain_character_max_length,
	t.typndims AS domain_array_dims
FROM
	pg_type t
	JOIN pg_namespace ns ON t.typnamespace = ns.oid
	LEFT JOIN pg_class cc ON t.typrelid = cc.oid
	LEFT JOIN pg_namespace c_ns ON cc.relnamespace = c_ns.oid
	LEFT JOIN pg_type ct ON cc.reltype = ct.oid
WHERE
	ns.nspname || '.' || t.typname = ANY($1)
