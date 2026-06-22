package geos

import (
	"fmt"

	gogeos "github.com/twpayne/go-geos"
)

// predicateTwo 对两个 *gogeos.Geom 执行二元谓词。
func predicateTwo(a, b *gogeos.Geom, fn func() bool) (bool, error) {
	if a == nil || b == nil {
		return false, fmt.Errorf("geometry 为 nil")
	}
	return safeRun(func() (bool, error) { return fn(), nil })
}

// Intersects 判断两几何是否有任意公共点（含边界接触）。
func Intersects(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.Intersects(b) }) }

// Contains 判断 a 是否严格包含 b（边界点不算，OGC 语义）。
func Contains(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.Contains(b) }) }

// Covers 判断 a 是否覆盖 b（边界点算，围栏命中场景）。
func Covers(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.Covers(b) }) }

// Within 判断 a 是否在 b 内。
func Within(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.Within(b) }) }

// Touches 判断两几何是否仅边界接触。
func Touches(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.Touches(b) }) }

// Disjoint 判断两几何是否完全无交集。
func Disjoint(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.Disjoint(b) }) }

// Equals 判断两几何在拓扑上是否相等。
func Equals(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.Equals(b) }) }

// Overlaps 判断两几何是否部分重叠。
func Overlaps(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.Overlaps(b) }) }

// Crosses 判断两几何是否穿越。
func Crosses(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.Crosses(b) }) }

// CoveredBy 判断 a 是否被 b 覆盖。
func CoveredBy(a, b *gogeos.Geom) (bool, error) { return predicateTwo(a, b, func() bool { return a.CoveredBy(b) }) }

// EqualsExact 判断两几何在 tolerance 内精确相等。
func EqualsExact(a, b *gogeos.Geom, tolerance float64) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.EqualsExact(b, tolerance) })
}
