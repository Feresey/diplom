SELECT
    -- table
    c.oid::INT AS index_table_oid,
    cs.nspname AS index_table_schema,
    c.relname AS index_table_name,
    -- index
    ci.oid::INT AS index_oid,
    ns.nspname AS index_schema,
    ci.relname AS index_name,
    -- related constraint
    co.oid AS constraint_oid,
    cos.nspname AS constraint_schema,
    co.conname AS constraint_name,
    -- index attributes
    i.indisunique AS is_unique,
    i.indisprimary AS is_primary,
    i.indnullsnotdistinct AS is_nulls_not_distinct,
    COALESCE(
        (SELECT
            ARRAY_AGG(a.attname)
        FROM
            pg_attribute a
            JOIN pg_class ac ON
                ac.oid = a.attrelid
                AND c.oid = ac.oid
                AND a.attnum = ANY(i.indkey)
        ),
        '{}'::TEXT[]
    ) AS index_columns,
    pg_get_indexdef(ci.oid) AS index_def
FROM
    pg_index i
    JOIN pg_class c ON i.indrelid = c.oid
    JOIN pg_namespace cs ON c.relnamespace = cs.oid
    JOIN pg_class ci ON i.indexrelid = ci.oid
    JOIN pg_namespace ns ON ci.relnamespace = ns.oid
    LEFT JOIN pg_constraint co ON co.conindid = ci.oid
    LEFT JOIN pg_namespace cos ON ci.relnamespace = cos.oid
WHERE
    i.indislive = True
    AND ns.nspname || '.' || c.relname = ANY($1)
    AND (
        co.oid IS NULL
        OR cos.nspname || '.' || co.conname = ANY($2)
    );
