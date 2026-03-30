package skills

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/zeromicro/go-zero/core/logx"
)

// Skill 单个 skill 的元数据和内容
type Skill struct {
	Name         string   // skill 名称
	Description  string   // 描述
	AllowedTools []string // 允许的工具列表
	Content      string   // 完整内容（不含 frontmatter）
	FilePath     string   // 文件路径
}

// Frontmatter SKILL.md 的 frontmatter 格式
type Frontmatter struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
}

// Loader 动态加载 skills 的加载器
type Loader struct {
	mu         sync.RWMutex
	dir        string
	skills     map[string]*Skill
	autoReload bool
	watcher    *fsnotify.Watcher // 文件监控器
	stopCh     chan struct{}     // 停止信号
}

// NewLoader 创建新的 skills 加载器
func NewLoader(dir string, autoReload bool) (*Loader, error) {
	// 确保目录存在
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("创建 skills 目录失败: %w", err)
	}

	loader := &Loader{
		dir:        dir,
		skills:     make(map[string]*Skill),
		autoReload: autoReload,
	}

	// 初始加载
	if err := loader.Load(); err != nil {
		return nil, err
	}

	return loader, nil
}

// Load 扫描目录并加载所有 SKILL.md 文件
func (l *Loader) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return filepath.Walk(l.dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 只处理 SKILL.md 文件
		if info.IsDir() || info.Name() != "SKILL.md" {
			return nil
		}

		// 获取 skill 目录名作为 skill name
		skillDir := filepath.Dir(path)
		skillName := filepath.Base(skillDir)

		if err := l.loadSkillFile(skillName, path); err != nil {
			logx.Errorf("加载 skill [%s] 失败: %v", skillName, err)
		}

		return nil
	})
}

// loadSkillFile 加载单个 SKILL.md 文件
func (l *Loader) loadSkillFile(name, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("读取文件失败: %w", err)
	}

	content := string(data)

	// 解析 frontmatter
	fm, body, err := parseFrontmatter(content)
	if err != nil {
		// 如果解析失败，假设没有 frontmatter，整个内容作为 body
		body = content
		fm = &Frontmatter{Name: name}
	}

	// 如果 frontmatter 没有 name，使用目录名
	if fm.Name == "" {
		fm.Name = name
	}

	l.skills[name] = &Skill{
		Name:         fm.Name,
		Description:  fm.Description,
		AllowedTools: fm.AllowedTools,
		Content:      strings.TrimSpace(body),
		FilePath:     path,
	}

	logx.Infof("加载 skill [%s]: %s", name, fm.Description)
	return nil
}

// GetSkill 根据名称获取 skill
func (l *Loader) GetSkill(name string) (*Skill, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	skill, ok := l.skills[name]
	return skill, ok
}

// ListSkills 返回所有 skills 的元数据列表
func (l *Loader) ListSkills() []*Skill {
	l.mu.RLock()
	defer l.mu.RUnlock()

	skills := make([]*Skill, 0, len(l.skills))
	for _, skill := range l.skills {
		skills = append(skills, skill)
	}
	return skills
}

// StartWatcher 启动文件监控（热加载）
func (l *Loader) StartWatcher() error {
	if !l.autoReload {
		return nil
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("创建文件监控器失败: %w", err)
	}

	l.watcher = watcher
	l.stopCh = make(chan struct{})

	// 递归监控目录
	if err := watcher.Add(l.dir); err != nil {
		watcher.Close()
		return fmt.Errorf("监控目录失败: %w", err)
	}

	// 启动监控 goroutine
	go l.watchLoop()

	logx.Infof("Skills 热加载已启用，监控目录: %s", l.dir)
	return nil
}

// watchLoop 文件变化监控循环
func (l *Loader) watchLoop() {
	for {
		select {
		case <-l.stopCh:
			return
		case event, ok := <-l.watcher.Events:
			if !ok {
				return
			}
			if filepath.Base(event.Name) == "SKILL.md" {
				l.handleFileChange(event)
			}
		case err, ok := <-l.watcher.Errors:
			if !ok {
				return
			}
			logx.Errorf("文件监控错误: %v", err)
		}
	}
}

// handleFileChange 处理文件变化
func (l *Loader) handleFileChange(event fsnotify.Event) {
	skillDir := filepath.Dir(event.Name)
	skillName := filepath.Base(skillDir)

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		logx.Infof("检测到新 skill: %s", skillName)
		l.Load()
	case event.Op&fsnotify.Write == fsnotify.Write:
		logx.Infof("Skill 内容更新: %s", skillName)
		l.Load()
	case event.Op&fsnotify.Remove == fsnotify.Remove:
		logx.Infof("Skill 已删除: %s", skillName)
		l.mu.Lock()
		delete(l.skills, skillName)
		l.mu.Unlock()
	case event.Op&fsnotify.Rename == fsnotify.Rename:
		logx.Infof("Skill 已重命名: %s", skillName)
		l.Load()
	}
}

// Stop 停止 Loader（关闭文件监控）
func (l *Loader) Stop() {
	if l.stopCh != nil {
		close(l.stopCh)
	}
	if l.watcher != nil {
		l.watcher.Close()
	}
}

// parseFrontmatter 解析 YAML frontmatter
// 格式:
// ---
// name: xxx
// description: xxx
// ---
// content
func parseFrontmatter(content string) (*Frontmatter, string, error) {
	const fmStart = "---\n"
	const fmEnd = "\n---\n"

	// 检查是否有 frontmatter
	if !strings.HasPrefix(content, fmStart) {
		return nil, content, fmt.Errorf("无 frontmatter")
	}

	// 找到 end marker
	endIdx := strings.Index(content[len(fmStart)-1:], fmEnd)
	if endIdx == -1 {
		return nil, content, fmt.Errorf("frontmatter 未闭合")
	}

	// 提取 frontmatter 内容
	fmStartIdx := len(fmStart) - 1
	fmEndIdx := fmStartIdx + endIdx + len(fmEnd)
	fmContent := content[fmStartIdx : fmEndIdx-len(fmEnd)]

	// 解析 YAML（简化实现，手动解析关键字段）
	fm := &Frontmatter{}

	lines := strings.Split(fmContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "name:") {
			fm.Name = strings.TrimSpace(strings.TrimPrefix(line, "name:"))
		} else if strings.HasPrefix(line, "description:") {
			fm.Description = strings.TrimSpace(strings.TrimPrefix(line, "description:"))
			// 处理多行 description
			fm.Description = strings.Trim(fm.Description, "\"|'")
		} else if strings.HasPrefix(line, "allowed-tools:") {
			toolsStr := strings.TrimSpace(strings.TrimPrefix(line, "allowed-tools:"))
			if strings.HasPrefix(toolsStr, "[") {
				// 解析数组格式: [tool1, tool2, tool3]
				toolsStr = strings.Trim(toolsStr, "[]")
				for _, t := range strings.Split(toolsStr, ",") {
					t = strings.TrimSpace(t)
					if t != "" {
						fm.AllowedTools = append(fm.AllowedTools, t)
					}
				}
			}
		}
	}

	// 返回 frontmatter 和剩余内容
	body := strings.TrimLeft(content[fmEndIdx:], "\n")
	return fm, body, nil
}
