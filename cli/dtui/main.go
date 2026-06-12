package main

import (
	"fmt"
	"os"

	"zero-service/cli/dtui/plugins/test"
	"zero-service/cli/uix"
)

func main() {
	app := uix.NewApp("dtui > ")
	app.RegisterModule(test.New())
	app.AppendMessage(uix.RoleSystem, "DTUI test host ready. Type /test to exercise the uix shell; Docker is not required.")
	app.AppendMessage(uix.RoleSystem, "Prompt modes: / commands, @ references, # file picker, ! disabled shell prefix.")

	if err := app.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
