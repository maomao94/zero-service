package uix

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Runner interface {
	Run(input string, history []Message) tea.Cmd
}

type MockRunner struct{}

type runnerResultMsg struct {
	tool    string
	content string
	err     error
}

func (MockRunner) Run(input string, _ []Message) tea.Cmd {
	return func() tea.Msg {
		input = strings.TrimSpace(input)
		if input == "" {
			return runnerResultMsg{}
		}
		return runnerResultMsg{
			tool:    "mock runner processed the prompt locally; no provider credentials were used",
			content: "Mock assistant received: " + input,
		}
	}
}
