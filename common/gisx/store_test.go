package gisx

import (
	"context"
	"errors"
	"testing"

	"github.com/paulmach/orb"
)

func TestErrFenceStoreNotImplemented(t *testing.T) {
	if ErrFenceStoreNotImplemented == nil {
		t.Fatal("ErrFenceStoreNotImplemented 不应为 nil")
	}
	if ErrFenceStoreNotImplemented.Error() == "" {
		t.Fatal("错误消息不应为空")
	}
}

func TestNoopFenceStore_CreateFence(t *testing.T) {
	s := &NoopFenceStore{}
	err := s.CreateFence(context.Background(), "f1", "test", orb.Polygon{orb.Ring{{0, 0}, {1, 0}, {0, 0}}}, 9, 7, nil, nil)
	if !errors.Is(err, ErrFenceStoreNotImplemented) {
		t.Errorf("期望 ErrFenceStoreNotImplemented，得到 %v", err)
	}
}

func TestNoopFenceStore_LoadFencePolygon(t *testing.T) {
	s := &NoopFenceStore{}
	poly, err := s.LoadFencePolygon(context.Background(), "f1")
	if !errors.Is(err, ErrFenceStoreNotImplemented) {
		t.Errorf("期望 ErrFenceStoreNotImplemented，得到 %v", err)
	}
	if poly != nil {
		t.Error("poly 应为 nil")
	}
}

func TestNoopFenceStore_FindNearbyFenceIds(t *testing.T) {
	s := &NoopFenceStore{}
	ids, err := s.FindNearbyFenceIds(context.Background(), 116.39, 39.9, 5)
	if !errors.Is(err, ErrFenceStoreNotImplemented) {
		t.Errorf("期望 ErrFenceStoreNotImplemented，得到 %v", err)
	}
	if ids != nil {
		t.Error("ids 应为 nil")
	}
}

func TestNoopFenceStore_FindFenceIdsByCellIds(t *testing.T) {
	s := &NoopFenceStore{}
	ids, err := s.FindFenceIdsByCellIds(context.Background(), []string{"cell1"})
	if !errors.Is(err, ErrFenceStoreNotImplemented) {
		t.Errorf("期望 ErrFenceStoreNotImplemented，得到 %v", err)
	}
	if ids != nil {
		t.Error("ids 应为 nil")
	}
}

func TestNoopFenceStore_UpdateFence(t *testing.T) {
	s := &NoopFenceStore{}
	err := s.UpdateFence(context.Background(), "f1", "test", orb.Polygon{orb.Ring{{0, 0}, {1, 0}, {0, 0}}}, 9, 7, nil, nil)
	if !errors.Is(err, ErrFenceStoreNotImplemented) {
		t.Errorf("期望 ErrFenceStoreNotImplemented，得到 %v", err)
	}
}

func TestNoopFenceStore_RemoveFence(t *testing.T) {
	s := &NoopFenceStore{}
	err := s.RemoveFence(context.Background(), "f1")
	if !errors.Is(err, ErrFenceStoreNotImplemented) {
		t.Errorf("期望 ErrFenceStoreNotImplemented，得到 %v", err)
	}
}

func TestNoopFenceStore_ListFences(t *testing.T) {
	s := &NoopFenceStore{}
	list, total, err := s.ListFences(context.Background(), 1, 20, "")
	if !errors.Is(err, ErrFenceStoreNotImplemented) {
		t.Errorf("期望 ErrFenceStoreNotImplemented，得到 %v", err)
	}
	if list != nil {
		t.Error("list 应为 nil")
	}
	if total != 0 {
		t.Error("total 应为 0")
	}
}

func TestNoopFenceStore_GetFence(t *testing.T) {
	s := &NoopFenceStore{}
	fence, err := s.GetFence(context.Background(), "f1")
	if !errors.Is(err, ErrFenceStoreNotImplemented) {
		t.Errorf("期望 ErrFenceStoreNotImplemented，得到 %v", err)
	}
	if fence != nil {
		t.Error("fence 应为 nil")
	}
}

func TestFenceInfo_PolygonType(t *testing.T) {
	fi := FenceInfo{
		Polygon: orb.Polygon{
			orb.Ring{{0, 0}, {4, 0}, {4, 4}, {0, 4}, {0, 0}},
			orb.Ring{{1, 1}, {3, 1}, {3, 3}, {1, 3}, {1, 1}},
		},
	}
	if len(fi.Polygon) != 2 {
		t.Fatalf("期望 2 个 ring（外环+洞），得到 %d", len(fi.Polygon))
	}
	if len(fi.Polygon[0]) != 5 {
		t.Errorf("外环期望 5 个点，得到 %d", len(fi.Polygon[0]))
	}
	if len(fi.Polygon[1]) != 5 {
		t.Errorf("洞期望 5 个点，得到 %d", len(fi.Polygon[1]))
	}
}
