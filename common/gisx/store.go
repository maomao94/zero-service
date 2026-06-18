package gisx

import (
	"context"
	"errors"
	"time"

	"github.com/paulmach/orb"
)

var ErrFenceStoreNotImplemented = errors.New("FenceStore 未实现，请注入具体实现")

// FenceInfo 围栏详情（store 层返回的通用结构）
type FenceInfo struct {
	FenceId          string
	Name             string
	Points           []orb.Point
	H3Resolution     int
	GeohashPrecision int
	H3Cells          []string
	Geohashes        []string
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// FenceStore 电子围栏数据存取接口。
// 具体实现可基于 Postgres/PostGIS、Redis 或内存存储。
type FenceStore interface {
	// CreateFence 创建围栏，同时保存多边形顶点、索引精度和覆盖的 H3 + geohash cells。
	CreateFence(ctx context.Context, fenceId, name string, points []orb.Point, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error

	// LoadFencePolygon 按 fence_id 加载围栏多边形顶点。
	LoadFencePolygon(ctx context.Context, fenceId string) ([]orb.Point, error)

	// FindNearbyFenceIds 查询附近 km 范围内的围栏 ID（粗过滤）。
	FindNearbyFenceIds(ctx context.Context, lat, lon, km float64) ([]string, error)

	// FindFenceIdsByCellIds 按 cell ID 反查关联的围栏 ID。
	FindFenceIdsByCellIds(ctx context.Context, cellIDs []string) ([]string, error)

	// UpdateFence 更新围栏多边形、索引精度和 cells（覆盖写入）。
	UpdateFence(ctx context.Context, fenceId, name string, points []orb.Point, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error

	// RemoveFence 删除围栏及其关联的 cell 映射。
	RemoveFence(ctx context.Context, fenceId string) error

	// ListFences 分页查询围栏列表。
	ListFences(ctx context.Context, page, pageSize int64, name string) ([]FenceInfo, int64, error)

	// GetFence 按 fence_id 获取围栏详情。
	GetFence(ctx context.Context, fenceId string) (*FenceInfo, error)
}

// NoopFenceStore 空实现，所有方法返回 ErrFenceStoreNotImplemented。
// 生产环境需替换为真实的存储实现。
type NoopFenceStore struct{}

func (s *NoopFenceStore) CreateFence(ctx context.Context, fenceId, name string, points []orb.Point, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error {
	return ErrFenceStoreNotImplemented
}

func (s *NoopFenceStore) LoadFencePolygon(ctx context.Context, fenceId string) ([]orb.Point, error) {
	return nil, ErrFenceStoreNotImplemented
}

func (s *NoopFenceStore) FindNearbyFenceIds(ctx context.Context, lat, lon, km float64) ([]string, error) {
	return nil, ErrFenceStoreNotImplemented
}

func (s *NoopFenceStore) FindFenceIdsByCellIds(ctx context.Context, cellIDs []string) ([]string, error) {
	return nil, ErrFenceStoreNotImplemented
}

func (s *NoopFenceStore) UpdateFence(ctx context.Context, fenceId, name string, points []orb.Point, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error {
	return ErrFenceStoreNotImplemented
}

func (s *NoopFenceStore) RemoveFence(ctx context.Context, fenceId string) error {
	return ErrFenceStoreNotImplemented
}

func (s *NoopFenceStore) ListFences(ctx context.Context, page, pageSize int64, name string) ([]FenceInfo, int64, error) {
	return nil, 0, ErrFenceStoreNotImplemented
}

func (s *NoopFenceStore) GetFence(ctx context.Context, fenceId string) (*FenceInfo, error) {
	return nil, ErrFenceStoreNotImplemented
}
