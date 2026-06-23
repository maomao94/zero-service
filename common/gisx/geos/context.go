package geos

import (
	"fmt"
	"sync"

	gogeos "github.com/twpayne/go-geos"
)

// GEOSVersion 返回当前链接的 GEOS C 库主、次、补丁版本号。
func GEOSVersion() (major, minor, patch int) {
	return gogeos.VersionMajor, gogeos.VersionMinor, gogeos.VersionPatch
}

// GEOSVersionString 返回 GEOS 版本字符串，格式 "major.minor.patch"。
func GEOSVersionString() string {
	major, minor, patch := GEOSVersion()
	return fmt.Sprintf("%d.%d.%d", major, minor, patch)
}

// defaultContext 是包级懒加载的 GEOS Context，用于短生命周期操作。
// 长生命周期对象（PreparedPolygon、STRtree）持有自己的 Context 引用。
var (
	defaultContext     *gogeos.Context
	defaultContextOnce sync.Once
)

func getDefaultContext() *gogeos.Context {
	defaultContextOnce.Do(func() {
		defaultContext = gogeos.NewContext()
	})
	return defaultContext
}

// safeRun 统一 recover panic → error。所有对 GEOS 的调用必须经过 safeRun。
func safeRun[T any](fn func() (T, error)) (result T, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = zeroValue[T]()
			err = fmt.Errorf("geos: %v", r)
		}
	}()
	return fn()
}

// safeRunErr 用于只返回 error 的操作。
func safeRunErr(fn func() error) error {
	_, err := safeRun(func() (struct{}, error) { return struct{}{}, fn() })
	return err
}

// zeroValue 返回 T 的零值。
func zeroValue[T any]() T {
	var z T
	return z
}

// oneAttr 是内部辅助函数，用于单个几何的属性查询（bool/float64/int 等）。
// 统一处理 nil 检查和 panic 捕获。
func oneAttr[T any](g *gogeos.Geom, fn func(*gogeos.Geom) T) (T, error) {
	if g == nil {
		var zero T
		return zero, errNil
	}
	return safeRun(func() (T, error) { return fn(g), nil })
}
