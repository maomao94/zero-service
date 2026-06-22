package geos

import (
	"fmt"

	gogeos "github.com/twpayne/go-geos"
)

func Relate(a, b *gogeos.Geom) (string, error) {
	if a == nil || b == nil {
		return "", fmt.Errorf("geometry 为 nil")
	}
	return safeRun(func() (string, error) { return a.Relate(b), nil })
}

func RelatePattern(a, b *gogeos.Geom, pattern string) (bool, error) {
	if a == nil || b == nil {
		return false, fmt.Errorf("geometry 为 nil")
	}
	return safeRun(func() (bool, error) { return a.RelatePattern(b, pattern), nil })
}

func DistanceWithin(a, b *gogeos.Geom, dist float64) (bool, error) {
	return predicateTwo(a, b, func() bool { return a.DistanceWithin(b, dist) })
}

func HausdorffDistance(a, b *gogeos.Geom) (float64, error) {
	if a == nil || b == nil {
		return 0, fmt.Errorf("geometry 为 nil")
	}
	return safeRun(func() (float64, error) { return a.HausdorffDistance(b), nil })
}

type pointPair struct{ x1, y1, x2, y2 float64 }

func NearestPoints(a, b *gogeos.Geom) (ax, ay, bx, by float64, err error) {
	if a == nil || b == nil {
		return 0, 0, 0, 0, fmt.Errorf("geometry 为 nil")
	}
	result, err := safeRun(func() (pointPair, error) {
		coords := a.NearestPoints(b)
		if len(coords) != 2 {
			return pointPair{}, fmt.Errorf("NearestPoints 返回异常坐标数: %d", len(coords))
		}
		return pointPair{coords[0][0], coords[0][1], coords[1][0], coords[1][1]}, nil
	})
	if err != nil {
		return 0, 0, 0, 0, err
	}
	return result.x1, result.y1, result.x2, result.y2, nil
}
