package geos

import gogeos "github.com/twpayne/go-geos"

// IsEmpty 判断几何是否为空。nil 几何视为空。
func IsEmpty(g *gogeos.Geom) (bool, error) {
	if g == nil {
		return true, nil
	}
	return oneBool(g, func(gg *gogeos.Geom) bool { return gg.IsEmpty() })
}

// IsSimple 判断几何是否简单（无自相交）。
func IsSimple(g *gogeos.Geom) (bool, error) {
	return oneBool(g, func(gg *gogeos.Geom) bool { return gg.IsSimple() })
}

// IsClosed 判断几何外环是否闭合。
func IsClosed(g *gogeos.Geom) (bool, error) {
	return oneBool(g, func(gg *gogeos.Geom) bool { return gg.IsClosed() })
}

// IsRing 判断几何是否为环形（闭合且简单）。
func IsRing(g *gogeos.Geom) (bool, error) {
	return oneBool(g, func(gg *gogeos.Geom) bool { return gg.IsRing() })
}

// HasZ 判断几何是否有 Z 坐标。
func HasZ(g *gogeos.Geom) (bool, error) {
	return oneBool(g, func(gg *gogeos.Geom) bool { return gg.HasZ() })
}

func oneBool(g *gogeos.Geom, fn func(*gogeos.Geom) bool) (bool, error) {
	if g == nil {
		return false, errNil
	}
	return safeRun(func() (bool, error) { return fn(g), nil })
}
