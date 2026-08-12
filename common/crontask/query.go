package crontask

import (
	"time"

	"github.com/teambition/rrule-go"
)

// ShiftSetForQuery 返回一份供查询使用的临时 RRULE Set：
// 将 DTSTART 从原始值按整周期平移到 after 之前最近一个符合 INTERVAL 相位的锚点（相位保持不变），
// 使 rrule-go 的迭代从 after 附近开始，避免从远古 DTSTART 逐点遍历导致的线性退化。
// 返回的查询集只用于 After / Iterator 等向前查询；RDATE 与 EXDATE 原样保留。
// 无法安全平移的规则（COUNT、BYWEEKNO、BYYEARDAY、BYEASTER、BYSETPOS，
// 或整周期平移被 Go AddDate 钳制而改变相位）返回 nil，调用方应继续使用原始 Set。
func ShiftSetForQuery(set *rrule.Set, after time.Time) *rrule.Set {
	if set == nil || set.GetRRule() == nil {
		return nil
	}
	option := set.GetRRule().OrigOptions
	if option.Count > 0 ||
		len(option.Byweekno) > 0 ||
		len(option.Byyearday) > 0 ||
		len(option.Byeaster) > 0 ||
		len(option.Bysetpos) > 0 {
		return nil
	}
	dtstart := set.GetDTStart()
	shifted, ok := shiftDtStartByPeriod(dtstart, after, option.Freq, option.Interval)
	if !ok {
		return nil
	}
	queryOption := option
	queryOption.Dtstart = shifted
	queryRule, err := rrule.NewRRule(queryOption)
	if err != nil {
		return nil
	}
	query := &rrule.Set{}
	query.RRule(queryRule)
	query.SetRDates(set.GetRDate())
	query.SetExDates(set.GetExDate())
	return query
}

// parseQuerySet 解析 RRULE Set，并返回平移迭代起点后的查询集。
// 无法安全平移的规则返回原始 Set，调用方继续按原语义查询。
func parseQuerySet(value string, after time.Time) (*rrule.Set, error) {
	set, err := parseRRuleSet(value)
	if err != nil {
		return nil, err
	}
	if query := ShiftSetForQuery(set, after); query != nil {
		return query, nil
	}
	return set, nil
}

// NextRuns 返回严格晚于 after 的至多 count 个未来计划时间。
// 通过 ShiftSetForQuery 平移迭代起点，并用单个迭代器顺序收集：
// 首段遍历只付一次，避免循环调用 Set.After 从起点反复重走造成的二次方退化。
// count <= 0 返回空切片；空字符串规则按解析错误处理（一次性任务由调用方用空规则直接触发）。
func NextRuns(value string, after time.Time, count int) ([]time.Time, error) {
	if count <= 0 {
		return []time.Time{}, nil
	}
	set, err := parseQuerySet(value, after)
	if err != nil {
		return nil, err
	}
	runs := make([]time.Time, 0, count)
	next := set.Iterator()
	for len(runs) < count {
		dt, ok := next()
		if !ok {
			break
		}
		if !dt.After(after) {
			continue
		}
		runs = append(runs, dt)
		after = dt
	}
	return runs, nil
}

// shiftDtStartByPeriod 将 dtstart 按整周期平移到 after 之前最近一个符合 INTERVAL 相位的锚点。
// 整周期平移保证 rrule-go 从 DTSTART 继承的隐含相位（缺省 BYHOUR/BYMINUTE/BYSECOND、
// BYMONTHDAY、BYMONTH、BYWEEKDAY 等）保持不变，因此从新起点开始的未来发生点与原始一致。
// 周期计数按 INTERVAL 取整对齐：rrule-go 迭代以锚点所在周期为第一个候选周期，
// 锚点不在间隔相位上会使后续整条序列错位，必须回退（INTERVAL 缺省为 1）。
// ok 为 false 表示无需平移或平移会破坏相位，调用方应使用原起点。
func shiftDtStartByPeriod(dtstart, after time.Time, freq rrule.Frequency, interval int) (shifted time.Time, ok bool) {
	if dtstart.IsZero() || !after.After(dtstart) {
		return time.Time{}, false
	}
	if interval < 1 {
		interval = 1
	}
	switch freq {
	case rrule.YEARLY:
		years := (after.Year() - dtstart.Year()) / interval * interval
		if years <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.AddDate(years, 0, 0)
		if shifted.After(after) {
			shifted = dtstart.AddDate(years-interval, 0, 0)
		}
		if shifted.Month() != dtstart.Month() || shifted.Day() != dtstart.Day() {
			return time.Time{}, false
		}
	case rrule.MONTHLY:
		months := ((after.Year()-dtstart.Year())*12 + int(after.Month()) - int(dtstart.Month())) / interval * interval
		if months <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.AddDate(0, months, 0)
		if shifted.After(after) {
			shifted = dtstart.AddDate(0, months-interval, 0)
		}
		if shifted.Day() != dtstart.Day() {
			return time.Time{}, false
		}
	case rrule.WEEKLY:
		days := int(after.Sub(dtstart).Hours()/24) / (7 * interval) * (7 * interval)
		if days <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.AddDate(0, 0, days)
	case rrule.DAILY:
		days := int(after.Sub(dtstart).Hours()/24) / interval * interval
		if days <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.AddDate(0, 0, days)
	case rrule.HOURLY:
		hours := int(after.Sub(dtstart).Hours()) / interval * interval
		if hours <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.Add(time.Duration(hours) * time.Hour)
		if shifted.Minute() != dtstart.Minute() || shifted.Second() != dtstart.Second() {
			return time.Time{}, false
		}
	case rrule.MINUTELY:
		minutes := int(after.Sub(dtstart).Minutes()) / interval * interval
		if minutes <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.Add(time.Duration(minutes) * time.Minute)
		if shifted.Second() != dtstart.Second() {
			return time.Time{}, false
		}
	default: // SECONDLY
		seconds := int(after.Sub(dtstart).Seconds()) / interval * interval
		if seconds <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.Add(time.Duration(seconds) * time.Second)
	}
	// duration 加法（HOURLY/MINUTELY/SECONDLY）跨 DST 时仍保证锚点墙钟时间不晚于 after：
	// 整周期步长落在绝对时间上，而时区偏移变化是整小时，锚点最多与 after 相等。
	// 分/秒相位校验在非整小时偏移的时区（如 30 分钟 DST）回退原集。
	if shifted.Equal(dtstart) {
		return time.Time{}, false
	}
	return shifted, true
}
