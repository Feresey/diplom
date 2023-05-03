tests

## fuckup

``` sql
BEGIN;
CREATE TABLE delme(text_id TEXT UNIQUE, pk SERIAL PRIMARY KEY);
CREATE TABLE delme_ref(ref_text_id TEXT REFERENCES delme(text_id));
SELECT
    d.*,
    c.conname AS objname,
    c_ref.relname AS refobjname
FROM pg_depend d
    JOIN pg_constraint c ON c.oid = d.objid
    JOIN pg_class c_ref ON c_ref.oid = d.refobjid
WHERE
    d.classid = 'pg_constraint'::regclass
    AND d.refclassid = 'pg_class'::regclass;

DROP TABLE delme_ref;
DROP TABLE delme;
END;
```

``` sql
BEGIN;
CREATE TABLE delme(text_id TEXT, pk SERIAL PRIMARY KEY);
CREATE UNIQUE INDEX delme_uniq ON delme(text_id);
CREATE TABLE delme_ref(ref_text_id TEXT REFERENCES delme(text_id));
SELECT
    d.*,
    c.conname AS objname,
    c_ref.relname AS refobjname
FROM pg_depend d
    JOIN pg_constraint c ON c.oid = d.objid
    JOIN pg_class c_ref ON c_ref.oid = d.refobjid
WHERE
    d.classid = 'pg_constraint'::regclass
    AND d.refclassid = 'pg_class'::regclass;

DROP TABLE delme_ref;
DROP TABLE delme;
END;
```

``` sql
SELECT
    ns.nspname AS schema_name,
    c.conname  AS constraint_name,
    nc.relname AS table_name,
    a.attname  AS column_name,
    fc.relname AS foreign_table_name,
    fa.attname AS foreign_column_name
FROM pg_constraint c
    JOIN pg_namespace ns ON ns.oid = c.connamespace
    JOIN pg_class nc ON c.conrelid = nc.oid
    JOIN pg_class fc ON c.confrelid = fc.oid
    JOIN pg_attribute a ON a.attrelid = nc.oid
        AND a.attnum = ANY(c.conkey)
    JOIN pg_attribute fa ON fa.attrelid = nc.oid
        AND fa.attnum = ANY(c.confkey)
WHERE c.conname LIKE '%delme%'
```
