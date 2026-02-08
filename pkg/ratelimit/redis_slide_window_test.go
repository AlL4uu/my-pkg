package ratelimit

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
)

func TestRedisSlidingWindowLimiter_Limit(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("failed to start miniredis: %v", err)
	}
	defer mr.Close()

	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	ctx := context.Background()

	testCases := []struct {
		name string

		interval time.Duration
		rate     int
		key      string
		// requests 定义每次调用期望是否被限流
		requests []bool

		// delayAfter 用于窗口滑动测试（可选）
		delayAfter time.Duration
		// extraKey 用于验证 key 隔离（可选）
		extraKey string
	}{
		{
			name:     "允许3次，第4次限流",
			interval: 5 * time.Second,
			rate:     3,
			key:      "user1",
			requests: []bool{false, false, false, true},
		},
		{
			name:       "窗口滑动后恢复",
			interval:   200 * time.Millisecond,
			rate:       1,
			key:        "user2",
			requests:   []bool{false, true},
			delayAfter: 250 * time.Millisecond, // 等待窗口过期后再测一次
		},
		{
			name:     "不同 key 互不影响",
			interval: 5 * time.Second,
			rate:     1,
			key:      "user3",
			requests: []bool{false, true},
			extraKey: "user4", // 额外验证 user4 不受影响
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			limiter := NewRedisSlidingWindowLimiter(client, tc.interval, tc.rate)

			// 执行主请求序列
			for i, expectLimited := range tc.requests {
				gotLimited, err := limiter.Limit(ctx, tc.key)
				if err != nil {
					t.Errorf("request #%d: unexpected error: %v", i+1, err)
					continue
				}
				if gotLimited != expectLimited {
					t.Errorf("request #%d: expected limited=%v, got=%v", i+1, expectLimited, gotLimited)
				}
			}

			// 可选：窗口滑动后恢复测试
			// 等待一段时间后再发一次，看是否恢复
			if tc.delayAfter > 0 {
				time.Sleep(tc.delayAfter)
				gotLimited, err := limiter.Limit(ctx, tc.key)
				if err != nil {
					t.Fatalf("after delay: unexpected error: %v", err)
				}
				if gotLimited {
					t.Error("after window slide: expected to be allowed, but got limited")
				}
			}

			// 可选：验证 key 隔离
			// 用另一个 key 发请求，看是否被限
			if tc.extraKey != "" {
				gotLimited, err := limiter.Limit(ctx, tc.extraKey)
				if err != nil {
					t.Fatalf("extra key test: unexpected error: %v", err)
				}
				if gotLimited {
					t.Error("extra key should not be limited, but got limited")
				}
			}
		})
	}
}
