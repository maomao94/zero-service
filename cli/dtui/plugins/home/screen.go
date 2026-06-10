package home

import (
	"zero-service/cli/uix/components"
)

const logo = `
    ██████╗ ████████╗██╗   ██╗██╗
    ██╔══██╗╚══██╔══╝██║   ██║██║
    ██║  ██║   ██║   ██║   ██║██║
    ██║  ██║   ██║   ██║   ██║██║
    ██████╔╝   ██║   ╚██████╔╝██║
    ╚═════╝    ╚═╝    ╚═════╝ ╚═╝

  Docker Terminal User Interface
`

// Screen is the welcome/home screen (not a Plugin).
type Screen struct {
	inner components.WelcomeScreen
}

func NewScreen() *Screen {
	return &Screen{inner: components.NewWelcomeScreen(logo)}
}

// SetSize updates the area for centering.
func (s *Screen) SetSize(w, h int) {
	s.inner.SetSize(w, h)
}

// View renders the centered welcome screen.
func (s Screen) View() string {
	return s.inner.View()
}
