package ratelimit

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

//go:embed slide_window.lua
var luaSlideWindow string

var ErrRateLimited = errors.New("rate limited")

type RedisSlidingWindowLimiter struct {
	cmd      redis.Cmdable
	interval time.Duration
	rate     int
}

func NewRedisSlidingWindowLimiter(cmd redis.Cmdable, interval time.Duration, rate int) Limiter {
	return &RedisSlidingWindowLimiter{
		cmd:      cmd,
		interval: interval,
		rate:     rate,
	}
}

// Limit 实现滑动窗口限流
// key: 限流维度，如 user_id、ip 等
// 返回 true 表示被限流，false 表示允许
func (r *RedisSlidingWindowLimiter) Limit(ctx context.Context, key string) (bool, error) {
	nowMs := time.Now().UnixMilli()
	// 使用纳秒保证同一毫秒内 member 唯一
	// 格式: {timestamp_ms}-{nanosecond}，例如 "1710000000123-456789012"
	member := fmt.Sprintf("%d-%d", nowMs, time.Now().UnixNano())

	result, err := r.cmd.Eval(ctx, luaSlideWindow, []string{key},
		r.interval.Milliseconds(), r.rate, nowMs, member).Int64()
	if err != nil {
		// 可选：记录日志（此处省略）
		return false, err
	}
	return result == 1, nil
}
