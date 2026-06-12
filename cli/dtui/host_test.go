package main

import (
	"strings"
	"testing"

	"zero-service/cli/dtui/internal/config"
	"zero-service/cli/uix"
)

func TestBuildAppStartsWithoutDockerDaemon(t *testing.T) {
	cfg := config.Config{}
	app := buildApp(cfg)
	if app == nil {
		t.Fatal("buildApp returned nil without Docker daemon")
	}
}

func TestAllProductionModulesRegistered(t *testing.T) {
	cfg := config.Config{}
	app := buildApp(cfg)

	wantModules := []string{"test", "containers", "images", "compose", "deploy", "config"}
	modules := app.Modules()
	names := make(map[string]bool, len(modules))
	for _, m := range modules {
		names[m.Name()] = true
	}
	for _, want := range wantModules {
		if !names[want] {
			t.Errorf("module %q not registered; registered: %v", want, moduleNames(modules))
		}
	}
}

func TestModuleAliasesResolve(t *testing.T) {
	cfg := config.Config{}
	app := buildApp(cfg)

	aliases := map[string]string{
		"ctr": "containers",
		"img": "images",
		"cfg": "config",
	}
	commands := app.Commands()
	cmdNames := make(map[string]bool, len(commands))
	for _, c := range commands {
		cmdNames[c.Name] = true
		for _, a := range c.Aliases {
			cmdNames[a] = true
		}
	}
	for alias, module := range aliases {
		if !cmdNames[alias] {
			t.Errorf("alias %q for module %q does not resolve as command; available: %v", alias, module, commandNames(commands))
		}
	}
}

func TestStartupMessagesDescribeProductionApp(t *testing.T) {
	cfg := config.Config{}
	app := buildApp(cfg)
	messages := app.Messages()

	if len(messages) == 0 {
		t.Fatal("no startup messages")
	}

	var texts []string
	for _, m := range messages {
		texts = append(texts, m.Content)
	}
	all := strings.Join(texts, "\n")

	if !strings.Contains(all, "/containers") {
		t.Error("startup messages should mention /containers")
	}
	if !strings.Contains(all, "/images") {
		t.Error("startup messages should mention /images")
	}
	if !strings.Contains(all, "/compose") {
		t.Error("startup messages should mention /compose")
	}
	if !strings.Contains(all, "/deploy") {
		t.Error("startup messages should mention /deploy")
	}
	if !strings.Contains(all, "/config") {
		t.Error("startup messages should mention /config")
	}
	if !strings.Contains(all, "Docker") {
		t.Error("startup messages should mention Docker behavior")
	}
}

func TestBuiltinCommandsRegistered(t *testing.T) {
	cfg := config.Config{}
	app := buildApp(cfg)

	wantBuiltins := []string{"help", "clear", "exit"}
	commands := app.Commands()
	cmdNames := make(map[string]bool, len(commands))
	for _, c := range commands {
		cmdNames[c.Name] = true
	}
	for _, want := range wantBuiltins {
		if !cmdNames[want] {
			t.Errorf("builtin command %q not registered", want)
		}
	}
}

func moduleNames(modules []uix.Module) []string {
	names := make([]string, len(modules))
	for i, m := range modules {
		names[i] = m.Name()
	}
	return names
}

func commandNames(commands []uix.Command) []string {
	names := make([]string, len(commands))
	for i, c := range commands {
		names[i] = c.Name
	}
	return names
}
