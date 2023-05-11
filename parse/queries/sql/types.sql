SELECT
	t.oid::INT      AS type_oid,
	t.typnamespace::regnamespace::TEXT AS type_schema,
	t.typname       AS type_name,
	t.typtype::TEXT AS type_type,
	-- array types
	t.typcategory = 'A' AS is_array,
	et.oid::INT AS element_type_oid,
	-- domain types
	dt.oid       AS domain_type_oid,
	t.typnotnull AS domain_is_not_nullable,
	information_schema._pg_char_max_length(dt.oid, t.typtypmod)::INT AS domain_character_max_length,
	COALESCE(dt.oid = 1700, False) AS domain_is_numeric,
	information_schema._pg_numeric_precision(dt.oid, t.typtypmod)::INT AS domain_precision,
	information_schema._pg_numeric_scale(dt.oid, t.typtypmod)::INT AS domain_scale,
	t.typndims AS domain_array_dims,
	-- range types
	rng.rngtypid::INT AS range_element_type_oid
FROM
	pg_type t
	LEFT JOIN pg_type  et  ON et.oid =   t.typelem
	LEFT JOIN pg_type  dt  ON dt.oid =   t.typbasetype
	LEFT JOIN pg_range rng ON t.oid  = rng.rngtypid
WHERE
	t.oid = ANY($1);
