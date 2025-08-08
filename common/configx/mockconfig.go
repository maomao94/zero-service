package configx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/brianvoe/gofakeit/v7"
	"github.com/zeromicro/go-zero/core/logx"
)

type MockConfig struct {
	mu        sync.RWMutex
	rawData   map[string]map[string]json.RawMessage // 改成 RawMessage 保留原始字节
	templates map[string]map[string]*template.Template
}

func MustNewMockConfig(path string) *MockConfig {
	cli, err := NewMockConfig(path)
	logx.Must(err)
	return cli
}

func NewMockConfig(path string) (*MockConfig, error) {
	gofakeit.Seed(0)

	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 改成 RawMessage，保留原始字节
	var raw map[string]map[string]json.RawMessage
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, err
	}

	mc := &MockConfig{
		rawData:   raw,
		templates: make(map[string]map[string]*template.Template),
	}

	mc.initTemplates()
	return mc, nil
}

func (mc *MockConfig) funcMap() template.FuncMap {
	return template.FuncMap{
		"fake": func(kind string) string {
			switch kind {
			case "name":
				return gofakeit.Name()
			case "city":
				return gofakeit.City()
			case "phone":
				return gofakeit.Phone()
			case "email":
				return gofakeit.Email()
			case "date":
				return gofakeit.Date().Format("2006-01-02 15:04:05")
			case "word":
				return gofakeit.Word()
			case "appName":
				return gofakeit.AppName()
			default:
				return ""
			}
		},
	}
}

func (mc *MockConfig) initTemplates() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	for key, scenes := range mc.rawData {
		mc.templates[key] = make(map[string]*template.Template)
		for scene, raw := range scenes {
			// delay 类型特殊处理
			if str := string(raw); strings.HasPrefix(str, "\"delay:") {
				continue
			}

			// 直接将 json.RawMessage 转为字符串，即原始模板内容，不会有转义问题
			tmplStr := string(raw)

			tmpl, err := template.New(scene).Funcs(mc.funcMap()).Parse(tmplStr)
			if err != nil {
				logx.Errorf("template parse error key=%s scene=%s: %v", key, scene, err)
				continue
			}

			mc.templates[key][scene] = tmpl
		}
	}
}

func (mc *MockConfig) GetResponse(method, path, scene string) (string, error) {
	key := fmt.Sprintf("%s %s", method, path)
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	sceneMap, ok := mc.rawData[key]
	if !ok {
		return "", fmt.Errorf("no mock config for %s", key)
	}

	resp, ok := sceneMap[scene]
	if !ok {
		return "", fmt.Errorf("no scene %s for %s", scene, key)
	}

	// delay 字符串直接返回
	if str := string(resp); strings.HasPrefix(str, "\"delay:") && len(str) > 7 {
		msStr := str[7 : len(str)-1]
		msInt, err := strconv.Atoi(msStr)
		if err != nil {
			return "", err
		}
		time.Sleep(time.Duration(msInt) * time.Millisecond)
		resp, ok = sceneMap["default"]
		if !ok {
			return "", fmt.Errorf("no scene %s for %s", scene, key)
		}
		scene = "default"
	}

	tmpl, ok := mc.templates[key][scene]
	if !ok {
		return "", fmt.Errorf("no compiled template for %s scene %s", key, scene)
	}

	var buf bytes.Buffer
	err := tmpl.Execute(&buf, nil)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}
