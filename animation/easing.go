package animation

import "math"

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// EasingFunc maps a linear progress value (0.0-1.0) to an eased value.
type EasingFunc func(t float64) float64

// EasingName identifies a built-in easing function by name.
type EasingName string

const (
	Linear     EasingName = "linear"
	InQuad     EasingName = "inQuad"
	OutQuad    EasingName = "outQuad"
	InOutQuad  EasingName = "inOutQuad"
	InCubic    EasingName = "inCubic"
	OutCubic   EasingName = "outCubic"
	InOutCubic EasingName = "inOutCubic"
	InExpo     EasingName = "inExpo"
	OutExpo    EasingName = "outExpo"
	InOutExpo  EasingName = "inOutExpo"
	InOutSine  EasingName = "inOutSine"
	InCirc     EasingName = "inCirc"
	OutCirc    EasingName = "outCirc"
	InOutCirc  EasingName = "inOutCirc"
	InBack     EasingName = "inBack"
	OutBack    EasingName = "outBack"
	InOutBack  EasingName = "inOutBack"
	OutBounce  EasingName = "outBounce"
	InBounce   EasingName = "inBounce"
	OutElastic EasingName = "outElastic"
)

// ---------------------------------------------------------------------------
// Lookup
// ---------------------------------------------------------------------------

// easingFuncs maps easing names to their implementations.
var easingFuncs = map[EasingName]EasingFunc{
	Linear:     easeLinear,
	InQuad:     easeInQuad,
	OutQuad:    easeOutQuad,
	InOutQuad:  easeInOutQuad,
	InCubic:    easeInCubic,
	OutCubic:   easeOutCubic,
	InOutCubic: easeInOutCubic,
	InExpo:     easeInExpo,
	OutExpo:    easeOutExpo,
	InOutExpo:  easeInOutExpo,
	InOutSine:  easeInOutSine,
	InCirc:     easeInCirc,
	OutCirc:    easeOutCirc,
	InOutCirc:  easeInOutCirc,
	InBack:     easeInBack,
	OutBack:    easeOutBack,
	InOutBack:  easeInOutBack,
	OutBounce:  easeOutBounce,
	InBounce:   easeInBounce,
	OutElastic: easeOutElastic,
}

// GetEasingFunc returns the easing function for the given name.
// Returns easeLinear if the name is not recognized.
func GetEasingFunc(name EasingName) EasingFunc {
	if fn, ok := easingFuncs[name]; ok {
		return fn
	}
	return easeLinear
}

// ---------------------------------------------------------------------------
// Implementations — all are pure math with no allocations.
// ---------------------------------------------------------------------------

func easeLinear(t float64) float64 {
	return t
}

func easeInQuad(t float64) float64 {
	return t * t
}

func easeOutQuad(t float64) float64 {
	return t * (2 - t)
}

func easeInOutQuad(t float64) float64 {
	if t < 0.5 {
		return 2 * t * t
	}
	return -1 + (4-2*t)*t
}

func easeInCubic(t float64) float64 {
	return t * t * t
}

func easeOutCubic(t float64) float64 {
	t--
	return t*t*t + 1
}

func easeInOutCubic(t float64) float64 {
	if t < 0.5 {
		return 4 * t * t * t
	}
	t--
	return 1 + 4*t*t*t // Equivalent to: (2t-2)^3/2 + 1 simplified
}

func easeInExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	return math.Pow(2, 10*(t-1))
}

func easeOutExpo(t float64) float64 {
	if t == 1 {
		return 1
	}
	return -math.Pow(2, -10*t) + 1
}

func easeInOutExpo(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	if t < 0.5 {
		return math.Pow(2, 20*t-10) / 2
	}
	return (2 - math.Pow(2, -20*t+10)) / 2
}

func easeInOutSine(t float64) float64 {
	return -(math.Cos(math.Pi*t) - 1) / 2
}

func easeInCirc(t float64) float64 {
	return 1 - math.Sqrt(1-t*t)
}

func easeOutCirc(t float64) float64 {
	t--
	return math.Sqrt(1 - t*t)
}

func easeInOutCirc(t float64) float64 {
	if t < 0.5 {
		return (1 - math.Sqrt(1-4*t*t)) / 2
	}
	t = 2*t - 2
	return (math.Sqrt(1-t*t) + 1) / 2
}

const backOvershoot = 1.70158

func easeInBack(t float64) float64 {
	const s = backOvershoot + 1 // 2.70158
	return t * t * (s*t - backOvershoot)
}

func easeOutBack(t float64) float64 {
	const s = backOvershoot + 1 // 2.70158
	t--
	return t*t*(s*t+backOvershoot) + 1
}

func easeInOutBack(t float64) float64 {
	const s = backOvershoot * 1.525 // Standard overshoot for in-out
	if t < 0.5 {
		t *= 2
		return (t * t * ((s+1)*t - s)) / 2
	}
	t = t*2 - 2
	return (t*t*((s+1)*t+s) + 2) / 2
}

func easeOutBounce(t float64) float64 {
	const (
		n1 = 7.5625
		d1 = 2.75
	)
	switch {
	case t < 1/d1:
		return n1 * t * t
	case t < 2/d1:
		t -= 1.5 / d1
		return n1*t*t + 0.75
	case t < 2.5/d1:
		t -= 2.25 / d1
		return n1*t*t + 0.9375
	default:
		t -= 2.625 / d1
		return n1*t*t + 0.984375
	}
}

func easeInBounce(t float64) float64 {
	return 1 - easeOutBounce(1-t)
}

func easeOutElastic(t float64) float64 {
	if t == 0 {
		return 0
	}
	if t == 1 {
		return 1
	}
	const p = 0.3
	return math.Pow(2, -10*t)*math.Sin((t-0.075)*2*math.Pi/p) + 1
}
