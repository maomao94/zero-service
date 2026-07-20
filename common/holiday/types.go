package holiday

import (
	"context"
	"time"
)

// Service 定义节假日查询能力，便于服务层按接口依赖。
type Service interface {
	// Lookup 查询指定日期的节假日信息。
	Lookup(t time.Time) DayInfo
	// IsHoliday 判断指定日期是否为节假日。
	IsHoliday(t time.Time) bool
	// IsWorkday 判断指定日期是否为工作日。
	IsWorkday(t time.Time) bool
	// SupportedYears 返回已配置特殊日期的年份列表。
	SupportedYears() []int
	// IsSupportedYear 判断指定年份是否有特殊日期配置。
	IsSupportedYear(year int) bool
	// GetFestival 查询指定年份的单个节假日详情。
	GetFestival(year int, name string) (FestivalInfo, bool)
	// ListFestivals 按年份区间和可选节日名称列出节假日详情。
	ListFestivals(ctx context.Context, req ListFestivalsReq) ([]FestivalInfo, error)
	// GetYearSummary 查询指定年份的特殊日期汇总。
	GetYearSummary(year int) (YearSummaryInfo, bool)
}

// Store 定义可编辑的节假日数据源能力。
type Store interface {
	Source
	// Save 保存单日特殊日期配置，存在则更新，不存在则创建。
	Save(ctx context.Context, item StoredEntry) error
	// SetEnabled 设置单日特殊日期配置是否启用。
	SetEnabled(ctx context.Context, date string, enabled bool) error
	// List 按年份列出特殊日期源配置。
	List(ctx context.Context, year int, includeDisabled bool) ([]StoredEntry, error)
}

// StoredEntry 表示数据源中的单日特殊日期配置。
type StoredEntry struct {
	// Date 是配置日期，格式 yyyy-MM-dd。
	Date string
	// Entry 是特殊日期内容。
	Entry Entry
	// Enabled 表示配置是否启用。
	Enabled bool
}

// DayType 表示日期最终业务类型。
type DayType string

const (
	// DayTypeHoliday 表示非工作日。
	DayTypeHoliday DayType = "holiday"
	// DayTypeWorkday 表示工作日。
	DayTypeWorkday DayType = "workday"
)

// DayKind 表示日期被判定为节假日或工作日的原因。
type DayKind string

const (
	// DayKindStatutoryHoliday 表示法定节假日或调休假日。
	DayKindStatutoryHoliday DayKind = "statutory_holiday"
	// DayKindWeekend 表示未被节假日数据覆盖的普通周末。
	DayKindWeekend DayKind = "weekend"
	// DayKindMakeupWorkday 表示调休补班日。
	DayKindMakeupWorkday DayKind = "makeup_workday"
	// DayKindNormalWorkday 表示未被节假日数据覆盖的普通工作日。
	DayKindNormalWorkday DayKind = "normal_workday"
)

// Entry 表示配置中的特殊日期记录。
type Entry struct {
	// Name 是节假日分组名称，如“国庆节”“中秋国庆”。
	Name string `json:"name"`
	// Type 是日期最终业务类型。
	Type DayType `json:"type"`
	// IsFestivalDay 表示是否为节日当天，如春节正月初一、国庆节 10 月 1 日。
	IsFestivalDay bool `json:"isFestivalDay"`
	// Note 是日期说明。
	Note string `json:"note"`
}

// DayInfo 表示日期查询结果。
type DayInfo struct {
	// Date 是查询日期，格式 yyyy-MM-dd。
	Date string
	// Name 是节假日分组名称，普通周末和普通工作日为空。
	Name string
	// Type 是日期最终业务类型。
	Type DayType
	// Kind 是日期被判定为节假日或工作日的原因。
	Kind DayKind
	// Note 是日期说明。
	Note string
	// IsFestivalDay 表示是否为节日当天。
	IsFestivalDay bool
	// IsHoliday 表示是否为节假日。
	IsHoliday bool
	// IsWorkday 表示是否为工作日。
	IsWorkday bool
}

// ListFestivalsReq 表示按年份区间和节日名称查询节假日详情的请求。
type ListFestivalsReq struct {
	// StartYear 是起始年份，为 0 时不限制起始年份。
	StartYear int
	// EndYear 是结束年份，为 0 时不限制结束年份。
	EndYear int
	// Name 是节日名称，空表示不过滤。
	Name string
}

// FestivalInfo 表示某一年单个节假日的特殊日期集合。
type FestivalInfo struct {
	// Year 是年份。
	Year int
	// Name 是节日名称。
	Name string
	// StartDate 是假日开始日期，格式 yyyy-MM-dd。
	StartDate string
	// EndDate 是假日结束日期，格式 yyyy-MM-dd。
	EndDate string
	// HolidayDays 是假日日期列表。
	HolidayDays []string
	// MakeupWorkdays 是调休补班日期列表。
	MakeupWorkdays []string
	// FestivalDays 是节日当天日期列表。
	FestivalDays []string
}

// YearSummaryInfo 表示某一年全部特殊日期的汇总。
type YearSummaryInfo struct {
	// Year 是年份。
	Year int
	// HolidayDays 是假日日期列表。
	HolidayDays []string
	// MakeupWorkdays 是调休补班日期列表。
	MakeupWorkdays []string
	// FestivalDays 是节日当天日期列表。
	FestivalDays []string
	// Names 是按首次特殊日期出现时间排序的节日名称列表。
	Names []string
}

func dateKey(t time.Time, loc *time.Location) string {
	return t.In(loc).Format(time.DateOnly)
}

func isWeekend(t time.Time) bool {
	weekday := t.Weekday()
	return weekday == time.Saturday || weekday == time.Sunday
}
