package geos

// errors.go — 包内统一错误定义
//
// 所有哨兵错误（sentinel error）集中定义于此，调用方可通过 errors.Is 区分错误类型。

import "fmt"

var (
	// ErrNil 几何对象为 nil。
	ErrNil = fmt.Errorf("geom 为 nil")

	// ErrClosed 对象已关闭或未初始化。
	ErrClosed = fmt.Errorf("对象已关闭或未初始化")

	// ErrNotPolygon 几何对象不是 Polygon 类型。
	ErrNotPolygon = fmt.Errorf("geometry 不是 Polygon 类型")

	// ErrEmptyRing 环坐标为空。
	ErrEmptyRing = fmt.Errorf("环坐标为空")

	// ErrEmptyOuterRing 外环为空，无法构造 Polygon。
	ErrEmptyOuterRing = fmt.Errorf("外环坐标为空")

	// ErrNotSupported 不支持的操作或几何类型。
	ErrNotSupported = fmt.Errorf("不支持的操作")

	// ErrEmptyGeoms 传入的几何对象切片为空，无法构造集合。
	ErrEmptyGeoms = fmt.Errorf("几何对象切片为空")

	// ErrEmptyResult 运算结果为空几何（例如不相交的交集、退化几何的质心等）。
	ErrEmptyResult = fmt.Errorf("运算结果为空几何")
)

// errNil 保留兼容（等价于 ErrNil），包内可使用。
var errNil = ErrNil
