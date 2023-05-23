local FloatDomain = {}
FloatDomain.__index = FloatDomain

function FloatDomain:new(step, top)
    local fd = {index = -1, value = 0, step = step, top = top}
    setmetatable(fd, self)
    return fd
end
function FloatDomain:value()
    return tostring(self.value)
end
function FloatDomain:reset()
    self.index = -1
    self.value = 0
end

function FloatDomain:next()
    self.index = self.index + 1
    if self.index == 0 then
        -- возвращаем ноль
        return true
    end
    if self.index % 2 == 0 then
        self.value = -self.value
    else
        self.value = math.abs(self.value) + self.step
    end
    return self.value <= self.top
end

