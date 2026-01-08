-- KEYS[1] = key

local val = tonumber(redis.call("GET", KEYS[1]) or "0")
redis.call("SET", KEYS[1], 0)
return val
