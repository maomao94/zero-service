package holiday

import (
	"context"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
)

const (
	defaultLocationName = "Asia/Shanghai"
	noteWeekend         = "周末"
	noteNormalWorkday   = "普通工作日"
)

type options struct {
	source   Source
	location *time.Location
}

// Option 配置 Calendar。
type Option func(*options)

// WithSource 设置特殊日期记录的数据源。
func WithSource(source Source) Option {
	return func(o *options) {
		o.source = source
	}
}

// WithLocation 设置把输入时间转换成日期时使用的时区。
func WithLocation(location *time.Location) Option {
	return func(o *options) {
		o.location = location
	}
}

// Calendar 基于一份固定特殊日期快照提供节假日查询。
type Calendar struct {
	source   Source
	location *time.Location
	entries  map[string]Entry
	years    []int
	mu       sync.RWMutex
}

var _ Service = (*Calendar)(nil)

// NewCalendar 创建 Calendar，默认使用内嵌节假日数据和 Asia/Shanghai 时区。
func NewCalendar(ctx context.Context, opts ...Option) (*Calendar, error) {
	location, err := time.LoadLocation(defaultLocationName)
	if err != nil {
		return nil, err
	}
	o := options{
		source:   NewEmbeddedSource(),
		location: location,
	}
	for _, opt := range opts {
		opt(&o)
	}
	if o.source == nil {
		o.source = NewEmbeddedSource()
	}
	if o.location == nil {
		o.location = location
	}
	entries, err := o.source.Load(ctx)
	if err != nil {
		return nil, err
	}
	return &Calendar{
		source:   o.source,
		location: o.location,
		entries:  entries,
		years:    yearsFromEntries(entries),
	}, nil
}

// MustNewCalendar 创建 Calendar，失败时 panic，并按默认间隔在后台自动刷新。
func MustNewCalendar(opts ...Option) *Calendar {
	ctx := context.Background()
	cal, err := NewCalendar(ctx, opts...)
	logx.Must(err)
	cal.startAutoReloadLoop(ctx, time.Minute)
	return cal
}

// Lookup 先查特殊日期配置，再按周末和工作日兜底判定日期类型。
func (c *Calendar) Lookup(t time.Time) DayInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	date := t.In(c.location)
	key := date.Format(time.DateOnly)
	if entry, ok := c.entries[key]; ok {
		switch entry.Type {
		case DayTypeHoliday:
			return DayInfo{Date: key, Name: entry.Name, Type: DayTypeHoliday, Kind: DayKindStatutoryHoliday, Note: entry.Note, IsFestivalDay: entry.IsFestivalDay, IsHoliday: true}
		case DayTypeWorkday:
			return DayInfo{Date: key, Name: entry.Name, Type: DayTypeWorkday, Kind: DayKindMakeupWorkday, Note: entry.Note, IsFestivalDay: entry.IsFestivalDay, IsWorkday: true}
		}
	}
	if isWeekend(date) {
		return DayInfo{Date: key, Type: DayTypeHoliday, Kind: DayKindWeekend, Note: noteWeekend, IsHoliday: true}
	}
	return DayInfo{Date: key, Type: DayTypeWorkday, Kind: DayKindNormalWorkday, Note: noteNormalWorkday, IsWorkday: true}
}

// IsHoliday 判断日期是否为节假日。
func (c *Calendar) IsHoliday(t time.Time) bool {
	return c.Lookup(t).IsHoliday
}

// IsWorkday 判断日期是否为工作日。
func (c *Calendar) IsWorkday(t time.Time) bool {
	return c.Lookup(t).IsWorkday
}

// SupportedYears 返回已配置特殊日期的年份列表。
func (c *Calendar) SupportedYears() []int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	years := make([]int, len(c.years))
	copy(years, c.years)
	return years
}

// IsSupportedYear 判断指定年份是否有特殊日期配置。
func (c *Calendar) IsSupportedYear(year int) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	i := sort.SearchInts(c.years, year)
	return i < len(c.years) && c.years[i] == year
}

// GetFestival 查询指定年份的单个节假日详情。
func (c *Calendar) GetFestival(year int, name string) (FestivalInfo, bool) {
	if name == "" {
		return FestivalInfo{}, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return festivalFromEntries(c.entries, year, name)
}

// ListFestivals 按年份区间和可选节日名称列出节假日详情。
func (c *Calendar) ListFestivals(ctx context.Context, req ListFestivalsReq) ([]FestivalInfo, error) {
	if err := validateListFestivalsReq(req); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return festivalsFromEntries(c.entries, req), nil
}

// GetYearSummary 查询指定年份的特殊日期汇总。
func (c *Calendar) GetYearSummary(year int) (YearSummaryInfo, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return yearSummaryFromEntries(c.entries, year)
}

// Reload 重新从数据源加载节假日快照。
func (c *Calendar) Reload(ctx context.Context) error {
	if c.source == nil {
		return nil
	}
	entries, err := c.source.Load(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.entries = entries
	c.years = yearsFromEntries(entries)
	c.mu.Unlock()
	return nil
}

// startAutoReloadLoop 周期性从数据源重新加载 Calendar。
func (c *Calendar) startAutoReloadLoop(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.Reload(context.Background()); err != nil {
					logx.Errorf("刷新节假日本地缓存失败: %v", err)
				}
			}
		}
	}()
}

var (
	defaultOnce sync.Once
	defaultCal  *Calendar
	defaultErr  error
)

// Default 返回使用内嵌节假日数据的默认 Calendar。
func Default() *Calendar {
	defaultOnce.Do(func() {
		defaultCal, defaultErr = NewCalendar(context.Background())
	})
	if defaultErr != nil {
		panic(defaultErr)
	}
	return defaultCal
}

// Lookup 使用默认 Calendar 判定日期类型。
func Lookup(t time.Time) DayInfo {
	return Default().Lookup(t)
}

// IsHoliday 使用默认 Calendar 判断日期是否为节假日。
func IsHoliday(t time.Time) bool {
	return Default().IsHoliday(t)
}

// IsWorkday 使用默认 Calendar 判断日期是否为工作日。
func IsWorkday(t time.Time) bool {
	return Default().IsWorkday(t)
}

// SupportedYears 返回默认 Calendar 已配置特殊日期的年份列表。
func SupportedYears() []int {
	return Default().SupportedYears()
}

// IsSupportedYear 判断默认 Calendar 是否有指定年份的特殊日期配置。
func IsSupportedYear(year int) bool {
	return Default().IsSupportedYear(year)
}

// GetFestival 使用默认 Calendar 查询指定年份的单个节假日详情。
func GetFestival(year int, name string) (FestivalInfo, bool) {
	return Default().GetFestival(year, name)
}

// ListFestivals 使用默认 Calendar 列出节假日详情。
func ListFestivals(ctx context.Context, req ListFestivalsReq) ([]FestivalInfo, error) {
	return Default().ListFestivals(ctx, req)
}

// GetYearSummary 使用默认 Calendar 查询指定年份的特殊日期汇总。
func GetYearSummary(year int) (YearSummaryInfo, bool) {
	return Default().GetYearSummary(year)
}

func festivalFromEntries(entries map[string]Entry, year int, name string) (FestivalInfo, bool) {
	infos := festivalsFromEntries(entries, ListFestivalsReq{StartYear: year, EndYear: year, Name: name})
	if len(infos) == 0 {
		return FestivalInfo{}, false
	}
	return infos[0], true
}

func festivalsFromEntries(entries map[string]Entry, req ListFestivalsReq) []FestivalInfo {
	groups := make(map[string]*FestivalInfo)
	for date, entry := range entries {
		year, ok := yearFromDate(date)
		if !ok || !matchesFestivalRequest(year, entry.Name, req) {
			continue
		}
		key := strconv.Itoa(year) + "\x00" + entry.Name
		info, ok := groups[key]
		if !ok {
			info = &FestivalInfo{Year: year, Name: entry.Name}
			groups[key] = info
		}
		appendFestivalEntry(info, date, entry)
	}
	infos := make([]FestivalInfo, 0, len(groups))
	for _, info := range groups {
		normalized, ok := normalizeFestivalInfo(*info)
		if ok {
			infos = append(infos, normalized)
		}
	}
	sortFestivals(infos)
	return infos
}

func yearSummaryFromEntries(entries map[string]Entry, year int) (YearSummaryInfo, bool) {
	info := YearSummaryInfo{Year: year}
	nameDates := make(map[string]string)
	for date, entry := range entries {
		if !isYearDate(date, year) {
			continue
		}
		switch entry.Type {
		case DayTypeHoliday:
			info.HolidayDays = append(info.HolidayDays, date)
		case DayTypeWorkday:
			info.MakeupWorkdays = append(info.MakeupWorkdays, date)
		}
		if entry.IsFestivalDay {
			info.FestivalDays = append(info.FestivalDays, date)
		}
		if firstDate, exists := nameDates[entry.Name]; !exists || date < firstDate {
			nameDates[entry.Name] = date
		}
	}
	if len(info.HolidayDays) == 0 && len(info.MakeupWorkdays) == 0 {
		return YearSummaryInfo{}, false
	}
	sort.Strings(info.HolidayDays)
	sort.Strings(info.MakeupWorkdays)
	sort.Strings(info.FestivalDays)
	for name := range nameDates {
		info.Names = append(info.Names, name)
	}
	sort.Slice(info.Names, func(i, j int) bool {
		left, right := info.Names[i], info.Names[j]
		if nameDates[left] == nameDates[right] {
			return left < right
		}
		return nameDates[left] < nameDates[right]
	})
	return info, true
}

func appendFestivalEntry(info *FestivalInfo, date string, entry Entry) {
	switch entry.Type {
	case DayTypeHoliday:
		info.HolidayDays = append(info.HolidayDays, date)
	case DayTypeWorkday:
		info.MakeupWorkdays = append(info.MakeupWorkdays, date)
	}
	if entry.IsFestivalDay {
		info.FestivalDays = append(info.FestivalDays, date)
	}
}

func normalizeFestivalInfo(info FestivalInfo) (FestivalInfo, bool) {
	if len(info.HolidayDays) == 0 && len(info.MakeupWorkdays) == 0 {
		return FestivalInfo{}, false
	}
	sort.Strings(info.HolidayDays)
	sort.Strings(info.MakeupWorkdays)
	sort.Strings(info.FestivalDays)
	if len(info.HolidayDays) > 0 {
		info.StartDate = info.HolidayDays[0]
		info.EndDate = info.HolidayDays[len(info.HolidayDays)-1]
	}
	return info, true
}

func sortFestivals(infos []FestivalInfo) {
	sort.Slice(infos, func(i, j int) bool {
		if infos[i].Year != infos[j].Year {
			return infos[i].Year < infos[j].Year
		}
		if infos[i].StartDate != infos[j].StartDate {
			return infos[i].StartDate < infos[j].StartDate
		}
		return infos[i].Name < infos[j].Name
	})
}

func matchesFestivalRequest(year int, name string, req ListFestivalsReq) bool {
	if req.StartYear != 0 && year < req.StartYear {
		return false
	}
	if req.EndYear != 0 && year > req.EndYear {
		return false
	}
	return req.Name == "" || matchesFestivalName(name, req.Name)
}

func matchesFestivalName(name, query string) bool {
	if name == query {
		return true
	}
	query = strings.TrimSuffix(query, "节")
	return query != "" && strings.Contains(name, query)
}

func validateListFestivalsReq(req ListFestivalsReq) error {
	if req.StartYear < 0 || req.StartYear > 9999 || req.EndYear < 0 || req.EndYear > 9999 {
		return errors.New("holiday year is invalid")
	}
	if req.StartYear != 0 && req.EndYear != 0 && req.StartYear > req.EndYear {
		return errors.New("holiday year range is invalid")
	}
	return nil
}

func isYearDate(date string, year int) bool {
	dateYear, ok := yearFromDate(date)
	return ok && dateYear == year
}

func yearFromDate(date string) (int, bool) {
	if len(date) < 4 {
		return 0, false
	}
	year, err := strconv.Atoi(date[:4])
	return year, err == nil
}

func yearsFromEntries(entries map[string]Entry) []int {
	seen := make(map[int]struct{})
	for date := range entries {
		if len(date) < 4 {
			continue
		}
		year, err := strconv.Atoi(date[:4])
		if err != nil {
			continue
		}
		seen[year] = struct{}{}
	}
	years := make([]int, 0, len(seen))
	for year := range seen {
		years = append(years, year)
	}
	sort.Ints(years)
	return years
}
