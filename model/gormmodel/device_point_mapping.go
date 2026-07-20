package gormmodel

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"zero-service/common/gormx"
	"zero-service/common/iec104/types"
	"zero-service/model"

	"github.com/zeromicro/go-zero/core/collection"
	"gorm.io/gorm"
)

const devicePointMappingCacheExpiration = 24 * time.Hour

// GormDevicePointMapping maps the shared device_point_mapping table.
type GormDevicePointMapping struct {
	gormx.LegacyStringBaseModel // id / create_time / update_time / delete_time / is_deleted

	CreateUser string `gorm:"column:create_user;size:64;default:'';comment:创建人" json:"create_user"`
	UpdateUser string `gorm:"column:update_user;size:64;default:'';comment:更新人" json:"update_user"`
	DeptCode   string `gorm:"column:dept_code;size:64;default:'';comment:机构code" json:"dept_code"`

	TagStation      string         `gorm:"column:tag_station;size:64;not null;default:'';uniqueIndex:uk_device_point_mapping_tag_station_coa_ioa,priority:1;comment:与 TDengine tag_station 对应" json:"tag_station"`
	Coa             int            `gorm:"column:coa;not null;default:0;uniqueIndex:uk_device_point_mapping_tag_station_coa_ioa,priority:2;comment:与 TDengine coa 对应" json:"coa"`
	Ioa             int            `gorm:"column:ioa;not null;default:0;uniqueIndex:uk_device_point_mapping_tag_station_coa_ioa,priority:3;comment:与 TDengine ioa 对应" json:"ioa"`
	DeviceId        string         `gorm:"column:device_id;size:64;not null;default:'';comment:设备编号/ID" json:"device_id"`
	DeviceName      string         `gorm:"column:device_name;size:128;not null;default:'';comment:设备名称" json:"device_name"`
	TdTableType     string         `gorm:"column:td_table_type;size:255;default:'';comment:TDengine 表类型（遥信表/遥测表等，逗号分隔）" json:"td_table_type"`
	EnablePush      int            `gorm:"column:enable_push;not null;default:1;comment:是否允许caller服务推送数据：0-不允许，1-允许" json:"enable_push"`
	EnableRawInsert int            `gorm:"column:enable_raw_insert;not null;default:1;comment:是否允许插入 raw 原生数据：0-否，1-是" json:"enable_raw_insert"`
	Description     sql.NullString `gorm:"column:description;size:256;default:'';comment:备注信息" json:"description"`
	Ext1            sql.NullString `gorm:"column:ext_1;size:64;default:'';comment:扩展字段1，如：alarm, normal, control等，用于主题拆分" json:"ext_1"`
	Ext2            sql.NullString `gorm:"column:ext_2;size:64;default:'';comment:扩展字段2" json:"ext_2"`
	Ext3            sql.NullString `gorm:"column:ext_3;size:64;default:'';comment:扩展字段3" json:"ext_3"`
	Ext4            sql.NullString `gorm:"column:ext_4;size:64;default:'';comment:扩展字段4" json:"ext_4"`
	Ext5            sql.NullString `gorm:"column:ext_5;size:64;default:'';comment:扩展字段5" json:"ext_5"`
}

func (GormDevicePointMapping) TableName() string { return "device_point_mapping" }

type DevicePointMappingFilter struct {
	TagStation string
	Coa        int64
	DeviceId   string
}

type DevicePointMappingStore struct {
	db    *gormx.DB
	cache *collection.Cache
}

func NewDevicePointMappingStore(db *gormx.DB) *DevicePointMappingStore {
	pmc, _ := collection.NewCache(devicePointMappingCacheExpiration, collection.WithName("pm-cache"))
	return &DevicePointMappingStore{db: db, cache: pmc}
}

func (s *DevicePointMappingStore) FindOne(ctx context.Context, id int64) (*GormDevicePointMapping, error) {
	var mapping GormDevicePointMapping
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&mapping).Error
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

func (s *DevicePointMappingStore) FindOneByTagStationCoaIoa(ctx context.Context, tagStation string, coa, ioa int64) (*GormDevicePointMapping, error) {
	var mapping GormDevicePointMapping
	err := s.db.WithContext(ctx).
		Where("tag_station = ? AND coa = ? AND ioa = ?", tagStation, coa, ioa).
		First(&mapping).Error
	if err != nil {
		return nil, err
	}
	return &mapping, nil
}

func (s *DevicePointMappingStore) FindPage(ctx context.Context, filter DevicePointMappingFilter, page, pageSize int64) ([]GormDevicePointMapping, int64, error) {
	db := s.db.WithContext(ctx).Model(&GormDevicePointMapping{})
	if filter.TagStation != "" {
		db = db.Where("tag_station = ?", filter.TagStation)
	}
	if filter.Coa > 0 {
		db = db.Where("coa = ?", filter.Coa)
	}
	if filter.DeviceId != "" {
		db = db.Where("device_id = ?", filter.DeviceId)
	}

	var mappings []GormDevicePointMapping
	result, err := gormx.QueryPage(db.Order("id DESC"), int(page), int(pageSize), &mappings)
	if err != nil {
		return nil, 0, err
	}
	return mappings, result.Total, nil
}

func (s *DevicePointMappingStore) RemoveCache(ctx context.Context, keys ...string) error {
	for _, key := range keys {
		s.cache.Del(key)
	}
	return nil
}

func (s *DevicePointMappingStore) GetCache(ctx context.Context, key string) (any, bool) {
	return s.cache.Get(key)
}

func (s *DevicePointMappingStore) GenerateCacheKey(tagStation string, coa, ioa int64) string {
	return fmt.Sprintf("pm:%s:%d:%d", tagStation, coa, ioa)
}

func (s *DevicePointMappingStore) FindCacheOneByTagStationCoaIoa(ctx context.Context, tagStation string, coa, ioa int64) (*GormDevicePointMapping, bool, error) {
	key := s.GenerateCacheKey(tagStation, coa, ioa)
	val, err := s.cache.Take(key, func() (any, error) {
		v, err := s.FindOneByTagStationCoaIoa(ctx, tagStation, coa, ioa)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return model.CacheEntry[GormDevicePointMapping]{}, nil
			}
			return nil, err
		}
		if v == nil {
			return model.CacheEntry[GormDevicePointMapping]{}, nil
		}
		return model.CacheEntry[GormDevicePointMapping]{Data: *v, Valid: true}, nil
	})
	if err != nil {
		return nil, false, err
	}
	entry, ok := val.(model.CacheEntry[GormDevicePointMapping])
	if !ok {
		return nil, false, errors.New("cache entry type assertion failed")
	}
	if !entry.Valid {
		return nil, false, nil
	}
	cacheData := entry.Data
	return &cacheData, true, nil
}

func (m *GormDevicePointMapping) ToPointMapping() *types.PointMapping {
	if m == nil {
		return nil
	}
	return &types.PointMapping{
		DeviceId:    m.DeviceId,
		DeviceName:  m.DeviceName,
		TdTableType: m.TdTableType,
		Ext1:        m.Ext1.String,
		Ext2:        m.Ext2.String,
		Ext3:        m.Ext3.String,
		Ext4:        m.Ext4.String,
		Ext5:        m.Ext5.String,
	}
}
