SELECT
    n.nspname AS schema_name,
    t.typname AS enum_name,
    array_agg(e.enumlabel) AS enum_values
FROM
    pg_type t
    JOIN pg_enum e ON t.oid = e.enumtypid
    JOIN pg_namespace n ON n.oid = t.typnamespace
WHERE
    n.nspname || '.' || t.typname = ANY($1)
GROUP BY
    n.nspname,
    t.typname;