SELECT
    -- table
    i.indrelid::INT AS index_table_oid,
    -- index
    ci.oid::INT AS index_oid,
    ci.relnamespace::regnamespace::TEXT AS index_schema,
    ci.relname AS index_name,
    -- related constraint
    co.oid AS constraint_oid,
    -- index attributes
    i.indisunique AS is_unique,
    i.indisprimary AS is_primary,
    i.indnullsnotdistinct AS is_nulls_not_distinct,
    COALESCE(i.indkey, '{}'::INT[]) AS index_colnums,
    pg_get_indexdef(ci.oid) AS index_def
FROM
    pg_index i
    JOIN pg_class ci ON i.indexrelid = ci.oid
    LEFT JOIN pg_constraint co ON co.conindid = ci.oid
WHERE
    i.indislive = True
    AND i.indrelid = ANY($1)
    AND (
        co.oid IS NULL
        OR co.oid = ANY($2)
    );
