local domains = require("domains.lua")

local defaultTopElements = 1000
local defaultStepFloatDomain = 0.1
local defaultTopFloatDomain = 10.0

local module = {
    -- домены по умолчанию для базовых типов
    DefaultTypeDomains = {
        pg_catalog = {
            bool = domains.Bool:new(),
            int2 = domains.Int:new(defaultTopElements),
            int4 = domains.Int:new(defaultTopElements),
            int8 = domains.Int:new(defaultTopElements),
            float4 = domains.Float:new(0, defaultStepFloatDomain, defaultTopFloatDomain),
            float8 = domains.Float:new(0, defaultStepFloatDomain, defaultTopFloatDomain),
            uuid = domains.UUID:new(),
            bytea = domains.UUID:new(),
            bit = domains.UUID:new(),
            varbit = domains.UUID:new(),
            char = domains.UUID:new(),
            varchar = domains.UUID:new(),
            text = domains.UUID:new(),
            date = domains.Time:new(os.time(), defaultTopElements),
            time = domains.Time:new(os.time(), defaultTopElements),
            timetz = domains.Time:new(os.time(), defaultTopElements),
            timestamp = domains.Time:new(os.time(), defaultTopElements),
            timestamptz = domains.Time:new(os.time(), defaultTopElements),
        }
    },
    -- переопределение доменов для конкретных колонок
    TableColumnDomains = {
        -- users = {
        --     id = domains.Serial:new(),
        --     -- all other by default
        --     role = function() return domains.Enum:new(conn.Query("SELECT DISTINCT role_name FROM test.roles")) end,
        -- },
    },
    -- порядок обработки таблиц. В рамках одной группы можно менять элементы местами, менять группы местами нельзя.
    TablesProcessOrder = {
        -- {"users", "products"},
        -- {"orders"},
    },
    -- правила для создания/загрузки частичных записей
    PartialRecords = {
        -- ["users"] = function() return go.LoadPartialChecksFromFile("users.csv") end,
        -- ["users"] = {
        --     [{ "id", "role" }] = {
        --         { "2", "admin" },
        --         { "3", "user" },
        --     },
        --     [{ "name" }] = {
        --         { "trailing spaces in name   " }
        --     },
        -- },
        -- ["products"] = {
        --     [{ "id", "admin_only" }] = {
        --         { "1", "true" },
        --     },
        -- },
        -- ["orders"] = {
        --     [{ "user_id", "product_id" }] = {
        --         { "2", "1" },
        --         { "3", "1" },
        --     },
        -- },
    },
}

return module
