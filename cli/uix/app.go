package uix

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/components"
)

type ShowModalMsg struct {
	Title   string
	Message string
	Buttons []components.ModalButton
}

type ConfirmMsg struct {
	Button string
}

type FileSelectedMsg struct {
	Path string
}

type FrameworkApp struct {
	width      int
	height     int
	bodyHeight int

	cmdbar     components.CmdBar
	statusbar  components.StatusBar
	dropdown   components.Dropdown
	filepicker *components.FilePicker
	modal      *components.Modal

	showModal      bool
	showFilepicker bool
	dropdownMode   components.InputMode

	registry *PluginRegistry
	active   Plugin
	homeView func() string
}

func NewApp(prompt string) FrameworkApp {
	cmdbar := components.NewCmdBar(prompt)
	statusbar := components.NewStatusBar()
	statusbar.SetRight("Ctrl+C 退出 | / 指令 | # 文件")

	return FrameworkApp{
		width:      80,
		height:     24,
		bodyHeight: 22,
		cmdbar:     cmdbar,
		statusbar:  statusbar,
		dropdown:   components.NewDropdown(80, 12),
		registry:   NewRegistry(),
	}
}

func (app *FrameworkApp) Register(plugin Plugin) {
	app.registry.Register(plugin)
}

func (app *FrameworkApp) SetHome(view func() string) {
	app.homeView = view
	app.statusbar.SetLeft("home")
}

func (app FrameworkApp) Init() tea.Cmd {
	cmds := []tea.Cmd{app.cmdbar.Init()}
	if app.active != nil {
		cmds = append(cmds, app.active.Init())
	}
	return tea.Batch(cmds...)
}

func (app FrameworkApp) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.recalculate(msg.Width, msg.Height)
		return app, nil

	case ShowModalMsg:
		app.ShowModal(msg.Title, msg.Message, msg.Buttons)
		return app, nil

	case FileSelectedMsg:
		if app.active != nil {
			model, cmd := app.active.Update(msg)
			if plugin, ok := model.(Plugin); ok {
				app.active = plugin
			}
			return app, cmd
		}
		return app, nil

	case tea.MouseMsg:
		return app.handleMouse(msg)

	case tea.KeyMsg:
		return app.handleKey(msg)
	}

	var cmds []tea.Cmd

	if app.active != nil && !app.showModal && !app.showFilepicker {
		model, pluginCmd := app.active.Update(msg)
		if plugin, ok := model.(Plugin); ok {
			app.active = plugin
		}
		cmds = append(cmds, pluginCmd)
	}

	if app.active == nil && !app.showModal && !app.showFilepicker {
		cbar, cmd := app.cmdbar.Update(msg)
		app.cmdbar = cbar
		cmds = append(cmds, cmd)
		if c := app.syncMode(); c != nil {
			cmds = append(cmds, c)
		}
	}

	return app, tea.Batch(cmds...)
}

func (app FrameworkApp) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if app.showModal {
		return app, nil
	}
	if app.showFilepicker && app.filepicker != nil {
		fp, cmd := app.filepicker.Update(msg)
		app.filepicker = &fp
		return app, cmd
	}
	return app, nil
}

func (app FrameworkApp) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if app.showModal {
		return app.handleModalKey(msg)
	}

	key := msg.String()

	if key == "ctrl+c" || key == "q" {
		return app, tea.Quit
	}

	if app.showFilepicker {
		return app.handleFilepickerKey(msg)
	}

	if app.dropdownMode != components.ModeFree {
		return app.handleDropdownKey(msg)
	}

	if key == "esc" {
		return app.handleEscape(msg)
	}

	if app.active != nil {
		return app.routePluginKey(msg)
	}

	return app.routeCmdBarKey(msg)
}

func (app FrameworkApp) routePluginKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	var cmds []tea.Cmd

	model, pluginCmd := app.active.Update(msg)
	if plugin, ok := model.(Plugin); ok {
		app.active = plugin
	}
	cmds = append(cmds, pluginCmd)

	if key == "/" || key == "#" {
		cbar, cmd := app.cmdbar.Update(msg)
		app.cmdbar = cbar
		cmds = append(cmds, cmd)
		if c := app.syncMode(); c != nil {
			cmds = append(cmds, c)
		}
	}

	return app, tea.Batch(cmds...)
}

func (app FrameworkApp) routeCmdBarKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cbar, cmd := app.cmdbar.Update(msg)
	app.cmdbar = cbar
	return app, tea.Batch(cmd, app.syncMode())
}

func (app FrameworkApp) handleEscape(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if app.active != nil {
		wasRoot := app.active.IsRoot()
		model, cmd := app.active.Update(msg)
		if plugin, ok := model.(Plugin); ok {
			app.active = plugin
		}
		if wasRoot {
			app.active = nil
			app.statusbar.SetLeft("home")
			app.statusbar.SetRight("Ctrl+C 退出 | / 指令 | # 文件")
		}
		app.cmdbar.SetValue("")
		app.dropdown.Hide()
		return app, cmd
	}
	app.cmdbar.SetValue("")
	app.dropdown.Hide()
	return app, nil
}

func (app FrameworkApp) handleDropdownKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		app.cmdbar.SetValue("")
		app.dropdown.Hide()
		app.dropdownMode = components.ModeFree
		return app, nil

	case "up", "k":
		app.dropdown.MoveUp()
		return app, nil

	case "down", "j":
		app.dropdown.MoveDown()
		return app, nil

	case "enter":
		sel := app.dropdown.Selected()
		if sel == nil {
			return app, nil
		}
		name := sel.Label
		app.cmdbar.SetValue("")
		app.dropdown.Hide()
		app.dropdownMode = components.ModeFree
		app.SwitchPlugin(name)
		if app.active != nil {
			return app, app.active.Init()
		}
		return app, nil

	default:
		cbar, cmd := app.cmdbar.Update(msg)
		app.cmdbar = cbar
		return app, tea.Batch(cmd, app.syncMode())
	}
}

func (app FrameworkApp) handleFilepickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	if key == "esc" {
		app.filepicker = nil
		app.showFilepicker = false
		app.cmdbar.SetValue("")
		return app, nil
	}

	fp, cmd := app.filepicker.Update(msg)
	app.filepicker = &fp

	if didSelect, path := app.filepicker.DidSelectFile(msg); didSelect {
		app.filepicker = nil
		app.showFilepicker = false
		app.cmdbar.SetValue("")
		return app, func() tea.Msg { return FileSelectedMsg{Path: path} }
	}

	return app, cmd
}

func (app FrameworkApp) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()

	switch key {
	case "esc":
		app.showModal = false
		app.modal = nil
		return app, nil

	case "enter":
		if btn := app.modal.SelectedButton(); btn != nil {
			label := btn.Label
			app.showModal = false
			app.modal = nil
			return app, func() tea.Msg { return ConfirmMsg{Button: label} }
		}
		return app, nil

	case "left", "h":
		app.modal.PrevButton()
		return app, nil

	case "right", "l":
		app.modal.NextButton()
		return app, nil
	}

	return app, nil
}

func (app FrameworkApp) View() string {
	if app.showModal && app.modal != nil {
		return app.modal.View()
	}

	body := app.bodyView()
	footer := app.statusbar.View()
	cmdbar := app.cmdbar.View()

	footerHeight := lipgloss.Height(footer)
	cmdbarHeight := lipgloss.Height(cmdbar)

	var middle string
	middleHeight := 0
	if app.showFilepicker && app.filepicker != nil {
		middle = app.filepicker.View()
		middleHeight = app.filepicker.Height()
	} else {
		middle = app.dropdown.View()
		middleHeight = app.dropdown.Height()
	}

	bodyHeight := app.height - cmdbarHeight - footerHeight - middleHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	var parts []string
	parts = append(parts, lipgloss.NewStyle().Width(app.SafeWidth()).Height(bodyHeight).MaxHeight(bodyHeight).Render(body))
	if middleHeight > 0 {
		parts = append(parts, middle)
	}
	parts = append(parts, footer)
	parts = append(parts, cmdbar)

	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (app FrameworkApp) Run() error {
	_, err := tea.NewProgram(app, tea.WithAltScreen()).Run()
	return err
}

func (app *FrameworkApp) SwitchPlugin(name string) {
	plugin := app.registry.Resolve(name)
	if plugin == nil || plugin == app.active {
		return
	}
	app.activatePlugin(plugin)
}

func (app *FrameworkApp) activatePlugin(plugin Plugin) {
	app.active = plugin
	plugin.SetSize(app.width, app.bodyHeight)
	app.statusbar.SetLeft(plugin.Name())
	app.statusbar.SetRight(HelpText(plugin.Bindings()) + " | esc 返回 | / 指令 | # 文件")
}

func (app *FrameworkApp) ShowModal(title, message string, buttons []components.ModalButton) {
	m := components.NewModal(title, message, buttons, app.width)
	app.modal = &m
	app.showModal = true
}

func (app *FrameworkApp) syncMode() tea.Cmd {
	prefix := app.cmdbar.Prefix()
	query := app.cmdbar.Query()

	switch prefix {
	case "/":
		if app.dropdownMode != components.ModeCommand {
			entries := app.buildCommandEntries()
			app.dropdown.SetEntries(entries)
			app.dropdownMode = components.ModeCommand
		}
		app.dropdown.Filter(query)

	case "#":
		app.dropdown.Hide()
		app.dropdownMode = components.ModeFree
		if !app.showFilepicker {
			fp := components.NewFilePicker(app.width)
			app.filepicker = &fp
			app.showFilepicker = true
			return app.filepicker.Init()
		}

	default:
		app.dropdown.Hide()
		app.dropdownMode = components.ModeFree
	}
	return nil
}

func (app FrameworkApp) buildCommandEntries() []components.DropdownEntry {
	plugins := app.registry.List()
	entries := make([]components.DropdownEntry, 0, len(plugins))
	for _, p := range plugins {
		if p.Name() == "welcome" {
			continue
		}
		entries = append(entries, components.DropdownEntry{
			Label:       p.Name(),
			Description: p.Description(),
			Prefix:      "/",
		})
	}
	return entries
}

func (app *FrameworkApp) recalculate(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}

	app.width = width
	app.height = height
	app.cmdbar.SetWidth(width)
	app.statusbar.SetWidth(width)
	app.dropdown.SetWidth(width)
	if app.filepicker != nil {
		fpHeight := min(15, height/3)
		app.filepicker.SetSize(width, fpHeight)
	}
	if app.modal != nil {
		app.modal.Width = width
	}

	cmdbarHeight := 2
	footerHeight := 2
	middleHeight := 0
	if app.showFilepicker && app.filepicker != nil {
		middleHeight = app.filepicker.Height()
	} else {
		middleHeight = app.dropdown.Height()
	}
	app.bodyHeight = height - cmdbarHeight - footerHeight - middleHeight
	if app.bodyHeight < 1 {
		app.bodyHeight = 1
	}
	if app.active != nil {
		app.active.SetSize(width, app.bodyHeight)
	}
}

func (app FrameworkApp) bodyView() string {
	if app.active == nil {
		if app.homeView != nil {
			return app.homeView()
		}
		return "Welcome."
	}
	return app.active.View()
}

func (app FrameworkApp) SafeWidth() int {
	if app.width <= 0 {
		return 80
	}
	return app.width
}

func (app FrameworkApp) BodyHeight() int {
	return app.bodyHeight
}

func HelpText(bindings []HelpBinding) string {
	parts := make([]string, 0, len(bindings))
	for _, binding := range bindings {
		if len(binding.Keys) == 0 || binding.Desc == "" {
			continue
		}
		parts = append(parts, strings.Join(binding.Keys, "/")+" "+binding.Desc)
	}
	return strings.Join(parts, " | ")
}
