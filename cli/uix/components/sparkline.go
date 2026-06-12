package components

import (
	"math"

	"github.com/NimbleMarkets/ntcharts/sparkline"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

// Sparkline wraps ntcharts/sparkline with project theme styling.
// It displays time-series data as a mini line chart with columns.
type Sparkline struct {
	model  sparkline.Model
	width  int
	height int
}

// NewSparkline creates a new Sparkline with the given dimensions.
// Safe defaults: width <= 0 → 20, height <= 0 → 5.
func NewSparkline(width, height int) *Sparkline {
	if width <= 0 {
		width = 20
	}
	if height <= 0 {
		height = 5
	}
	style := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent))
	m := sparkline.New(width, height, sparkline.WithStyle(style))
	return &Sparkline{
		model:  m,
		width:  width,
		height: height,
	}
}

// SetData replaces all sparkline data with the given values.
func (s *Sparkline) SetData(data []float64) {
	s.model.Clear()
	s.model.PushAll(data)
	s.model.Draw()
}

// AddData appends data values to the sparkline.
func (s *Sparkline) AddData(data ...float64) {
	s.model.PushAll(data)
	s.model.Draw()
}

// SetSize updates the sparkline dimensions and redraws.
func (s *Sparkline) SetSize(width, height int) {
	if width <= 0 {
		width = 20
	}
	if height <= 0 {
		height = 5
	}
	s.width = width
	s.height = height
	s.model.Resize(width, height)
	s.model.Draw()
}

// View renders the sparkline as a string.
func (s *Sparkline) View() string {
	return s.model.View()
}

// SetStyle sets a custom Lip Gloss style for the sparkline columns.
func (s *Sparkline) SetStyle(style lipgloss.Style) {
	s.model.Style = style
	s.model.Draw()
}

// SetMax sets the expected maximum value for scaling.
func (s *Sparkline) SetMax(max float64) {
	s.model.SetMax(math.Max(max, 1))
	s.model.Draw()
}

// Clear resets the sparkline data.
func (s *Sparkline) Clear() {
	s.model.Clear()
}
