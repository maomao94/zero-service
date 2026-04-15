package quick

import "os"

// SkillsConfig Skills 配置
type SkillsConfig struct {
	Dir     string `json:"dir"`     // Skills 目录路径
	Enabled bool   `json:"enabled"` // 是否启用
}

// CheckSkillsDir 检查 Skills 目录是否存在
func CheckSkillsDir(dir string) bool {
	if dir == "" {
		return false
	}
	_, err := os.Stat(dir)
	return err == nil
}

// StringPtr 返回字符串指针
func StringPtr(s string) *string {
	return &s
}

// IntPtr 返回 int 指针
func IntPtr(i int) *int {
	return &i
}
