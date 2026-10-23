package gormmodel

import (
	"context"
	"time"

	"zero-service/common/gormx"
)

// OpenAppSecret 开放平台应用密钥，仅保存 sha256 十六进制摘要。
type OpenAppSecret struct {
	AppId      string    `gorm:"column:app_id;type:varchar(32);index:idx_open_app_secret_app_status;comment:应用ID"`
	SecretHash string    `gorm:"column:secret_hash;type:char(64);comment:sha256(secret)"`
	Status     int       `gorm:"column:status;default:1;index:idx_open_app_secret_app_status;comment:状态 1-生效 2-待失效 3-已失效"`
	ExpireAt   time.Time `gorm:"column:expire_at;comment:失效时间"`
	gormx.TimeMixin
	gormx.IDModel
}

func (OpenAppSecret) TableName() string { return "open_app_secret" }

type OpenAppSecretStore struct {
	db *gormx.DB
}

func NewOpenAppSecretStore(db *gormx.DB) *OpenAppSecretStore {
	return &OpenAppSecretStore{db: db}
}

func (s *OpenAppSecretStore) Create(ctx context.Context, secret *OpenAppSecret) error {
	return s.db.WithContext(ctx).Create(secret).Error
}

// ListUsable 返回当前可用于鉴权的密钥：
// status=1 生效中，status=2 且尚未到 expire_at（重置后的旧密钥保留窗口）。
func (s *OpenAppSecretStore) ListUsable(ctx context.Context, now time.Time) ([]OpenAppSecret, error) {
	var secrets []OpenAppSecret
	err := s.db.WithContext(ctx).
		Where("status = ? OR (status = ? AND expire_at > ?)", 1, 2, now).
		Order("id ASC").
		Find(&secrets).Error
	if err != nil {
		return nil, err
	}
	return secrets, nil
}

// Replace 将应用现有密钥标记为 24 小时后失效，并写入新密钥。
func (s *OpenAppSecretStore) Replace(ctx context.Context, appID, newHash string, expireAt time.Time) error {
	return s.db.WithContext(ctx).Transact(func(tx *gormx.DB) error {
		if err := tx.Model(&OpenAppSecret{}).
			Where("app_id = ? AND status IN ?", appID, []int{1, 2}).
			Updates(map[string]any{"status": 2, "expire_at": expireAt}).Error; err != nil {
			return err
		}
		return tx.Create(&OpenAppSecret{
			AppId:      appID,
			SecretHash: newHash,
			Status:     1,
		}).Error
	})
}
