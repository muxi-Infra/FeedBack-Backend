-- KEYS[1] = keyA
-- KEYS[2] = keyB

local a = redis.call("GET", KEYS[1])
if not a then
	redis.call("SET", KEYS[1], 0)
end

local b = redis.call("GET", KEYS[2])
if not b then
	redis.call("SET", KEYS[2], 0)
	b = 0
else
	b = tonumber(b)
end

-- A + 1
local newA = redis.call("INCR", KEYS[1])

-- B - 1（不允许负数）
local newB
if b <= 0 then
	redis.call("SET", KEYS[2], 0)
	newB = 0
else
	newB = redis.call("DECR", KEYS[2])
end

return { newA, newB }
