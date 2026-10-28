package gormmodel

import (
	"context"
	"time"

	"zero-service/common/gormx"
)

// OpenApp 开放平台应用。
type OpenApp struct {
	AppId        string    `gorm:"column:app_id;type:varchar(40);primaryKey;comment:应用ID"`
	Name         string    `gorm:"column:name;type:varchar(64);comment:应用名称"`
	Remark       string    `gorm:"column:remark;type:varchar(255);default:'';comment:备注"`
	Status       int       `gorm:"column:status;default:1;comment:状态 1-启用 2-停用"`
	SignRequired bool      `gorm:"column:sign_required;default:false;comment:是否要求请求签名"`
	LastUsedAt   time.Time `gorm:"column:last_used_at;comment:最近调用时间"`
	gormx.TimeMixin
}

func (OpenApp) TableName() string { return "open_app" }

type OpenAppStore struct {
	db *gormx.DB
}

func NewOpenAppStore(db *gormx.DB) *OpenAppStore {
	return &OpenAppStore{db: db}
}

func (s *OpenAppStore) Create(ctx context.Context, app *OpenApp) error {
	return s.db.WithContext(ctx).Create(app).Error
}

func (s *OpenAppStore) FindByAppId(ctx context.Context, appID string) (*OpenApp, error) {
	var app OpenApp
	if err := s.db.WithContext(ctx).Where("app_id = ?", appID).First(&app).Error; err != nil {
		return nil, err
	}
	return &app, nil
}

func (s *OpenAppStore) ListActive(ctx context.Context) ([]OpenApp, error) {
	var apps []OpenApp
	if err := s.db.WithContext(ctx).Where("status = ?", 1).Order("app_id ASC").Find(&apps).Error; err != nil {
		return nil, err
	}
	return apps, nil
}

func (s *OpenAppStore) FindPage(ctx context.Context, keyword string, status int, page, pageSize int64) ([]OpenApp, int64, error) {
	db := s.db.WithContext(ctx).Model(&OpenApp{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("app_id LIKE ? OR name LIKE ?", like, like)
	}
	if status > 0 {
		db = db.Where("status = ?", status)
	}

	var apps []OpenApp
	result, err := gormx.QueryPage(db.Order("created_at DESC, app_id ASC"), page, pageSize, &apps)
	if err != nil {
		return nil, 0, err
	}
	return apps, result.Total, nil
}

func (s *OpenAppStore) UpdateStatus(ctx context.Context, appID string, status int) (int64, error) {
	result := s.db.WithContext(ctx).Model(&OpenApp{}).
		Where("app_id = ?", appID).
		Updates(map[string]any{"status": status})
	return result.RowsAffected, result.Error
}

func (s *OpenAppStore) UpdateSignRequired(ctx context.Context, appID string, required bool) (int64, error) {
	result := s.db.WithContext(ctx).Model(&OpenApp{}).
		Where("app_id = ?", appID).
		Update("sign_required", required)
	return result.RowsAffected, result.Error
}

// TouchLastUsedAt 只更新 last_used_at，不刷新 updated_at。
func (s *OpenAppStore) TouchLastUsedAt(ctx context.Context, appID string, at time.Time) error {
	return s.db.WithContext(ctx).Model(&OpenApp{}).
		Where("app_id = ?", appID).
		UpdateColumn("last_used_at", at).Error
}
