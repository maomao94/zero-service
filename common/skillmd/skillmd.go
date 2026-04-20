// Package skillmd 扫描 SKILL.md 并解析 YAML frontmatter，供 aisolo ListSkills 等只读场景使用。
package skillmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Info 单个 skill 的展示元数据（不含正文）。
type Info struct {
	ID           string
	Name         string
	Description  string
	Tags         []string
	LaunchPrompt string
}

type docFront struct {
	Name         string   `yaml:"name"`
	Description  string   `yaml:"description"`
	AllowedTools []string `yaml:"allowed-tools"`
	Tags         []string `yaml:"tags"`
	LaunchPrompt string   `yaml:"launch_prompt"`
}

// ScanDir 扫描 baseDir 下各子目录中的 SKILL.md（不递归子 skill）。
func ScanDir(baseDir string) ([]Info, error) {
	if baseDir == "" {
		return nil, nil
	}
	abs, err := filepath.Abs(baseDir)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(abs)
	if err != nil || !fi.IsDir() {
		return nil, fmt.Errorf("skillmd: not a directory: %s", baseDir)
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	var out []Info
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		id := ent.Name()
		p := filepath.Join(abs, id, "SKILL.md")
		st, err := os.Stat(p)
		if err != nil || st.IsDir() {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		info, err := parseSkillFile(id, string(data))
		if err != nil {
			continue
		}
		out = append(out, info)
	}
	return out, nil
}

func parseSkillFile(id, content string) (Info, error) {
	fm, err := parseFrontmatterYAML(content)
	if err != nil {
		return Info{ID: id, Name: id}, nil
	}
	name := fm.Name
	if name == "" {
		name = id
	}
	desc := strings.TrimSpace(fm.Description)
	lp := strings.TrimSpace(fm.LaunchPrompt)
	if lp == "" && desc != "" {
		lp = "请根据以下技能说明协助用户完成任务：\n\n" + desc
	}
	return Info{
		ID: id, Name: name, Description: desc,
		Tags: append([]string(nil), fm.Tags...), LaunchPrompt: lp,
	}, nil
}

func parseFrontmatterYAML(content string) (*docFront, error) {
	rest := content
	if strings.HasPrefix(rest, "---\n") {
		rest = rest[4:]
	} else if strings.HasPrefix(rest, "---\r\n") {
		rest = rest[5:]
	} else {
		return nil, fmt.Errorf("no frontmatter")
	}
	var end int
	if i := strings.Index(rest, "\n---\n"); i >= 0 {
		end = i
	} else if i := strings.Index(rest, "\r\n---\r\n"); i >= 0 {
		end = i
	} else {
		return nil, fmt.Errorf("unclosed frontmatter")
	}
	fmYAML := rest[:end]
	var fm docFront
	if err := yaml.Unmarshal([]byte(fmYAML), &fm); err != nil {
		return nil, err
	}
	return &fm, nil
}
