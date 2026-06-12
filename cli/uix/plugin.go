package uix

import tea "github.com/charmbracelet/bubbletea"

type Module interface {
	Name() string
	Description() string
	Aliases() []string
	Init() tea.Cmd
	Update(msg tea.Msg) (tea.Model, tea.Cmd)
	View() string
	SetSize(width, height int)
	Bindings() []HelpBinding
	IsRoot() bool
}

type HelpBinding struct {
	Keys []string
	Desc string
}

type Command struct {
	Name        string
	Description string
	Aliases     []string
	Run         func(*Shell) tea.Cmd
}
