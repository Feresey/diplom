SELECT c.column_name,
    ct.column_type,
    (
        CASE
            WHEN c.character_maximum_length != 0 THEN (
                ct.column_type || '(' || c.character_maximum_length || ')'
            )
            ELSE c.udt_name
        END
    ) AS column_full_type,
    c.udt_name,
    e.data_type AS array_type,
    c.domain_name,
    c.column_default,
    COALESCE(
        col_description(
            (
                '"' || c.table_schema || '"."' || c.table_name || '"'
            )::regclass::oid,
            ordinal_position
        ),
        ''
    ) AS column_comment,
    c.is_nullable = 'YES' AS is_nullable,
    (
        CASE
            WHEN (
                SELECT CASE
                        WHEN column_name = 'is_identity' THEN (
                            SELECT c.is_identity = 'YES' AS is_identity
                        )
                        ELSE false
                    END AS is_identity
                FROM information_schema.columns
                WHERE table_schema = 'information_schema'
                    AND table_name = 'columns'
                    AND column_name = 'is_identity'
            ) IS NULL THEN 'NO'
            ELSE is_identity
        END
    ) = 'YES' AS is_identity,
    (
        SELECT EXISTS(
                SELECT 1
                FROM information_schema.table_constraints tc
                    INNER JOIN information_schema.constraint_column_usage AS ccu ON tc.constraint_name = ccu.constraint_name
                WHERE tc.table_schema = $1
                    AND tc.constraint_type = 'UNIQUE'
                    AND ccu.constraint_schema = $1
                    AND ccu.table_name = c.table_name
                    AND ccu.column_name = c.column_name
                    AND (
                        SELECT count(*)
                        FROM information_schema.constraint_column_usage
                        WHERE constraint_schema = $1
                            AND constraint_name = tc.constraint_name
                    ) = 1
            )
    )
    OR (
        SELECT EXISTS(
                SELECT 1
                FROM pg_indexes pgix
                    INNER JOIN pg_class pgc ON pgix.indexname = pgc.relname
                    AND pgc.relkind = 'i'
                    AND pgc.relnatts = 1
                    INNER JOIN pg_index pgi ON pgi.indexrelid = pgc.oid
                    INNER JOIN pg_attribute pga ON pga.attrelid = pgi.indrelid
                    AND pga.attnum = ANY(pgi.indkey)
                WHERE pgix.schemaname = $1
                    AND pgix.tablename = c.table_name
                    AND pga.attname = c.column_name
                    AND pgi.indisunique = true
            )
    ) AS is_unique
FROM information_schema.columns AS c
    INNER JOIN pg_namespace AS pgn ON pgn.nspname = c.udt_schema
    LEFT JOIN pg_type pgt ON c.data_type = 'USER-DEFINED'
    AND pgn.oid = pgt.typnamespace
    AND c.udt_name = pgt.typname
    LEFT JOIN information_schema.element_types e ON (
        (
            c.table_catalog,
            c.table_schema,
            c.table_name,
            'TABLE',
            c.dtd_identifier
        ) = (
            e.object_catalog,
            e.object_schema,
            e.object_name,
            e.object_type,
            e.collection_type_identifier
        )
    ),
    LATERAL (
        SELECT (
                CASE
                    WHEN pgt.typtype = 'e' THEN (
                        SELECT 'enum.' || c.udt_name || '(''' || string_agg(labels.label, ''',''') || ''')'
                        FROM (
                                SELECT pg_enum.enumlabel AS label
                                FROM pg_enum
                                WHERE pg_enum.enumtypid = (
                                        SELECT typelem
                                        FROM pg_type
                                            INNER JOIN pg_namespace ON pg_type.typnamespace = pg_namespace.oid
                                        WHERE pg_type.typtype = 'b'
                                            AND pg_type.typname = ('_' || c.udt_name)
                                            AND pg_namespace.nspname = $1
                                        LIMIT 1
                                    )
                                ORDER BY pg_enum.enumsortorder
                            ) AS labels
                    )
                    ELSE c.data_type
                END
            ) AS column_type
    ) ct
WHERE c.table_name = $2
    AND c.table_schema = $1
    AND c.is_generated = 'NEVER'
ORDER BY c.ordinal_position