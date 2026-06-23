package geos

// strtree.go — STRtree 空间索引
//
// STRtree (Sort-Tile-Recursive Tree) 是 GEOS 提供的 R-Tree 空间索引。
// 用于加速空间查询：在大量几何中快速找到与查询范围相交的几何。
//
// 什么是 R-Tree？
// R-Tree 是一种空间索引数据结构，将空间对象组织成层次化的矩形包围盒。
// 查询时先检查包围盒，快速排除不相关的对象，再精确检查剩余对象。
//
// 适用场景：
//   - 围栏命中预筛选：从 10000 个围栏中快速找到可能命中的候选
//   - 空间范围查询：找到某个区域内的所有几何
//   - 最近邻查询：（注意：go-geos 的 Nearest 目前有问题，不建议使用）
//
// 使用方式：
//
//	// 1. 创建索引
//	tree := geos.NewSTRtree(10)  // nodeCapacity=10 是推荐值
//	defer tree.Close()
//
//	// 2. 插入几何和关联值
//	tree.Insert(fenceGeom1, "fence-1")
//	tree.Insert(fenceGeom2, "fence-2")
//
//	// 3. 查询
//	queryGeom, _ := geos.NewPoint(50, 50)
//	results, _ := tree.Query(queryGeom)  // 返回与查询范围相交的所有值
//
//	// 4. 使用结果
//	for _, v := range results {
//	    fenceID := v.(string)
//	    // 精确判断...
//	}
//
// 工作原理：
//   - Insert 时，几何被放入最小包围盒（MBR），按 Sort-Tile-Recursive 算法组织
//   - Query 时，先用查询几何的包围盒在 R-Tree 中快速筛选
//   - 返回所有包围盒相交的几何关联值（可能有假阳性，需要精确判断）
//
// 注意事项：
//   - go-geos 标注 STRtree 的 Nearest 方法 "currently broken"，会 segfault
//   - Insert/Query/Iterate/Remove 可正常使用
//   - nodeCapacity 推荐 10，太大太小都会影响性能

import (
	"fmt"

	gogeos "github.com/twpayne/go-geos"
)

// STRtree 是 R-Tree 空间索引的封装。
//
// 内部持有 go-geos 的 STRtree 对象。
// 所有方法都通过 safeRun/safeRunErr 包装，统一处理 panic。
type STRtree struct {
	tree *gogeos.STRtree
}

// NewSTRtree 创建一个新的 STRtree 空间索引。
//
// 参数：
//   - nodeCapacity: 每个节点的最大子节点数，推荐值 10
//     - 太小（如 2）：树太深，查询慢
//     - 太大（如 1000）：树太平，每个节点检查太多
//     - 10 是 GEOS 文档推荐的默认值
//
// 使用包级默认 Context 创建。
//
// 示例：
//
//	tree := geos.NewSTRtree(10)
//	defer tree.Close()
func NewSTRtree(nodeCapacity int) *STRtree {
	return &STRtree{tree: getDefaultContext().NewSTRtree(nodeCapacity)}
}

// Insert 向索引中插入一个几何和关联值。
//
// 参数：
//   - g: 几何对象，用于计算包围盒
//   - value: 任意关联值，查询时会原样返回
//
// value 可以是任何类型：字符串（围栏 ID）、结构体指针、索引号等。
// 查询时通过类型断言取回。
//
// 注意：同一个 value 不要插入多次，否则 Remove 时会有问题。
//
// 示例：
//
//	tree.Insert(fenceGeom, "fence-123")
//	tree.Insert(fenceGeom, 42)           // 也可以用索引
//	tree.Insert(fenceGeom, &FenceData{}) // 或结构体指针
func (t *STRtree) Insert(g *gogeos.Geom, value any) error {
	if t == nil || t.tree == nil {
		return fmt.Errorf("STRtree 已关闭")
	}
	return safeRunErr(func() error { return t.tree.Insert(g, value) })
}

// Query 查询索引中与查询几何相交的所有关联值。
//
// 参数：
//   - g: 查询几何，用于计算查询范围的包围盒
//
// 返回值是所有包围盒与查询范围相交的关联值列表。
// 注意：返回的是包围盒相交的结果，可能有假阳性。
// 如果需要精确判断，拿到结果后还需要用 Covers/Contains 等谓词精确判断。
//
// 示例：
//
//	queryPoint, _ := geos.NewPoint(50, 50)
//	results, _ := tree.Query(queryPoint)
//	for _, v := range results {
//	    fenceID := v.(string)
//	    // 精确判断点是否在围栏内
//	    hit, _ := orbconv.CoversPointOrb(fences[fenceID], userPoint)
//	}
func (t *STRtree) Query(g *gogeos.Geom) ([]any, error) {
	if t == nil || t.tree == nil {
		return nil, fmt.Errorf("STRtree 已关闭")
	}
	return safeRun(func() ([]any, error) {
		var values []any
		t.tree.Query(g, func(v any) { values = append(values, v) })
		return values, nil
	})
}

// Iterate 遍历索引中的所有关联值。
//
// 对索引中每个已插入的值调用回调函数。
// 如果遍历过程中 GEOS 内部出错，返回 error。
//
// 示例：
//
//	if err := tree.Iterate(func(v any) {
//	    fmt.Println(v)
//	}); err != nil {
//	    // 处理错误
//	}
func (t *STRtree) Iterate(fn func(value any)) error {
	if t == nil || t.tree == nil {
		return nil
	}
	return safeRunErr(func() error {
		t.tree.Iterate(fn)
		return nil
	})
}

// Remove 从索引中移除一个几何和关联值。
//
// 参数：
//   - g: 几何对象（必须与 Insert 时相同）
//   - value: 关联值（必须与 Insert 时相同）
//
// 返回 (bool, error)：
//   - true: 成功移除
//   - false: 未找到匹配项（可能已移除或从未插入）
//
// 示例：
//
//	ok, _ := tree.Remove(fenceGeom, "fence-123")
//	if !ok {
//	    // 未找到，可能已移除
//	}
func (t *STRtree) Remove(g *gogeos.Geom, value any) (bool, error) {
	if t == nil || t.tree == nil {
		return false, fmt.Errorf("STRtree 已关闭")
	}
	return safeRun(func() (bool, error) { return t.tree.Remove(g, value), nil })
}

// Close 释放索引引用。重复调用安全。
//
// go-geos 通过 runtime.AddCleanup 自动管理 C 内存，此处仅置空 Go 引用帮助 GC。
// 可以不调用 Close()，依赖 GC 自动回收，但显式调用是好习惯。
//
// 示例：
//
//	tree := geos.NewSTRtree(10)
//	defer tree.Close()  // 函数结束时释放
func (t *STRtree) Close() {
	if t != nil {
		t.tree = nil
	}
}
