SELECT
    t.oid::INT AS type_oid,
    COALESCE(
        array_agg(e.enumlabel::TEXT),
        '{}'::TEXT[]
    ) AS enum_values
FROM
    pg_type t
    JOIN pg_enum e ON t.oid = e.enumtypid
WHERE
    t.oid = ANY($1)
GROUP BY
    t.oid;