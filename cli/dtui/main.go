package main

import (
	"fmt"
	"os"

	"zero-service/cli/dtui/internal/config"
	"zero-service/cli/dtui/plugins/compose"
	"zero-service/cli/dtui/plugins/deploy"
	"zero-service/cli/dtui/plugins/images"
	"zero-service/cli/dtui/plugins/test"
	"zero-service/cli/uix"
)

func main() {
	cfg := config.Load(config.DefaultPath())
	app := uix.NewApp("dtui > ")
	app.RegisterModule(test.New())
	app.RegisterModule(images.New())
	app.RegisterModule(compose.New(cfg))
	app.RegisterModule(deploy.New(cfg))
	app.AppendMessage(uix.RoleSystem, "DTUI test host ready. Type /test to exercise the uix shell; Docker is not required.")
	app.AppendMessage(uix.RoleSystem, "Type /images to manage Docker images, /compose for Compose projects, /deploy for deployments (requires Docker daemon).")
	app.AppendMessage(uix.RoleSystem, "Prompt modes: / commands, @ references, # file picker, ! disabled shell prefix.")

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
