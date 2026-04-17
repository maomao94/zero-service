package checkpoint

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// JSONLStore 把每个 key 的最新快照单独存成一个 .jsonl 文件。
//
// 实现取舍：
//   - checkpoint 的访问模式是 key-value 覆盖写，不需要追加顺序，所以用单文件覆盖写入
//     而不是像 memory 消息那样按时间追加，读写都更简单，损坏恢复也容易。
//   - 文件名用 key 的 sha1，避免非法字符与超长名。
type JSONLStore struct {
	baseDir string
	mu      sync.Mutex // 保证并发安全 + 原子替换
}

// NewJSONLStore 创建 JSONL 存储。会自动创建 baseDir。
func NewJSONLStore(baseDir string) (*JSONLStore, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, fmt.Errorf("checkpoint.jsonl: mkdir %s: %w", baseDir, err)
	}
	return &JSONLStore{baseDir: baseDir}, nil
}

type jsonlRecord struct {
	Key   string `json:"key"`
	Value []byte `json:"value"` // []byte 在 JSON 里自动 base64，天然安全
}

// Set 写入快照（原子覆盖：临时文件 + rename）。
func (s *JSONLStore) Set(_ context.Context, key string, value []byte) error {
	if key == "" {
		return fmt.Errorf("checkpoint.jsonl: empty key")
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	path := s.keyPath(key)
	tmp := path + ".tmp"

	data, err := json.Marshal(jsonlRecord{Key: key, Value: value})
	if err != nil {
		return fmt.Errorf("checkpoint.jsonl: marshal: %w", err)
	}
	data = append(data, '\n')

	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return fmt.Errorf("checkpoint.jsonl: write tmp: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		return fmt.Errorf("checkpoint.jsonl: rename: %w", err)
	}
	return nil
}

// Get 读取快照。不存在时返回 (nil,false,nil)。
func (s *JSONLStore) Get(_ context.Context, key string) ([]byte, bool, error) {
	if key == "" {
		return nil, false, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.keyPath(key))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("checkpoint.jsonl: read: %w", err)
	}

	var rec jsonlRecord
	if err := json.Unmarshal(data, &rec); err != nil {
		return nil, false, fmt.Errorf("checkpoint.jsonl: unmarshal: %w", err)
	}
	return rec.Value, true, nil
}

// Delete 删除快照。
func (s *JSONLStore) Delete(_ context.Context, key string) error {
	if key == "" {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := os.Remove(s.keyPath(key)); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("checkpoint.jsonl: remove: %w", err)
	}
	return nil
}

// Close 无资源需要释放。
func (s *JSONLStore) Close() error { return nil }

func (s *JSONLStore) keyPath(key string) string {
	sum := sha1.Sum([]byte(key))
	return filepath.Join(s.baseDir, hex.EncodeToString(sum[:])+".jsonl")
}

var _ Store = (*JSONLStore)(nil)
