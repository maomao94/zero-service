package images

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/table"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	dt "zero-service/cli/dtui/internal/docker"
	"zero-service/cli/uix"
	"zero-service/cli/uix/components"
	"zero-service/cli/uix/theme"
)

var panelBorder = lipgloss.NewStyle().
	BorderStyle(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color(theme.ColorBorder)).
	Padding(0, 1)

type Plugin struct {
	client *dt.Client
	table  table.Model
	width  int
	height int
	images []dt.Image
	cursor int
}

func New(client *dt.Client) *Plugin {
	cols := []table.Column{
		{Title: "Repository", Width: 30},
		{Title: "Tag", Width: 12},
		{Title: "ID", Width: 14},
		{Title: "Size", Width: 10},
	}
	t := table.New(table.WithColumns(cols), table.WithFocused(true))
	s := table.DefaultStyles()
	s.Header = s.Header.BorderForeground(lipgloss.Color(theme.ColorBorder)).BorderBottom(true).Foreground(lipgloss.Color(theme.ColorAccent))
	s.Selected = s.Selected.Background(lipgloss.Color(theme.ColorSelected)).Foreground(lipgloss.Color(theme.ColorFg))
	s.Cell = s.Cell.Foreground(lipgloss.Color(theme.ColorFg))
	t.SetStyles(s)
	return &Plugin{client: client, table: t}
}

func (p *Plugin) Name() string        { return "images" }
func (p *Plugin) Description() string { return "Manage Docker images" }
func (p *Plugin) Aliases() []string   { return []string{"i", "img"} }
func (p *Plugin) IsRoot() bool        { return true }
func (p *Plugin) OnActivate() tea.Cmd { return p.loadImages() }
func (p *Plugin) OnDeactivate()       {}

func (p *Plugin) Init() tea.Cmd { return p.loadImages() }

func (p *Plugin) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case imagesLoadedMsg:
		p.images = msg.images
		p.updateTable()
		return p, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if p.cursor > 0 {
				p.cursor--
			}
		case "down", "j":
			if p.cursor < len(p.images)-1 {
				p.cursor++
			}
		case "r":
			return p, p.loadImages()
		case "x":
			return p.confirmRemove()
		case "p":
			return p.pruneImages()
		}
	}
	var cmd tea.Cmd
	p.table, cmd = p.table.Update(msg)
	return p, cmd
}

func (p *Plugin) View() string {
	return panelBorder.Width(p.width - 2).Render(p.table.View())
}

func (p *Plugin) SetSize(w, h int) {
	p.width = w
	p.height = h
	p.table.SetWidth(w - 6)
	p.table.SetHeight(h - 4)
}

func (p *Plugin) Bindings() []uix.HelpBinding {
	return []uix.HelpBinding{
		{Keys: []string{"↑↓"}, Desc: "选择"},
		{Keys: []string{"x"}, Desc: "删除"},
		{Keys: []string{"p"}, Desc: "清理"},
		{Keys: []string{"r"}, Desc: "刷新"},
	}
}

func (p *Plugin) updateTable() {
	rows := make([]table.Row, len(p.images))
	for i, img := range p.images {
		repo, tag := splitImageName(img.Repository)
		rows[i] = table.Row{repo, tag, img.ID[:12], img.Size}
	}
	p.table.SetRows(rows)
	if p.cursor >= len(p.images) {
		p.cursor = max(0, len(p.images)-1)
	}
	p.table.SetCursor(p.cursor)
}

func (p *Plugin) confirmRemove() (tea.Model, tea.Cmd) {
	if p.cursor < 0 || p.cursor >= len(p.images) {
		return p, nil
	}
	img := p.images[p.cursor]
	return p, func() tea.Msg {
		return uix.ShowModalMsg{
			Title:   "Confirm Remove",
			Message: fmt.Sprintf("Remove image %s?", img.Repository),
			Buttons: []components.ModalButton{
				{Label: "Cancel", Key: "esc"},
				{Label: "Remove", Key: "enter"},
			},
		}
	}
}

func (p *Plugin) pruneImages() (tea.Model, tea.Cmd) {
	return p, func() tea.Msg {
		reclaimed, err := p.client.PruneImages()
		if err != nil {
			return imagesLoadedMsg{err: err}
		}
		imgs, _ := p.client.ListImages("")
		return imagesLoadedMsg{images: imgs, reclaimed: reclaimed}
	}
}

func (p *Plugin) loadImages() tea.Cmd {
	return func() tea.Msg {
		imgs, _ := p.client.ListImages("")
		return imagesLoadedMsg{images: imgs}
	}
}

func splitImageName(full string) (repo, tag string) {
	parts := strings.SplitN(full, ":", 2)
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return full, "latest"
}

type imagesLoadedMsg struct {
	images    []dt.Image
	err       error
	reclaimed uint64
}
