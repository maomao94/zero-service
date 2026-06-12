package components

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

// ListItem wraps list.Item for use with the List component.
type ListItem struct {
	title       string
	description string
}

// NewListItem creates a new ListItem.
func NewListItem(title, description string) ListItem {
	return ListItem{title: title, description: description}
}

func (i ListItem) Title() string       { return i.title }
func (i ListItem) Description() string { return i.description }
func (i ListItem) FilterValue() string { return i.title }

// List wraps bubbles/list with project theme styling.
type List struct {
	model  list.Model
	width  int
	height int
}

// NewList creates a new List with the given items and dimensions.
func NewList(items []list.Item, width, height int) List {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	l := list.New(items, list.NewDefaultDelegate(), width, height)
	l.Title = ""
	l.SetShowStatusBar(false)
	l.SetFilteringEnabled(false)
	l.SetShowTitle(false)
	l.SetShowPagination(true)
	l.SetShowHelp(false)
	l.Styles.PaginationStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	l.Styles.HelpStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	return List{
		model:  l,
		width:  width,
		height: height,
	}
}

// SetFilteringEnabled enables or disables filtering.
func (l *List) SetFilteringEnabled(enabled bool) {
	l.model.SetFilteringEnabled(enabled)
}

// SetTitle sets the list title.
func (l *List) SetTitle(title string) {
	l.model.Title = title
	l.model.SetShowTitle(title != "")
}

// SetShowStatusBar sets whether to show the status bar.
func (l *List) SetShowStatusBar(show bool) {
	l.model.SetShowStatusBar(show)
}

// Items returns the current list items.
func (l List) Items() []list.Item {
	return l.model.Items()
}

// SetItems replaces all items in the list.
func (l *List) SetItems(items []list.Item) {
	l.model.SetItems(items)
}

// SelectedItem returns the currently selected item.
func (l List) SelectedItem() list.Item {
	item := l.model.SelectedItem()
	if item == nil {
		return nil
	}
	return item
}

// Index returns the current index.
func (l List) Index() int {
	return l.model.Index()
}

// Update processes list messages.
func (l List) Update(msg tea.Msg) (List, tea.Cmd) {
	var cmd tea.Cmd
	l.model, cmd = l.model.Update(msg)
	return l, cmd
}

// View renders the list.
func (l List) View() string {
	return l.model.View()
}

// SetSize updates the list dimensions.
func (l *List) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 20
	}
	l.width = width
	l.height = height
	l.model.SetWidth(width)
	l.model.SetHeight(height)
}
