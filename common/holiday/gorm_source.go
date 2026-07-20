package holiday

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"

	"zero-service/common/gormx"
)

const defaultGormTable = "sys_holiday"

// GormHoliday 是 GormSource 默认读取和初始化的表结构。
type GormHoliday struct {
	Date          string  `gorm:"column:date;primaryKey;size:10;comment:日期，格式 YYYY-MM-DD"`
	Name          string  `gorm:"column:name;size:32;not null;index;comment:节假日分组名称"`
	Type          DayType `gorm:"column:type;size:32;not null;comment:日期类型，holiday=假日，workday=工作日"`
	IsFestivalDay bool    `gorm:"column:is_festival_day;not null;comment:是否节日当天"`
	Note          string  `gorm:"column:note;size:128;comment:日期说明"`
	Enabled       bool    `gorm:"column:enabled;not null;comment:是否启用"`
}

// TableName 返回 GormHoliday 的默认表名。
func (GormHoliday) TableName() string {
	return defaultGormTable
}

type gormOptions struct {
	autoMigrate  bool
	initEmbedded bool
}

// GormOption 配置 GormSource。
type GormOption func(*gormOptions)

// WithGormAutoMigrate 控制 GormSource 加载前是否自动建表或更新表结构。
func WithGormAutoMigrate(enabled bool) GormOption {
	return func(o *gormOptions) {
		o.autoMigrate = enabled
	}
}

// WithGormInitEmbedded 控制空表是否用内嵌 JSON 数据初始化。
func WithGormInitEmbedded(enabled bool) GormOption {
	return func(o *gormOptions) {
		o.initEmbedded = enabled
	}
}

// GormSource 从 gormx.DB 加载节假日数据。
type GormSource struct {
	db           *gormx.DB
	autoMigrate  bool
	initEmbedded bool
	prepareOnce  sync.Once
	prepareErr   error
}

var _ Store = (*GormSource)(nil)

// NewGormSource 创建使用 gormx.DB 的数据源。
func NewGormSource(db *gormx.DB, opts ...GormOption) *GormSource {
	o := gormOptions{autoMigrate: true, initEmbedded: true}
	for _, opt := range opts {
		opt(&o)
	}
	return &GormSource{db: db, autoMigrate: o.autoMigrate, initEmbedded: o.initEmbedded}
}

// Load 从配置表加载 enabled=true 的节假日数据。
func (s *GormSource) Load(ctx context.Context) (map[string]Entry, error) {
	if s.db == nil {
		return nil, errors.New("holiday gorm source db is nil")
	}
	if err := s.prepare(ctx); err != nil {
		return nil, err
	}
	return s.loadRows(ctx)
}

func (s *GormSource) migrateSchema(ctx context.Context) error {
	if s.db == nil {
		return errors.New("holiday gorm source db is nil")
	}
	return s.db.WithContext(ctx).AutoMigrate(&GormHoliday{})
}

// Save 保存单日特殊日期配置，存在则更新，不存在则创建。
func (s *GormSource) Save(ctx context.Context, item StoredEntry) error {
	if s.db == nil {
		return errors.New("holiday gorm source db is nil")
	}
	item.Entry = normalizeEntry(item.Entry)
	if err := validateStoredEntry(item); err != nil {
		return err
	}
	if err := s.prepare(ctx); err != nil {
		return err
	}
	if err := s.db.WithContext(ctx).Save(gormHolidayFromStoredEntry(item)).Error; err != nil {
		return err
	}
	return nil
}

// SetEnabled 设置单日特殊日期配置是否启用。
func (s *GormSource) SetEnabled(ctx context.Context, date string, enabled bool) error {
	if s.db == nil {
		return errors.New("holiday gorm source db is nil")
	}
	if err := validateDate(date); err != nil {
		return err
	}
	if err := s.prepare(ctx); err != nil {
		return err
	}
	var row GormHoliday
	db := s.db.WithContext(ctx)
	if err := db.Where("date = ?", date).First(&row).Error; err != nil {
		return err
	}
	row.Enabled = enabled
	return db.Save(&row).Error
}

// List 按年份列出特殊日期源配置。
func (s *GormSource) List(ctx context.Context, year int, includeDisabled bool) ([]StoredEntry, error) {
	if s.db == nil {
		return nil, errors.New("holiday gorm source db is nil")
	}
	if year < 1 || year > 9999 {
		return nil, errors.New("holiday year is invalid")
	}
	if err := s.prepare(ctx); err != nil {
		return nil, err
	}
	var rows []GormHoliday
	startDate, endDate := yearBounds(year)
	query := s.db.WithContext(ctx).Model(&GormHoliday{}).Where("date >= ? AND date <= ?", startDate, endDate)
	if !includeDisabled {
		query = query.Where("enabled = ?", true)
	}
	if err := query.Order("date ASC").Find(&rows).Error; err != nil {
		return nil, err
	}
	entries := make([]StoredEntry, 0, len(rows))
	for _, row := range rows {
		item := storedEntryFromGormHoliday(row)
		if err := validateStoredEntry(item); err != nil {
			return nil, err
		}
		entries = append(entries, item)
	}
	return entries, nil
}

func (s *GormSource) initializeEmbedded(ctx context.Context) error {
	if s.db == nil {
		return errors.New("holiday gorm source db is nil")
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&GormHoliday{}).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}
	entries, err := NewEmbeddedSource().Load(ctx)
	if err != nil {
		return err
	}
	rows := make([]GormHoliday, 0, len(entries))
	dates := make([]string, 0, len(entries))
	for date := range entries {
		dates = append(dates, date)
	}
	sort.Strings(dates)
	for _, date := range dates {
		entry := entries[date]
		rows = append(rows, GormHoliday{Date: date, Name: entry.Name, Type: entry.Type, IsFestivalDay: entry.IsFestivalDay, Note: entry.Note, Enabled: true})
	}
	if len(rows) == 0 {
		return nil
	}
	return s.db.WithContext(ctx).CreateInBatches(rows, 100).Error
}

func (s *GormSource) loadRows(ctx context.Context) (map[string]Entry, error) {
	var rows []GormHoliday
	if err := s.db.WithContext(ctx).Model(&GormHoliday{}).Where("enabled = ?", true).Find(&rows).Error; err != nil {
		return nil, err
	}
	entries := make(map[string]Entry, len(rows))
	for _, row := range rows {
		item := storedEntryFromGormHoliday(row)
		if err := validateStoredEntry(item); err != nil {
			return nil, err
		}
		entries[item.Date] = item.Entry
	}
	return entries, nil
}

func gormHolidayFromStoredEntry(item StoredEntry) *GormHoliday {
	return &GormHoliday{
		Date:          item.Date,
		Name:          item.Entry.Name,
		Type:          item.Entry.Type,
		IsFestivalDay: item.Entry.IsFestivalDay,
		Note:          item.Entry.Note,
		Enabled:       item.Enabled,
	}
}

func storedEntryFromGormHoliday(row GormHoliday) StoredEntry {
	return StoredEntry{
		Date: row.Date,
		Entry: normalizeEntry(Entry{
			Name:          row.Name,
			Type:          row.Type,
			IsFestivalDay: row.IsFestivalDay,
			Note:          row.Note,
		}),
		Enabled: row.Enabled,
	}
}

func validateStoredEntry(item StoredEntry) error {
	return validateEntry(item.Date, item.Entry)
}

func (s *GormSource) prepare(ctx context.Context) error {
	s.prepareOnce.Do(func() {
		if s.autoMigrate {
			if err := s.migrateSchema(ctx); err != nil {
				s.prepareErr = err
				return
			}
		}
		if s.initEmbedded {
			s.prepareErr = s.initializeEmbedded(ctx)
		}
	})
	return s.prepareErr
}

func yearBounds(year int) (string, string) {
	prefix := fmt.Sprintf("%04d", year)
	return prefix + "-01-01", prefix + "-12-31"
}
