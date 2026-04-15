package router

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// =============================================================================
// RedisCheckPointStore Redis 存储（生产用）
// =============================================================================

// RedisCheckPointStore Redis 实现的检查点存储
// 适用于分布式部署，多实例共享中断状态
type RedisCheckPointStore struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// RedisCheckPointStoreConfig Redis 存储配置
type RedisCheckPointStoreConfig struct {
	Addr     string        // Redis 地址
	Password string        // Redis 密码
	DB       int           // Redis DB
	Prefix   string        // Key 前缀
	TTL      time.Duration // 生存时间
}

// NewRedisCheckPointStore 创建 Redis 检查点存储
func NewRedisCheckPointStore(cfg *RedisCheckPointStoreConfig) (*RedisCheckPointStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// 测试连接
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "checkpoint:"
	}

	return &RedisCheckPointStore{
		client: client,
		prefix: prefix,
		ttl:    cfg.TTL,
	}, nil
}

// NewRedisCheckPointStoreWithClient 使用已有 Redis 客户端创建
func NewRedisCheckPointStoreWithClient(client *redis.Client, prefix string, ttl time.Duration) *RedisCheckPointStore {
	if prefix == "" {
		prefix = "checkpoint:"
	}
	return &RedisCheckPointStore{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// key 构建带前缀的 key
func (s *RedisCheckPointStore) key(checkPointID string) string {
	return s.prefix + checkPointID
}

// Set 保存检查点
func (s *RedisCheckPointStore) Set(ctx context.Context, checkPointID string, checkPoint []byte) error {
	key := s.key(checkPointID)

	if s.ttl > 0 {
		return s.client.Set(ctx, key, checkPoint, s.ttl).Err()
	}
	return s.client.SetNX(ctx, key, checkPoint, 0).Err()
}

// SetWithTTL 设置检查点并指定 TTL
func (s *RedisCheckPointStore) SetWithTTL(ctx context.Context, checkPointID string, checkPoint []byte, ttl time.Duration) error {
	key := s.key(checkPointID)
	return s.client.Set(ctx, key, checkPoint, ttl).Err()
}

// Get 获取检查点
func (s *RedisCheckPointStore) Get(ctx context.Context, checkPointID string) ([]byte, bool, error) {
	key := s.key(checkPointID)

	data, err := s.client.Get(ctx, key).Bytes()
	if err == redis.Nil {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	// 返回数据副本
	result := make([]byte, len(data))
	copy(result, data)
	return result, true, nil
}

// Delete 删除检查点
func (s *RedisCheckPointStore) Delete(ctx context.Context, checkPointID string) error {
	key := s.key(checkPointID)
	return s.client.Del(ctx, key).Err()
}

// Exists 检查点是否存在
func (s *RedisCheckPointStore) Exists(ctx context.Context, checkPointID string) (bool, error) {
	key := s.key(checkPointID)
	n, err := s.client.Exists(ctx, key).Result()
	return n > 0, err
}

// TTL 获取检查点剩余 TTL
func (s *RedisCheckPointStore) TTL(ctx context.Context, checkPointID string) (time.Duration, error) {
	key := s.key(checkPointID)
	return s.client.TTL(ctx, key).Result()
}

// Keys 获取所有检查点 ID
func (s *RedisCheckPointStore) Keys(ctx context.Context) ([]string, error) {
	pattern := s.prefix + "*"
	return s.client.Keys(ctx, pattern).Result()
}

// Close 关闭连接
func (s *RedisCheckPointStore) Close() error {
	return s.client.Close()
}

// Client 返回原始 Redis 客户端
func (s *RedisCheckPointStore) Client() *redis.Client {
	return s.client
}

// ListBySession 获取会话下的所有检查点
func (s *RedisCheckPointStore) ListBySession(ctx context.Context, sessionID string) ([]*CheckPoint, error) {
	keys, err := s.Keys(ctx)
	if err != nil {
		return nil, err
	}

	var results []*CheckPoint
	for _, key := range keys {
		// 去掉前缀
		checkPointID := key[len(s.prefix):]
		data, exists, err := s.Get(ctx, checkPointID)
		if err != nil || !exists {
			continue
		}

		var checkPoint CheckPoint
		if err := json.Unmarshal(data, &checkPoint); err != nil {
			continue
		}

		if checkPoint.SessionID == sessionID {
			results = append(results, &checkPoint)
		}
	}
	return results, nil
}

// ListByUser 获取用户下的所有检查点
func (s *RedisCheckPointStore) ListByUser(ctx context.Context, userID string) ([]*CheckPoint, error) {
	keys, err := s.Keys(ctx)
	if err != nil {
		return nil, err
	}

	var results []*CheckPoint
	for _, key := range keys {
		// 去掉前缀
		checkPointID := key[len(s.prefix):]
		data, exists, err := s.Get(ctx, checkPointID)
		if err != nil || !exists {
			continue
		}

		var checkPoint CheckPoint
		if err := json.Unmarshal(data, &checkPoint); err != nil {
			continue
		}

		if checkPoint.UserID == userID {
			results = append(results, &checkPoint)
		}
	}
	return results, nil
}

// CleanupExpired 清理过期的检查点
// Redis 自动过期，这里统计已过期但尚未被清理的数量
func (s *RedisCheckPointStore) CleanupExpired(ctx context.Context) (int, error) {
	// 执行 SCAN 遍历所有 key，检查 TTL
	var count int
	iter := s.client.Scan(ctx, 0, s.prefix+"*", 1000).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		ttl, err := s.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		if ttl < 0 {
			// 已过期
			count++
			// 主动删除
			s.client.Del(ctx, key)
		}
	}
	return count, iter.Err()
}

// UpdateStatus 更新检查点状态
func (s *RedisCheckPointStore) UpdateStatus(ctx context.Context, checkPointID string, status string, errMsg string) error {
	data, exists, err := s.Get(ctx, checkPointID)
	if err != nil || !exists {
		return err
	}

	var checkPoint CheckPoint
	if err := json.Unmarshal(data, &checkPoint); err != nil {
		return err
	}

	checkPoint.Status = status
	checkPoint.Error = errMsg
	checkPoint.UpdatedAt = time.Now()

	newData, err := json.Marshal(checkPoint)
	if err != nil {
		return err
	}

	// 保留原有 TTL
	ttl, err := s.client.TTL(ctx, s.key(checkPointID)).Result()
	if err != nil {
		return err
	}

	return s.client.Set(ctx, s.key(checkPointID), newData, ttl).Err()
}

// IncrementRetry 增加重试次数
func (s *RedisCheckPointStore) IncrementRetry(ctx context.Context, checkPointID string) (int, error) {
	data, exists, err := s.Get(ctx, checkPointID)
	if err != nil || !exists {
		return 0, err
	}

	var checkPoint CheckPoint
	if err := json.Unmarshal(data, &checkPoint); err != nil {
		return 0, err
	}

	checkPoint.RetryCount++
	checkPoint.UpdatedAt = time.Now()

	newData, err := json.Marshal(checkPoint)
	if err != nil {
		return 0, err
	}

	// 保留原有 TTL
	ttl, err := s.client.TTL(ctx, s.key(checkPointID)).Result()
	if err != nil {
		return 0, err
	}

	if err := s.client.Set(ctx, s.key(checkPointID), newData, ttl).Err(); err != nil {
		return 0, err
	}

	return checkPoint.RetryCount, nil
}

// =============================================================================
// CheckPointData 检查点数据结构
// =============================================================================

// CheckPointData 检查点数据
type CheckPointData struct {
	UserID     string          `json:"user_id"`
	SessionID  string          `json:"session_id"`
	State      json.RawMessage `json:"state"`      // Agent 状态
	Messages   json.RawMessage `json:"messages"`   // 消息历史
	ToolCalls  json.RawMessage `json:"tool_calls"` // 待确认工具调用
	Metadata   json.RawMessage `json:"metadata"`   // 元数据
	CreatedAt  time.Time       `json:"created_at"`
	ResumeData json.RawMessage `json:"resume_data"` // 恢复数据
}

// NewCheckPointData 创建检查点数据
func NewCheckPointData(userID, sessionID string) *CheckPointData {
	return &CheckPointData{
		UserID:    userID,
		SessionID: sessionID,
		CreatedAt: time.Now(),
	}
}

// Encode 序列化为字节
func (c *CheckPointData) Encode() ([]byte, error) {
	return json.Marshal(c)
}

// Decode 从字节反序列化
func DecodeCheckPoint(data []byte) (*CheckPointData, error) {
	var cp CheckPointData
	if err := json.Unmarshal(data, &cp); err != nil {
		return nil, err
	}
	return &cp, nil
}
