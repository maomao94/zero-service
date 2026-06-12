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

type AppendMessageMsg struct {
	Role    MessageRole
	Content string
}

type StatusMsg struct {
	Left  string
	Right string
}

type Shell struct {
	width      int
	height     int
	bodyHeight int

	prompt    components.CmdBar
	statusbar components.StatusBar
	dropdown  components.Dropdown
	filepick  *components.FilePicker
	modal     *components.Modal

	showModal      bool
	showFilepicker bool
	dropdownMode   components.InputMode

	modules  *ModuleRegistry
	commands *CommandRegistry
	active   Module
	timeline Timeline
	runner   Runner
}

func NewShell(prompt string) *Shell {
	cmdbar := components.NewCmdBar(prompt)
	statusbar := components.NewStatusBar()
	statusbar.SetLeft("chat")
	statusbar.SetRight("enter 发送 | / 指令 | @ 引用 | # 文件 | esc 返回")

	app := &Shell{
		width:      80,
		height:     24,
		bodyHeight: 20,
		prompt:     cmdbar,
		statusbar:  statusbar,
		dropdown:   components.NewDropdown(80, 12),
		modules:    NewModuleRegistry(),
		commands:   NewCommandRegistry(),
		timeline:   NewTimeline(80, 20),
		runner:     MockRunner{},
	}
	app.registerBuiltins()
	return app
}

func NewApp(prompt string) *Shell { return NewShell(prompt) }

func (app *Shell) RegisterModule(module Module) {
	if module == nil {
		return
	}
	app.modules.Register(module)
	name := module.Name()
	app.RegisterCommand(Command{
		Name:        name,
		Description: module.Description(),
		Aliases:     module.Aliases(),
		Run: func(app *Shell) tea.Cmd {
			return app.EnterModule(name)
		},
	})
}

func (app *Shell) RegisterCommand(command Command) {
	app.commands.Register(command)
}

func (app *Shell) SetRunner(runner Runner) {
	if runner != nil {
		app.runner = runner
	}
}

func (app *Shell) AppendMessage(role MessageRole, content string) {
	app.timeline.Append(role, content)
}

func (app *Shell) Init() tea.Cmd {
	cmds := []tea.Cmd{app.prompt.Init()}
	if app.active != nil {
		cmds = append(cmds, app.active.Init())
	}
	return tea.Batch(cmds...)
}

func (app *Shell) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		app.recalculate(msg.Width, msg.Height)
		return app, nil
	case ShowModalMsg:
		app.ShowModal(msg.Title, msg.Message, msg.Buttons)
		return app, nil
	case ConfirmMsg:
		return app.routeToActive(msg)
	case FileSelectedMsg:
		if app.active != nil {
			return app.routeToActive(msg)
		}
		app.timeline.Append(RoleTool, "selected file: "+msg.Path)
		return app, nil
	case AppendMessageMsg:
		role := msg.Role
		if role == "" {
			role = RoleModule
		}
		app.timeline.Append(role, msg.Content)
		return app, nil
	case StatusMsg:
		if strings.TrimSpace(msg.Left) != "" {
			app.statusbar.SetLeft(msg.Left)
		}
		if strings.TrimSpace(msg.Right) != "" {
			app.statusbar.SetRight(msg.Right)
		}
		return app, nil
	case runnerResultMsg:
		if msg.err != nil {
			app.timeline.Append(RoleSystem, "runner error: "+msg.err.Error())
			return app, nil
		}
		app.timeline.Append(RoleTool, msg.tool)
		app.timeline.Append(RoleAssistant, msg.content)
		return app, nil
	case tea.MouseMsg:
		return app.handleMouse(msg)
	case tea.KeyMsg:
		return app.handleKey(msg)
	}

	if app.active != nil && !app.showModal && !app.showFilepicker {
		return app.routeToActive(msg)
	}
	return app, nil
}

func (app *Shell) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if app.showModal {
		return app, nil
	}
	if app.showFilepicker && app.filepick != nil {
		fp, cmd := app.filepick.Update(msg)
		app.filepick = &fp
		return app, cmd
	}
	return app, nil
}

func (app *Shell) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if app.showModal {
		return app.handleModalKey(msg)
	}

	key := msg.String()
	if key == "ctrl+c" {
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
		return app.routeModuleKey(msg)
	}
	return app.routePromptKey(msg)
}

func (app *Shell) routePromptKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "enter":
		return app.submitPrompt()
	case "pgup":
		app.timeline.PageUp()
		return app, nil
	case "pgdown":
		app.timeline.PageDown()
		return app, nil
	case "ctrl+g", "home":
		app.timeline.GotoTop()
		return app, nil
	case "end":
		app.timeline.GotoBottom()
		return app, nil
	}

	cbar, cmd := app.prompt.Update(msg)
	app.prompt = cbar
	return app, tea.Batch(cmd, app.syncMode())
}

func (app *Shell) routeModuleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "/" || key == "@" || key == "#" {
		cbar, cmd := app.prompt.Update(msg)
		app.prompt = cbar
		return app, tea.Batch(cmd, app.syncMode())
	}
	return app.routeToActive(msg)
}

func (app *Shell) routeToActive(msg tea.Msg) (tea.Model, tea.Cmd) {
	if app.active == nil {
		return app, nil
	}
	model, cmd := app.active.Update(msg)
	if module, ok := model.(Module); ok {
		app.active = module
	}
	return app, cmd
}

func (app *Shell) submitPrompt() (tea.Model, tea.Cmd) {
	value := strings.TrimSpace(app.prompt.Value())
	if value == "" {
		return app, nil
	}
	app.prompt.SetValue("")
	app.dropdown.Hide()
	app.dropdownMode = components.ModeFree

	switch {
	case strings.HasPrefix(value, "/"):
		name := strings.TrimSpace(strings.TrimPrefix(value, "/"))
		return app, app.runCommand(name)
	case strings.HasPrefix(value, "!"):
		app.timeline.Append(RoleSystem, "shell command execution is disabled in this build: "+value)
		return app, nil
	default:
		app.timeline.Append(RoleUser, value)
		if app.runner == nil {
			return app, nil
		}
		return app, app.runner.Run(value, app.timeline.Messages())
	}
}

func (app *Shell) handleEscape(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	app.prompt.SetValue("")
	app.dropdown.Hide()
	app.dropdownMode = components.ModeFree
	if app.active == nil {
		return app, nil
	}
	if app.active.IsRoot() {
		name := app.active.Name()
		app.active = nil
		app.timeline.Append(RoleModule, "left module: "+name)
		app.statusbar.SetLeft("chat")
		app.statusbar.SetRight("enter 发送 | / 指令 | @ 引用 | # 文件 | esc 返回")
		return app, nil
	}
	model, cmd := app.active.Update(msg)
	if module, ok := model.(Module); ok {
		app.active = module
	}
	return app, cmd
}

func (app *Shell) handleDropdownKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	switch key {
	case "esc":
		app.prompt.SetValue("")
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
			return app.submitPrompt()
		}
		label := sel.Label
		app.prompt.SetValue("")
		app.dropdown.Hide()
		mode := app.dropdownMode
		app.dropdownMode = components.ModeFree
		if mode == components.ModeCommand {
			return app, app.runCommand(label)
		}
		if mode == components.ModeShell {
			app.timeline.Append(RoleSystem, "shell command execution is disabled; ! is reserved for a future safe mode")
			return app, nil
		}
		app.timeline.Append(RoleSystem, sel.Prefix+label+" selected")
		return app, nil
	default:
		cbar, cmd := app.prompt.Update(msg)
		app.prompt = cbar
		return app, tea.Batch(cmd, app.syncMode())
	}
}

func (app *Shell) handleFilepickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	key := msg.String()
	if key == "esc" {
		app.filepick = nil
		app.showFilepicker = false
		app.prompt.SetValue("")
		return app, nil
	}

	fp, cmd := app.filepick.Update(msg)
	app.filepick = &fp
	if didSelect, path := app.filepick.DidSelectFile(msg); didSelect {
		app.filepick = nil
		app.showFilepicker = false
		app.prompt.SetValue("")
		return app, func() tea.Msg { return FileSelectedMsg{Path: path} }
	}
	return app, cmd
}

func (app *Shell) handleModalKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if app.modal == nil {
		app.showModal = false
		return app, nil
	}
	switch msg.String() {
	case "esc":
		var label string
		if btn := app.modal.ButtonByKey("esc"); btn != nil {
			label = btn.Label
		}
		app.showModal = false
		app.modal = nil
		if label != "" {
			return app, func() tea.Msg { return ConfirmMsg{Button: label} }
		}
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

func (app *Shell) View() string {
	if app.showModal && app.modal != nil {
		app.modal.Width = app.SafeWidth()
		app.modal.Height = app.safeHeight()
		return app.modal.View()
	}

	footer := app.statusbar.View()
	prompt := app.prompt.View()
	footerHeight := lipgloss.Height(footer)
	promptHeight := lipgloss.Height(prompt)

	middle := ""
	middleHeight := 0
	if app.showFilepicker && app.filepick != nil {
		middle = app.filepick.View()
		middleHeight = app.filepick.Height()
	} else {
		middle = app.dropdown.View()
		middleHeight = app.dropdown.Height()
	}

	bodyHeight := app.height - footerHeight - promptHeight - middleHeight
	if bodyHeight < 1 {
		bodyHeight = 1
	}
	body := app.bodyView(bodyHeight)

	parts := []string{lipgloss.NewStyle().Width(app.SafeWidth()).Height(bodyHeight).MaxHeight(bodyHeight).Render(body)}
	if middleHeight > 0 {
		parts = append(parts, middle)
	}
	parts = append(parts, footer, prompt)
	return lipgloss.JoinVertical(lipgloss.Left, parts...)
}

func (app *Shell) Run() error {
	_, err := tea.NewProgram(app, tea.WithAltScreen()).Run()
	return err
}

func (app *Shell) EnterModule(name string) tea.Cmd {
	module := app.modules.Resolve(name)
	if module == nil {
		app.timeline.Append(RoleSystem, "unknown module: "+name)
		return nil
	}
	app.active = module
	module.SetSize(app.width, app.bodyHeight)
	app.statusbar.SetLeft(module.Name())
	app.statusbar.SetRight(HelpText(module.Bindings()) + " | esc 返回 | / 指令 | @ 引用 | # 文件")
	app.timeline.Append(RoleModule, "entered module: "+module.Name())
	return module.Init()
}

func (app *Shell) ShowModal(title, message string, buttons []components.ModalButton) {
	m := components.NewModal(title, message, buttons, app.width)
	m.Height = app.safeHeight()
	app.modal = &m
	app.showModal = true
}

func (app *Shell) syncMode() tea.Cmd {
	prefix := app.prompt.Prefix()
	query := app.prompt.Query()
	switch prefix {
	case "/":
		if app.dropdownMode != components.ModeCommand {
			app.dropdown.SetEntries(app.buildCommandEntries())
			app.dropdownMode = components.ModeCommand
		}
		app.dropdown.Filter(query)
	case "@":
		if app.dropdownMode != components.ModeReference {
			app.dropdown.SetEntries(app.buildReferenceEntries())
			app.dropdownMode = components.ModeReference
		}
		app.dropdown.Filter(query)
	case "#":
		app.dropdown.Hide()
		app.dropdownMode = components.ModeFree
		if !app.showFilepicker {
			fp := components.NewFilePicker(app.width)
			app.filepick = &fp
			app.showFilepicker = true
			return app.filepick.Init()
		}
	case "!":
		if app.dropdownMode != components.ModeShell {
			app.dropdown.SetEntries([]components.DropdownEntry{{Label: "disabled", Description: "shell execution is reserved for a future safe mode", Prefix: "!"}})
			app.dropdownMode = components.ModeShell
		}
		app.dropdown.Filter(query)
	default:
		app.dropdown.Hide()
		app.dropdownMode = components.ModeFree
	}
	return nil
}

func (app *Shell) buildCommandEntries() []components.DropdownEntry {
	commands := app.commands.List()
	entries := make([]components.DropdownEntry, 0, len(commands))
	for _, command := range commands {
		entries = append(entries, components.DropdownEntry{Label: command.Name, Description: command.Description, Prefix: "/"})
	}
	return entries
}

func (app *Shell) buildReferenceEntries() []components.DropdownEntry {
	return []components.DropdownEntry{
		{Label: "file", Description: "Use # to select a local file for context", Prefix: "@"},
		{Label: "module", Description: "Use / to open a module", Prefix: "@"},
	}
}

func (app *Shell) runCommand(name string) tea.Cmd {
	command := app.commands.Resolve(name)
	if command == nil {
		app.timeline.Append(RoleSystem, "unknown command: /"+name)
		return nil
	}
	return command.Run(app)
}

func (app *Shell) Messages() []Message { return app.timeline.Messages() }

func (app *Shell) ActiveModule() Module { return app.active }

func (app *Shell) Commands() []Command { return app.commands.List() }

func (app *Shell) Modules() []Module { return app.modules.List() }

func (app *Shell) recalculate(width, height int) {
	if width <= 0 {
		width = 80
	}
	if height <= 0 {
		height = 24
	}
	app.width = width
	app.height = height
	app.prompt.SetWidth(width)
	app.statusbar.SetWidth(width)
	app.dropdown.SetWidth(width)
	if app.filepick != nil {
		app.filepick.SetSize(width, min(15, height/3))
	}
	if app.modal != nil {
		app.modal.Width = width
	}
	app.bodyHeight = height - 4
	if app.bodyHeight < 1 {
		app.bodyHeight = 1
	}
	app.timeline.SetSize(width, app.bodyHeight)
	if app.active != nil {
		app.active.SetSize(width, app.bodyHeight)
	}
}

func (app *Shell) bodyView(height int) string {
	if app.active != nil {
		app.active.SetSize(app.SafeWidth(), height)
		return app.active.View()
	}
	app.timeline.SetSize(app.SafeWidth(), height)
	return app.timeline.View()
}

func (app *Shell) SafeWidth() int {
	if app.width <= 0 {
		return 80
	}
	return app.width
}

func (app *Shell) safeHeight() int {
	if app.height <= 0 {
		return 24
	}
	return app.height
}

func (app *Shell) BodyHeight() int { return app.bodyHeight }

func (app *Shell) registerBuiltins() {
	app.RegisterCommand(Command{
		Name:        "help",
		Description: "Show shell interaction help",
		Aliases:     []string{"h"},
		Run: func(app *Shell) tea.Cmd {
			app.timeline.Append(RoleSystem, "Commands: /help, /clear, /exit, plus registered modules. Use @ for references and # for file selection.")
			return nil
		},
	})
	app.RegisterCommand(Command{
		Name:        "clear",
		Description: "Clear chat timeline",
		Aliases:     []string{"new"},
		Run: func(app *Shell) tea.Cmd {
			app.timeline.Clear()
			return nil
		},
	})
	app.RegisterCommand(Command{
		Name:        "exit",
		Description: "Exit the TUI",
		Aliases:     []string{"quit", "q"},
		Run:         func(app *Shell) tea.Cmd { return tea.Quit },
	})
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
