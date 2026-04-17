package sessionworkdir

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"zero-service/aiapp/aisolo/internal/config"
)

// EnsureSession 在已配置的会话工作区父目录下创建 sessionID 子目录。
func EnsureSession(c config.Config, sessionID string) error {
	base := c.Agent.Deep.FilesystemSessionBaseDir
	if base == "" || sessionID == "" {
		return nil
	}
	if strings.ContainsAny(sessionID, `/\:`) {
		return fmt.Errorf("sessionworkdir: invalid session id")
	}
	abs, err := filepath.Abs(base)
	if err != nil {
		return err
	}
	abs = filepath.Clean(abs)
	return os.MkdirAll(filepath.Join(abs, sessionID), 0755)
}
