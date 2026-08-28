package tool

import (
	"math/rand"
	"time"
)

// JitterDelay 返回 [base, max] 范围内的随机延迟（jitter）。
// 用于避免同时重试风暴：多个节点同时失败时，随机偏移错开重试时间。
func JitterDelay(base, max time.Duration) time.Duration {
	if base >= max {
		return base
	}
	return base + time.Duration(rand.Int63n(int64(max-base)))
}
