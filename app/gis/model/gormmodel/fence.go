package gormmodel

import "zero-service/common/gormx"

// GisFence 电子围栏主表
type GisFence struct {
	gormx.LegacyIDMixin
	gormx.LegacyTimeMixin
	FenceId          string `gorm:"column:fence_id;type:varchar(36);uniqueIndex;not null;comment:围栏业务ID"`
	Name             string `gorm:"column:name;type:varchar(255);not null;default:''"`
	Points           string `gorm:"column:points;type:text;not null;comment:多边形顶点JSON [[lon,lat],...]"`
	H3Resolution     int    `gorm:"column:h3_resolution;not null;default:9"`
	GeohashPrecision int    `gorm:"column:geohash_precision;not null;default:7"`
}

func (GisFence) TableName() string { return "gis_fence" }

// GisFenceCell 围栏-Cell 映射表（用于反查）
type GisFenceCell struct {
	gormx.LegacyIDMixin
	FenceId  string `gorm:"column:fence_id;type:varchar(36);not null;index:idx_fence_cell_fence_id"`
	CellId   string `gorm:"column:cell_id;type:varchar(64);not null;index:idx_fence_cell_cell_id"`
	CellType string `gorm:"column:cell_type;type:varchar(10);not null;comment:h3 或 geohash"`
}

func (GisFenceCell) TableName() string { return "gis_fence_cell" }
