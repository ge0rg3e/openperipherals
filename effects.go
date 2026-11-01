package main

import (
	"math"

	"openperipherals/internal/razer"
)

// Software effects have no firmware equivalent, so the lighting loop renders
// them and streams custom frames to keyboards that take them (see
// lighting.go). Their preview formulas live in frontend/src/lib/effects.ts and
// must stay in step with the ones here.

// softwareEffect reports whether an effect id can be drawn by the app. Some
// ids (ripple) run in another brand's firmware, so use softwareOn to decide
// whether a particular keyboard needs the app-drawn path.
func softwareEffect(effect string) bool {
	switch effect {
	case "fire", "ripple", "rain", "comet":
		return true
	}
	return false
}

// fallbackLighting is the closest hardware effect for a keyboard that can't
// show a software effect (or when the app stops and hands the board back).
func fallbackLighting(l KeyboardLighting) KeyboardLighting {
	switch l.Effect {
	case "fire":
		l.Effect = "breathing"
	case "ripple":
		l.Effect = "reactive"
		l.Speed = min(max(l.Speed, 1), 4)
	case "rain":
		l.Effect = "starlight"
		l.Speed = min(max(l.Speed, 1), 3)
	case "comet":
		l.Effect = "wave"
	}
	return l
}

// frameEffectColor is the colour of one LED for a software effect. c1 is the
// parsed effect colour, row and col are matrix coordinates and t is seconds
// since the lighting loop started.
func frameEffectColor(l KeyboardLighting, c1 [3]float64, row, col, rows, cols int, t float64) razer.RGB {
	x := float64(col) / float64(max(cols-1, 1))
	y := float64(row) / float64(max(rows-1, 1))
	seed := effectNoise(x*7.13 + y*3.71)
	rate := effectRate(l.Speed)
	var c [3]float64

	switch l.Effect {
	case "fire":
		// Flames lick upward: heat falls off toward the top of the board and
		// every key flickers on its own beat.
		flicker := effectNoise(seed*53 + math.Floor(t*12*rate))
		wave := 0.5 + 0.5*math.Sin(y*7+t*3.2*rate+seed*5)
		heat := clamp01(0.12 + (1-y)*0.95 + wave*0.16 + (flicker-0.5)*0.38)
		c = effectScale(effectMix(c1, [3]float64{255, 238, 190}, heat*0.5), 0.18+0.82*heat)
	case "ripple":
		// Rings travel out from the middle of the board.
		dx, dy := (x-0.5)*1.6, y-0.5
		d := math.Min(math.Hypot(dx, dy)/0.72, 1)
		p := effectFrac(t/(2.4/rate) - d*1.8)
		level := 0.0
		if p < 0.22 {
			level = math.Sin(p / 0.22 * math.Pi)
		}
		c = effectScale(c1, 0.05+0.95*level)
	case "rain":
		// Two drops fall down each column, offset so the rain never stops.
		phase := effectNoise(x*22+3.1) * 7
		level := rainLevel(t*0.35*rate+phase, y, 0.16)
		level = math.Max(level, 0.6*rainLevel(t*0.35*rate+phase+0.55, y, 0.10))
		c = effectScale(c1, level)
	case "comet":
		// A bright head with a fading tail crosses the board.
		xx := x
		if l.Direction == "left" {
			xx = 1 - x
		}
		d := effectFrac(t*0.28*rate) - xx
		if d < 0 {
			d++
		}
		level := 0.0
		if d < 0.30 {
			level = math.Pow(1-d/0.30, 2.2)
		}
		c = c1
		if d < 0.05 {
			c = effectMix(c1, [3]float64{255, 255, 255}, 1-d/0.05)
		}
		c = effectScale(c, level)
	}
	return razer.RGB{R: toByte(c[0]), G: toByte(c[1]), B: toByte(c[2])}
}

// rainLevel is a falling drop's brightness at height y, with the head at head.
func rainLevel(head, y, tail float64) float64 {
	d := head - y
	if d < 0 {
		d++
	}
	if d >= tail {
		return 0
	}
	return math.Pow(1-d/tail, 2)
}

func effectRate(speed int) float64 { return 0.6 + 0.2*float64(min(max(speed, 1), 4)) }

func effectColor(hex string) [3]float64 {
	c, err := parseHex(hex)
	if err != nil {
		return [3]float64{255, 255, 255}
	}
	return [3]float64{float64(c[0]), float64(c[1]), float64(c[2])}
}

func effectScale(c [3]float64, k float64) [3]float64 {
	return [3]float64{c[0] * k, c[1] * k, c[2] * k}
}

func effectMix(a, b [3]float64, k float64) [3]float64 {
	return [3]float64{a[0] + (b[0]-a[0])*k, a[1] + (b[1]-a[1])*k, a[2] + (b[2]-a[2])*k}
}

// effectNoise is the same stable pseudo-random function as the UI preview.
func effectNoise(n float64) float64 { return effectFrac(math.Sin(n*91.345+47.853) * 24634.6345) }

func effectFrac(v float64) float64 { return v - math.Floor(v) }

func clamp01(v float64) float64 { return min(max(v, 0), 1) }

func toByte(v float64) uint8 { return uint8(math.Round(min(max(v, 0), 255))) }
