package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"zero-service/cli/uix"
)

type basePlugin struct {
	name        string
	description string
	aliases     []string
	width       int
	height      int
}

func (p *basePlugin) Name() string                  { return p.name }
func (p *basePlugin) Description() string           { return p.description }
func (p *basePlugin) Aliases() []string             { return p.aliases }
func (p *basePlugin) Init() tea.Cmd                 { return nil }
func (p *basePlugin) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return p, nil }
func (p *basePlugin) SetSize(w, h int)              { p.width = w; p.height = h }
func (p *basePlugin) View() string {
	return fmt.Sprintf("Plugin: %s\n\n%s\n\nSize: %dx%d\n\nType / to switch plugins.",
		p.name, p.description, p.width, p.height)
}
func (p *basePlugin) IsRoot() bool                         { return true }
func (p *basePlugin) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{{Keys: []string{"/"}, Desc: "切换插件"}, {Keys: []string{"q"}, Desc: "退出"}}
}

func main() {
	app := uix.NewApp("dtui > ")

	app.Register(&basePlugin{name: "containers", description: "Manage Docker containers", aliases: []string{"c"}})
	app.Register(&basePlugin{name: "images", description: "Manage Docker images", aliases: []string{"i"}})
	app.Register(&basePlugin{name: "compose", description: "Docker Compose orchestration", aliases: []string{"co"}})
	app.Register(&basePlugin{name: "deploy", description: "Frontend deployment", aliases: []string{"d"}})
	app.Register(&basePlugin{name: "config", description: "Configuration management", aliases: []string{"cfg"}})

	if err := app.Run(); err != nil {
		panic(err)
	}
}
