package logic

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"zero-service/app/trigger/internal/svc"
	"zero-service/app/trigger/trigger"
	"zero-service/common/holiday"
)

func TestSaveHolidaySourceReloadsCalendar(t *testing.T) {
	ctx := context.Background()
	store := newMemoryHolidayStore(nil)
	cal, err := holiday.NewCalendar(ctx, holiday.WithSource(store))
	if err != nil {
		t.Fatal(err)
	}
	logic := NewSaveHolidaySourceLogic(ctx, &svc.ServiceContext{HolidaySource: store, HolidayCalendar: cal})

	if info := cal.Lookup(time.Date(2037, time.January, 1, 9, 0, 0, 0, time.UTC)); info.Kind != holiday.DayKindNormalWorkday {
		t.Fatalf("before save info = %+v, want normal workday", info)
	}
	_, err = logic.SaveHolidaySource(&trigger.SaveHolidaySourceReq{Date: "2037-01-01", Name: "测试节", Type: string(holiday.DayTypeHoliday), Note: "测试节", IsFestivalDay: true, Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if info := cal.Lookup(time.Date(2037, time.January, 1, 9, 0, 0, 0, time.UTC)); info.Kind != holiday.DayKindStatutoryHoliday || info.Name != "测试节" || !info.IsFestivalDay {
		t.Fatalf("after save info = %+v, want reloaded 测试节 holiday", info)
	}
}

func TestSetHolidaySourceEnabledReloadsCalendar(t *testing.T) {
	ctx := context.Background()
	store := newMemoryHolidayStore(map[string]holiday.StoredEntry{
		"2037-01-01": {Date: "2037-01-01", Entry: holiday.Entry{Name: "测试节", Type: holiday.DayTypeHoliday, IsFestivalDay: true, Note: "测试节"}, Enabled: true},
	})
	cal, err := holiday.NewCalendar(ctx, holiday.WithSource(store))
	if err != nil {
		t.Fatal(err)
	}
	logic := NewSetHolidaySourceEnabledLogic(ctx, &svc.ServiceContext{HolidaySource: store, HolidayCalendar: cal})

	if info := cal.Lookup(time.Date(2037, time.January, 1, 9, 0, 0, 0, time.UTC)); info.Kind != holiday.DayKindStatutoryHoliday {
		t.Fatalf("before disable info = %+v, want holiday", info)
	}
	_, err = logic.SetHolidaySourceEnabled(&trigger.SetHolidaySourceEnabledReq{Date: "2037-01-01", Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if info := cal.Lookup(time.Date(2037, time.January, 1, 9, 0, 0, 0, time.UTC)); info.Kind != holiday.DayKindNormalWorkday || info.Name != "" {
		t.Fatalf("after disable info = %+v, want reloaded normal workday fallback", info)
	}
}

type memoryHolidayStore struct {
	mu    sync.RWMutex
	items map[string]holiday.StoredEntry
}

func newMemoryHolidayStore(items map[string]holiday.StoredEntry) *memoryHolidayStore {
	store := &memoryHolidayStore{items: make(map[string]holiday.StoredEntry, len(items))}
	for date, item := range items {
		store.items[date] = item
	}
	return store
}

func (s *memoryHolidayStore) Load(context.Context) (map[string]holiday.Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make(map[string]holiday.Entry, len(s.items))
	for date, item := range s.items {
		if item.Enabled {
			entries[date] = item.Entry
		}
	}
	return entries, nil
}

func (s *memoryHolidayStore) Save(_ context.Context, item holiday.StoredEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.items[item.Date] = item
	return nil
}

func (s *memoryHolidayStore) SetEnabled(_ context.Context, date string, enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	item := s.items[date]
	item.Enabled = enabled
	s.items[date] = item
	return nil
}

func (s *memoryHolidayStore) List(_ context.Context, year int, includeDisabled bool) ([]holiday.StoredEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	items := make([]holiday.StoredEntry, 0, len(s.items))
	prefix := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Format("2006")
	for date, item := range s.items {
		if len(date) >= 4 && date[:4] == prefix && (includeDisabled || item.Enabled) {
			items = append(items, item)
		}
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Date < items[j].Date })
	return items, nil
}
