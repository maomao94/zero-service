package components

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"

	"zero-service/cli/uix/theme"
)

// Progress wraps bubbles/progress with project theme styling.
type Progress struct {
	progress progress.Model
	percent  float64
	width    int
}

// NewProgress creates a new Progress bar with the given width.
func NewProgress(width int) Progress {
	if width <= 0 {
		width = 80
	}
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithWidth(width),
		progress.WithoutPercentage(),
	)
	return Progress{
		progress: p,
		width:    width,
	}
}

// SetPercent sets the progress percentage (0.0 to 1.0).
func (p *Progress) SetPercent(pct float64) {
	if pct < 0 {
		pct = 0
	}
	if pct > 1 {
		pct = 1
	}
	p.percent = pct
}

// Percent returns the current progress percentage.
func (p Progress) Percent() float64 {
	return p.percent
}

// Update processes progress animation messages.
func (p Progress) Update(msg tea.Msg) (Progress, tea.Cmd) {
	m, cmd := p.progress.Update(msg)
	p.progress = m.(progress.Model)
	return p, cmd
}

// View renders the progress bar.
func (p Progress) View() string {
	return p.progress.ViewAs(p.percent)
}

// SetSize updates the progress bar width.
func (p *Progress) SetSize(width, height int) {
	if width <= 0 {
		width = 80
	}
	p.width = width
	p.progress.Width = width
}

// SetFullColor sets a custom full color for the progress bar.
func (p *Progress) SetFullColor(color string) {
	p.progress.FullColor = color
}

// SetEmptyColor sets a custom empty color for the progress bar.
func (p *Progress) SetEmptyColor(color string) {
	p.progress.EmptyColor = color
}

// SetCustomColors sets both full and empty colors from theme palette.
func (p *Progress) SetCustomColors() {
	p.progress.FullColor = theme.ColorAccent
	p.progress.EmptyColor = theme.ColorDim
}
