-- foreign keys
SELECT
    *
FROM (
    SELECT
        c.oid      AS constraint_oid,
        ns.nspname AS schema_name,
        c.conname  AS constraint_name,
        nc.relname AS table_name,
        a.attname  AS column_name,
        fc.oid     AS foreign_constraint_oid,
        fs.nspname AS foreign_schema_name,
        fc.relname AS foreign_table_name,
        fa.attname AS foreign_column_name
    FROM pg_constraint c
        JOIN pg_namespace ns ON ns.oid = c.connamespace
        JOIN pg_class nc ON c.conrelid = nc.oid
        JOIN pg_attribute a ON a.attrelid = nc.oid
            AND a.attnum = ANY(c.conkey)
        LEFT JOIN pg_class fc ON c.confrelid = fc.oid
        LEFT JOIN pg_namespace fs ON fs.oid = fc.relnamespace
        LEFT JOIN pg_attribute fa ON fa.attrelid = fc.oid
            AND fa.attnum = ANY(c.confkey)
) AS coninfo
WHERE
	ns.nspname || '.' || c.conname = ANY($1)