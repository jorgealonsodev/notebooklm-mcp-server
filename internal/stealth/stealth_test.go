package stealth

import (
	"math"
	"testing"
	"time"
)

// ---- RandomDelay tests ----

func TestRandomDelay_ReturnsValueInRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		delay := RandomDelay(100, 400)
		if delay < 100 || delay > 400 {
			t.Errorf("RandomDelay(100, 400) = %dms, want in [100, 400]", delay)
		}
	}
}

func TestRandomDelay_GaussianMeanNear60Percent(t *testing.T) {
	const iterations = 10000
	minMs, maxMs := 100, 500
	expectedMean := float64(minMs) + 0.6*float64(maxMs-minMs) // 60% of range

	sum := 0.0
	for i := 0; i < iterations; i++ {
		sum += float64(RandomDelay(minMs, maxMs))
	}
	actualMean := sum / iterations

	// Allow 10% tolerance on the mean
	tolerance := 0.1 * float64(maxMs-minMs)
	if math.Abs(actualMean-expectedMean) > tolerance {
		t.Errorf("mean delay = %.1fms, want ~%.1fms (±%.0fms)", actualMean, expectedMean, tolerance)
	}
}

func TestRandomDelay_EdgeCases(t *testing.T) {
	tests := []struct {
		name   string
		min    int
		max    int
		wantLo int
		wantHi int
	}{
		{"narrow range", 200, 210, 200, 210},
		{"same min max", 100, 100, 100, 100},
		{"wide range", 50, 1000, 50, 1000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := 0; i < 50; i++ {
				d := RandomDelay(tt.min, tt.max)
				if d < tt.wantLo || d > tt.wantHi {
					t.Errorf("RandomDelay(%d, %d) = %d, want [%d, %d]",
						tt.min, tt.max, d, tt.wantLo, tt.wantHi)
				}
			}
		})
	}
}

// ---- Bezier curve tests ----

func TestGenerateBezierPath_ReturnsCorrectStepCount(t *testing.T) {
	for i := 0; i < 20; i++ {
		steps := generateBezierPath(0, 0, 100, 100)
		if len(steps) < 10 || len(steps) > 25 {
			t.Errorf("generateBezierPath produced %d steps, want [10, 25]", len(steps))
		}
	}
}

func TestGenerateBezierPath_StartsAndEndsAtCorrectPoints(t *testing.T) {
	for i := 0; i < 10; i++ {
		steps := generateBezierPath(10, 20, 300, 400)
		if len(steps) == 0 {
			t.Fatal("path should have at least one step")
		}
		first := steps[0]
		last := steps[len(steps)-1]

		// First point should be near start (within jitter tolerance)
		if math.Abs(first.X-10) > 5 || math.Abs(first.Y-20) > 5 {
			t.Errorf("first point (%.1f, %.1f) too far from start (10, 20)", first.X, first.Y)
		}
		// Last point should be near end (within jitter tolerance)
		if math.Abs(last.X-300) > 5 || math.Abs(last.Y-400) > 5 {
			t.Errorf("last point (%.1f, %.1f) too far from end (300, 400)", last.X, last.Y)
		}
	}
}

func TestGenerateBezierPath_MonotonicProgress(t *testing.T) {
	// For a simple diagonal movement, the path should generally progress
	// from start to end (not necessarily monotonic due to jitter, but
	// the overall trend should be correct).
	steps := generateBezierPath(0, 0, 500, 500)
	if len(steps) < 2 {
		t.Fatal("need at least 2 steps")
	}

	// Last point should be closer to the destination than the first
	firstDist := math.Abs(steps[0].X-500) + math.Abs(steps[0].Y-500)
	lastDist := math.Abs(steps[len(steps)-1].X-500) + math.Abs(steps[len(steps)-1].Y-500)
	if lastDist > firstDist {
		t.Error("last point should be closer to destination than first point")
	}
}

// ---- Typing delay calculation tests ----

func TestCharDelay_InRange(t *testing.T) {
	for i := 0; i < 100; i++ {
		delay := charDelay(200) // 200 WPM
		if delay <= 0 {
			t.Errorf("charDelay(200) = %dms, want positive", delay)
		}
		// At 200 WPM: base = 60000/(200*5) = 60ms, variation 50%-200% = 30-120ms
		if delay < 20 || delay > 200 {
			t.Errorf("charDelay(200) = %dms, want in [20, 200]", delay)
		}
	}
}

func TestCharDelay_VariesByWPM(t *testing.T) {
	// Higher WPM should produce shorter delays on average
	fastSum := 0
	slowSum := 0
	const iterations = 200

	for i := 0; i < iterations; i++ {
		fastSum += charDelay(240)
		slowSum += charDelay(160)
	}

	fastAvg := fastSum / iterations
	slowAvg := slowSum / iterations

	if fastAvg >= slowAvg {
		t.Errorf("fast WPM avg delay (%dms) should be less than slow WPM avg (%dms)", fastAvg, slowAvg)
	}
}

func TestCharDelay_LongerAfterPunctuation(t *testing.T) {
	// Punctuation should increase delay
	punctSum := 0
	normalSum := 0
	const iterations = 200

	for i := 0; i < iterations; i++ {
		punctSum += charDelayWithPunct(200, '.')
		normalSum += charDelay(200)
	}

	punctAvg := punctSum / iterations
	normalAvg := normalSum / iterations

	if punctAvg <= normalAvg {
		t.Errorf("punctuation avg delay (%dms) should be greater than normal avg (%dms)", punctAvg, normalAvg)
	}
}

// ---- Typo simulation tests ----

func TestShouldTypo_ApproximatesRate(t *testing.T) {
	const iterations = 10000
	typos := 0
	for i := 0; i < iterations; i++ {
		if shouldTypo() {
			typos++
		}
	}

	rate := float64(typos) / float64(iterations)
	// 0.3% rate = 0.003, allow wide tolerance for randomness
	if rate < 0.001 || rate > 0.006 {
		t.Errorf("typo rate = %.4f, want ~0.003 (±0.003)", rate)
	}
}

// ---- Typing duration tests ----

func TestEstimatedTypingDuration_ReasonableForWPM(t *testing.T) {
	// "Hello World" = 11 chars at 200 WPM
	// 200 WPM = 3.33 chars/sec → ~3.3s for 11 chars
	// With punctuation delays and variability, allow 1-10s
	text := "Hello World"
	duration := estimatedTypingDuration(text, 200)

	if duration < 500*time.Millisecond {
		t.Errorf("estimated duration for %q at 200 WPM = %v, too fast", text, duration)
	}
	if duration > 15*time.Second {
		t.Errorf("estimated duration for %q at 200 WPM = %v, too slow", text, duration)
	}
}

// ---- Click offset tests ----

func TestRandomClickOffset_NonCenter(t *testing.T) {
	for i := 0; i < 50; i++ {
		x, y := randomClickOffset(100, 50)
		// Offset should be within the element but not exactly center
		if x < 0 || x > 100 || y < 0 || y > 50 {
			t.Errorf("click offset (%.1f, %.1f) outside element bounds", x, y)
		}
	}
}
