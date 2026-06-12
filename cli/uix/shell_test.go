package uix

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"zero-service/cli/uix/components"
)

type testModule struct {
	name      string
	confirmed string
}

func (m *testModule) Name() string        { return m.name }
func (m *testModule) Description() string { return "test module" }
func (m *testModule) Aliases() []string   { return []string{"tm"} }
func (m *testModule) Init() tea.Cmd       { return nil }
func (m *testModule) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if msg, ok := msg.(ConfirmMsg); ok {
		m.confirmed = msg.Button
	}
	return m, nil
}
func (m *testModule) View() string              { return "test" }
func (m *testModule) SetSize(width, height int) {}
func (m *testModule) Bindings() []HelpBinding   { return nil }
func (m *testModule) IsRoot() bool              { return true }

func TestRegisterModuleRegistersSlashCommand(t *testing.T) {
	shell := NewShell("test > ")
	module := &testModule{name: "demo"}

	shell.RegisterModule(module)
	command := shell.commands.Resolve("tm")
	if command == nil {
		t.Fatal("expected module alias to resolve as command")
	}

	command.Run(shell)
	if shell.ActiveModule() != module {
		t.Fatalf("expected active module %q, got %#v", module.Name(), shell.ActiveModule())
	}
}

func TestSubmitPromptRunsMockRunnerLifecycle(t *testing.T) {
	shell := NewShell("test > ")
	shell.prompt.SetValue("hello")

	_, cmd := shell.submitPrompt()
	if cmd == nil {
		t.Fatal("expected mock runner command")
	}

	_, _ = shell.Update(cmd())
	messages := shell.Messages()
	if len(messages) != 3 {
		t.Fatalf("expected user/tool/assistant messages, got %d", len(messages))
	}
	if messages[0].Role != RoleUser || messages[1].Role != RoleTool || messages[2].Role != RoleAssistant {
		t.Fatalf("unexpected message roles: %#v", messages)
	}
}

func TestShellPrefixIsDisabled(t *testing.T) {
	shell := NewShell("test > ")
	shell.prompt.SetValue("! rm -rf /")

	_, cmd := shell.submitPrompt()
	if cmd != nil {
		t.Fatal("expected disabled shell prefix to avoid command execution")
	}
	messages := shell.Messages()
	if len(messages) != 1 || messages[0].Role != RoleSystem {
		t.Fatalf("expected one system message, got %#v", messages)
	}
	if !strings.Contains(messages[0].Content, "disabled") {
		t.Fatalf("expected disabled warning, got %q", messages[0].Content)
	}
}

func TestHashModeInitializesFilePicker(t *testing.T) {
	shell := NewShell("test > ")
	shell.prompt.SetValue("#")

	cmd := shell.syncMode()
	if !shell.showFilepicker || shell.filepick == nil {
		t.Fatal("expected # mode to open file picker")
	}
	if cmd == nil {
		t.Fatal("expected file picker init command")
	}
}

func TestModalEscapeSendsCancelButton(t *testing.T) {
	shell := NewShell("test > ")
	module := &testModule{name: "demo"}
	shell.active = module
	shell.ShowModal("Confirm", "Cancel me", nil)
	shell.modal.Buttons = append(shell.modal.Buttons, testCancelButton(), testActionButton())

	_, cmd := shell.handleModalKey(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd == nil {
		t.Fatal("expected cancel confirmation command")
	}
	_, _ = shell.Update(cmd())
	if module.confirmed != "Cancel" {
		t.Fatalf("expected Cancel confirmation, got %q", module.confirmed)
	}
}

func testCancelButton() components.ModalButton {
	return components.ModalButton{Label: "Cancel", Key: "esc"}
}

func testActionButton() components.ModalButton {
	return components.ModalButton{Label: "Run", Key: "enter"}
}
