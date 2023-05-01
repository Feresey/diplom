SELECT
	ns.nspname      AS schema_name,
	t.typname       AS type_name,
	t.typtype::TEXT AS type_type,
	-- array types
	t.typcategory = 'A' AS is_array,
	et_ns.nspname       AS element_type_shema,
	et.typname          AS element_type_name,
	-- domain types
	t.typnotnull AS domain_is_not_nullable,
	nd.nspname   AS domain_schema,
	dt.typname   AS domain_type,
	information_schema._pg_char_max_length(dt.oid, t.typtypmod)::INT AS domain_character_max_length,
	t.typndims   AS domain_array_dims,
	-- range types
	rst_ns.nspname AS range_element_type_schema,
	rst.typname    AS range_element_type_name,
	rmt_ns.nspname AS range_multi_element_type_schema,
	rmt.typname    AS range_multi_type_name
FROM
	pg_type t
	JOIN pg_namespace ns          ON t.typnamespace = ns.oid
	LEFT JOIN pg_type et          ON t.typelem = et.oid
	LEFT JOIN pg_namespace et_ns   ON et.typnamespace = et_ns.oid
	LEFT JOIN pg_type dt          ON t.typbasetype = dt.oid
	LEFT JOIN pg_namespace nd     ON dt.typnamespace = nd.oid
	LEFT JOIN pg_range rng        ON rng.rngtypid = t.oid
	LEFT JOIN pg_type rst         ON rng.rngsubtype = rst.oid
	LEFT JOIN pg_namespace rst_ns ON rst.typnamespace = rst_ns.oid
	LEFT JOIN pg_type rmt         ON rng.rngmultitypid = rmt.oid
	LEFT JOIN pg_namespace rmt_ns ON rmt.typnamespace = rmt_ns.oid
WHERE
	ns.nspname || '.' || t.typname = ANY($1)
