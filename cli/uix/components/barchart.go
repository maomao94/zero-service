package components

import (
	"github.com/NimbleMarkets/ntcharts/barchart"
	"github.com/charmbracelet/lipgloss"

	"zero-service/cli/uix/theme"
)

// BarChart wraps ntcharts/barchart with project theme styling.
// It displays categorical data as vertical or horizontal bars.
type BarChart struct {
	model  barchart.Model
	width  int
	height int
}

// NewBarChart creates a new BarChart with the given dimensions and data.
// Safe defaults: width <= 0 → 40, height <= 0 → 10.
func NewBarChart(width, height int, labels []string, values []float64) *BarChart {
	if width <= 0 {
		width = 40
	}
	if height <= 0 {
		height = 10
	}
	m := barchart.New(width, height)
	m.AxisStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorDim))
	m.LabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorFg))

	bc := &BarChart{
		model:  m,
		width:  width,
		height: height,
	}

	if len(labels) > 0 && len(values) > 0 {
		bc.SetData(labels, values)
	}

	return bc
}

// SetData replaces all bar chart data with the given labels and values.
// Labels and values must have the same length.
func (bc *BarChart) SetData(labels []string, values []float64) {
	bc.model.Clear()
	bc.pushData(labels, values)
	bc.model.Draw()
}

// pushData adds data to the model without clearing or drawing.
func (bc *BarChart) pushData(labels []string, values []float64) {
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorAccent))
	green := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorGreen))
	yellow := lipgloss.NewStyle().Foreground(lipgloss.Color(theme.ColorYellow))
	styles := []lipgloss.Style{accent, green, yellow}

	for i, label := range labels {
		if i >= len(values) {
			break
		}
		style := styles[i%len(styles)]
		bc.model.Push(barchart.BarData{
			Label: label,
			Values: []barchart.BarValue{
				{Name: label, Value: values[i], Style: style},
			},
		})
	}
}

// SetSize updates the bar chart dimensions and redraws.
func (bc *BarChart) SetSize(width, height int) {
	if width <= 0 {
		width = 40
	}
	if height <= 0 {
		height = 10
	}
	bc.width = width
	bc.height = height
	bc.model.Resize(width, height)
	bc.model.Draw()
}

// View renders the bar chart as a string.
func (bc *BarChart) View() string {
	return bc.model.View()
}

// SetHorizontal sets whether bars are displayed horizontally.
func (bc *BarChart) SetHorizontal(horizontal bool) {
	bc.model.SetHorizontal(horizontal)
	bc.model.Draw()
}

// SetShowAxis sets whether to display axis and labels.
func (bc *BarChart) SetShowAxis(show bool) {
	bc.model.SetShowAxis(show)
	bc.model.Draw()
}

// Clear resets the bar chart data.
func (bc *BarChart) Clear() {
	bc.model.Clear()
}
