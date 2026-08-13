package rrulex

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
	set, err := ParseSet(value)
	if err != nil {
		return nil, err
	}
	if query := ShiftSetForQuery(set, after); query != nil {
		return query, nil
	}
	return set, nil
}

// NextRuns 返回满足边界条件的至多 count 个有效计划时间。
// 边界语义与官方 rrule.Set.After 一致：inc 为 true 时接受不早于 dt 的候选（!v.Before(dt)），
// 为 false 时接受严格晚于 dt 的候选（v.After(dt)）；后续候选由迭代器单调性保证严格递增。
// invalid 为 nil 或返回 false 表示候选有效；true 表示该候选无效应跳过。
// 本封装只解决性能问题：解析与 ShiftSetForQuery 平移只做一次，用单个迭代器顺序收集，
// 避免循环调用官方 After 反复从起点重走；只统计最终接受的有效点，耗尽时返回已收集结果。
// 谓词跳过不推进游标：迭代器本身单调递增，后续候选必然晚于已跳过候选；
// 仅已接受结果推进 dt 游标，保证返回序列严格递增。
// count <= 0 返回空切片；空字符串规则按解析错误处理（一次性任务由调用方用空规则直接触发）。
func NextRuns(value string, dt time.Time, inc bool, count int, invalid func(time.Time) bool) ([]time.Time, error) {
	if count <= 0 {
		return []time.Time{}, nil
	}
	set, err := parseQuerySet(value, dt)
	if err != nil {
		return nil, err
	}
	runs := make([]time.Time, 0, count)
	next := set.Iterator()
	for len(runs) < count {
		v, ok := next()
		if !ok {
			break
		}
		if invalid != nil && invalid(v) {
			continue
		}
		if inc && !v.Before(dt) || !inc && v.After(dt) {
			runs = append(runs, v)
			dt = v
		}
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
		// 年数差向下取整到 INTERVAL 的倍数：锚点必须落在「与 dtstart 同相位」的间隔周期上，
		// 否则后续整条序列的周期计数错位。
		years := (after.Year() - dtstart.Year()) / interval * interval
		if years <= 0 {
			// 未跨过第一个完整间隔（最近锚点就是 dtstart 本身），平移无意义。
			return time.Time{}, false
		}
		shifted = dtstart.AddDate(years, 0, 0)
		if shifted.After(after) {
			// 锚点整体越过 after（同一年内移动到 after 之后），回退一个间隔到最近锚点。
			shifted = dtstart.AddDate(years-interval, 0, 0)
		}
		// 相位校验：rrule-go 缺省 BYMONTH/BYMONTHDAY 继承自 dtstart 的月/日，
		// AddDate 对 2/29 等日期钳制会改变相位，破坏则放弃平移、沿用原起点慢迭代。
		if shifted.Month() != dtstart.Month() || shifted.Day() != dtstart.Day() {
			return time.Time{}, false
		}
	case rrule.MONTHLY:
		// 月数差（含跨年）向下取整到 INTERVAL 的倍数，原理同 YEARLY。
		months := ((after.Year()-dtstart.Year())*12 + int(after.Month()) - int(dtstart.Month())) / interval * interval
		if months <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.AddDate(0, months, 0)
		if shifted.After(after) {
			shifted = dtstart.AddDate(0, months-interval, 0)
		}
		// 相位校验：缺省 BYMONTHDAY 继承 dtstart 的日号，月末（如 31 日）跨短月被钳制后日号改变即放弃。
		if shifted.Day() != dtstart.Day() {
			return time.Time{}, false
		}
	case rrule.WEEKLY:
		// WEEKLY/DAILY 是墙钟日历频率。起点与查询点 UTC offset 不同时，绝对小时数
		// 无法安全代表日历天数（DST 前拨会少算、回拨会多算），回退原始 Set。
		if !sameUTCOffset(dtstart, after) {
			return time.Time{}, false
		}
		// 天数差向下取整到 7*INTERVAL 的倍数：整数周保证星期相位不变。
		days := int(after.Sub(dtstart).Hours()/24) / (7 * interval) * (7 * interval)
		if days <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.AddDate(0, 0, days)
	case rrule.DAILY:
		if !sameUTCOffset(dtstart, after) {
			return time.Time{}, false
		}
		// 天数差向下取整到 INTERVAL 的倍数，避免出现半个周期的错位。
		days := int(after.Sub(dtstart).Hours()/24) / interval * interval
		if days <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.AddDate(0, 0, days)
	case rrule.HOURLY:
		// 小时差向下取整到 INTERVAL 的倍数；锚点的分/秒相位随后校验。
		hours := int(after.Sub(dtstart).Hours()) / interval * interval
		if hours <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.Add(time.Duration(hours) * time.Hour)
		// 相位校验：缺省 BYMINUTE/BYSECOND 继承自 dtstart，非整小时时区偏移（如 30 分钟 DST）会错开分位，放弃平移。
		if shifted.Minute() != dtstart.Minute() || shifted.Second() != dtstart.Second() {
			return time.Time{}, false
		}
	case rrule.MINUTELY:
		// 分钟差向下取整到 INTERVAL 的倍数；锚点的秒相位随后校验。
		minutes := int(after.Sub(dtstart).Minutes()) / interval * interval
		if minutes <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.Add(time.Duration(minutes) * time.Minute)
		// 相位校验：缺省 BYSECOND 继承自 dtstart，非整分钟时区偏移下的秒位保护。
		if shifted.Second() != dtstart.Second() {
			return time.Time{}, false
		}
	default: // SECONDLY
		// 秒差向下取整到 INTERVAL 的倍数，SECONDLY 无更低粒度相位可破坏。
		seconds := int(after.Sub(dtstart).Seconds()) / interval * interval
		if seconds <= 0 {
			return time.Time{}, false
		}
		shifted = dtstart.Add(time.Duration(seconds) * time.Second)
	}
	// duration 加法（HOURLY/MINUTELY/SECONDLY）跨 DST 时仍保证锚点墙钟时间不晚于 after：
	// 整周期步长落在绝对时间上，而时区偏移变化是整小时，锚点最多与 after 相等。
	// 分/秒相位校验在非整小时偏移的时区（如 30 分钟 DST）回退原集。
	if shifted.Equal(dtstart) || shifted.After(after) {
		return time.Time{}, false
	}
	return shifted, true
}

func sameUTCOffset(a, b time.Time) bool {
	_, aOffset := a.Zone()
	_, bOffset := b.In(a.Location()).Zone()
	return aOffset == bOffset
}
