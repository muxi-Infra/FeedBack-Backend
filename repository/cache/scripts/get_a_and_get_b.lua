-- KEYS[1] = keyA
-- KEYS[2] = keyB

local a = redis.call("GET", KEYS[1])
if not a then
	redis.call("SET", KEYS[1], 0)
	a=0
else
    a = tonumber(a)
end

local b = redis.call("GET", KEYS[2])
if not b then
	redis.call("SET", KEYS[2], 0)
	b = 0
else
	b = tonumber(b)
end

return { a, b }
