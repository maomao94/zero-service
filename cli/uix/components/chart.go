package components

// ChartComponent defines the interface for chart components.
// All chart implementations must support resizing and rendering.
type ChartComponent interface {
	// SetSize updates the chart dimensions.
	SetSize(width, height int)
	// View renders the chart as a string.
	View() string
}
