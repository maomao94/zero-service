package main

import (
	"fmt"
	"runtime/debug"
	"strings"

	"zero-service/common/gisx/geos"

	"github.com/mattn/go-sqlite3"
)

const h3GoModulePath = "github.com/uber/h3-go/v4"

// printDependencyVersions 打印 gis 依赖的 C 组件版本，便于现场核对运行时环境。
// H3 由 h3-go 内置 C 源码编译，模块未导出 C 版本 API，这里取 Go 模块版本；
// 按 h3-go 的版本约定，模块 major/minor 与 H3 core 一致，patch 可能只含绑定修复。
func printDependencyVersions() {
	fmt.Printf("GEOS Version: %s\n", geos.GEOSVersionString())
	fmt.Printf("H3 Version: %s\n", h3Version())
	sqliteVersion, _, _ := sqlite3.Version()
	fmt.Printf("SQLite Version: %s\n", sqliteVersion)
}

func h3Version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range info.Deps {
		if dep.Path == h3GoModulePath {
			return strings.TrimPrefix(dep.Version, "v")
		}
	}
	return "unknown"
}
