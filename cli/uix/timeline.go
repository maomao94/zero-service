package uix

import (
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

type MessageRole string

const (
	RoleUser      MessageRole = "user"
	RoleAssistant MessageRole = "assistant"
	RoleSystem    MessageRole = "system"
	RoleTool      MessageRole = "tool"
	RoleModule    MessageRole = "module"
)

type Message struct {
	Role      MessageRole
	Content   string
	Status    string
	Timestamp time.Time
}

type Timeline struct {
	messages []Message
	viewport viewport.Model
	width    int
	height   int
}

func NewTimeline(width, height int) Timeline {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	return Timeline{viewport: viewport.New(width, height), width: width, height: height}
}

func (t *Timeline) Append(role MessageRole, content string) {
	if strings.TrimSpace(content) == "" {
		return
	}
	t.messages = append(t.messages, Message{Role: role, Content: content, Timestamp: time.Now()})
	t.updateContent()
	t.viewport.GotoBottom()
}

func (t Timeline) Messages() []Message {
	messages := make([]Message, len(t.messages))
	copy(messages, t.messages)
	return messages
}

func (t *Timeline) Clear() {
	t.messages = nil
	t.updateContent()
}

func (t *Timeline) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	t.width = width
	t.height = height
	t.viewport.Width = width
	t.viewport.Height = height
	t.updateContent()
}

func (t Timeline) View() string {
	if len(t.messages) == 0 {
		return t.emptyView()
	}
	return t.viewport.View()
}

func (t *Timeline) LineUp()     { t.viewport.LineUp(1) }
func (t *Timeline) LineDown()   { t.viewport.LineDown(1) }
func (t *Timeline) PageUp()     { t.viewport.PageUp() }
func (t *Timeline) PageDown()   { t.viewport.PageDown() }
func (t *Timeline) GotoTop()    { t.viewport.GotoTop() }
func (t *Timeline) GotoBottom() { t.viewport.GotoBottom() }

func (t *Timeline) updateContent() {
	var b strings.Builder
	for i, message := range t.messages {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(t.renderMessage(message))
	}
	t.viewport.SetContent(b.String())
}

func (t Timeline) renderMessage(message Message) string {
	label := string(message.Role)
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorFg))

	switch message.Role {
	case RoleUser:
		label = "you"
		style = style.BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color(theme.ColorAccent)).PaddingLeft(1)
	case RoleAssistant:
		label = "assistant"
		style = style.BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color(theme.ColorGreen)).PaddingLeft(1)
	case RoleSystem:
		label = "system"
		style = style.Foreground(lipgloss.Color(theme.ColorDim)).BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color(theme.ColorDim)).PaddingLeft(1)
	case RoleTool:
		label = "tool"
		style = style.BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color(theme.ColorYellow)).PaddingLeft(1)
	case RoleModule:
		label = "module"
		style = style.BorderLeft(true).BorderStyle(lipgloss.ThickBorder()).BorderForeground(lipgloss.Color(theme.ColorAccentDark)).PaddingLeft(1)
	}

	header := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim)).Render(label + "  " + message.Timestamp.Format("15:04:05"))
	contentWidth := t.width - 4
	if contentWidth < 20 {
		contentWidth = 20
	}
	content := lipgloss.NewStyle().Width(contentWidth).Render(message.Content)
	return style.Width(contentWidth + 2).Render(header + "\n" + content)
}

func (t Timeline) emptyView() string {
	width := t.width - 4
	if width < 30 {
		width = 30
	}
	title := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color(theme.ColorAccent)).Render("uix shell")
	lines := []string{
		title,
		"",
		"输入消息开始对话，或输入 / 打开模块与命令。",
		"@ 引用文件或上下文，# 选择资源，! 已预留给 shell 命令。",
	}
	box := theme.Border("home").Width(width).Render(strings.Join(lines, "\n"))
	return lipgloss.Place(t.width, t.height, lipgloss.Center, lipgloss.Center, box)
}
