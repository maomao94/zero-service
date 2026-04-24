package triggerutil

import "time"

// RedisLockExpireSeconds 与 ExecuteCallback / CallbackPlanExecItem 使用的「请求超时(ms) + 120s」对齐，供 redis.NewRedisLock 之后 SetExpire 使用。
//
// go-zero RedisLock 未 SetExpire 时内部 seconds 为 0，加锁 TTL 仅约 500ms（见 go-zero lockscript 与 tolerance），
// 下游 RPC 略慢即过期，defer Release 会得到 ok=false，并打出「锁已过期或不存在」类日志。
func RedisLockExpireSeconds(requestTimeoutMs int64) int {
	ttl := time.Duration(requestTimeoutMs)*time.Millisecond + 120*time.Second
	s := int(ttl / time.Second)
	if s < 1 {
		return 1
	}
	return s
}
