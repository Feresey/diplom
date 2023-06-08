local checks = {}
checks.__index = checks

checks.aliases = {
    int2 = "int",
    int4 = "int",
    int8 = "int",
    float4 = "float",
    float8 = "float",

    uuid = "text",
    bytea = "text",
    bit = "text",
    varbit = "text",
    char = "text",
    varchar = "text",

    timetz = "time",
    date = "datetime",
    timestamp = "datetime",
    timestamptz = "datetime",
}

checks.checks = {
    bool = { "True", "False" },

    int = { "0", "-1", "1" },
    int2 = { "-32768", "32767" },
    int4 = { "-2147483648", "2147483647" },
    int8 = { "-9223372036854775808", "9223372036854775807" },
    -- numeric типы с явно указанными precision и scale не могут хранить +-Inf
    numeric = { "'NaN'::NUMERIC" },
    float = { "0", "'NaN'::REAL", "'infinity'::REAL", "'-infinity'::REAL" },

    -- нет текстовых типов с длиной меньше 1, а нолик почти для любого текстового типа валидный
    text = { "''", "' '", "'0'" },

    datetime = {
        "'epoch'::TIMESTAMP",
        "'infinity'::TIMESTAMP",
        "'-infinity'::TIMESTAMP",
    },
    time = {
        "'allballs'::TIME",
    },
}

checks.col_replaces = {
    -- test.users.col1 = {1,2,3},
}
checks.col_appends = {
    -- test.users.col1 = {1,2,3},
}

checks.type_replaces = {
    -- test.typ1 = {1,2,3},
}
checks.type_appends = {
    -- test.typ1 = {1,2,3},
}

checks.skip_tables = {
    -- "test.users"
}

local function mergeLists(dst, src)
    if not src or not dst then return end
    for _, val in ipairs(src) do
        table.insert(dst, val)
    end
end

function checks:get_type_checks(typename)
    local replaces = self.type_replaces[typename]
    if replaces then
        return replaces
    end
    local res = {}
    local aliased = self.checks[self.aliases[typename]]
    mergeLists(res, aliased)
    local base = self.checks[typename]
    mergeLists(res, base)
    local appends = self.type_appends[typename]
    mergeLists(res, appends)
    return res
end

function checks:get_column_checks(t, column)
    local replaces = self.col_replaces[t.name .. "." .. column.name]
    if replaces then
        return replaces
    end
    local col_checks = {}
    if column.attr.notnull then
        table.insert(col_checks, "NULL")
    end

    local max_char = column.attr.char_max_length
    if max_char then
        table.insert(col_checks, "'" .. string.rep(" ", max_char) .. "'")
        table.insert(col_checks, "'" .. string.rep("0", max_char) .. "'")
    end

    if column.attr.is_numeric then
        -- только для типа NUMERIC без явно заданных precision, scale возможны такие значения
        if column.attr.precision == 0 and column.attr.scale == 0 then
            table.insert(col_checks, "'infinity'::NUMERIC")
            table.insert(col_checks, "'-infinity'::NUMERIC")
        end
    end

    local appends = self.col_appends[t.name .. "." .. column.name]

    local res = {}
    mergeLists(res, col_checks)
    mergeLists(res, checks:get_type_checks(column.type.name))
    mergeLists(res, appends)
    return res
end

function checks:get_table_checks(t)
    if self.skip_tables[t.name] then return end

    local foreign_columns = {}
    for _, foreign in pairs(t.fk) do
        for _, colname in ipairs(foreign.columns) do
            foreign_columns[colname] = true
        end
    end

    local res = {}

    for colname, col in pairs(t.columns) do
        if not foreign_columns[col] then
            res[colname] = checks:get_column_checks(t, col)
        end
    end

    return res
end

return checks
