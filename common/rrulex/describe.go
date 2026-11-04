package rrulex

import (
	"cmp"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// Describe 将包含 DTSTART 和 RRULE 的完整 RRULE Set 转换为稳定的简体中文业务描述。
// 对无法准确表达的高级筛选返回 ErrUnsupportedDescription。
func Describe(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	parsed, err := parseDescriptionRule(value)
	if err != nil {
		return "", err
	}
	if err := validateDescribable(parsed); err != nil {
		return "", err
	}

	fixedDailyTimes := hasDailyFixedTimeSet(parsed)
	frequency := parsed.option.Freq
	if fixedDailyTimes {
		frequency = rrule.DAILY
	}
	parts := []string{frequencyText(frequency, parsed.option.Interval, hasLimitingDateFilter(parsed.option, frequency))}
	if frequency == rrule.WEEKLY && (parsed.option.Interval > 1 || len(parsed.option.Bysetpos) > 0) {
		parts[0] = fmt.Sprintf("以%s为一周起始，%s", weekdayText(parsed.option.Wkst), parts[0])
	}
	filters := renderDateFilters(parsed.option, parsed.dtstart)
	if len(filters) > 0 {
		parts = append(parts, strings.Join(filters, "，且"))
	}
	if text := renderTime(parsed.option, parsed.dtstart, parsed.option.Freq <= rrule.DAILY || fixedDailyTimes); text != "" {
		parts = append(parts, text)
	}
	var description strings.Builder
	description.WriteString(parts[0])
	for i, part := range parts[1:] {
		description.WriteString(partSeparator(i, part, parts[0], len(filters) > 0))
		description.WriteString(part)
	}
	if text := renderSetPositions(parsed.option.Bysetpos); text != "" {
		description.WriteString("；每个周期先按上述条件形成候选，再选择" + text + "执行（仅在该位置存在候选时）")
	} else {
		separator := " "
		if len(parts) == 1 && parsed.option.Freq == rrule.SECONDLY {
			separator = ""
		}
		description.WriteString(separator + "执行（仅在上述条件形成匹配候选时）")
	}

	if parsed.option.Count > 0 {
		description.WriteString(fmt.Sprintf("，周期规则最多生成 %d 次", parsed.option.Count))
	}
	description.WriteString(renderBoundary(parsed.dtstart, parsed.option.Until, parsed.location))
	description.WriteString(renderDates("，额外纳入候选：", parsed.rdates, parsed.location))
	description.WriteString(renderDates("，从周期规则与额外候选的合并结果中排除：", parsed.exdates, parsed.location))
	description.WriteString(renderTimezoneNotice(parsed.location))
	return description.String(), nil
}

// standaloneFrequencySuffixes 是不需要额外分隔符就能直接接续日期筛选文案的频率后缀。
var standaloneFrequencySuffixes = [...]string{"每年", "每月", "每周", "每天"}

// partSeparator 返回描述片段之间的分隔符。只有第一个附加片段会走特殊规则，
// 其余片段统一用空格连接。
func partSeparator(index int, part, first string, hasDateFilters bool) string {
	if index != 0 {
		return " "
	}
	switch {
	case strings.Contains(first, "生成候选"):
		return "，并"
	case strings.HasPrefix(part, "仅"):
		return "，"
	case hasDateFilters:
		for _, suffix := range standaloneFrequencySuffixes {
			if strings.HasSuffix(first, suffix) {
				return ""
			}
		}
		return "，"
	default:
		return " "
	}
}

// hasDailyFixedTimeSet 判断低频规则是否可等价展示为每日固定时刻集合。
// INTERVAL 大于 1 时，候选仍受 DTSTART 的小时/分钟相位影响，不能做此简化。
func hasDailyFixedTimeSet(parsed descriptionRule) bool {
	if parsed.option.Interval > 1 || len(parsed.option.Bysetpos) > 0 {
		return false
	}

	option := parsed.option
	switch parsed.option.Freq {
	case rrule.HOURLY:
		return len(option.Byhour) > 0
	case rrule.MINUTELY:
		return len(option.Byhour) > 0 && len(option.Byminute) > 0
	case rrule.SECONDLY:
		return len(option.Byhour) > 0 && len(option.Byminute) > 0 && len(option.Bysecond) > 0
	default:
		return false
	}
}

type descriptionRule struct {
	option   rrule.ROption
	dtstart  time.Time
	rdates   []time.Time
	exdates  []time.Time
	location *time.Location
}

type clockDimension struct {
	name   string
	values []int
}

func parseDescriptionRule(value string) (descriptionRule, error) {
	set, err := ParseSet(value)
	if err != nil {
		return descriptionRule{}, fmt.Errorf("parse RRULE description: %w", err)
	}
	rule := set.GetRRule()
	dtstart := set.GetDTStart()
	return descriptionRule{
		option:   rule.Options,
		dtstart:  dtstart,
		rdates:   append([]time.Time(nil), set.GetRDate()...),
		exdates:  append([]time.Time(nil), set.GetExDate()...),
		location: dtstart.Location(),
	}, nil
}

func validateDescribable(parsed descriptionRule) error {
	option := parsed.option
	unsupported := make([]string, 0, 3)
	if len(option.Byyearday) > 0 {
		unsupported = append(unsupported, "BYYEARDAY")
	}
	if len(option.Byweekno) > 0 {
		unsupported = append(unsupported, "BYWEEKNO")
	}
	if len(option.Byeaster) > 0 {
		unsupported = append(unsupported, "BYEASTER")
	}
	if len(unsupported) > 0 {
		return fmt.Errorf("%w: %s", ErrUnsupportedDescription, strings.Join(unsupported, ","))
	}
	if len(option.Bysetpos) > 0 {
		for _, clock := range bySetPosClockDimensions(option) {
			if hasDuplicateInts(clock.values) {
				return fmt.Errorf("%w: duplicate %s with BYSETPOS", ErrUnsupportedDescription, clock.name)
			}
		}
		hours, minutes, seconds := effectiveClockParts(option, parsed.dtstart)
		if option.Freq != rrule.SECONDLY &&
			!(option.Freq == rrule.YEARLY && len(option.Bymonth) > 0) &&
			len(option.Bymonthday) == 0 && len(option.Byweekday) == 0 &&
			len(hours) == 0 && len(minutes) == 0 && len(seconds) == 0 {
			return fmt.Errorf("%w: BYSETPOS without a selectable filter", ErrUnsupportedDescription)
		}
	}
	if option.Freq > rrule.MONTHLY {
		for _, weekday := range option.Byweekday {
			if weekday.N() != 0 {
				return fmt.Errorf("%w: ordinal BYDAY requires MONTHLY or YEARLY", ErrUnsupportedDescription)
			}
		}
	}
	hasPlainWeekday := false
	hasOrdinalWeekday := false
	for _, weekday := range option.Byweekday {
		if weekday.N() < -5 && (option.Freq == rrule.MONTHLY || option.Freq == rrule.YEARLY && len(option.Bymonth) > 0) {
			return fmt.Errorf("%w: negative ordinal BYDAY exceeds monthly range", ErrUnsupportedDescription)
		}
		if weekday.N() == 0 {
			hasPlainWeekday = true
		} else {
			hasOrdinalWeekday = true
		}
	}
	if hasPlainWeekday && hasOrdinalWeekday {
		return fmt.Errorf("%w: mixed ordinal and plain BYDAY", ErrUnsupportedDescription)
	}
	return nil
}

// bySetPosClockDimensions returns only fields that expand rrule-go's timeset
// for this frequency. Higher clock fields merely filter the current period and
// duplicate values in those filters do not occupy additional BYSETPOS slots.
func bySetPosClockDimensions(option rrule.ROption) []clockDimension {
	dimensions := make([]clockDimension, 0, 3)
	if option.Freq < rrule.HOURLY {
		dimensions = append(dimensions, clockDimension{name: "BYHOUR", values: option.Byhour})
	}
	if option.Freq < rrule.MINUTELY {
		dimensions = append(dimensions, clockDimension{name: "BYMINUTE", values: option.Byminute})
	}
	if option.Freq < rrule.SECONDLY {
		dimensions = append(dimensions, clockDimension{name: "BYSECOND", values: option.Bysecond})
	}
	return dimensions
}

var (
	frequencyUnitTexts = map[rrule.Frequency]string{
		rrule.YEARLY: "年", rrule.MONTHLY: "个月", rrule.WEEKLY: "周", rrule.DAILY: "天",
		rrule.HOURLY: "小时", rrule.MINUTELY: "分钟", rrule.SECONDLY: "秒",
	}
	frequencyLabelTexts = map[rrule.Frequency]string{
		rrule.YEARLY: "每年", rrule.MONTHLY: "每月", rrule.WEEKLY: "每周", rrule.DAILY: "每天",
		rrule.HOURLY: "每小时", rrule.MINUTELY: "每分钟", rrule.SECONDLY: "每秒",
	}
)

func frequencyText(freq rrule.Frequency, interval int, hasFilters bool) string {
	if interval < 1 {
		interval = 1
	}
	unit := frequencyUnitTexts[freq]
	if interval == 1 {
		text := frequencyLabelTexts[freq]
		if hasFilters {
			return "以" + text + "为周期生成候选"
		}
		return text
	}
	text := fmt.Sprintf("按 %d %s间隔", interval, unit)
	if hasFilters {
		return "以开始时间为基准，" + text + "生成候选"
	}
	return "以开始时间为基准，" + text
}

// hasLimitingDateFilter reports whether calendar BY parts filter the candidate
// represented by the frequency instead of expanding dates inside that period.
func hasLimitingDateFilter(option rrule.ROption, freq rrule.Frequency) bool {
	if freq > rrule.YEARLY && len(option.Bymonth) > 0 {
		return true
	}
	if freq > rrule.MONTHLY && len(option.Bymonthday) > 0 {
		return true
	}
	if freq <= rrule.WEEKLY || len(option.Byweekday) == 0 {
		return false
	}
	weekdays := sortedUniqueWeekdays(option.Byweekday)
	text, grouped := groupedWeekdayText(weekdays)
	return !grouped || text != "周一至周日"
}

func renderDateFilters(option rrule.ROption, dtstart time.Time) []string {
	months := sortedUniqueInts(option.Bymonth)
	monthDays := sortedUniqueInts(option.Bymonthday)
	weekdays := sortedUniqueWeekdays(option.Byweekday)
	if option.Freq == rrule.YEARLY && len(months) == 0 && len(monthDays) == 0 && len(weekdays) == 0 && !dtstart.IsZero() {
		months = []int{int(dtstart.Month())}
		monthDays = []int{dtstart.Day()}
	}
	if option.Freq == rrule.MONTHLY && len(monthDays) == 0 && len(weekdays) == 0 && !dtstart.IsZero() {
		monthDays = []int{dtstart.Day()}
	}
	if option.Freq == rrule.WEEKLY && len(weekdays) == 0 && !dtstart.IsZero() {
		weekdays = []rrule.Weekday{weekdayFromTime(dtstart.Weekday())}
	}

	var filters []string
	if len(months) > 0 {
		filters = append(filters, renderMonthFilter(option, months))
	}
	if len(monthDays) > 0 {
		filters = append(filters, renderMonthDayFilter(option, months, monthDays))
	}
	if len(weekdays) > 0 {
		filters = append(filters, renderWeekdayFilter(option, months, monthDays, weekdays))
	}
	return filters
}

// renderMonthFilter 渲染 BYMONTH 的月份筛选文案。
func renderMonthFilter(option rrule.ROption, months []int) string {
	values := make([]string, len(months))
	for i, month := range months {
		values[i] = fmt.Sprintf("%d月", month)
	}
	text := strings.Join(values, "、")
	if option.Freq != rrule.YEARLY {
		text = "仅限" + text
	}
	return text
}

// renderMonthDayFilter 渲染 BYMONTHDAY 的日期筛选文案，负值表示从月末倒数。
func renderMonthDayFilter(option rrule.ROption, months, monthDays []int) string {
	values := make([]string, len(monthDays))
	for i, day := range monthDays {
		switch {
		case day == -1:
			values[i] = "最后一天"
		case day < -1:
			values[i] = fmt.Sprintf("倒数第 %d 天", -day)
		default:
			values[i] = fmt.Sprintf("第 %d 天", day)
		}
	}
	text := strings.Join(values, "、")
	switch option.Freq {
	case rrule.YEARLY:
		if len(months) == 0 {
			text = "各月的" + text
		} else {
			text = "各指定月份的" + text
		}
	case rrule.MONTHLY:
	default:
		text = "仅限每月" + text
	}
	return text
}

// renderWeekdayFilter 渲染 BYDAY 的星期筛选文案，允许序号（如第 1 个周一）。
func renderWeekdayFilter(option rrule.ROption, months, monthDays []int, weekdays []rrule.Weekday) string {
	text, grouped := groupedWeekdayText(weekdays)
	if !grouped {
		values := make([]string, len(weekdays))
		for i, weekday := range weekdays {
			values[i] = weekdayText(weekday)
		}
		text = strings.Join(values, "、")
	}
	hasOrdinal := slices.ContainsFunc(weekdays, func(weekday rrule.Weekday) bool {
		return weekday.N() != 0
	})
	switch option.Freq {
	case rrule.YEARLY:
		switch {
		case len(monthDays) > 0 && hasOrdinal && len(months) > 0:
			text = "仅限各指定月份的" + text
		case len(monthDays) > 0:
			text = "仅限" + text
		case hasOrdinal && len(months) > 0:
			text = "各指定月份的" + text
		case hasOrdinal:
			// The leading YEARLY frequency already supplies the scope.
		case len(months) > 0:
			text = "各指定月份内的" + weekdayScopeText(text, grouped)
		default:
			text = weekdayScopeText(text, grouped)
		}
	case rrule.MONTHLY:
		if len(monthDays) > 0 {
			text = "仅限" + text
		}
	case rrule.WEEKLY:
	default:
		text = "仅限" + text
	}
	return text
}

func groupedWeekdayText(weekdays []rrule.Weekday) (string, bool) {
	if matchesPlainWeekdays(weekdays, 0, 1, 2, 3, 4) {
		return "工作日", true
	}
	if matchesPlainWeekdays(weekdays, 5, 6) {
		return "周末", true
	}
	if matchesPlainWeekdays(weekdays, 0, 1, 2, 3, 4, 5, 6) {
		return "周一至周日", true
	}
	return "", false
}

func matchesPlainWeekdays(weekdays []rrule.Weekday, days ...int) bool {
	if len(weekdays) != len(days) {
		return false
	}
	for i, weekday := range weekdays {
		if weekday.N() != 0 || weekday.Day() != days[i] {
			return false
		}
	}
	return true
}

func weekdayScopeText(text string, grouped bool) string {
	if grouped {
		return text
	}
	return "每个" + text
}

func renderSetPositions(positions []int) string {
	positions = sortedUniqueInts(positions)
	// 正数按升序排在前面，负数按数值升序（即绝对值降序）排在后面。
	slices.SortFunc(positions, func(a, b int) int {
		if (a < 0) != (b < 0) {
			if a < 0 {
				return 1
			}
			return -1
		}
		return cmp.Compare(a, b)
	})
	values := make([]string, len(positions))
	for i, position := range positions {
		switch {
		case position == -1:
			values[i] = "最后一个"
		case position > 0:
			values[i] = fmt.Sprintf("第 %d 个", position)
		default:
			values[i] = fmt.Sprintf("倒数第 %d 个", -position)
		}
	}
	return joinChineseList(values)
}

func joinChineseList(values []string) string {
	switch len(values) {
	case 0:
		return ""
	case 1:
		return values[0]
	default:
		return strings.Join(values[:len(values)-1], "、") + "和" + values[len(values)-1]
	}
}

// maxExpandedClockPoints 是逐个列出 HH:MM[:SS] 的展开上限；
// 超过该数量时改按小时/分钟/秒维度描述，避免文案长度爆炸。
const maxExpandedClockPoints = 24

func renderTime(option rrule.ROption, dtstart time.Time, expandClockTimes bool) string {
	hours, minutes, seconds := effectiveClockParts(option, dtstart)

	if expandClockTimes && len(hours) > 0 && len(minutes) > 0 && len(seconds) > 0 {
		if len(hours)*len(minutes)*len(seconds) <= maxExpandedClockPoints {
			values := make([]string, 0, len(hours)*len(minutes)*len(seconds))
			omitSeconds := len(seconds) == 1 && seconds[0] == 0
			for _, hour := range hours {
				for _, minute := range minutes {
					for _, second := range seconds {
						if omitSeconds {
							values = append(values, fmt.Sprintf("%02d:%02d", hour, minute))
						} else {
							values = append(values, fmt.Sprintf("%02d:%02d:%02d", hour, minute, second))
						}
					}
				}
			}
			return strings.Join(sortedUniqueStrings(values), "、")
		}
	}

	var dimensions []string
	if len(hours) > 0 {
		dimensions = append(dimensions, "小时为 "+joinPadded(hours))
	}
	if len(minutes) > 0 {
		dimensions = append(dimensions, "分钟为 "+joinPadded(minutes))
	}
	if len(seconds) > 0 {
		dimensions = append(dimensions, "秒为 "+joinPadded(seconds))
	}
	if len(dimensions) == 0 {
		return ""
	}
	return "时间条件：" + strings.Join(dimensions, "，")
}

func effectiveClockParts(option rrule.ROption, dtstart time.Time) ([]int, []int, []int) {
	hours := sortedUniqueInts(option.Byhour)
	minutes := sortedUniqueInts(option.Byminute)
	seconds := sortedUniqueInts(option.Bysecond)
	if !dtstart.IsZero() {
		if len(hours) == 0 && option.Freq < rrule.HOURLY {
			hours = []int{dtstart.Hour()}
		}
		if len(minutes) == 0 && option.Freq < rrule.MINUTELY {
			minutes = []int{dtstart.Minute()}
		}
		if len(seconds) == 0 && option.Freq < rrule.SECONDLY {
			seconds = []int{dtstart.Second()}
		}
	}
	return hours, minutes, seconds
}

func renderBoundary(dtstart, until time.Time, location *time.Location) string {
	const layout = "2006-01-02 15:04:05"
	if !dtstart.IsZero() && !until.IsZero() {
		if until.Before(dtstart) {
			return fmt.Sprintf("，周期规则边界已倒置：开始于 %s，截止于 %s", dtstart.In(location).Format(layout), until.In(location).Format(layout))
		}
		return fmt.Sprintf("，周期规则有效期：%s 至 %s", dtstart.In(location).Format(layout), until.In(location).Format(layout))
	}
	if !dtstart.IsZero() {
		return "，周期规则开始于：" + dtstart.In(location).Format(layout)
	}
	if !until.IsZero() {
		return "，周期规则有效至：" + until.In(location).Format(layout)
	}
	return ""
}

func renderTimezoneNotice(location *time.Location) string {
	if location == nil || location.String() == "UTC" {
		return ""
	}
	return fmt.Sprintf("，时区：%s；遇不存在或重复的本地时间，以时区库实际解析结果为准", location)
}

func renderDates(prefix string, dates []time.Time, location *time.Location) string {
	if len(dates) == 0 {
		return ""
	}
	dates = slices.Clone(dates)
	slices.SortFunc(dates, func(a, b time.Time) int {
		switch {
		case a.Before(b):
			return -1
		case b.Before(a):
			return 1
		default:
			return 0
		}
	})
	dates = slices.CompactFunc(dates, time.Time.Equal)
	values := make([]string, 0, len(dates))
	for _, date := range dates {
		values = append(values, date.In(location).Format("2006-01-02 15:04:05 -07:00"))
	}
	return prefix + strings.Join(values, "、")
}

var (
	weekdayNames = [...]string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	// weekdayByTimeWeekday 把 time.Weekday 映射为 rrule.Weekday 常量。
	weekdayByTimeWeekday = [...]rrule.Weekday{rrule.SU, rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR, rrule.SA}
)

func weekdayText(weekday rrule.Weekday) string {
	name := weekdayNames[weekday.Day()]
	if weekday.N() > 0 {
		return fmt.Sprintf("第 %d 个%s", weekday.N(), name)
	}
	if weekday.N() < 0 {
		return fmt.Sprintf("倒数第 %d 个%s", -weekday.N(), name)
	}
	return name
}

func weekdayFromTime(weekday time.Weekday) rrule.Weekday {
	return weekdayByTimeWeekday[weekday]
}

func sortedUniqueInts(values []int) []int {
	result := slices.Clone(values)
	slices.Sort(result)
	return slices.Compact(result)
}

func hasDuplicateInts(values []int) bool {
	sorted := slices.Clone(values)
	slices.Sort(sorted)
	return len(slices.Compact(sorted)) != len(sorted)
}

func sortedUniqueWeekdays(values []rrule.Weekday) []rrule.Weekday {
	result := slices.Clone(values)
	slices.SortFunc(result, func(a, b rrule.Weekday) int {
		if a.Day() != b.Day() {
			return cmp.Compare(a.Day(), b.Day())
		}
		return cmp.Compare(a.N(), b.N())
	})
	return slices.CompactFunc(result, func(a, b rrule.Weekday) bool {
		return a.Day() == b.Day() && a.N() == b.N()
	})
}

func sortedUniqueStrings(values []string) []string {
	slices.Sort(values)
	return slices.Compact(values)
}

func joinPadded(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprintf("%02d", value)
	}
	return strings.Join(parts, "、")
}
