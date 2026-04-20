// Package fsrestrict 将 filesystem.Backend 限制在若干允许根目录之下（类 chroot），并支持会话分区与读写策略。
//
// Eino filesystem 中间件默认将超大工具结果卸载到绝对路径 /large_tool_result/...；
// policyBackend 将该前缀映射到会话根或首个用户根下的 large_tool_result 子目录。
package fsrestrict

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudwego/eino/adk/filesystem"
)

// Wrap 兼容旧调用：仅用户根目录、在用户区内读写改均允许（等价于 PermissivePolicy）。
func Wrap(inner filesystem.Backend, allowedAbsRoots []string) filesystem.Backend {
	if inner == nil || len(allowedAbsRoots) == 0 {
		return inner
	}
	roots := make([]string, 0, len(allowedAbsRoots))
	for _, r := range allowedAbsRoots {
		if r == "" {
			continue
		}
		roots = append(roots, filepath.Clean(r))
	}
	if len(roots) == 0 {
		return inner
	}
	return WrapConfigured(inner, Config{UserRoots: roots, Policy: PermissivePolicy()})
}

func underRoot(root, abs string) bool {
	root = filepath.Clean(root)
	abs = filepath.Clean(abs)
	if root == abs {
		return true
	}
	rel, err := filepath.Rel(root, abs)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
