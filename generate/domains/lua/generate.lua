local domains = require("domains.lua")

local defaultTopElements = 1000
local defaultStepFloatDomain = 0.1
local defaultTopFloatDomain = 10.0

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
}

DefaultTableDomains = {
    users = {
        full_name = domains.UUID:new(),
    }
}
