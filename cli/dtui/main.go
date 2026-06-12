package main

import (
	"fmt"
	"os"

	"zero-service/cli/dtui/internal/config"
	"zero-service/cli/dtui/plugins/compose"
	cfgplugin "zero-service/cli/dtui/plugins/config"
	"zero-service/cli/dtui/plugins/containers"
	"zero-service/cli/dtui/plugins/deploy"
	"zero-service/cli/dtui/plugins/images"
	"zero-service/cli/dtui/plugins/test"
	"zero-service/cli/uix"
)

func main() {
	cfg := config.Load(config.DefaultPath())
	app := buildApp(cfg)
	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func buildApp(cfg config.Config) *uix.Shell {
	app := uix.NewApp("dtui > ")
	app.RegisterModule(test.New())
	app.RegisterModule(containers.New())
	app.RegisterModule(images.New())
	app.RegisterModule(compose.New(cfg))
	app.RegisterModule(deploy.New(cfg))
	app.RegisterModule(cfgplugin.New(cfg, ""))
	app.AppendMessage(uix.RoleSystem, "DTUI — Docker Terminal UI. Type / to open the command palette.")
	app.AppendMessage(uix.RoleSystem, "Modules: /containers (ctr), /images (img), /compose, /deploy, /config (cfg), /test.")
	app.AppendMessage(uix.RoleSystem, "Docker modules require the Docker daemon; the app starts without it.")
	app.AppendMessage(uix.RoleSystem, "Prompt modes: / commands, @ references, # file picker, ! disabled shell prefix.")
	return app
}
