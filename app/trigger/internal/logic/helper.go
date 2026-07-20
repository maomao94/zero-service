package logic

import (
	"time"

	"zero-service/app/trigger/trigger"
	"zero-service/common/holiday"
	"zero-service/common/tool"
	"zero-service/third_party/extproto"

	"github.com/dromara/carbon/v2"
)

func parseHolidayDate(date string) (time.Time, error) {
	if date == "" {
		return carbon.Now().StdTime(), nil
	}
	c := carbon.ParseByFormat(date, carbon.DateFormat)
	if c.Error != nil || c.IsInvalid() {
		if c.Error != nil {
			return time.Time{}, tool.NewErrorByPbCodeWrap(extproto.Code__1_01_PARAM_INVALID, c.Error, "日期格式错误，应为 yyyy-MM-dd")
		}
		return time.Time{}, tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "日期格式错误，应为 yyyy-MM-dd")
	}
	return c.StdTime(), nil
}

func validateHolidayDate(date string) error {
	if date == "" {
		return tool.NewErrorByPbCode(extproto.Code__1_01_PARAM_INVALID, "日期不能为空")
	}
	_, err := parseHolidayDate(date)
	return err
}

func toHolidayDayPb(info holiday.DayInfo) *trigger.HolidayDayPb {
	return &trigger.HolidayDayPb{
		Date:          info.Date,
		Name:          info.Name,
		Type:          string(info.Type),
		Kind:          toHolidayDayKindPb(info.Kind),
		Note:          info.Note,
		IsFestivalDay: info.IsFestivalDay,
		IsHoliday:     info.IsHoliday,
		IsWorkday:     info.IsWorkday,
	}
}

func toHolidayDayKindPb(kind holiday.DayKind) trigger.HolidayDayKindPb {
	switch kind {
	case holiday.DayKindStatutoryHoliday:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_STATUTORY_HOLIDAY
	case holiday.DayKindWeekend:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_WEEKEND
	case holiday.DayKindMakeupWorkday:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_MAKEUP_WORKDAY
	case holiday.DayKindNormalWorkday:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_NORMAL_WORKDAY
	default:
		return trigger.HolidayDayKindPb_HOLIDAY_DAY_KIND_UNSPECIFIED
	}
}

func toHolidayFestivalPb(info holiday.FestivalInfo) *trigger.HolidayFestivalPb {
	return &trigger.HolidayFestivalPb{
		Year:           int32(info.Year),
		Name:           info.Name,
		StartDate:      info.StartDate,
		EndDate:        info.EndDate,
		HolidayDays:    append([]string(nil), info.HolidayDays...),
		MakeupWorkdays: append([]string(nil), info.MakeupWorkdays...),
		FestivalDays:   append([]string(nil), info.FestivalDays...),
	}
}

func toHolidayYearSummaryPb(info holiday.YearSummaryInfo) *trigger.HolidayYearSummaryPb {
	return &trigger.HolidayYearSummaryPb{
		Year:           int32(info.Year),
		HolidayDays:    append([]string(nil), info.HolidayDays...),
		MakeupWorkdays: append([]string(nil), info.MakeupWorkdays...),
		FestivalDays:   append([]string(nil), info.FestivalDays...),
		Names:          append([]string(nil), info.Names...),
	}
}

func toHolidaySourcePb(item holiday.StoredEntry) *trigger.HolidaySourcePb {
	return &trigger.HolidaySourcePb{
		Date:          item.Date,
		Name:          item.Entry.Name,
		Type:          string(item.Entry.Type),
		Note:          item.Entry.Note,
		IsFestivalDay: item.Entry.IsFestivalDay,
		Enabled:       item.Enabled,
	}
}
