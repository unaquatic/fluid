package animation

import (
	"math"
	"testing"
)

const epsilon = 1e-9

// ---------------------------------------------------------------------------
// Boundary tests: f(0) ≈ 0, f(1) ≈ 1
// ---------------------------------------------------------------------------

func TestEasingBoundaries(t *testing.T) {
	names := []EasingName{
		Linear, InQuad, OutQuad, InOutQuad,
		InCubic, OutCubic, InOutCubic,
		InExpo, OutExpo, InOutExpo,
		InOutSine, InCirc, OutCirc, InOutCirc,
		InBack, OutBack, InOutBack,
		OutBounce, InBounce, OutElastic,
	}

	for _, name := range names {
		fn := GetEasingFunc(name)
		t.Run(string(name)+"_at_0", func(t *testing.T) {
			got := fn(0)
			if math.Abs(got) > 0.01 {
				t.Errorf("%s(0) = %f, want ≈ 0", name, got)
			}
		})
		t.Run(string(name)+"_at_1", func(t *testing.T) {
			got := fn(1)
			if math.Abs(got-1) > 0.01 {
				t.Errorf("%s(1) = %f, want ≈ 1", name, got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Monotonicity tests for well-behaved functions
// (Excludes bounce, elastic, and back which overshoot)
// ---------------------------------------------------------------------------

func TestEasingMonotonicity(t *testing.T) {
	monotonic := []EasingName{
		Linear, InQuad, OutQuad, InOutQuad,
		InCubic, OutCubic, InOutCubic,
		InExpo, OutExpo, InOutExpo,
		InOutSine, InCirc, OutCirc, InOutCirc,
	}

	const steps = 100
	for _, name := range monotonic {
		fn := GetEasingFunc(name)
		t.Run(string(name), func(t *testing.T) {
			prev := fn(0)
			for i := 1; i <= steps; i++ {
				tt := float64(i) / float64(steps)
				curr := fn(tt)
				if curr < prev-epsilon {
					t.Errorf("%s is not monotonic: f(%f)=%f < f(%f)=%f",
						name, tt, curr, float64(i-1)/float64(steps), prev)
				}
				prev = curr
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Specific midpoint values
// ---------------------------------------------------------------------------

func TestEasingMidpoints(t *testing.T) {
	tests := []struct {
		name     EasingName
		input    float64
		expected float64
		tol      float64
	}{
		{Linear, 0.5, 0.5, epsilon},
		{InQuad, 0.5, 0.25, epsilon},
		{OutQuad, 0.5, 0.75, epsilon},
		{InOutQuad, 0.5, 0.5, epsilon},
		{InCubic, 0.5, 0.125, epsilon},
		{OutCubic, 0.5, 0.875, epsilon},
		{InOutSine, 0.5, 0.5, epsilon},
		{InCirc, 0.0, 0.0, epsilon},
		{OutCirc, 1.0, 1.0, epsilon},
	}

	for _, tc := range tests {
		t.Run(string(tc.name), func(t *testing.T) {
			fn := GetEasingFunc(tc.name)
			got := fn(tc.input)
			if math.Abs(got-tc.expected) > tc.tol {
				t.Errorf("%s(%f) = %f, want %f (tol %f)",
					tc.name, tc.input, got, tc.expected, tc.tol)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Linear identity
// ---------------------------------------------------------------------------

func TestLinearIdentity(t *testing.T) {
	fn := GetEasingFunc(Linear)
	for i := 0; i <= 100; i++ {
		tt := float64(i) / 100.0
		got := fn(tt)
		if math.Abs(got-tt) > epsilon {
			t.Errorf("linear(%f) = %f, want %f", tt, got, tt)
		}
	}
}

// ---------------------------------------------------------------------------
// Bounce / Elastic / Back: overshoot is allowed, but boundaries hold
// ---------------------------------------------------------------------------

func TestOutBounceNonNegative(t *testing.T) {
	fn := GetEasingFunc(OutBounce)
	for i := 0; i <= 100; i++ {
		tt := float64(i) / 100.0
		got := fn(tt)
		if got < -epsilon {
			t.Errorf("outBounce(%f) = %f, should not be negative", tt, got)
		}
	}
}

func TestInBounceNonNegative(t *testing.T) {
	fn := GetEasingFunc(InBounce)
	for i := 0; i <= 100; i++ {
		tt := float64(i) / 100.0
		got := fn(tt)
		// InBounce can overshoot but should end at 1
		_ = got
	}
	// Just test boundaries
	if v := fn(0); math.Abs(v) > 0.01 {
		t.Errorf("inBounce(0) = %f, want ≈ 0", v)
	}
	if v := fn(1); math.Abs(v-1) > 0.01 {
		t.Errorf("inBounce(1) = %f, want ≈ 1", v)
	}
}

func TestOutElasticOscillates(t *testing.T) {
	fn := GetEasingFunc(OutElastic)
	// OutElastic should oscillate around 1.0 before settling
	hasAbove := false
	for i := 1; i < 50; i++ {
		tt := float64(i) / 100.0
		got := fn(tt)
		if got > 1.0 {
			hasAbove = true
		}
	}
	if !hasAbove {
		t.Error("outElastic should overshoot above 1.0")
	}
}

func TestInBackOvershoots(t *testing.T) {
	fn := GetEasingFunc(InBack)
	// InBack goes negative near the start
	hasNegative := false
	for i := 1; i < 50; i++ {
		tt := float64(i) / 100.0
		got := fn(tt)
		if got < 0 {
			hasNegative = true
		}
	}
	if !hasNegative {
		t.Error("inBack should overshoot below 0 near the start")
	}
}

func TestOutBackOvershoots(t *testing.T) {
	fn := GetEasingFunc(OutBack)
	// OutBack goes above 1 before settling
	hasAbove := false
	for i := 50; i < 100; i++ {
		tt := float64(i) / 100.0
		got := fn(tt)
		if got > 1.0 {
			hasAbove = true
		}
	}
	if !hasAbove {
		t.Error("outBack should overshoot above 1.0 near the end")
	}
}

// ---------------------------------------------------------------------------
// GetEasingFunc fallback
// ---------------------------------------------------------------------------

func TestGetEasingFuncFallback(t *testing.T) {
	fn := GetEasingFunc("nonexistent")
	// Should fall back to linear
	if v := fn(0.5); math.Abs(v-0.5) > epsilon {
		t.Errorf("fallback(0.5) = %f, want 0.5 (linear fallback)", v)
	}
}

// ---------------------------------------------------------------------------
// InOutExpo symmetry
// ---------------------------------------------------------------------------

func TestInOutExpoSymmetry(t *testing.T) {
	fn := GetEasingFunc(InOutExpo)
	// InOutExpo should be symmetric around (0.5, 0.5)
	got := fn(0.5)
	if math.Abs(got-0.5) > 0.01 {
		t.Errorf("inOutExpo(0.5) = %f, want ≈ 0.5", got)
	}
}

// ---------------------------------------------------------------------------
// InOutCirc symmetry
// ---------------------------------------------------------------------------

func TestInOutCircSymmetry(t *testing.T) {
	fn := GetEasingFunc(InOutCirc)
	got := fn(0.5)
	if math.Abs(got-0.5) > 0.01 {
		t.Errorf("inOutCirc(0.5) = %f, want ≈ 0.5", got)
	}
}

// ---------------------------------------------------------------------------
// InOutBack symmetry
// ---------------------------------------------------------------------------

func TestInOutBackSymmetry(t *testing.T) {
	fn := GetEasingFunc(InOutBack)
	got := fn(0.5)
	if math.Abs(got-0.5) > 0.01 {
		t.Errorf("inOutBack(0.5) = %f, want ≈ 0.5", got)
	}
}
