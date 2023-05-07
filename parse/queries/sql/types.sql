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
	rmt_ns.nspname AS range_multi_type_schema,
	rmt.typname    AS range_multi_type_name,
	-- enum types
	(SELECT
		COALESCE(
			array_agg(e.enumlabel::TEXT),
			'{}'::TEXT[]
		) AS enum_values
	FROM
		pg_type t
		JOIN pg_enum      e ON t.oid = e.enumtypid
		JOIN pg_namespace n ON n.oid = t.typnamespace
	WHERE
		n.nspname || '.' || t.typname = ANY($1)
	GROUP BY
		t.oid
	) AS enum_values
FROM
	pg_type t
	     JOIN pg_namespace ns     ON ns.oid     =   t.typnamespace
	LEFT JOIN pg_type      et     ON et.oid     =   t.typelem
	LEFT JOIN pg_namespace et_ns  ON et_ns.oid  =  et.typnamespace
	LEFT JOIN pg_type      dt     ON dt.oid     =   t.typbasetype
	LEFT JOIN pg_namespace nd     ON nd.oid     =  dt.typnamespace
	LEFT JOIN pg_range     rng    ON t.oid      = rng.rngtypid
	LEFT JOIN pg_type      rst    ON rst.oid    = rng.rngsubtype
	LEFT JOIN pg_namespace rst_ns ON rst_ns.oid = rst.typnamespace
	LEFT JOIN pg_type      rmt    ON rmt.oid    = rng.rngmultitypid
	LEFT JOIN pg_namespace rmt_ns ON rmt_ns.oid = rmt.typnamespace
WHERE
	ns.nspname || '.' || t.typname = ANY($1)
