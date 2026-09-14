package symbol

// Shade runes ordered by increasing cell coverage.
const (
	ShadeLight  = '░' // Light shade, approximately 25 percent coverage.
	ShadeMedium = '▒' // Medium shade, approximately 50 percent coverage.
	ShadeDark   = '▓' // Dark shade, approximately 75 percent coverage.
	ShadeFull   = '█' // Full-block coverage.
)

// GaugeFill draws the filled portion of a gauge.
const GaugeFill = '█'

// GaugeEmpty draws the unfilled portion of a gauge.
const GaugeEmpty = '░'
