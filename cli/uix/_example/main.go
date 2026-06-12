package main

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"

	"zero-service/cli/uix"
)

type baseModule struct {
	name        string
	description string
	aliases     []string
	width       int
	height      int
}

func (m *baseModule) Name() string                            { return m.name }
func (m *baseModule) Description() string                     { return m.description }
func (m *baseModule) Aliases() []string                       { return m.aliases }
func (m *baseModule) Init() tea.Cmd                           { return nil }
func (m *baseModule) Update(msg tea.Msg) (tea.Model, tea.Cmd) { return m, nil }
func (m *baseModule) SetSize(w, h int)                        { m.width = w; m.height = h }
func (m *baseModule) View() string {
	return fmt.Sprintf("Module: %s\n\n%s\n\nSize: %dx%d\n\nType / to switch modules.",
		m.name, m.description, m.width, m.height)
}
func (m *baseModule) IsRoot() bool { return true }
func (m *baseModule) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{{Keys: []string{"/"}, Desc: "切换模块"}, {Keys: []string{"esc"}, Desc: "返回"}}
}

func main() {
	app := uix.NewApp("dtui > ")

	app.RegisterModule(&baseModule{name: "notes", description: "Scratch module", aliases: []string{"n"}})
	app.RegisterModule(&baseModule{name: "logs", description: "Output module", aliases: []string{"l"}})
	app.AppendMessage(uix.RoleSystem, "Example ready. Type / to open modules or send a chat message.")

	if err := app.Run(); err != nil {
		panic(err)
	}
}
