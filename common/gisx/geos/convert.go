package geos

import gogeos "github.com/twpayne/go-geos"

func FromWKT(wkt string) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewGeomFromWKT(wkt)
	})
}
func ToWKT(g *gogeos.Geom) (string, error) {
	return safeRun(func() (string, error) { return g.ToWKT(), nil })
}
func FromWKB(wkb []byte) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewGeomFromWKB(wkb)
	})
}
func ToWKB(g *gogeos.Geom) ([]byte, error) {
	return safeRun(func() ([]byte, error) { return g.ToWKB(), nil })
}
func FromGeoJSON(geojson string) (*gogeos.Geom, error) {
	return safeRun(func() (*gogeos.Geom, error) {
		return getDefaultContext().NewGeomFromGeoJSON(geojson)
	})
}
func ToGeoJSON(g *gogeos.Geom, indent int) (string, error) {
	return safeRun(func() (string, error) { return g.ToGeoJSON(indent), nil })
}
