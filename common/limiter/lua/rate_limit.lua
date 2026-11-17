-- Token bucket rate limiter
-- KEYS[1]: unique limiter identifier
-- ARGV[1]: number of tokens requested (usually 1)
-- ARGV[2]: token generation rate (per second)
-- ARGV[3]: bucket capacity

local key = KEYS[1]
local requested = tonumber(ARGV[1])
local rate = tonumber(ARGV[2])
local capacity = tonumber(ARGV[3])

-- Get the current time (Redis server time)
local now = redis.call('TIME')
local nowInSeconds = tonumber(now[1])

-- Get the bucket state
local bucket = redis.call('HMGET', key, 'tokens', 'last_time')
local tokens = tonumber(bucket[1])
local last_time = tonumber(bucket[2])

-- Initialize the bucket (first request or expired)
if not tokens or not last_time then
    tokens = capacity
    last_time = nowInSeconds
else
    -- Compute newly added tokens
    local elapsed = nowInSeconds - last_time
    local add_tokens = elapsed * rate
    tokens = math.min(capacity, tokens + add_tokens)
    last_time = nowInSeconds
end

-- Determine whether the request is allowed
local allowed = false
if tokens >= requested then
    tokens = tokens - requested
    allowed = true
end

---- Update the bucket state and set the expiration
redis.call('HMSET', key, 'tokens', tokens, 'last_time', last_time)
--redis.call('EXPIRE', key, math.ceil(capacity / rate) + 60) -- Extend the expiration appropriately

return allowed and 1 or 0