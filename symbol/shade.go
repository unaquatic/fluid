package symbol

// Shade characters — used for backgrounds, progress bars, gauges, and fills.

const (
	ShadeLight  = '░' // U+2591 Light shade (25%)
	ShadeMedium = '▒' // U+2592 Medium shade (50%)
	ShadeDark   = '▓' // U+2593 Dark shade (75%)
	ShadeFull   = '█' // U+2588 Full block (100%)
)

// GaugeFill is the filled portion character for progress bars/gauges.
const GaugeFill = '█'

// GaugeEmpty is the empty portion character for progress bars/gauges.
const GaugeEmpty = '░'
