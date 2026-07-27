package crontask

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/teambition/rrule-go"
)

// DescribeRRule 将包含 DTSTART 和 RRULE 的完整 RRULE Set 转换为稳定的简体中文业务描述。
// 对无法准确表达的高级筛选返回 ErrUnsupportedDescription。
func DescribeRRule(value string) (string, error) {
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
	parts := []string{frequencyText(frequency, parsed.option.Interval, hasLimitingDateFilter(parsed.original, frequency))}
	if frequency == rrule.WEEKLY && (parsed.option.Interval > 1 || len(parsed.original.Bysetpos) > 0) {
		parts[0] = fmt.Sprintf("以%s为一周起始，%s", weekdayText(parsed.option.Wkst), parts[0])
	}
	filters := renderDateFilters(parsed.option, parsed.dtstart)
	if len(filters) > 0 {
		parts = append(parts, strings.Join(filters, "，且"))
	}
	if text := renderTime(parsed.option, parsed.dtstart, parsed.option.Freq <= rrule.DAILY || fixedDailyTimes); text != "" {
		parts = append(parts, text)
	}
	description := strings.Join(parts, " ")
	if text := renderSetPositions(parsed.original.Bysetpos); text != "" {
		description += "，每个周期内取符合上述条件的" + text
	}
	description += " 执行"

	if parsed.option.Count > 0 {
		description += fmt.Sprintf("，周期规则最多生成 %d 次", parsed.option.Count)
	}
	description += renderBoundary(parsed.dtstart, parsed.option.Until, parsed.location)
	description += renderDates("，额外执行：", parsed.rdates, parsed.location)
	description += renderDates("，排除执行：", parsed.exdates, parsed.location)
	description += renderTimezoneNotice(parsed.location)
	return description, nil
}

// hasDailyFixedTimeSet 判断低频推进规则是否可等价展示为每日固定时刻集合。
// INTERVAL 大于 1 时，候选仍受 DTSTART 的小时/分钟相位影响，不能做此简化。
func hasDailyFixedTimeSet(parsed descriptionRule) bool {
	if parsed.option.Interval > 1 || len(parsed.original.Bysetpos) > 0 {
		return false
	}

	original := parsed.original
	switch parsed.option.Freq {
	case rrule.HOURLY:
		return len(original.Byhour) > 0
	case rrule.MINUTELY:
		return len(original.Byhour) > 0 && len(original.Byminute) > 0
	case rrule.SECONDLY:
		return len(original.Byhour) > 0 && len(original.Byminute) > 0 && len(original.Bysecond) > 0
	default:
		return false
	}
}

type descriptionRule struct {
	option   rrule.ROption
	original rrule.ROption
	dtstart  time.Time
	rdates   []time.Time
	exdates  []time.Time
	location *time.Location
}

func parseDescriptionRule(value string) (descriptionRule, error) {
	if err := validateDescriptionSetShape(value); err != nil {
		return descriptionRule{}, err
	}
	set, err := parseRRuleSet(value)
	if err != nil {
		return descriptionRule{}, fmt.Errorf("parse RRULE description: %w", err)
	}
	rule := set.GetRRule()
	dtstart := set.GetDTStart()
	return descriptionRule{
		option:   rule.Options,
		original: rule.OrigOptions,
		dtstart:  dtstart,
		rdates:   append([]time.Time(nil), set.GetRDate()...),
		exdates:  append([]time.Time(nil), set.GetExDate()...),
		location: dtstart.Location(),
	}, nil
}

func validateDescriptionSetShape(value string) error {
	lines := strings.Split(strings.ReplaceAll(strings.TrimSpace(value), "\r\n", "\n"), "\n")
	counts := make(map[string]int, 4)
	for _, line := range lines {
		separator := strings.IndexAny(line, ";:")
		if separator <= 0 {
			return fmt.Errorf("parse RRULE description: invalid Set line %q", line)
		}
		name := strings.ToUpper(strings.TrimSpace(line[:separator]))
		switch name {
		case "DTSTART", "RRULE", "RDATE", "EXDATE":
		default:
			return fmt.Errorf("%w: unsupported Set component %s", ErrUnsupportedDescription, name)
		}
		counts[name]++
		if name == "DTSTART" && counts[name] > 1 {
			return fmt.Errorf("parse RRULE description: Set requires exactly one DTSTART")
		}
		if name == "RRULE" && counts[name] > 1 {
			return fmt.Errorf("%w: multiple RRULE components", ErrUnsupportedDescription)
		}
	}
	return nil
}

func validateDescribable(parsed descriptionRule) error {
	option := parsed.option
	original := parsed.original
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
	if len(original.Bysetpos) > 0 && !(option.Freq == rrule.YEARLY && len(original.Bymonth) > 0) &&
		len(original.Bymonthday) == 0 && len(original.Byweekday) == 0 &&
		len(original.Byhour) == 0 && len(original.Byminute) == 0 && len(original.Bysecond) == 0 {
		return fmt.Errorf("%w: BYSETPOS without a selectable filter", ErrUnsupportedDescription)
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

func frequencyText(freq rrule.Frequency, interval int, advancesWithFilters bool) string {
	if interval < 1 {
		interval = 1
	}
	unit := map[rrule.Frequency]string{
		rrule.YEARLY: "年", rrule.MONTHLY: "个月", rrule.WEEKLY: "周", rrule.DAILY: "天",
		rrule.HOURLY: "小时", rrule.MINUTELY: "分钟", rrule.SECONDLY: "秒",
	}[freq]
	if interval == 1 {
		text := map[rrule.Frequency]string{
			rrule.YEARLY: "每年", rrule.MONTHLY: "每月", rrule.WEEKLY: "每周", rrule.DAILY: "每天",
			rrule.HOURLY: "每小时", rrule.MINUTELY: "每分钟", rrule.SECONDLY: "每秒",
		}[freq]
		if advancesWithFilters {
			return text + "推进"
		}
		return text
	}
	text := fmt.Sprintf("按 %d %s间隔", interval, unit)
	if advancesWithFilters {
		return text + "推进"
	}
	return text
}

// hasLimitingDateFilter reports whether calendar BY parts filter the candidate
// represented by the frequency instead of expanding dates inside that period.
func hasLimitingDateFilter(original rrule.ROption, freq rrule.Frequency) bool {
	if freq > rrule.YEARLY && len(original.Bymonth) > 0 {
		return true
	}
	if freq > rrule.MONTHLY && len(original.Bymonthday) > 0 {
		return true
	}
	return freq > rrule.WEEKLY && len(original.Byweekday) > 0
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
		values := make([]string, len(months))
		for i, month := range months {
			values[i] = fmt.Sprintf("%d月", month)
		}
		text := strings.Join(values, "、")
		if option.Freq != rrule.YEARLY {
			text = "仅在" + text
		}
		filters = append(filters, text)
	}
	if len(monthDays) > 0 {
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
				text = "每年内各月" + text
			} else {
				text = "各指定月份" + text
			}
		case rrule.MONTHLY:
		default:
			text = "仅每月" + text
		}
		filters = append(filters, text)
	}
	if len(weekdays) > 0 {
		values := make([]string, len(weekdays))
		for i, weekday := range weekdays {
			values[i] = weekdayText(weekday)
		}
		text := strings.Join(values, "、")
		hasOrdinal := false
		for _, weekday := range weekdays {
			if weekday.N() != 0 {
				hasOrdinal = true
				break
			}
		}
		switch option.Freq {
		case rrule.YEARLY:
			switch {
			case len(monthDays) > 0 && hasOrdinal && len(months) > 0:
				text = "仅限各指定月份的" + text
			case len(monthDays) > 0 && hasOrdinal:
				text = "仅限每年内的" + text
			case len(monthDays) > 0:
				text = "仅限" + text
			case hasOrdinal && len(months) > 0:
				text = "各指定月份的" + text
			case hasOrdinal:
				text = "每年内的" + text
			case len(months) > 0:
				text = "各指定月份内每个" + text
			default:
				text = "每年内每个" + text
			}
		case rrule.MONTHLY:
			if len(monthDays) > 0 {
				text = "仅限" + text
			}
		case rrule.WEEKLY:
		default:
			text = "仅" + text
		}
		filters = append(filters, text)
	}
	return filters
}

func renderSetPositions(positions []int) string {
	positions = sortedUniqueInts(positions)
	values := make([]string, len(positions))
	for i, position := range positions {
		if position > 0 {
			values[i] = fmt.Sprintf("第 %d 个", position)
		} else {
			values[i] = fmt.Sprintf("倒数第 %d 个", -position)
		}
	}
	return strings.Join(values, "、")
}

func renderTime(option rrule.ROption, dtstart time.Time, expandClockTimes bool) string {
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

	if expandClockTimes && len(hours) > 0 && len(minutes) > 0 && len(seconds) > 0 {
		if len(hours)*len(minutes)*len(seconds) <= 24 {
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

func renderBoundary(dtstart, until time.Time, location *time.Location) string {
	const layout = "2006-01-02 15:04:05"
	if !dtstart.IsZero() && !until.IsZero() {
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
	dates = append([]time.Time(nil), dates...)
	sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })
	values := make([]string, 0, len(dates))
	for i, date := range dates {
		if i > 0 && date.Equal(dates[i-1]) {
			continue
		}
		values = append(values, date.In(location).Format("2006-01-02 15:04:05 -07:00"))
	}
	return prefix + strings.Join(values, "、")
}

func weekdayText(weekday rrule.Weekday) string {
	name := []string{"周一", "周二", "周三", "周四", "周五", "周六", "周日"}[weekday.Day()]
	if weekday.N() > 0 {
		return fmt.Sprintf("第 %d 个%s", weekday.N(), name)
	}
	if weekday.N() < 0 {
		return fmt.Sprintf("倒数第 %d 个%s", -weekday.N(), name)
	}
	return name
}

func weekdayFromTime(weekday time.Weekday) rrule.Weekday {
	return []rrule.Weekday{rrule.SU, rrule.MO, rrule.TU, rrule.WE, rrule.TH, rrule.FR, rrule.SA}[weekday]
}

func sortedUniqueInts(values []int) []int {
	result := append([]int(nil), values...)
	sort.Ints(result)
	return compactSorted(result)
}

func compactSorted(values []int) []int {
	if len(values) == 0 {
		return values
	}
	result := values[:1]
	for _, value := range values[1:] {
		if value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func sortedUniqueWeekdays(values []rrule.Weekday) []rrule.Weekday {
	result := append([]rrule.Weekday(nil), values...)
	sort.Slice(result, func(i, j int) bool {
		if result[i].Day() == result[j].Day() {
			return result[i].N() < result[j].N()
		}
		return result[i].Day() < result[j].Day()
	})
	compacted := result[:0]
	for _, value := range result {
		if len(compacted) == 0 || value.Day() != compacted[len(compacted)-1].Day() || value.N() != compacted[len(compacted)-1].N() {
			compacted = append(compacted, value)
		}
	}
	return compacted
}

func sortedUniqueStrings(values []string) []string {
	sort.Strings(values)
	result := values[:0]
	for _, value := range values {
		if len(result) == 0 || value != result[len(result)-1] {
			result = append(result, value)
		}
	}
	return result
}

func joinPadded(values []int) string {
	parts := make([]string, len(values))
	for i, value := range values {
		parts[i] = fmt.Sprintf("%02d", value)
	}
	return strings.Join(parts, "、")
}
