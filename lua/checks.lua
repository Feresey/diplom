local checks = {
    TypeChecks = {
        bool = { "True", "False" },

        int2 = { "0", "-1", "1", "-32768", "32767" },
        int4 = { "0", "-1", "1", "-2147483648", "2147483647" },
        int8 = { "0", "-1", "1", "-9223372036854775808", "9223372036854775807" },
        -- numeric типы с явно указанными precision и scale не могут хранить +-Inf и иногда даже нолик
        numeric = { "NaN" },
        float4 = { "0", "NaN", "infinity", "-infinity" },
        float8 = { "0", "NaN", "infinity", "-infinity" },

        -- нет текстовых типов с длиной меньше 1, а нолик почти для любого текстового типа валидный
        text = { "", " ", "0" },
        char = { "", " ", "0" },
        bit = { "", " ", "0" },
        bytea = { "", " ", "0" },

        time = { "allballs" },
        timetz = { "allballs" },
        date = { "epoch", "infinity", "-infinity" },
        timestamp = { "epoch", "infinity", "-infinity" },
        timestamptz = { "epoch", "infinity", "-infinity" },

        -- test.name_domain = {""},
    },
    TableColumnChecks = {
        -- test.users.col1 = {1,2,3},
    },
    Skip = {
        -- ["test.roles"]="all",
        -- ["test.users"]={"name", "email"},
    },
}

checks.__index = checks

local function mergeLists(dst, src)
    if not src or not dst then return end
    for _, val in ipairs(src) do
        table.insert(dst, val)
    end
end

function checks:GetTypeChecks(t, column)
    local typname = column.type.schema
    if column.type.schema ~= "pg_catalog" then
        typname = typname .. "."
    else
        typname = ""
    end
    typname = typname .. column.type.name
    return self.TypeChecks[typname]
end

function checks:GetColumnChecks(t, column)
    local skip = self.Skip[t.name]
    if type(skip) == "string" and skip == "all" then return end
    if type(skip) ~= "table" then return end

    local replaces = self.TableColumnChecks[t.schema .. "." .. t.name .. "." .. column.name]
    if replaces then
        return replaces
    end
    local col_checks = {}
    if column.attr.notnull then
        table.insert(col_checks, "NULL")
    end

    local max_char = column.attr.char_max_length
    if max_char then
        table.insert(col_checks, string.rep(" ", max_char))
        table.insert(col_checks, string.rep("0", max_char))
    end

    if column.attr.is_numeric then
        -- только для типа NUMERIC без явно заданных precision, scale возможны такие значения
        if column.attr.precision == 0 and column.attr.scale == 0 then
            table.insert(col_checks, "infinity")
            table.insert(col_checks, "-infinity")
        end
        -- TODO add min/max for numeric values with scale and precision
    end

    local res = {}
    mergeLists(res, col_checks)
    mergeLists(res, checks:GetTypeChecks(t, column))
    return res
end

-- function checks:GetTableChecks(t)
--     local skip = self.Skip[t.name]
--     if type(skip) == "string" and skip == "all" then return end
--     if type(skip) ~= "table" then return end

--     local foreign_columns = {}
--     for _, foreign in pairs(t.fk) do
--         for _, colname in ipairs(foreign.columns) do
--             foreign_columns[colname] = true
--         end
--     end

--     local res = {}

--     for colname, col in pairs(t.columns) do
--         if not foreign_columns[col] and not skip[col] then
--             res[colname] = checks:GetColumnChecks(t, col)
--         end
--     end

--     return res
-- end

return checks
