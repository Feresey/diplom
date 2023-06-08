local Int = {}
Int.__index = Int

function Int:new(start, step, top, allow_negative)
    local id = {
        start = start,
        step = step,
        top = top,
        allow_negative = allow_negative,
        index = -1,
        value = start,
    }
    return setmetatable(id, self)
end

function Int:reset()
    self.index = -1
    self.value = self.start
end

function Int:next()
    self.index = self.index + 1
    if self.index == 0 then return self.value end
    if self.index % 2 == 0 and self.allow_negative then
        return -self.value
    end
    self.value = self.value + self.step
    if self.value > self.top then
        return
    end
    return self.value
end

local Time = {}
Time.__index = Time

function Time:new(params)
    if not params.now then params.now = os.time() end
    if not params.step then params.step = 1 end
    if not params.top then params.top = 1000 end
    -- default format is RFC3339
    if not params.format then params.format = '!%Y-%m-%dT%H:%M:%SZ' end
    local td = {
        params = {
            now = params.now,
            step = params.step,
            top = params.top,
            allow_negative = params.allow_negative,
            format = params.format,
        },
        index = -1,
        value = 0,
    }
    return setmetatable(td, self)
end

function Time:reset()
    self.index = -1
    self.value = 0
end

function Time:next()
    self.index = self.index + 1
    if self.index == 0 then
        return os.date(self.params.format, self.params.now)
    end
    if self.index >= self.params.top then
        return
    end
    if self.index % 2 == 1 then
        self.value = self.value + self.params.step
    end
    local diff = self.value
    if self.index % 2 == 0 and self.params.allow_negative then
        diff = -self.value
    end
    return os.date(self.params.format, self.params.now + diff)
end

local Float = {}
Float.__index = Float

function Float:new(start, step, top, allow_negative)
    local fd = {
        start = start,
        step = step or 0.1,
        top = top or 10.0,
        allow_negative = allow_negative,
        index = -1,
        value = start,
    }
    return setmetatable(fd, self)
end

function Float:reset()
    self.index = -1
    self.value = self.start
end

function Float:next()
    self.index = self.index + 1
    if self.index == 0 then
        -- возвращаем ноль
        return self.value
    end
    if self.index % 2 == 0 and self.allow_negative then
        return -self.value
    end
    self.value = self.value + self.step
    if self.value > self.top then
        return
    end
    return self.value
end

local Enum = {}
Enum.__index = Enum

function Enum:new(values)
    local ed = {
        index = 0,
        values = values,
    }
    return setmetatable(ed, self)
end

function Enum:reset()
    self.index = 0
end

function Enum:next()
    self.index = self.index + 1
    if self.index > #self.values then
        return
    end
    return self.values[self.index]
end

local uuid = require("go_uuid")
local UUID = {}
UUID.__index = UUID

function UUID:new() return setmetatable({}, self) end

function UUID:next() return uuid.new() end

function UUID:reset() end

local domains = {
    UUID = UUID,
    Bool = function() return Enum:new({ "True", "False" }) end,
    Enum = Enum,
    Int = Int,
    Float = Float,
    Time = Time,
}

return domains
