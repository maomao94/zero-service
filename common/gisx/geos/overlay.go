package geos

import gogeos "github.com/twpayne/go-geos"

// overlayTwo 对两个 *gogeos.Geom 执行二元 overlay。
func overlayTwo(a, b *gogeos.Geom, fn func() *gogeos.Geom) (*gogeos.Geom, error) {
	if a == nil || b == nil {
		return nil, errNil
	}
	return safeRun(func() (*gogeos.Geom, error) { return fn(), nil })
}

// transformOne 对单个 *gogeos.Geom 执行变换。
func transformOne(g *gogeos.Geom, fn func() *gogeos.Geom) (*gogeos.Geom, error) {
	if g == nil {
		return nil, errNil
	}
	return safeRun(func() (*gogeos.Geom, error) { return fn(), nil })
}

// --- Overlay ---

func Intersection(a, b *gogeos.Geom) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.Intersection(b) })
}
func Union(a, b *gogeos.Geom) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.Union(b) })
}
func Difference(a, b *gogeos.Geom) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.Difference(b) })
}
func SymDifference(a, b *gogeos.Geom) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.SymDifference(b) })
}
func UnaryUnion(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.UnaryUnion() })
}
func Envelope(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Envelope() })
}
func Boundary(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Boundary() })
}
func BuildArea(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.BuildArea() })
}
func LineMerge(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.LineMerge() })
}
func Node(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Node() })
}
func MinimumRotatedRectangle(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.MinimumRotatedRectangle() })
}

// --- Valid ---

func IsValid(g *gogeos.Geom) (bool, error) {
	return oneBool(g, func(gg *gogeos.Geom) bool { return gg.IsValid() })
}
func IsValidReason(g *gogeos.Geom) (string, error) {
	if g == nil {
		return "", errNil
	}
	return safeRun(func() (string, error) { return g.IsValidReason(), nil })
}
func MakeValid(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.MakeValid() })
}

// --- Measure ---

func Area(g *gogeos.Geom) (float64, error) {
	return oneFloat(g, func(gg *gogeos.Geom) float64 { return gg.Area() })
}
func Length(g *gogeos.Geom) (float64, error) {
	return oneFloat(g, func(gg *gogeos.Geom) float64 { return gg.Length() })
}
func Distance(a, b *gogeos.Geom) (float64, error) {
	if a == nil || b == nil {
		return 0, errNil
	}
	return safeRun(func() (float64, error) { return a.Distance(b), nil })
}
func Centroid(g *gogeos.Geom) (x, y float64, err error) {
	v, e := safeRun(func() (pointPair, error) {
		c := g.Centroid()
		if c == nil || c.IsEmpty() {
			return pointPair{}, errNil
		}
		return pointPair{x1: c.X(), y1: c.Y()}, nil
	})
	if e != nil {
		return 0, 0, e
	}
	return v.x1, v.y1, nil
}
func PointOnSurface(g *gogeos.Geom) (x, y float64, err error) {
	v, e := safeRun(func() (pointPair, error) {
		p := g.PointOnSurface()
		if p == nil || p.IsEmpty() {
			return pointPair{}, errNil
		}
		return pointPair{x1: p.X(), y1: p.Y()}, nil
	})
	if e != nil {
		return 0, 0, e
	}
	return v.x1, v.y1, nil
}

// --- Simplify / Buffer ---

func Buffer(g *gogeos.Geom, width float64, quadsegs int) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Buffer(width, quadsegs) })
}
func Simplify(g *gogeos.Geom, tolerance float64) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Simplify(tolerance) })
}
func TopologyPreserveSimplify(g *gogeos.Geom, tolerance float64) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.TopologyPreserveSimplify(tolerance) })
}
func ConvexHull(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.ConvexHull() })
}
func ConcaveHull(g *gogeos.Geom, ratio float64, allowHoles bool) (*gogeos.Geom, error) {
	var holes uint
	if allowHoles {
		holes = 1
	}
	return transformOne(g, func() *gogeos.Geom { return g.ConcaveHull(ratio, holes) })
}

// --- Transform ---

func Normalize(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Normalize() })
}
func Reverse(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Reverse() })
}
func Snap(a, b *gogeos.Geom, tolerance float64) (*gogeos.Geom, error) {
	return overlayTwo(a, b, func() *gogeos.Geom { return a.Snap(b, tolerance) })
}
func ClipByRect(g *gogeos.Geom, minX, minY, maxX, maxY float64) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.ClipByRect(minX, minY, maxX, maxY) })
}
func Densify(g *gogeos.Geom, tolerance float64) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.Densify(tolerance) })
}
func OffsetCurve(g *gogeos.Geom, width float64, quadsegs int) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.OffsetCurve(width, quadsegs, gogeos.BufJoinStyleRound, 5.0) })
}
func EndPoint(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.EndPoint() })
}
func StartPoint(g *gogeos.Geom) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.StartPoint() })
}

// --- Meta ---

func MinimumClearance(g *gogeos.Geom) (float64, error) {
	return oneFloat(g, func(gg *gogeos.Geom) float64 { return gg.MinimumClearance() })
}
func SRID(g *gogeos.Geom) (int, error) {
	return oneInt(g, func(gg *gogeos.Geom) int { return gg.SRID() })
}
func SetSRID(g *gogeos.Geom, srid int) (*gogeos.Geom, error) {
	return transformOne(g, func() *gogeos.Geom { return g.SetSRID(srid) })
}
func Precision(g *gogeos.Geom) (float64, error) {
	return oneFloat(g, func(gg *gogeos.Geom) float64 { return gg.Precision() })
}
func FrechetDistance(a, b *gogeos.Geom) (float64, error) {
	if a == nil || b == nil {
		return 0, errNil
	}
	return safeRun(func() (float64, error) { return a.FrechetDistance(b), nil })
}

// --- Helpers ---

func oneFloat(g *gogeos.Geom, fn func(*gogeos.Geom) float64) (float64, error) {
	if g == nil {
		return 0, errNil
	}
	return safeRun(func() (float64, error) { return fn(g), nil })
}
func oneInt(g *gogeos.Geom, fn func(*gogeos.Geom) int) (int, error) {
	if g == nil {
		return 0, errNil
	}
	return safeRun(func() (int, error) { return fn(g), nil })
}
