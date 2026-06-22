package geos

import (
	"fmt"

	gogeos "github.com/twpayne/go-geos"
)

type STRtree struct {
	tree *gogeos.STRtree
}

func NewSTRtree(nodeCapacity int) *STRtree {
	return &STRtree{tree: getDefaultContext().NewSTRtree(nodeCapacity)}
}

func (t *STRtree) Insert(g *gogeos.Geom, value any) error {
	if t == nil || t.tree == nil {
		return fmt.Errorf("STRtree 已关闭")
	}
	return safeRunErr(func() error { return t.tree.Insert(g, value) })
}

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

func (t *STRtree) Iterate(fn func(value any)) {
	if t == nil || t.tree == nil {
		return
	}
	_ = safeRunErr(func() error {
		t.tree.Iterate(fn)
		return nil
	})
}

func (t *STRtree) Remove(g *gogeos.Geom, value any) (bool, error) {
	if t == nil || t.tree == nil {
		return false, fmt.Errorf("STRtree 已关闭")
	}
	return safeRun(func() (bool, error) { return t.tree.Remove(g, value), nil })
}

func (t *STRtree) Close() {
	if t != nil {
		t.tree = nil
	}
}
