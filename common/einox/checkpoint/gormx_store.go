package checkpoint

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"zero-service/common/gormx"
)

// GormxStore 基于 common/gormx 的关系型数据库存储实现。
//
// 适合多实例/分布式部署场景：任何实例触发的 interrupt 快照都能被其它实例 Resume。
type GormxStore struct {
	db *gormx.DB
}

// NewGormxStore 创建 gormx 存储，调用方传入已建立连接的 *gormx.DB。
func NewGormxStore(db *gormx.DB) (*GormxStore, error) {
	if db == nil {
		return nil, fmt.Errorf("checkpoint.gormx: db is nil")
	}
	s := &GormxStore{db: db}
	if err := s.db.DB.AutoMigrate(&CheckpointRecord{}); err != nil {
		return nil, fmt.Errorf("checkpoint.gormx: auto migrate: %w", err)
	}
	return s, nil
}

// Set 写入快照。使用 upsert 保证幂等。
func (s *GormxStore) Set(ctx context.Context, key string, value []byte) error {
	if key == "" {
		return fmt.Errorf("checkpoint.gormx: empty key")
	}
	rec := CheckpointRecord{
		Key:       key,
		Value:     append([]byte(nil), value...),
		UpdatedAt: time.Now(),
	}
	return s.db.DB.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).
		Create(&rec).Error
}

// Get 读取快照。
func (s *GormxStore) Get(ctx context.Context, key string) ([]byte, bool, error) {
	if key == "" {
		return nil, false, nil
	}
	var rec CheckpointRecord
	err := s.db.DB.WithContext(ctx).Where("`key` = ?", key).Take(&rec).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("checkpoint.gormx: query: %w", err)
	}
	return append([]byte(nil), rec.Value...), true, nil
}

// Delete 删除快照。
func (s *GormxStore) Delete(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	return s.db.DB.WithContext(ctx).Where("`key` = ?", key).Delete(&CheckpointRecord{}).Error
}

// Close 不主动关闭底层连接（由调用方统一管理）。
func (s *GormxStore) Close() error { return nil }

// =============================================================================
// 表模型
// =============================================================================

// CheckpointRecord 快照记录。Key 为主键，Value 存储 adk 序列化后的二进制快照。
type CheckpointRecord struct {
	Key       string    `gorm:"column:key;type:varchar(255);primaryKey"`
	Value     []byte    `gorm:"column:value;type:longblob"`
	UpdatedAt time.Time `gorm:"column:updated_at"`
}

// TableName 自定义表名。
func (CheckpointRecord) TableName() string {
	return "einox_agent_checkpoint"
}

var _ Store = (*GormxStore)(nil)
