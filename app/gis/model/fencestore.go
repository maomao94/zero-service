package model

import (
	"context"
	"encoding/json"
	"fmt"

	"zero-service/app/gis/model/gormmodel"
	"zero-service/common/gisx"
	"zero-service/common/gormx"

	"github.com/mmcloughlin/geohash"
	"github.com/paulmach/orb"
	"gorm.io/gorm"
)

// GormFenceStore 基于 GORM 的 FenceStore 实现
type GormFenceStore struct {
	db *gormx.DB
}

func NewGormFenceStore(db *gormx.DB) *GormFenceStore {
	return &GormFenceStore{db: db}
}

func (s *GormFenceStore) CreateFence(ctx context.Context, fenceId, name string, points []orb.Point, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error {
	pointsJSON, err := json.Marshal(points)
	if err != nil {
		return fmt.Errorf("序列化多边形顶点失败: %w", err)
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

	if err := s.batchInsertCells(tx, fenceId, h3Cells, geohashCells); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit().Error
}

func (s *GormFenceStore) LoadFencePolygon(ctx context.Context, fenceId string) ([]orb.Point, error) {
	var fence gormmodel.GisFence
	if err := s.db.DB.WithContext(ctx).Where("fence_id = ?", fenceId).First(&fence).Error; err != nil {
		return nil, fmt.Errorf("围栏不存在: %s", fenceId)
	}

	var points []orb.Point
	if err := json.Unmarshal([]byte(fence.Points), &points); err != nil {
		return nil, fmt.Errorf("解析围栏顶点失败: %w", err)
	}
	return points, nil
}

func (s *GormFenceStore) FindNearbyFenceIds(ctx context.Context, lat, lon, km float64) ([]string, error) {
	precision := kmToGeohashPrecision(km)
	hash := geohash.EncodeWithPrecision(lat, lon, uint(precision))

	candidates := []string{hash}
	for _, n := range geohash.Neighbors(hash) {
		candidates = append(candidates, n)
	}
	exactMatches, likePatterns := geohashLookupKeys(candidates)

	var cellRecords []gormmodel.GisFenceCell
	query := s.db.DB.WithContext(ctx).Where("cell_type = ? AND cell_id IN ?", "geohash", exactMatches)
	for _, pattern := range likePatterns {
		query = query.Or("cell_type = ? AND cell_id LIKE ?", "geohash", pattern)
	}
	if err := query.Find(&cellRecords).Error; err != nil {
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

func (s *GormFenceStore) UpdateFence(ctx context.Context, fenceId, name string, points []orb.Point, h3Resolution, geohashPrecision int, h3Cells, geohashCells []string) error {
	pointsJSON, err := json.Marshal(points)
	if err != nil {
		return fmt.Errorf("序列化多边形顶点失败: %w", err)
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

	if err := s.batchInsertCells(tx, fenceId, h3Cells, geohashCells); err != nil {
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

func (s *GormFenceStore) batchInsertCells(tx *gorm.DB, fenceId string, h3Cells, geohashCells []string) error {
	cells := make([]gormmodel.GisFenceCell, 0, len(h3Cells)+len(geohashCells))
	for _, c := range h3Cells {
		cells = append(cells, gormmodel.GisFenceCell{FenceId: fenceId, CellId: c, CellType: "h3"})
	}
	for _, c := range geohashCells {
		cells = append(cells, gormmodel.GisFenceCell{FenceId: fenceId, CellId: c, CellType: "geohash"})
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
		var pts []orb.Point
		_ = json.Unmarshal([]byte(f.Points), &pts)
		cells := cellMap[f.FenceId]
		list[i] = gisx.FenceInfo{
			FenceId:          f.FenceId,
			Name:             f.Name,
			Points:           pts,
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

	var pts []orb.Point
	_ = json.Unmarshal([]byte(fence.Points), &pts)

	h3Cells, geohashes := s.loadCellsByFenceId(ctx, fenceId)

	return &gisx.FenceInfo{
		FenceId:          fence.FenceId,
		Name:             fence.Name,
		Points:           pts,
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
		case "h3":
			fc.h3 = append(fc.h3, c.CellId)
		case "geohash":
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
		case "h3":
			h3Cells = append(h3Cells, c.CellId)
		case "geohash":
			geohashes = append(geohashes, c.CellId)
		}
	}
	return
}

func geohashLookupKeys(candidates []string) ([]string, []string) {
	exactSet := make(map[string]struct{}, len(candidates)*3)
	likeSet := make(map[string]struct{}, len(candidates))
	for _, candidate := range candidates {
		// 同精度 exact match
		exactSet[candidate] = struct{}{}
		// 入库精度更细时，LIKE 匹配多余尾串
		likeSet[candidate+"%"] = struct{}{}
		// 入库精度更粗时，最多回退 2 级前缀（超过则空间跨度过大，不适合"附近"过滤）
		for i := len(candidate) - 1; i >= max(1, len(candidate)-2); i-- {
			exactSet[candidate[:i]] = struct{}{}
		}
	}

	exactMatches := make([]string, 0, len(exactSet))
	for key := range exactSet {
		exactMatches = append(exactMatches, key)
	}
	likePatterns := make([]string, 0, len(likeSet))
	for key := range likeSet {
		likePatterns = append(likePatterns, key)
	}
	return exactMatches, likePatterns
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
