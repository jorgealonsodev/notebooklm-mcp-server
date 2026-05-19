// Package stealth provides human-like browser behavior utilities
// to reduce bot-detection signals during automation.
package stealth

import (
	"math"
	"math/rand"
	"time"

	"github.com/jorge/notebooklm-mcp-server/internal/config"
	"github.com/playwright-community/playwright-go"
)

// Point represents a 2D coordinate for mouse movement.
type Point struct {
	X, Y float64
}

// StealthOptions holds per-call stealth configuration.
type StealthOptions struct {
	Enabled        *bool
	RandomDelays   *bool
	HumanTyping    *bool
	MouseMovements *bool
	TypingWPMMin   *int
	TypingWPMMax   *int
	DelayMinMs     *int
	DelayMaxMs     *int
}

// resolveOptions merges per-call options with base config values.
func resolveOptions(cfg config.Config, opts *StealthOptions) config.Config {
	out := cfg
	if opts == nil {
		return out
	}
	if opts.Enabled != nil {
		out.StealthEnabled = *opts.Enabled
	}
	if opts.RandomDelays != nil {
		out.StealthRandomDelays = *opts.RandomDelays
	}
	if opts.HumanTyping != nil {
		out.StealthHumanTyping = *opts.HumanTyping
	}
	if opts.MouseMovements != nil {
		out.StealthMouseMovements = *opts.MouseMovements
	}
	if opts.TypingWPMMin != nil {
		out.TypingWPMMin = *opts.TypingWPMMin
	}
	if opts.TypingWPMMax != nil {
		out.TypingWPMMax = *opts.TypingWPMMax
	}
	if opts.DelayMinMs != nil {
		out.MinDelayMs = *opts.DelayMinMs
	}
	if opts.DelayMaxMs != nil {
		out.MaxDelayMs = *opts.DelayMaxMs
	}
	return out
}

// RandomDelay returns a delay in milliseconds using a Gaussian distribution
// centered at 60% of the range with a standard deviation of 20% of the range.
// Values are clamped to [min, max].
func RandomDelay(min, max int) int {
	if min >= max {
		return min
	}
	rng := max - min
	mean := float64(min) + 0.6*float64(rng)
	stdDev := 0.2 * float64(rng)

	val := rand.NormFloat64()*stdDev + mean
	clamped := int(math.Round(val))
	if clamped < min {
		clamped = min
	}
	if clamped > max {
		clamped = max
	}
	return clamped
}

// charDelay computes a variable per-character typing delay based on WPM.
// At 200 WPM, average delay is ~300ms per character.
func charDelay(wpm int) int {
	// chars per minute = WPM * 5 (avg word = 5 chars)
	// ms per char = 60000 / (WPM * 5)
	baseMs := 60000 / (wpm * 5)
	// Add variability: 50% to 200% of base
	variation := 0.5 + rand.Float64()*1.5
	return int(float64(baseMs) * variation)
}

// charDelayWithPunct adds extra delay after punctuation marks.
func charDelayWithPunct(wpm int, ch rune) int {
	delay := charDelay(wpm)
	switch ch {
	case '.', '!', '?', ';', ':':
		delay += charDelay(wpm) * 2 // 3x normal delay after sentence-ending punctuation
	case ',':
		delay += charDelay(wpm) // 2x normal delay after comma
	}
	return delay
}

// shouldTypo returns true with approximately 0.3% probability.
func shouldTypo() bool {
	return rand.Float64() < 0.003
}

// randomTypoChar returns a random nearby keyboard character.
func randomTypoChar(original rune) rune {
	keyboard := "qwertyuiopasdfghjklzxcvbnm"
	for _, c := range keyboard {
		if c == original {
			// Return a random adjacent key
			idx := rand.Intn(len(keyboard))
			return rune(keyboard[idx])
		}
	}
	// Fallback: random lowercase letter
	return rune('a' + rand.Intn(26))
}

// generateBezierPath creates a curved mouse path from (x1,y1) to (x2,y2)
// with 10-25 intermediate steps and jitter.
func generateBezierPath(x1, y1, x2, y2 float64) []Point {
	steps := 10 + rand.Intn(15) // 10-24 iterations → 11-25 points
	path := make([]Point, 0, steps+1)

	// Control points for the Bezier curve (offset from the direct line)
	dx, dy := x2-x1, y2-y1
	// Random control point offset
	offsetX := (rand.Float64() - 0.5) * dx * 0.5
	offsetY := (rand.Float64() - 0.5) * dy * 0.5

	cpx1 := x1 + dx*0.3 + offsetX
	cpy1 := y1 + dy*0.3 + offsetY
	cpx2 := x1 + dx*0.7 - offsetX
	cpy2 := y1 + dy*0.7 - offsetY

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		// Cubic Bezier formula
		mt := 1 - t
		mt2 := mt * mt
		mt3 := mt2 * mt
		t2 := t * t
		t3 := t2 * t

		x := mt3*x1 + 3*mt2*t*cpx1 + 3*mt*t2*cpx2 + t3*x2
		y := mt3*y1 + 3*mt2*t*cpy1 + 3*mt*t2*cpy2 + t3*y2

		// Add jitter
		jitter := 2.0
		x += (rand.Float64() - 0.5) * jitter
		y += (rand.Float64() - 0.5) * jitter

		path = append(path, Point{X: x, Y: y})
	}

	return path
}

// randomClickOffset returns a click position within the element bounds,
// avoiding the exact center to simulate human imprecision.
func randomClickOffset(width, height float64) (float64, float64) {
	// Avoid the center 20% of the element
	margin := 0.15
	x := (margin + rand.Float64()*(1-2*margin)) * width
	y := (margin + rand.Float64()*(1-2*margin)) * height
	return x, y
}

// estimatedTypingDuration returns an approximate duration for typing the
// given text at the specified WPM.
func estimatedTypingDuration(text string, wpm int) time.Duration {
	totalMs := 0
	for _, ch := range text {
		totalMs += charDelayWithPunct(wpm, ch)
	}
	return time.Duration(totalMs) * time.Millisecond
}

// HumanType types the given text character by character with human-like
// delays, variable WPM, and occasional typos with backspace correction.
func HumanType(page playwright.Page, text string, cfg config.Config, opts *StealthOptions) error {
	resolved := resolveOptions(cfg, opts)
	if !resolved.StealthEnabled || !resolved.StealthHumanTyping {
		// Fast type: just send the whole text
		return page.Keyboard().Press(text)
	}

	wpm := resolved.TypingWPMMin + rand.Intn(resolved.TypingWPMMax-resolved.TypingWPMMin+1)

	for _, ch := range text {
		// Simulate typo with backspace correction
		if shouldTypo() {
			wrongChar := randomTypoChar(ch)
			if err := page.Keyboard().Press(string(wrongChar)); err != nil {
				return err
			}
			time.Sleep(time.Duration(RandomDelay(50, 150)) * time.Millisecond)
			// Backspace
			if err := page.Keyboard().Press("Backspace"); err != nil {
				return err
			}
			time.Sleep(time.Duration(RandomDelay(80, 200)) * time.Millisecond)
		}

		// Type the correct character
		if err := page.Keyboard().Press(string(ch)); err != nil {
			return err
		}

		// Variable delay after this character
		delay := charDelayWithPunct(wpm, ch)
		time.Sleep(time.Duration(delay) * time.Millisecond)
	}

	return nil
}

// HumanMouseMove moves the mouse from the current position to (x, y)
// following a Bezier curve with jitter, simulating human movement.
func HumanMouseMove(page playwright.Page, x, y float64, cfg config.Config, opts *StealthOptions) error {
	resolved := resolveOptions(cfg, opts)
	if !resolved.StealthEnabled || !resolved.StealthMouseMovements {
		// Instant move
		return page.Mouse().Move(x, y)
	}

	// Get current mouse position (approximate — we use 0,0 as start if unknown)
	path := generateBezierPath(0, 0, x, y)

	for _, p := range path {
		if err := page.Mouse().Move(p.X, p.Y); err != nil {
			return err
		}
		// Small delay between movement steps
		time.Sleep(time.Duration(RandomDelay(5, 20)) * time.Millisecond)
	}

	return nil
}

// HumanClick combines mouse movement and a non-center click offset.
func HumanClick(page playwright.Page, x, y float64, cfg config.Config, opts *StealthOptions) error {
	if err := HumanMouseMove(page, x, y, cfg, opts); err != nil {
		return err
	}
	// Click with slight offset from center
	offsetX, offsetY := randomClickOffset(10, 10) // small element offset
	return page.Mouse().Click(x+offsetX-5, y+offsetY-5)
}
