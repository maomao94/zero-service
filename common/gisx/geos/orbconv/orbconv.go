// Package orbconv provides conversion between orb types and go-geos geometries,
// plus convenience wrappers that accept orb types directly.
//
// This is the recommended entry point for application code that works with orb types.
// For code that doesn't use orb, use the parent geos package directly.
package orbconv

import (
	"github.com/paulmach/orb"
	gogeos "github.com/twpayne/go-geos"

	"zero-service/common/gisx/geos"
)

// --- Conversion functions ---

// GeomToRing extracts a ring's coordinates from a LineString/LinearRing geometry.
func GeomToRing(g *gogeos.Geom) (orb.Ring, error) {
	if g == nil {
		return nil, nil
	}
	coords := g.CoordSeq().ToCoords()
	if len(coords) == 0 {
		return nil, nil
	}
	ring := make(orb.Ring, 0, len(coords))
	for _, c := range coords {
		ring = append(ring, orb.Point{c[0], c[1]})
	}
	return ring, nil
}

// GeomToPolygon extracts a polygon's rings from a Polygon geometry.
func GeomToPolygon(g *gogeos.Geom) (orb.Polygon, error) {
	if g == nil {
		return nil, nil
	}
	switch g.TypeID() {
	case gogeos.TypeIDPolygon:
		return singlePolyToOrb(g)
	case gogeos.TypeIDMultiPolygon:
		n := g.NumGeometries()
		if n == 0 {
			return orb.Polygon{}, nil
		}
		return singlePolyToOrb(g.Geometry(0))
	default:
		return nil, nil
	}
}

func singlePolyToOrb(g *gogeos.Geom) (orb.Polygon, error) {
	extRing, err := GeomToRing(g.ExteriorRing())
	if err != nil {
		return nil, err
	}
	poly := orb.Polygon{extRing}
	for i := 0; i < g.NumInteriorRings(); i++ {
		hole, err := GeomToRing(g.InteriorRing(i))
		if err != nil {
			return nil, err
		}
		poly = append(poly, hole)
	}
	return poly, nil
}

// RingToGeom converts an orb.Ring to a GEOS LinearRing.
func RingToGeom(ring orb.Ring) (*gogeos.Geom, error) {
	coords := ringToCoords(ring)
	if len(coords) == 0 {
		return nil, nil
	}
	return geos.NewLinearRing(coords)
}

// PolygonToGeom converts an orb.Polygon to a GEOS Polygon.
func PolygonToGeom(poly orb.Polygon) (*gogeos.Geom, error) {
	if len(poly) == 0 {
		return nil, nil
	}
	coordss := make([][][]float64, 0, len(poly))
	for _, ring := range poly {
		coords := ringToCoords(ring)
		if len(coords) == 0 {
			continue
		}
		coordss = append(coordss, coords)
	}
	if len(coordss) == 0 {
		return nil, nil
	}
	return geos.NewPolygon(coordss)
}

// PointToGeom converts an orb.Point to a GEOS Point.
func PointToGeom(p orb.Point) (*gogeos.Geom, error) {
	return geos.NewPoint(p[0], p[1])
}

// --- Convenience wrappers (orb types in, standard types out) ---

// IntersectsOrb 判断两 orb.Polygon 是否有交集。
func IntersectsOrb(a, b orb.Polygon) (bool, error) {
	ag, bg, err := both(a, b)
	if err != nil {
		return false, err
	}
	return geos.Intersects(ag, bg)
}

// ContainsOrb 判断 outer 是否包含 inner（边界不算）。
func ContainsOrb(outer, inner orb.Polygon) (bool, error) {
	og, ig, err := both(outer, inner)
	if err != nil {
		return false, err
	}
	return geos.Contains(og, ig)
}

// CoversOrb 判断 outer 是否覆盖 inner（边界算，围栏命中场景）。
func CoversOrb(outer, inner orb.Polygon) (bool, error) {
	og, ig, err := both(outer, inner)
	if err != nil {
		return false, err
	}
	return geos.Covers(og, ig)
}

// CoversPointOrb 判断 poly 是否覆盖点 pt（边界算）。
func CoversPointOrb(poly orb.Polygon, pt orb.Point) (bool, error) {
	g, err := PolygonToGeom(poly)
	if err != nil || g == nil {
		return false, err
	}
	pg, err := geos.NewPoint(pt[0], pt[1])
	if err != nil {
		return false, err
	}
	return geos.Covers(g, pg)
}

// ContainsPointOrb 判断 poly 是否包含点 pt（边界不算）。
func ContainsPointOrb(poly orb.Polygon, pt orb.Point) (bool, error) {
	g, err := PolygonToGeom(poly)
	if err != nil || g == nil {
		return false, err
	}
	pg, err := geos.NewPoint(pt[0], pt[1])
	if err != nil {
		return false, err
	}
	return geos.Contains(g, pg)
}

// ValidOrb 判断 orb.Polygon 是否有效。
func ValidOrb(poly orb.Polygon) (bool, error) {
	g, err := PolygonToGeom(poly)
	if err != nil || g == nil {
		return false, err
	}
	return geos.IsValid(g)
}

func both(a, b orb.Polygon) (*gogeos.Geom, *gogeos.Geom, error) {
	ag, err := PolygonToGeom(a)
	if err != nil {
		return nil, nil, err
	}
	bg, err := PolygonToGeom(b)
	if err != nil {
		return nil, nil, err
	}
	return ag, bg, nil
}

func ringToCoords(ring orb.Ring) [][]float64 {
	coords := make([][]float64, 0, len(ring)+1)
	for _, p := range ring {
		coords = append(coords, []float64{p[0], p[1]})
	}
	if len(coords) > 0 {
		first, last := coords[0], coords[len(coords)-1]
		if first[0] != last[0] || first[1] != last[1] {
			coords = append(coords, []float64{first[0], first[1]})
		}
	}
	return coords
}
