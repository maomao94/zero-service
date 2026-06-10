package main

import (
	"fmt"
	"os"

	"zero-service/cli/dtui/internal/config"
	"zero-service/cli/dtui/internal/docker"
	"zero-service/cli/dtui/plugins/compose"
	"zero-service/cli/dtui/plugins/containers"
	"zero-service/cli/dtui/plugins/deploy"
	"zero-service/cli/dtui/plugins/images"
	"zero-service/cli/dtui/plugins/settings"
	"zero-service/cli/dtui/plugins/home"
	"zero-service/cli/uix"
)

func main() {
	configPath := config.DefaultPath()
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg := config.Load(configPath)

	client, err := docker.NewClient()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Docker client error: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	if err := client.Ping(); err != nil {
		fmt.Fprintf(os.Stderr, "Docker daemon not running: %v\n", err)
		os.Exit(1)
	}

	app := uix.NewApp("dtui > ")

	containerPlugin := containers.New(client)
	imagePlugin := images.New(client)
	composePlugin := compose.New(client, cfg)
	deployPlugin := deploy.New(client, cfg)
	settingsPlugin := settings.New(cfg)

	app.Register(containerPlugin)
	app.Register(imagePlugin)
	app.Register(composePlugin)
	app.Register(deployPlugin)
	app.Register(settingsPlugin)

	welcomeScreen := home.NewScreen()
	app.SetHome(func() string {
		welcomeScreen.SetSize(app.SafeWidth(), app.BodyHeight())
		return welcomeScreen.View()
	})

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
