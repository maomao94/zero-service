package components

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

var (
	cmdbarPromptStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color(theme.ColorAccent)).
				Bold(true)

	cmdbarSeparator = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorBorder)).
			Render("│")

	cmdbarHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorDim))

	slashHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorYellow)).
			Background(lipgloss.Color(theme.ColorSelected)).
			Padding(0, 1)

	hashHintStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color(theme.ColorYellow)).
			Background(lipgloss.Color(theme.ColorSelected)).
			Padding(0, 1)
)

type InputMode int

const (
	ModeFree   InputMode = iota
	ModeCommand
	ModeFile
)

type CmdBar struct {
	input  textinput.Model
	prompt string
	width  int
}

func NewCmdBar(prompt string) CmdBar {
	ti := textinput.New()
	ti.Prompt = ""
	ti.Placeholder = "输入 / 选择指令 | # 选择文件"
	ti.CharLimit = 0
	ti.Width = 60
	ti.Focus()

	return CmdBar{
		input:  ti,
		prompt: prompt,
		width:  80,
	}
}

func (c *CmdBar) Init() tea.Cmd {
	return textinput.Blink
}

func (c CmdBar) Update(msg tea.Msg) (CmdBar, tea.Cmd) {
	var cmd tea.Cmd
	c.input, cmd = c.input.Update(msg)
	return c, cmd
}

func (c CmdBar) View() string {
	prompt := cmdbarPromptStyle.Render(c.prompt)
	input := c.input.View()

	line := lipgloss.JoinHorizontal(lipgloss.Top,
		prompt,
		cmdbarSeparator,
		" ",
		input,
	)

	lineWidth := lipgloss.Width(line)
	if lineWidth < c.width {
		pad := strings.Repeat("─", c.width-lineWidth)
		line += lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorBorder)).Render(pad)
	}

	prefix := c.Prefix()
	hints := ""
	switch prefix {
	case "/":
		hints = "  搜索指令"
	case "#":
		hints = "  搜索文件"
	default:
		hints = "  / 指令  # 文件"
	}
	helpLine := cmdbarHintStyle.Render(hints)

	return line + "\n" + helpLine
}

func (c CmdBar) Height() int {
	return 2
}

func (c *CmdBar) Prefix() string {
	v := c.input.Value()
	if strings.HasPrefix(v, "/") {
		return "/"
	}
	if strings.HasPrefix(v, "#") {
		return "#"
	}
	return ""
}

func (c *CmdBar) Query() string {
	v := c.input.Value()
	if strings.HasPrefix(v, "/") || strings.HasPrefix(v, "#") {
		return strings.TrimPrefix(v[1:], " ")
	}
	return ""
}

func (c CmdBar) Value() string {
	return c.input.Value()
}

func (c *CmdBar) SetValue(s string) {
	c.input.SetValue(s)
}

func (c *CmdBar) SetWidth(w int) {
	if w <= 0 {
		w = 80
	}
	c.width = w
	c.input.Width = max(10, w-lipgloss.Width(c.prompt)-4)
}

func (c *CmdBar) Focus() tea.Cmd {
	return c.input.Focus()
}

func (c *CmdBar) Blur() {
	c.input.Blur()
}
