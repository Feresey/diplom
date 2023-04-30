SELECT
    ns.nspname AS schema_name,
    arr_t.typname AS array_name,
    elem_t.typname AS elem_name,
FROM
    pg_type arr_t
    JOIN pg_namespace ns ON arr_t.typnamespace = ns.oid
    JOIN pg_type elem_t ON arr_t.oid = elem_t.typarray
WHERE
    ns.nspname || '.' || arr_t.typname = ANY($1);
