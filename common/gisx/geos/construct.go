package geos

import gogeos "github.com/twpayne/go-geos"

func NewPoint(x, y float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewPointFromXY(x, y), nil
	})
}
func NewLineString(coords [][]float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewLineString(coords), nil
	})
}
func NewLinearRing(coords [][]float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		if len(coords) > 0 && (coords[0][0] != coords[len(coords)-1][0] || coords[0][1] != coords[len(coords)-1][1]) {
			coords = append(coords, []float64{coords[0][0], coords[0][1]})
		}
		return getDefaultContext().NewLinearRing(coords), nil
	})
}
func NewPolygon(coordss [][][]float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewPolygon(coordss), nil
	})
}
func NewBoundsRect(minX, minY, maxX, maxY float64) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewGeomFromBounds(minX, minY, maxX, maxY), nil
	})
}
