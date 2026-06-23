package model

import (
	"context"
	"encoding/json"
	"fmt"
	"math"

	"zero-service/app/gis/model/gormmodel"
	"zero-service/common/gisx"
	"zero-service/common/gormx"

	"github.com/paulmach/orb"
	"github.com/uber/h3-go/v4"
	"gorm.io/gorm"
)

const (
	h3CellType             = "h3"
	geohashCellType        = "geohash"
	h3RecallResolution     = 9
	h3RecallCellType       = "h3_r9"
	h3RecallAverageEdgeKm  = 0.2
	h3PolygonMaxCellBudget = 1000
)

// GormFenceStore 基于 GORM 的 FenceStore 实现
type GormFenceStore struct {
	db *gormx.DB
}

func NewGormFenceStore(db *gormx.DB) *GormFenceStore {
	return &GormFenceStore{db: db}
}

func (s *GormFenceStore) CreateFence(ctx context.Context, fenceId, name string, polygon orb.Polygon, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error {
	pointsJSON, err := json.Marshal(polygon)
	if err != nil {
		return fmt.Errorf("序列化多边形顶点失败: %w", err)
	}
	recallH3Cells, err := computeH3RecallCells(polygon)
	if err != nil {
		return fmt.Errorf("生成召回 H3 cells 失败: %w", err)
	}

	fence := &gormmodel.GisFence{
		FenceId:          fenceId,
		Name:             name,
		Points:           string(pointsJSON),
		H3Resolution:     h3Resolution,
		GeohashPrecision: geohashPrecision,
	}

	tx := s.db.DB.WithContext(ctx).Begin()
	if err := tx.Create(fence).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("创建围栏记录失败: %w", err)
	}

	if err := s.batchInsertCells(tx, fenceId, h3Cells, geohashCells, recallH3Cells); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *GormFenceStore) LoadFencePolygon(ctx context.Context, fenceId string) (orb.Polygon, error) {
	var fence gormmodel.GisFence
	if err := s.db.DB.WithContext(ctx).Where("fence_id = ?", fenceId).First(&fence).Error; err != nil {
		return nil, fmt.Errorf("围栏不存在: %s", fenceId)
	}

	var polygon orb.Polygon
	if err := json.Unmarshal([]byte(fence.Points), &polygon); err != nil {
		return nil, fmt.Errorf("解析围栏顶点失败: %w", err)
	}
	return polygon, nil
}

func (s *GormFenceStore) FindNearbyFenceIds(ctx context.Context, lon, lat, km float64) ([]string, error) {
	k := kmToH3RecallK(km)
	origin, err := h3.LatLngToCell(h3.NewLatLng(lat, lon), h3RecallResolution)
	if err != nil {
		return nil, err
	}
	candidates, err := h3.GridDisk(origin, k)
	if err != nil {
		return nil, err
	}
	cellIds := make([]string, 0, len(candidates))
	for _, cell := range candidates {
		cellIds = append(cellIds, cell.String())
	}

	var cellRecords []gormmodel.GisFenceCell
	if err := s.db.DB.WithContext(ctx).
		Where("cell_type = ? AND cell_id IN ?", h3RecallCellType, cellIds).
		Find(&cellRecords).Error; err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, r := range cellRecords {
		seen[r.FenceId] = struct{}{}
	}

	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *GormFenceStore) FindFenceIdsByCellIds(ctx context.Context, cellIDs []string) ([]string, error) {
	var cellRecords []gormmodel.GisFenceCell
	if err := s.db.DB.WithContext(ctx).
		Where("cell_id IN ?", cellIDs).
		Find(&cellRecords).Error; err != nil {
		return nil, err
	}

	seen := make(map[string]struct{})
	for _, r := range cellRecords {
		seen[r.FenceId] = struct{}{}
	}

	ids := make([]string, 0, len(seen))
	for id := range seen {
		ids = append(ids, id)
	}
	return ids, nil
}

func (s *GormFenceStore) UpdateFence(ctx context.Context, fenceId, name string, polygon orb.Polygon, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error {
	pointsJSON, err := json.Marshal(polygon)
	if err != nil {
		return fmt.Errorf("序列化多边形顶点失败: %w", err)
	}
	recallH3Cells, err := computeH3RecallCells(polygon)
	if err != nil {
		return fmt.Errorf("生成召回 H3 cells 失败: %w", err)
	}

	tx := s.db.DB.WithContext(ctx).Begin()

	updates := map[string]interface{}{
		"points":            string(pointsJSON),
		"h3_resolution":     h3Resolution,
		"geohash_precision": geohashPrecision,
	}
	if name != "" {
		updates["name"] = name
	}
	if err := tx.Model(&gormmodel.GisFence{}).Where("fence_id = ?", fenceId).Updates(updates).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("更新围栏记录失败: %w", err)
	}

	if err := tx.Where("fence_id = ?", fenceId).Delete(&gormmodel.GisFenceCell{}).Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("清理旧 cells 失败: %w", err)
	}

	if err := s.batchInsertCells(tx, fenceId, h3Cells, geohashCells, recallH3Cells); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *GormFenceStore) RemoveFence(ctx context.Context, fenceId string) error {
	tx := s.db.DB.WithContext(ctx).Begin()

	if err := tx.Where("fence_id = ?", fenceId).Delete(&gormmodel.GisFenceCell{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	if err := tx.Where("fence_id = ?", fenceId).Delete(&gormmodel.GisFence{}).Error; err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *GormFenceStore) batchInsertCells(tx *gorm.DB, fenceId string, h3Cells, geohashCells, recallH3Cells []string) error {
	cells := make([]gormmodel.GisFenceCell, 0, len(h3Cells)+len(geohashCells)+len(recallH3Cells))
	for _, c := range h3Cells {
		cells = append(cells, gormmodel.GisFenceCell{FenceId: fenceId, CellId: c, CellType: h3CellType})
	}
	for _, c := range geohashCells {
		cells = append(cells, gormmodel.GisFenceCell{FenceId: fenceId, CellId: c, CellType: geohashCellType})
	}
	for _, c := range recallH3Cells {
		cells = append(cells, gormmodel.GisFenceCell{FenceId: fenceId, CellId: c, CellType: h3RecallCellType})
	}

	if len(cells) > 0 {
		if err := tx.CreateInBatches(cells, 500).Error; err != nil {
			return fmt.Errorf("批量插入 cells 失败: %w", err)
		}
	}
	return nil
}

func (s *GormFenceStore) ListFences(ctx context.Context, page, pageSize int64, name string) ([]gisx.FenceInfo, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	query := s.db.DB.WithContext(ctx).Model(&gormmodel.GisFence{})
	if name != "" {
		query = query.Where("name LIKE ?", "%"+name+"%")
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var fences []gormmodel.GisFence
	if err := query.Offset(int((page - 1) * pageSize)).Limit(int(pageSize)).
		Order("created_at DESC").Find(&fences).Error; err != nil {
		return nil, 0, err
	}

	fenceIds := make([]string, len(fences))
	for i, f := range fences {
		fenceIds[i] = f.FenceId
	}

	cellMap := s.batchLoadCells(ctx, fenceIds)

	list := make([]gisx.FenceInfo, len(fences))
	for i, f := range fences {
		var poly orb.Polygon
		_ = json.Unmarshal([]byte(f.Points), &poly)
		cells := cellMap[f.FenceId]
		list[i] = gisx.FenceInfo{
			FenceId:          f.FenceId,
			Name:             f.Name,
			Polygon:          poly,
			H3Resolution:     f.H3Resolution,
			GeohashPrecision: f.GeohashPrecision,
			H3Cells:          cells.h3,
			Geohashes:        cells.geohash,
			CreatedAt:        f.CreateTime,
			UpdatedAt:        f.UpdateTime,
		}
	}
	return list, total, nil
}

func (s *GormFenceStore) GetFence(ctx context.Context, fenceId string) (*gisx.FenceInfo, error) {
	var fence gormmodel.GisFence
	if err := s.db.DB.WithContext(ctx).Where("fence_id = ?", fenceId).First(&fence).Error; err != nil {
		return nil, fmt.Errorf("围栏不存在: %s", fenceId)
	}

	var poly orb.Polygon
	_ = json.Unmarshal([]byte(fence.Points), &poly)

	h3Cells, geohashes := s.loadCellsByFenceId(ctx, fenceId)

	return &gisx.FenceInfo{
		FenceId:          fence.FenceId,
		Name:             fence.Name,
		Polygon:          poly,
		H3Resolution:     fence.H3Resolution,
		GeohashPrecision: fence.GeohashPrecision,
		H3Cells:          h3Cells,
		Geohashes:        geohashes,
		CreatedAt:        fence.CreateTime,
		UpdatedAt:        fence.UpdateTime,
	}, nil
}

type fenceCells struct {
	h3      []string
	geohash []string
}

func (s *GormFenceStore) batchLoadCells(ctx context.Context, fenceIds []string) map[string]fenceCells {
	result := make(map[string]fenceCells, len(fenceIds))
	if len(fenceIds) == 0 {
		return result
	}
	var cells []gormmodel.GisFenceCell
	s.db.DB.WithContext(ctx).Where("fence_id IN ?", fenceIds).Find(&cells)
	for _, c := range cells {
		fc := result[c.FenceId]
		switch c.CellType {
		case h3CellType:
			fc.h3 = append(fc.h3, c.CellId)
		case geohashCellType:
			fc.geohash = append(fc.geohash, c.CellId)
		}
		result[c.FenceId] = fc
	}
	return result
}

func (s *GormFenceStore) loadCellsByFenceId(ctx context.Context, fenceId string) (h3Cells, geohashes []string) {
	var cells []gormmodel.GisFenceCell
	s.db.DB.WithContext(ctx).Where("fence_id = ?", fenceId).Find(&cells)
	for _, c := range cells {
		switch c.CellType {
		case h3CellType:
			h3Cells = append(h3Cells, c.CellId)
		case geohashCellType:
			geohashes = append(geohashes, c.CellId)
		}
	}
	return
}

func computeH3RecallCells(polygon orb.Polygon) ([]string, error) {
	geoPolygon, err := gisx.OrbPolygonToH3GeoPolygon(polygon)
	if err != nil {
		return nil, err
	}
	cells, err := h3.PolygonToCellsExperimental(geoPolygon, h3RecallResolution, h3.ContainmentOverlapping, h3PolygonMaxCellBudget)
	if err != nil {
		return nil, err
	}
	cellStrings := make([]string, len(cells))
	for i, c := range cells {
		cellStrings[i] = c.String()
	}
	return cellStrings, nil
}

func kmToH3RecallK(km float64) int {
	k := int(math.Ceil(km / h3RecallAverageEdgeKm))
	if k < 1 {
		k = 1
	}
	return k
}

func kmToGeohashPrecision(km float64) int {
	switch {
	case km > 1000:
		return 2
	case km > 100:
		return 3
	case km > 10:
		return 4
	case km > 1:
		return 5
	default:
		return 6
	}
}
