
-- KEYS[1] = key
-- ARGV[1] = window (ms)
-- ARGV[2] = threshold
-- ARGV[3] = now (ms)

local key = KEYS[1]                     -- 限流对象
local window = tonumber(ARGV[1])     -- 窗口大小（毫秒）
local threshold = tonumber(ARGV[2]) -- 阈值
local now = tonumber(ARGV[3])           -- 当前时间戳（毫秒）
local member = ARGV[4]               -- 唯一请求标识
local min = now - window

-- 清理窗口外的旧请求
redis.call('ZREMRANGEBYSCORE', key, '-inf', min)

-- 统计当前窗口内的请求数
local cnt = redis.call('ZCOUNT', key, min, '+inf')

if cnt >= threshold then
    return 1  -- 限流
else
    redis.call('ZADD', key, now, member)  -- score=时间戳，member=唯一ID
    redis.call('PEXPIRE', key, window)    -- 自动过期，单位毫秒
    return 0  -- 放行
end