-- foreign keys
SELECT
    ns.nspname    AS schema_name,
    -- constraint name
    ci.c_oid::INT AS constraint_oid,
    c.conname     AS constraint_name,
    -- constraint info
	c.contype::TEXT AS constraint_type,
	COALESCE(
		(SELECT NOT pg_index.indnullsnotdistinct
			FROM pg_index
			WHERE pg_index.indexrelid = c.conindid
		), False
	) AS nulls_not_distinct,
	pg_get_constraintdef(c.oid) as constraint_def,
    -- local table
    nc.oid::INT AS table_oid,
    nc.relname  AS table_name,
    ci.c_cols   AS table_columns,
    -- foreign table
    fc.oid::INT AS fc_table_oid,
    fs.nspname  AS fc_schema_name,
    fc.relname  AS fc_table_name,
    ci.fc_cols  AS fc_columns
FROM (
    SELECT
        c.oid                       AS c_oid,
        nc.oid                      AS c_table,
        ARRAY_AGG(a.attname::TEXT)  AS c_cols,
        -- fk only
        fc.oid                      AS fc_table,
        ARRAY_AGG(fa.attname::TEXT) AS fc_cols
    FROM pg_constraint    c
        JOIN pg_class    nc ON  c.conrelid = nc.oid
        JOIN pg_attribute a ON  a.attrelid = nc.oid
                            AND a.attnum   = ANY(c.conkey)
        -- fk only
        LEFT JOIN pg_class     fc ON   c.confrelid = fc.oid
        LEFT JOIN pg_attribute fa ON  fa.attrelid  = fc.oid
                                  AND fa.attnum    = ANY(c.confkey)
    GROUP BY
        c.oid,
        nc.oid,
        fc.oid
) AS ci
    JOIN pg_constraint      c ON  c.oid = ci.c_oid
    JOIN pg_class          nc ON nc.oid = ci.c_table
    JOIN pg_namespace      ns ON ns.oid =  c.connamespace
    LEFT JOIN pg_class     fc ON fc.oid = ci.fc_table
    LEFT JOIN pg_namespace fs ON fs.oid = fc.relnamespace
WHERE
	ns.nspname || '.' || nc.relname = ANY($1);
