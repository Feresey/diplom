SELECT
	-- column
	a.attrelid::INT AS table_oid,
	a.attnum        AS column_num,
	a.attname       AS column_name,
	-- type
	t.oid::INT AS type_oid,
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
	)::INT AS character_max_length,
	a.atttypid = 1700 AS is_numeric,
	COALESCE(
		information_schema._pg_numeric_precision(
			information_schema._pg_truetypid(a.*, t.*),
			information_schema._pg_truetypmod(a.*, t.*)
		),
		information_schema._pg_numeric_precision(
			elem_t.oid,
			a.atttypmod
		)
	)::INT AS numeric_precision,
	COALESCE(
		information_schema._pg_numeric_scale(
			information_schema._pg_truetypid(a.*, t.*),
			information_schema._pg_truetypmod(a.*, t.*)
		),
		information_schema._pg_numeric_scale(
			elem_t.oid,
			a.atttypmod
		)
	)::INT AS numeric_scale
FROM
	pg_attribute a
	JOIN pg_type t ON a.atttypid = t.oid
	LEFT JOIN pg_attrdef ad ON a.attrelid = ad.adrelid AND a.attnum = ad.adnum
	LEFT JOIN pg_type elem_t ON elem_t.oid = t.typelem
WHERE
	attnum > 0
	AND attisdropped = False
	AND a.attrelid = ANY($1)
ORDER BY
	a.attrelid,
	a.attnum;
