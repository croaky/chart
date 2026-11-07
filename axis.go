package chart

import (
	"fmt"
	"strconv"

	"github.com/croaky/hml"
)

// AxisTick is one axis label and where it sits: 0 at the start of the
// axis and 1 at the end.
//
// A tick carries its own position because a log axis does not space
// its ticks evenly. 20, 50, 100, 200 are four steps that a reader reads
// as even, and four gaps of different sizes on the plot.
type AxisTick struct {
	Label string  `json:"label"`
	At    float64 `json:"at"`
}

// PlacedTick is an AxisTick that a view can place. Style sets
// --chart-at to the tick's position as a percentage. The CSS of the
// axis spends it as a bottom offset on a y axis and as a left offset on
// an x axis, so one tick type serves both axes.
//
// A view draws a grid rule from the same style, so a rule cannot sit
// anywhere but on the label it belongs to.
type PlacedTick struct {
	Label string      `json:"label"`
	Style hml.SafeCSS `json:"style"`
}

// PlacedTicks places each tick of an axis.
func PlacedTicks(ticks []AxisTick) []PlacedTick {
	out := make([]PlacedTick, 0, len(ticks))
	for _, t := range ticks {
		out = append(out, PlacedTick{Label: t.Label, Style: atStyle(t.At)})
	}
	return out
}

// atStyle places an element at a fraction of the plot from 0 to 1.
func atStyle(at float64) hml.SafeCSS {
	return hml.SafeCSS(fmt.Sprintf("--chart-at: %.2f%%", 100*at))
}

// scale is an axis that maps a value to a position: 0 at the bottom or
// the left of the plot and 1 at the top or the right. [ValueAxis] and
// [LogAxis] are the two, so a builder has one code path for both.
type scale interface {
	Up(v float64) float64
	ends() (lo, hi float64)
	tickList() []AxisTick
}

// AxisScale returns the top of a bar chart's axis for small integers
// and its two labels: the top, then zero.
//
// A caller scales a bar against the returned top, not against its own
// largest value. The tallest bar then stops at a round number that a
// reader can name, and not at the top of the box.
//
// It returns two labels and not one for each step. The axis says how
// tall the plot is. A reader takes the count of one column from its
// tip, not from a tick beside it.
//
// The step table sets the top: the value rounded up to a step that
// widens with the range, so 39 tops out at 40 and 3,800 at 4,000. Use
// [NewValueAxis] for a value that is not a small integer.
//
// A maximum at or below zero returns the smallest axis, 10, so a
// caller that draws anyway divides by no zero.
func AxisScale(maxVal int) (int, []string) {
	if maxVal <= 0 {
		return 10, []string{"10", "0"}
	}

	var step int
	switch {
	case maxVal <= 10:
		step = 2
	case maxVal <= 25:
		step = 5
	case maxVal <= 50:
		step = 10
	case maxVal <= 100:
		step = 20
	case maxVal <= 200:
		step = 50
	case maxVal <= 500:
		step = 100
	default:
		step = wideStep(maxVal)
	}

	maxY := ((maxVal + step - 1) / step) * step
	return maxY, []string{strconv.Itoa(maxY), "0"}
}

// wideStep is the step for an axis over 500: the smallest of 1, 2, or
// 5 times a power of ten that divides the range into five gaps or
// fewer. A step fixed at 100 drew 39 ticks under 3,800.
func wideStep(maxVal int) int {
	for magnitude := 100; ; magnitude *= 10 {
		for _, m := range []int{1, 2, 5} {
			if step := m * magnitude; (maxVal+step-1)/step <= 5 {
				return step
			}
		}
	}
}
