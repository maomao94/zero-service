package components

import (
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

type FilePicker struct {
	fp     filepicker.Model
	width  int
	height int
}

func NewFilePicker(width int) FilePicker {
	if width <= 0 {
		width = 80
	}
	fp := filepicker.New()
	if home, err := os.UserHomeDir(); err == nil {
		fp.CurrentDirectory = home
	} else {
		fp.CurrentDirectory, _ = os.Getwd()
	}
	fp.DirAllowed = false
	fp.FileAllowed = true
	fp.ShowHidden = false
	fp.AutoHeight = false
	fp.Height = 8

	fp.KeyMap.Back = key.NewBinding(key.WithKeys("h", "backspace", "left"))

	fp.Styles = filepicker.Styles{
		DisabledCursor:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)),
		Cursor:           lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true),
		Symlink:          lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)),
		Directory:        lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true),
		File:             lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorFg)),
		DisabledFile:     lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)),
		Permission:       lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)),
		Selected:         lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent)).Bold(true),
		DisabledSelected: lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)),
		FileSize:         lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Width(7).Align(lipgloss.Right),
		EmptyDirectory:   lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).PaddingLeft(2),
	}
	fp.Cursor = "▶"

	return FilePicker{fp: fp, width: width, height: 24}
}

func (fp *FilePicker) Init() tea.Cmd {
	return fp.fp.Init()
}

func (fp FilePicker) Update(msg tea.Msg) (FilePicker, tea.Cmd) {
	m, cmd := fp.fp.Update(msg)
	fp.fp = m
	return fp, cmd
}

func (fp FilePicker) View() string {
	width := fp.width
	if width <= 0 {
		width = 80
	}
	panelWidth := width - 4
	if panelWidth < 20 {
		panelWidth = 20
	}
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorDim)).
		Render("  " + fp.fp.CurrentDirectory)

	body := fp.fp.View()

	border := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.ColorAccent)).
		BorderTop(true).BorderRight(true).BorderBottom(true).BorderLeft(true).
		Padding(0, 1)

	hints := lipgloss.NewStyle().
		Foreground(lipgloss.Color(theme.ColorDim)).
		Render("  j/k 导航  l/enter 进入  h 返回  esc 关闭  enter 选择文件")

	return header + "\n" + border.Width(panelWidth).Render(body) + "\n" + hints
}

func (fp FilePicker) DidSelectFile(msg tea.Msg) (bool, string) {
	return fp.fp.DidSelectFile(msg)
}

func (fp *FilePicker) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 12
	}
	fp.width = width
	fp.height = height
	if height > 5 {
		fp.fp.Height = height - 4
	}
}

func (fp FilePicker) Height() int {
	return fp.fp.Height + 4
}
