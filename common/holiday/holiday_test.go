package holiday

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"zero-service/common/gormx"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func TestLookupConfiguredHoliday(t *testing.T) {
	info := Lookup(time.Date(2026, time.October, 1, 12, 0, 0, 0, time.UTC))

	if info.Date != "2026-10-01" {
		t.Fatalf("date = %q, want 2026-10-01", info.Date)
	}
	if !info.IsHoliday || info.IsWorkday {
		t.Fatalf("holiday flags = holiday:%v workday:%v, want true false", info.IsHoliday, info.IsWorkday)
	}
	if info.Type != DayTypeHoliday || info.Kind != DayKindStatutoryHoliday || info.Note != "国庆节" {
		t.Fatalf("info = %+v, want statutory holiday 国庆节", info)
	}
	if !info.IsFestivalDay {
		t.Fatalf("IsFestivalDay = false, want true")
	}
}

func TestLookupFestivalDay(t *testing.T) {
	festivalDay := Lookup(time.Date(2026, time.February, 17, 12, 0, 0, 0, time.Local))
	if festivalDay.Name != "春节" || !festivalDay.IsFestivalDay {
		t.Fatalf("festivalDay = %+v, want 春节当天", festivalDay)
	}

	holidayDay := Lookup(time.Date(2026, time.February, 18, 12, 0, 0, 0, time.Local))
	if holidayDay.Name != "春节" || holidayDay.IsFestivalDay {
		t.Fatalf("holidayDay = %+v, want 春节假期非当天", holidayDay)
	}

	makeupDay := Lookup(time.Date(2026, time.February, 28, 12, 0, 0, 0, time.Local))
	if makeupDay.Name != "春节" || makeupDay.Kind != DayKindMakeupWorkday || makeupDay.IsFestivalDay {
		t.Fatalf("makeupDay = %+v, want 春节调休补班非当天", makeupDay)
	}
}

func TestLookupNewYearsDay2023(t *testing.T) {
	festivalDay := Lookup(time.Date(2023, time.January, 1, 12, 0, 0, 0, time.Local))
	if festivalDay.Name != "元旦" || !festivalDay.IsFestivalDay {
		t.Fatalf("festivalDay = %+v, want 元旦当天", festivalDay)
	}

	holidayDay := Lookup(time.Date(2023, time.January, 2, 12, 0, 0, 0, time.Local))
	if holidayDay.Name != "元旦" || holidayDay.IsFestivalDay {
		t.Fatalf("holidayDay = %+v, want 元旦假期非当天", holidayDay)
	}
}

func TestLookupConfiguredMakeupWorkday(t *testing.T) {
	info := Lookup(time.Date(2026, time.October, 10, 12, 0, 0, 0, time.UTC))

	if !IsWorkday(time.Date(2026, time.October, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("IsWorkday() = false, want true")
	}
	if IsHoliday(time.Date(2026, time.October, 10, 12, 0, 0, 0, time.UTC)) {
		t.Fatal("IsHoliday() = true, want false")
	}
	if !info.IsWorkday || info.IsHoliday {
		t.Fatalf("workday flags = holiday:%v workday:%v, want false true", info.IsHoliday, info.IsWorkday)
	}
	if info.Type != DayTypeWorkday || info.Kind != DayKindMakeupWorkday || info.Note != "国庆节补班" {
		t.Fatalf("info = %+v, want makeup workday 国庆节补班", info)
	}
}

func TestLookupWeekendFallback(t *testing.T) {
	info := Lookup(time.Date(2026, time.July, 18, 12, 0, 0, 0, time.Local))

	if !info.IsHoliday || info.Type != DayTypeHoliday || info.Kind != DayKindWeekend || info.Note != noteWeekend {
		t.Fatalf("info = %+v, want weekend holiday", info)
	}
}

func TestLookupNormalWorkdayFallback(t *testing.T) {
	info := Lookup(time.Date(2026, time.July, 20, 12, 0, 0, 0, time.Local))

	if !info.IsWorkday || info.Type != DayTypeWorkday || info.Kind != DayKindNormalWorkday || info.Note != noteNormalWorkday {
		t.Fatalf("info = %+v, want normal workday", info)
	}
}

func TestLookupUsesChinaDate(t *testing.T) {
	info := Lookup(time.Date(2026, time.September, 30, 16, 30, 0, 0, time.UTC))

	if info.Date != "2026-10-01" || info.Kind != DayKindStatutoryHoliday || info.Note != "国庆节" {
		t.Fatalf("info = %+v, want China date 2026-10-01 国庆节", info)
	}
}

func TestSupportedYears(t *testing.T) {
	want := []int{2023, 2024, 2025, 2026}
	if got := SupportedYears(); !reflect.DeepEqual(got, want) {
		t.Fatalf("SupportedYears() = %v, want %v", got, want)
	}
	if !IsSupportedYear(2026) {
		t.Fatal("IsSupportedYear(2026) = false, want true")
	}
	if IsSupportedYear(2022) {
		t.Fatal("IsSupportedYear(2022) = true, want false")
	}
}

func TestListFestivalsNationalDay2026(t *testing.T) {
	infos, err := ListFestivals(context.Background(), ListFestivalsReq{StartYear: 2026, EndYear: 2026, Name: "国庆节"})
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 {
		t.Fatalf("len(infos) = %d, want 1: %+v", len(infos), infos)
	}
	info := infos[0]
	if info.Year != 2026 || info.Name != "国庆节" {
		t.Fatalf("info = %+v, want 2026 国庆节", info)
	}
	if info.StartDate != "2026-10-01" || info.EndDate != "2026-10-07" {
		t.Fatalf("range = %s - %s, want 2026-10-01 - 2026-10-07", info.StartDate, info.EndDate)
	}
	wantHolidays := []string{"2026-10-01", "2026-10-02", "2026-10-03", "2026-10-04", "2026-10-05", "2026-10-06", "2026-10-07"}
	if !reflect.DeepEqual(info.HolidayDays, wantHolidays) {
		t.Fatalf("HolidayDays = %v, want %v", info.HolidayDays, wantHolidays)
	}
	wantWorkdays := []string{"2026-09-20", "2026-10-10"}
	if !reflect.DeepEqual(info.MakeupWorkdays, wantWorkdays) {
		t.Fatalf("MakeupWorkdays = %v, want %v", info.MakeupWorkdays, wantWorkdays)
	}
	if !reflect.DeepEqual(info.FestivalDays, []string{"2026-10-01"}) {
		t.Fatalf("FestivalDays = %v, want [2026-10-01]", info.FestivalDays)
	}
}

func TestListFestivalsCombinedHoliday(t *testing.T) {
	infos, err := ListFestivals(context.Background(), ListFestivalsReq{StartYear: 2025, EndYear: 2025, Name: "中秋国庆"})
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 {
		t.Fatalf("len(infos) = %d, want 1: %+v", len(infos), infos)
	}
	info := infos[0]
	if info.StartDate != "2025-10-01" || info.EndDate != "2025-10-08" {
		t.Fatalf("range = %s - %s, want 2025-10-01 - 2025-10-08", info.StartDate, info.EndDate)
	}
	if !containsString(info.HolidayDays, "2025-10-06") {
		t.Fatalf("HolidayDays = %v, want include 2025-10-06", info.HolidayDays)
	}
}

func TestGetFestivalMatchesCombinedHolidayByPartialName(t *testing.T) {
	info, ok := GetFestival(2025, "国庆节")
	if !ok {
		t.Fatal("GetFestival(2025, 国庆节) ok = false, want match 中秋国庆")
	}
	if info.Name != "中秋国庆" || info.StartDate != "2025-10-01" || info.EndDate != "2025-10-08" {
		t.Fatalf("info = %+v, want 2025 中秋国庆", info)
	}
}

func TestListFestivalsDoesNotContainYearSummaryPseudoItem(t *testing.T) {
	infos, err := ListFestivals(context.Background(), ListFestivalsReq{StartYear: 2026, EndYear: 2026})
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) == 0 {
		t.Fatal("ListFestivals() returned empty, want festivals")
	}
	for _, info := range infos {
		if info.Name == "" {
			t.Fatalf("ListFestivals() includes year summary pseudo item: %+v", info)
		}
	}
	if infos[0].Year != 2026 || infos[0].Name != "元旦" || infos[0].StartDate != "2026-01-01" {
		t.Fatalf("first info = %+v, want 2026 元旦", infos[0])
	}
}

func TestListFestivalsYearRangeAndNameFilter(t *testing.T) {
	infos, err := ListFestivals(context.Background(), ListFestivalsReq{StartYear: 2025, EndYear: 2026, Name: "元旦"})
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 2 {
		t.Fatalf("len(infos) = %d, want 2: %+v", len(infos), infos)
	}
	if infos[0].Year != 2025 || infos[1].Year != 2026 {
		t.Fatalf("infos = %+v, want 2025 and 2026 元旦", infos)
	}
	for _, info := range infos {
		if info.Name != "元旦" {
			t.Fatalf("info = %+v, want 元旦 only", info)
		}
	}
}

func TestListFestivalsMatchesCombinedHolidayByPartialName(t *testing.T) {
	infos, err := ListFestivals(context.Background(), ListFestivalsReq{StartYear: 2025, EndYear: 2025, Name: "国庆节"})
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 || infos[0].Name != "中秋国庆" {
		t.Fatalf("infos = %+v, want 中秋国庆", infos)
	}
}

func TestGetYearSummary(t *testing.T) {
	info, ok := GetYearSummary(2026)
	if !ok {
		t.Fatal("GetYearSummary(2026) ok = false, want true")
	}
	if info.Year != 2026 {
		t.Fatalf("Year = %d, want 2026", info.Year)
	}
	if !containsString(info.HolidayDays, "2026-10-01") || !containsString(info.MakeupWorkdays, "2026-10-10") || !containsString(info.FestivalDays, "2026-10-01") {
		t.Fatalf("info = %+v, want holiday, makeup workday and festival day", info)
	}
	wantNames := []string{"元旦", "春节", "清明节", "劳动节", "端午节", "国庆节", "中秋节"}
	if !reflect.DeepEqual(info.Names, wantNames) {
		t.Fatalf("Names = %v, want %v", info.Names, wantNames)
	}
}

func TestListFestivalsNotFound(t *testing.T) {
	infos, err := ListFestivals(context.Background(), ListFestivalsReq{StartYear: 2026, EndYear: 2026, Name: "不存在"})
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 0 {
		t.Fatalf("infos = %+v, want empty", infos)
	}
}

func TestGetFestivalNotFound(t *testing.T) {
	if _, ok := GetFestival(2026, "不存在"); ok {
		t.Fatal("GetFestival() ok = true, want false")
	}
}

func TestDirSource(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"2030-01-01":{"type":"holiday","isFestivalDay":true,"note":"测试假日"},"2030-01-06":{"type":"workday","note":"测试补班"}}`)
	if err := os.WriteFile(filepath.Join(dir, "2030.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	cal, err := NewCalendar(context.Background(), WithSource(NewDirSource(dir)))
	if err != nil {
		t.Fatal(err)
	}

	holidayInfo := cal.Lookup(time.Date(2030, time.January, 1, 9, 0, 0, 0, time.UTC))
	if holidayInfo.Kind != DayKindStatutoryHoliday || holidayInfo.Note != "测试假日" || !holidayInfo.IsFestivalDay {
		t.Fatalf("holidayInfo = %+v, want dir holiday", holidayInfo)
	}
	workdayInfo := cal.Lookup(time.Date(2030, time.January, 6, 9, 0, 0, 0, time.UTC))
	if workdayInfo.Kind != DayKindMakeupWorkday || workdayInfo.Note != "测试补班" {
		t.Fatalf("workdayInfo = %+v, want dir workday", workdayInfo)
	}
}

func TestDirSourceRejectsInvalidType(t *testing.T) {
	dir := t.TempDir()
	data := []byte(`{"2030-01-01":{"type":"bad","note":"错误类型"}}`)
	if err := os.WriteFile(filepath.Join(dir, "2030.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := NewCalendar(context.Background(), WithSource(NewDirSource(dir)))
	if err == nil {
		t.Fatal("NewCalendar() error = nil, want invalid type error")
	}
}

func TestDirSourceRejectsDuplicateDateAcrossFiles(t *testing.T) {
	dir := t.TempDir()
	first := []byte(`{"2030-01-01":{"name":"测试节","type":"holiday","note":"测试假日"}}`)
	second := []byte(`{"2030-01-01":{"name":"测试节","type":"workday","note":"测试补班"}}`)
	if err := os.WriteFile(filepath.Join(dir, "2030.json"), first, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "2030-extra.json"), second, 0o600); err != nil {
		t.Fatal(err)
	}

	_, err := NewCalendar(context.Background(), WithSource(NewDirSource(dir)))
	if err == nil || !strings.Contains(err.Error(), "duplicate date") {
		t.Fatalf("NewCalendar() error = %v, want duplicate date", err)
	}
}

func TestGormSource(t *testing.T) {
	db := newHolidayTestDB(t)
	if err := db.AutoMigrate(&GormHoliday{}); err != nil {
		t.Fatal(err)
	}
	rows := []GormHoliday{
		{Date: "2031-01-01", Type: DayTypeHoliday, IsFestivalDay: true, Note: "数据库假日", Enabled: true},
		{Date: "2031-01-05", Type: DayTypeWorkday, Note: "数据库补班", Enabled: true},
		{Date: "2031-02-01", Type: DayTypeHoliday, Note: "禁用假日", Enabled: false},
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	cal, err := NewCalendar(context.Background(), WithSource(NewGormSource(db)))
	if err != nil {
		t.Fatal(err)
	}

	holidayInfo := cal.Lookup(time.Date(2031, time.January, 1, 9, 0, 0, 0, time.UTC))
	if holidayInfo.Kind != DayKindStatutoryHoliday || holidayInfo.Note != "数据库假日" || !holidayInfo.IsFestivalDay {
		t.Fatalf("holidayInfo = %+v, want gorm holiday", holidayInfo)
	}
	workdayInfo := cal.Lookup(time.Date(2031, time.January, 5, 9, 0, 0, 0, time.UTC))
	if workdayInfo.Kind != DayKindMakeupWorkday || workdayInfo.Note != "数据库补班" {
		t.Fatalf("workdayInfo = %+v, want gorm workday", workdayInfo)
	}
	disabledInfo := cal.Lookup(time.Date(2031, time.February, 1, 9, 0, 0, 0, time.UTC))
	if disabledInfo.Kind != DayKindWeekend {
		t.Fatalf("disabledInfo = %+v, want weekend fallback", disabledInfo)
	}
}

func TestGormSourceInitializesEmptyTableFromEmbeddedData(t *testing.T) {
	db := newHolidayTestDB(t)

	cal, err := NewCalendar(context.Background(), WithSource(NewGormSource(db)))
	if err != nil {
		t.Fatal(err)
	}
	info := cal.Lookup(time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC))
	if info.Name != "国庆节" || info.Kind != DayKindStatutoryHoliday {
		t.Fatalf("info = %+v, want embedded 国庆节", info)
	}

	var count int64
	if err := db.Model(&GormHoliday{}).Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count == 0 {
		t.Fatal("gorm source did not initialize embedded rows")
	}
	var row GormHoliday
	if err := db.Where("date = ?", "2026-10-01").First(&row).Error; err != nil {
		t.Fatal(err)
	}
	if row.Name != "国庆节" || row.Note != "国庆节" || row.Type != DayTypeHoliday || !row.IsFestivalDay || !row.Enabled {
		t.Fatalf("row = %+v, want initialized 国庆节 row", row)
	}
}

func TestGormSourceSaveSetEnabledAndList(t *testing.T) {
	db := newHolidayTestDB(t)
	store := NewGormSource(db, WithGormInitEmbedded(false))
	ctx := context.Background()

	entry := Entry{Name: "测试节", Type: DayTypeHoliday, IsFestivalDay: true, Note: "测试节"}
	if err := store.Save(ctx, StoredEntry{Date: "2033-01-01", Entry: entry, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	cal, err := NewCalendar(ctx, WithSource(store))
	if err != nil {
		t.Fatal(err)
	}
	info := cal.Lookup(time.Date(2033, time.January, 1, 9, 0, 0, 0, time.Local))
	if info.Name != "测试节" || !info.IsFestivalDay || !info.IsHoliday {
		t.Fatalf("info = %+v, want saved festival holiday", info)
	}
	updatedEntry := Entry{Name: "测试节", Type: DayTypeHoliday, Note: "测试节调休"}
	if err := store.Save(ctx, StoredEntry{Date: "2033-01-01", Entry: updatedEntry, Enabled: false}); err != nil {
		t.Fatal(err)
	}
	var updatedRow GormHoliday
	if err := db.Where("date = ?", "2033-01-01").First(&updatedRow).Error; err != nil {
		t.Fatal(err)
	}
	if updatedRow.IsFestivalDay || updatedRow.Enabled || updatedRow.Note != "测试节调休" {
		t.Fatalf("updatedRow = %+v, want false flags and updated note", updatedRow)
	}

	if err := store.SetEnabled(ctx, "2033-01-01", false); err != nil {
		t.Fatal(err)
	}
	if err := store.SetEnabled(ctx, "2033-01-01", false); err != nil {
		t.Fatalf("SetEnabled() idempotent error = %v, want nil", err)
	}
	cal, err = NewCalendar(ctx, WithSource(store))
	if err != nil {
		t.Fatal(err)
	}
	disabled := cal.Lookup(time.Date(2033, time.January, 1, 9, 0, 0, 0, time.Local))
	if disabled.Name != "" || disabled.Kind != DayKindWeekend {
		t.Fatalf("disabled = %+v, want weekend fallback", disabled)
	}

	entries, err := store.List(ctx, 2033, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Date != "2033-01-01" || entries[0].Enabled || entries[0].Entry.IsFestivalDay || entries[0].Entry.Note != "测试节调休" {
		t.Fatalf("entries = %+v, want disabled updated non-festival entry", entries)
	}
	activeEntries, err := store.List(ctx, 2033, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(activeEntries) != 0 {
		t.Fatalf("activeEntries = %+v, want empty", activeEntries)
	}
}

func TestCalendarWithGormSourceListFestivalsUsesSnapshot(t *testing.T) {
	db := newHolidayTestDB(t)
	store := NewGormSource(db, WithGormInitEmbedded(false))
	ctx := context.Background()
	rows := []GormHoliday{
		{Date: "2036-01-01", Name: "测试节", Type: DayTypeHoliday, IsFestivalDay: true, Note: "测试节", Enabled: true},
		{Date: "2036-01-02", Name: "测试节", Type: DayTypeHoliday, Note: "测试节", Enabled: true},
		{Date: "2036-01-04", Name: "测试节", Type: DayTypeWorkday, Note: "测试节补班", Enabled: true},
		{Date: "2036-01-05", Name: "测试节", Type: DayTypeHoliday, Note: "禁用假日", Enabled: false},
		{Date: "2036-02-01", Name: "其他节", Type: DayTypeHoliday, IsFestivalDay: true, Note: "其他节", Enabled: true},
	}
	if err := db.AutoMigrate(&GormHoliday{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}

	cal, err := NewCalendar(ctx, WithSource(store))
	if err != nil {
		t.Fatal(err)
	}
	infos, err := cal.ListFestivals(ctx, ListFestivalsReq{StartYear: 2036, EndYear: 2036, Name: "测试节"})
	if err != nil {
		t.Fatal(err)
	}
	if len(infos) != 1 {
		t.Fatalf("len(infos) = %d, want 1: %+v", len(infos), infos)
	}
	info := infos[0]
	if info.Name != "测试节" || info.StartDate != "2036-01-01" || info.EndDate != "2036-01-02" {
		t.Fatalf("info = %+v, want enabled 测试节 range", info)
	}
	if !reflect.DeepEqual(info.HolidayDays, []string{"2036-01-01", "2036-01-02"}) {
		t.Fatalf("HolidayDays = %v, want enabled holidays only", info.HolidayDays)
	}
	if !reflect.DeepEqual(info.MakeupWorkdays, []string{"2036-01-04"}) {
		t.Fatalf("MakeupWorkdays = %v, want enabled makeup workday", info.MakeupWorkdays)
	}
	if !reflect.DeepEqual(info.FestivalDays, []string{"2036-01-01"}) {
		t.Fatalf("FestivalDays = %v, want enabled festival day", info.FestivalDays)
	}
}

func TestGormSourceSetEnabledMissingRow(t *testing.T) {
	db := newHolidayTestDB(t)
	store := NewGormSource(db, WithGormInitEmbedded(false))
	if err := store.SetEnabled(context.Background(), "2034-01-01", true); !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Fatalf("SetEnabled() error = %v, want gorm.ErrRecordNotFound", err)
	}
}

func TestCalendarReloadReflectsStoreChanges(t *testing.T) {
	db := newHolidayTestDB(t)
	store := NewGormSource(db, WithGormInitEmbedded(false))
	cal, err := NewCalendar(context.Background(), WithSource(store))
	if err != nil {
		t.Fatal(err)
	}
	if info := cal.Lookup(time.Date(2035, time.January, 1, 9, 0, 0, 0, time.UTC)); info.Kind != DayKindNormalWorkday {
		t.Fatalf("before save info = %+v, want normal workday", info)
	}
	if err := store.Save(context.Background(), StoredEntry{Date: "2035-01-01", Entry: Entry{Name: "测试节", Type: DayTypeHoliday, IsFestivalDay: true, Note: "测试节"}, Enabled: true}); err != nil {
		t.Fatal(err)
	}
	if info := cal.Lookup(time.Date(2035, time.January, 1, 9, 0, 0, 0, time.UTC)); info.Kind != DayKindNormalWorkday {
		t.Fatalf("before reload info = %+v, want stale normal workday", info)
	}
	if err := cal.Reload(context.Background()); err != nil {
		t.Fatal(err)
	}
	if info := cal.Lookup(time.Date(2035, time.January, 1, 9, 0, 0, 0, time.UTC)); info.Kind != DayKindStatutoryHoliday || info.Name != "测试节" {
		t.Fatalf("after reload info = %+v, want updated holiday", info)
	}
}

func TestGormSourceCanDisableEmbeddedInitialization(t *testing.T) {
	db := newHolidayTestDB(t)

	cal, err := NewCalendar(context.Background(), WithSource(NewGormSource(db, WithGormInitEmbedded(false))))
	if err != nil {
		t.Fatal(err)
	}
	info := cal.Lookup(time.Date(2026, time.October, 1, 9, 0, 0, 0, time.UTC))
	if info.Kind != DayKindNormalWorkday {
		t.Fatalf("info = %+v, want normal workday fallback", info)
	}
}

func TestGormSourceRequiresDB(t *testing.T) {
	_, err := NewCalendar(context.Background(), WithSource(NewGormSource(nil)))
	if err == nil {
		t.Fatal("NewCalendar() error = nil, want nil db error")
	}
}

func TestEmbeddedDataDeclaresNames(t *testing.T) {
	files, err := embeddedData.ReadDir("data")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
			continue
		}
		data, err := embeddedData.ReadFile("data/" + file.Name())
		if err != nil {
			t.Fatal(err)
		}
		var entries map[string]Entry
		if err := json.Unmarshal(data, &entries); err != nil {
			t.Fatalf("%s: %v", file.Name(), err)
		}
		for date, entry := range entries {
			if entry.Name == "" {
				t.Fatalf("%s %s missing name", file.Name(), date)
			}
			if err := validateEntry(date, entry); err != nil {
				t.Fatalf("%s %s invalid entry: %v", file.Name(), date, err)
			}
		}
	}
}

func newHolidayTestDB(t *testing.T) *gormx.DB {
	t.Helper()
	dialector := gorm.Dialector(sqlite.Open("file:" + t.Name() + "?mode=memory&cache=shared"))
	db, err := gormx.OpenWithDialector(&dialector, gormx.WithoutOpenTelemetry())
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
