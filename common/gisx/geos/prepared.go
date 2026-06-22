package geos

import (
	"fmt"

	gogeos "github.com/twpayne/go-geos"
)

// PreparedGeom 是 Prepared Geometry 的封装。适用于一个固定几何对大量候选做循环判定。
// 注意：传入的几何必须由 geos 包构造（使用同一个默认 Context），否则 GEOS 会 panic。
// prepare 后的谓词调用会缓存在 GEOS 内部 R-Tree，性能远优于直接调用原子谓词。
type PreparedGeom struct {
	geom *gogeos.Geom
	prep *gogeos.PrepGeom
}

// NewPreparedGeom 从 *gogeos.Geom 构造 PreparedGeom。
// 几何必须为 geos 包默认 Context 构造，否则内部会 panic（已被 prepRun recover）。
func NewPreparedGeom(g *gogeos.Geom) (*PreparedGeom, error) {
	if g == nil {
		return nil, fmt.Errorf("geometry 为 nil")
	}
	return &PreparedGeom{geom: g, prep: g.Prepare()}, nil
}

// Close 释放底层引用。重复调用安全。
// go-geos 通过 runtime.AddCleanup 自动管理 C 内存，此处仅置空 Go 引用帮助 GC。
func (p *PreparedGeom) Close() {
	if p != nil {
		p.prep = nil
		p.geom = nil
	}
}

func (p *PreparedGeom) prepRun(fn func() bool) (bool, error) {
	if p == nil || p.prep == nil {
		return false, fmt.Errorf("PreparedGeom 已关闭或未初始化")
	}
	return safeRun(func() (bool, error) { return fn(), nil })
}

// Intersects 判断 prepared 几何与 other 是否有交集。
func (p *PreparedGeom) Intersects(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Intersects(other) })
}

// Contains 判断 prepared 几何是否严格包含 other（边界不算）。
func (p *PreparedGeom) Contains(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Contains(other) })
}

// ContainsXY 判断 prepared 几何是否严格包含点 (x,y)。
func (p *PreparedGeom) ContainsXY(x, y float64) (bool, error) {
	return p.prepRun(func() bool { return p.prep.ContainsXY(x, y) })
}

// Covers 判断 prepared 几何是否覆盖 other（边界算）。
func (p *PreparedGeom) Covers(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Covers(other) })
}

// IntersectsXY 判断 prepared 几何是否与点 (x,y) 相交（等价 CoversPoint）。
func (p *PreparedGeom) IntersectsXY(x, y float64) (bool, error) {
	return p.prepRun(func() bool { return p.prep.IntersectsXY(x, y) })
}

// Disjoint 判断 prepared 几何是否与 other 完全无交集。
func (p *PreparedGeom) Disjoint(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Disjoint(other) })
}

// CoveredBy 判断 prepared 几何是否被 other 覆盖。
func (p *PreparedGeom) CoveredBy(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.CoveredBy(other) })
}

// Overlaps 判断是否部分重叠。
func (p *PreparedGeom) Overlaps(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Overlaps(other) })
}

// Touches 判断是否仅边界接触。
func (p *PreparedGeom) Touches(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Touches(other) })
}

// Within 判断是否在 other 内部。
func (p *PreparedGeom) Within(other *gogeos.Geom) (bool, error) {
	return p.prepRun(func() bool { return p.prep.Within(other) })
}

// DistanceWithin 判断与 other 距离是否在 dist 内。
func (p *PreparedGeom) DistanceWithin(other *gogeos.Geom, dist float64) (bool, error) {
	return p.prepRun(func() bool { return p.prep.DistanceWithin(other, dist) })
}
