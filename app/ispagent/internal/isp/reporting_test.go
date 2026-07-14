package isp

import (
	"testing"
	"time"

	protocol "zero-service/common/isp"
)

func TestApplyRegistrationIntervals(t *testing.T) {
	m := newReportManager()

	m.applyRegistrationIntervals([]protocol.Item{
		{"patroldevice_run_interval": "10"},
	})

	if got := m.intervals[ReportCategoryPatrolDeviceRunData]; got != 10*time.Second {
		t.Fatalf("run data interval = %s, want 10s", got)
	}
}

func TestParseItemInterval(t *testing.T) {
	if got := parseItemInterval(protocol.Item{}, "k", 60*time.Second); got != 60*time.Second {
		t.Fatalf("empty = %s, want 60s", got)
	}
	if got := parseItemInterval(protocol.Item{"k": "0"}, "k", 60*time.Second); got != 60*time.Second {
		t.Fatalf("zero = %s, want 60s", got)
	}
	if got := parseItemInterval(protocol.Item{"k": "bad"}, "k", 60*time.Second); got != 60*time.Second {
		t.Fatalf("bad = %s, want 60s", got)
	}
	if got := parseItemInterval(protocol.Item{"k": "-1"}, "k", 60*time.Second); got != 60*time.Second {
		t.Fatalf("neg = %s, want 60s", got)
	}
	if got := parseItemInterval(protocol.Item{"k": "10"}, "k", 0); got != 10*time.Second {
		t.Fatalf("valid = %s, want 10s", got)
	}
}

func TestReportManagerDueAndMarkSent(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second

	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{{"value": "42"}}, now)

	due := m.dueReports(now)
	if len(due) != 1 {
		t.Fatalf("due reports = %d, want 1", len(due))
	}

	m.markSent(ReportCategoryPatrolDeviceRunData, "station-1", now, time.Time{})
	if got := len(m.dueReports(now.Add(9 * time.Second))); got != 0 {
		t.Fatalf("due before interval = %d, want 0", got)
	}
	if got := len(m.dueReports(now.Add(10 * time.Second))); got != 1 {
		t.Fatalf("due at interval = %d, want 1", got)
	}
}

func TestReportManagerNewItemResetsLastSentAndDueImmediately(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second

	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "old"},
	}, now)
	m.markSent(ReportCategoryPatrolDeviceRunData, "station-1", now, time.Time{})

	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-2", "type": "1", "value": "new"},
	}, now.Add(2*time.Second))

	due := m.dueReports(now.Add(2 * time.Second))
	if len(due) != 1 {
		t.Fatalf("due reports after new item = %d, want 1", len(due))
	}
	if !containsItemValue(due[0].items, "new") {
		t.Fatal("new item was not included in immediate due report")
	}
}

func TestReportManagerExistingItemRefreshDoesNotResetLastSent(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second

	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "old"},
	}, now)
	m.markSent(ReportCategoryPatrolDeviceRunData, "station-1", now, time.Time{})

	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "refreshed"},
	}, now.Add(2*time.Second))

	if got := len(m.dueReports(now.Add(2 * time.Second))); got != 0 {
		t.Fatalf("due reports after existing item refresh = %d, want 0", got)
	}
	if got := len(m.dueReports(now.Add(10 * time.Second))); got != 1 {
		t.Fatalf("due reports at original interval = %d, want 1", got)
	}
}

func TestReportManagerSkipsStaleItemsOnDue(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{{"value": "42"}}, now)

	timeout := freshnessTimeout(10 * time.Second)
	if got := len(m.dueReports(now.Add(timeout - time.Second))); got != 1 {
		t.Fatalf("due before freshness = %d, want 1", got)
	}
	if got := len(m.dueReports(now.Add(timeout))); got != 0 {
		t.Fatalf("due after freshness = %d, want 0", got)
	}
}

func TestReportManagerRemovesExpiredItemsFromCache(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "old"},
		{"patroldevice_code": "robot-2", "type": "1", "value": "fresh"},
	}, now)
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-2", "type": "1", "value": "fresh"},
	}, now.Add(15*time.Second))

	m.dueReports(now.Add(freshnessTimeout(10 * time.Second)))

	report := m.cache[ReportCategoryPatrolDeviceRunData]["station-1"]
	if report == nil {
		t.Fatal("cached report was removed, want fresh item to remain")
	}
	if got := len(report.itemByKey); got != 1 {
		t.Fatalf("cached items after cleanup = %d, want 1", got)
	}
	if containsCachedItemValue(report.itemByKey, "old") {
		t.Fatal("expired item remained in cache")
	}
	if !containsCachedItemValue(report.itemByKey, "fresh") {
		t.Fatal("fresh item missing from cache")
	}
}

func TestReportManagerCleansExpiredItemsBeforeReportInterval(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = time.Minute
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "old"},
	}, now)
	m.markSent(ReportCategoryPatrolDeviceRunData, "station-1", now.Add(90*time.Second), time.Time{})

	if got := len(m.dueReports(now.Add(2 * time.Minute))); got != 0 {
		t.Fatalf("due reports before interval = %d, want 0", got)
	}
	if report := m.cache[ReportCategoryPatrolDeviceRunData]["station-1"]; report != nil {
		t.Fatalf("cached report remained after freshness cleanup before interval: %#v", report)
	}
}

func TestReportManagerRemovesCachedReportWhenAllItemsExpire(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "old"},
	}, now)

	m.dueReports(now.Add(freshnessTimeout(10 * time.Second)))

	if report := m.cache[ReportCategoryPatrolDeviceRunData]["station-1"]; report != nil {
		t.Fatalf("cached report remained after all items expired: %#v", report)
	}
}

func TestReportManagerDeleteExpiredSkipsConcurrentlyRefreshedItem(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "old"},
	}, now)

	key := itemKey(keyAttrsByCategory[ReportCategoryPatrolDeviceRunData], protocol.Item{"patroldevice_code": "robot-1", "type": "1"}, 0)
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "fresh"},
	}, now.Add(time.Second))
	m.deleteExpired([]expiredReportItem{{
		category:  ReportCategoryPatrolDeviceRunData,
		code:      "station-1",
		itemKey:   key,
		updatedAt: now,
	}}, nil)

	report := m.cache[ReportCategoryPatrolDeviceRunData]["station-1"]
	if report == nil {
		t.Fatal("refreshed cached report was deleted")
	}
	if !containsCachedItemValue(report.itemByKey, "fresh") {
		t.Fatal("refreshed item was deleted by stale expired ref")
	}
}

func TestReportManagerRemovesPreexistingEmptyCachedReport(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.cache[ReportCategoryPatrolDeviceRunData]["station-1"] = &cachedReport{itemByKey: map[string]*cachedItem{}}

	if got := len(m.dueReports(now)); got != 0 {
		t.Fatalf("due reports for empty cached report = %d, want 0", got)
	}
	if report := m.cache[ReportCategoryPatrolDeviceRunData]["station-1"]; report != nil {
		t.Fatalf("empty cached report remained after scan: %#v", report)
	}
}

func TestReportManagerNoFreshCheckDoesNotCleanStaleItems(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager(WithNoFreshCheck(ReportCategoryPatrolDeviceCoordinates))
	m.intervals[ReportCategoryPatrolDeviceCoordinates] = 10 * time.Second

	m.update(ReportCategoryPatrolDeviceCoordinates, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "coordinate_geography": "1,2"},
	}, now)

	due := m.dueReports(now.Add(freshnessTimeout(10 * time.Second)))
	if len(due) != 1 {
		t.Fatalf("due reports = %d, want 1", len(due))
	}
	report := m.cache[ReportCategoryPatrolDeviceCoordinates]["station-1"]
	if report == nil {
		t.Fatal("noFreshCheck cached report was cleaned")
	}
	if got := len(report.itemByKey); got != 1 {
		t.Fatalf("noFreshCheck cached items after scan = %d, want 1", got)
	}
}

func TestReportManagerStatusAndCoordinatesDefaultInterval(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager(WithNoFreshCheck(ReportCategoryPatrolDeviceCoordinates))

	m.update(ReportCategoryPatrolDeviceStatusData, "station-1", []protocol.Item{{"patroldevice_code": "robot-1", "type": "1", "value": "ok"}}, now)
	m.update(ReportCategoryPatrolDeviceCoordinates, "station-1", []protocol.Item{{"patroldevice_code": "robot-1", "coordinate_geography": "1,2"}}, now)

	// 坐标 noFreshCheck=true，状态默认 1 分钟
	if got := len(m.dueReports(now)); got != 2 {
		t.Fatalf("initial due reports = %d, want 2", got)
	}
	m.markSent(ReportCategoryPatrolDeviceStatusData, "station-1", now, time.Time{})
	m.markSent(ReportCategoryPatrolDeviceCoordinates, "station-1", now, time.Time{})

	// 坐标默认 2 秒间隔，状态默认 1 分钟
	if got := dueCountByCategory(m.dueReports(now.Add(defaultCoordInterval)), ReportCategoryPatrolDeviceCoordinates); got != 1 {
		t.Fatalf("coordinate due at 2s = %d, want 1", got)
	}
	if got := dueCountByCategory(m.dueReports(now.Add(defaultCoordInterval)), ReportCategoryPatrolDeviceStatusData); got != 0 {
		t.Fatalf("status due at 2s = %d, want 0", got)
	}
	// 1 分钟后状态到间隔
	if got := len(m.dueReports(now.Add(time.Minute))); got != 2 {
		t.Fatalf("both due at 1min = %d, want 2", got)
	}
}

func TestReportManagerClonesItems(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second
	items := []protocol.Item{{"value": "42"}}
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", items, now)
	items[0]["value"] = "changed"

	due := m.dueReports(now)
	if len(due) != 1 {
		t.Fatalf("due reports = %d, want 1", len(due))
	}
	if got := due[0].items[0]["value"]; got != "42" {
		t.Fatalf("cached value = %s, want 42", got)
	}
	due[0].items[0]["value"] = "mutated"
	if got := m.dueReports(now)[0].items[0]["value"]; got != "42" {
		t.Fatalf("snapshot mutation changed cache: %s", got)
	}
}

func TestReportManagerSeparateByCodeAndItemKey(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second

	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "11"},
		{"patroldevice_code": "robot-2", "type": "1", "value": "21"},
	}, now)
	m.update(ReportCategoryPatrolDeviceRunData, "station-2", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "31"},
	}, now)
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "12"},
	}, now.Add(time.Second))

	due := m.dueReports(now.Add(time.Second))
	if len(due) != 2 {
		t.Fatalf("due reports = %d, want 2", len(due))
	}

	itemsByCode := map[string][]protocol.Item{}
	for _, report := range due {
		itemsByCode[report.code] = report.items
	}
	if got := len(itemsByCode["station-1"]); got != 2 {
		t.Fatalf("station-1 items = %d, want 2", got)
	}
	if got := len(itemsByCode["station-2"]); got != 1 {
		t.Fatalf("station-2 items = %d, want 1", got)
	}
	if !containsItemValue(itemsByCode["station-1"], "12") {
		t.Fatal("station-1 missing updated robot-1 value")
	}
	if containsItemValue(itemsByCode["station-1"], "11") {
		t.Fatal("station-1 kept stale robot-1 value")
	}
}

func TestReportManagerStalePerItem(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second

	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "type": "1", "value": "old"},
	}, now)
	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-2", "type": "1", "value": "fresh"},
	}, now.Add(15*time.Second))

	timeout := freshnessTimeout(10 * time.Second)
	due := m.dueReports(now.Add(timeout))
	if len(due) != 1 {
		t.Fatalf("due reports = %d, want 1", len(due))
	}
	if containsItemValue(due[0].items, "old") {
		t.Fatal("stale old item was included in due report")
	}
	if !containsItemValue(due[0].items, "fresh") {
		t.Fatal("fresh item was not included in due report")
	}
}

func TestReportManagerIncompleteKeyAttrs(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceRunData] = 10 * time.Second

	m.update(ReportCategoryPatrolDeviceRunData, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "value": "11"},
		{"patroldevice_code": "robot-1", "value": "12"},
	}, now)

	due := m.dueReports(now)
	if got := len(due); got != 1 {
		t.Fatalf("due reports = %d, want 1", got)
	}
	if got := len(due[0].items); got != 2 {
		t.Fatalf("items = %d, want 2", got)
	}
}

func TestReportManagerNoFreshCheckReportsStaleItems(t *testing.T) {
	now := time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC)
	m := newReportManager()
	m.intervals[ReportCategoryPatrolDeviceCoordinates] = 10 * time.Second
	m.setNoFreshCheck(ReportCategoryPatrolDeviceCoordinates, true)

	m.update(ReportCategoryPatrolDeviceCoordinates, "station-1", []protocol.Item{
		{"patroldevice_code": "robot-1", "coordinate_geography": "1,2"},
	}, now)

	timeout := freshnessTimeout(10 * time.Second)
	due := m.dueReports(now.Add(timeout))
	if len(due) != 1 {
		t.Fatalf("due reports = %d, want 1", len(due))
	}
	if !containsItemAttr(due[0].items, "coordinate_geography", "1,2") {
		t.Fatal("stale item not included when noFreshCheck is set")
	}

	m.setNoFreshCheck(ReportCategoryPatrolDeviceCoordinates, false)
	if got := len(m.dueReports(now.Add(timeout))); got != 0 {
		t.Fatalf("due reports after disabling noFreshCheck = %d, want 0", got)
	}
}

func containsItemValue(items []protocol.Item, value string) bool {
	for _, item := range items {
		if item["value"] == value {
			return true
		}
	}
	return false
}

func dueCountByCategory(reports []reportSnapshot, category ReportCategory) int {
	count := 0
	for _, report := range reports {
		if report.category == category {
			count++
		}
	}
	return count
}

func containsItemAttr(items []protocol.Item, attr, value string) bool {
	for _, item := range items {
		if item[attr] == value {
			return true
		}
	}
	return false
}

func containsCachedItemValue(items map[string]*cachedItem, value string) bool {
	for _, item := range items {
		if item.item["value"] == value {
			return true
		}
	}
	return false
}
