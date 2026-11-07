package chart

import "math"

// tickTarget is about how many gaps a linear axis wants. The step
// rounds to a number a reader can add in their head, so the count
// lands between two and eight.
//
// Four and not five: a plot is often half a card wide, so the axis is
// short.
const tickTarget = 4

// ValueAxis is a linear axis for any range of numbers, negative ones
// included.
//
// The step is the 1/2/5 sequence at the range's own magnitude, the
// "nice" idea from d3. A reader gets 0, 500M, 1B, and not a count of
// ticks that grows with the data. Both ends of the sequence matter: 2
// keeps a range of 30,000 off a step of 5,000 that labels it seven
// times, and 5 keeps a range of 1B off a step of 200M.
//
// The axis always holds zero. A bar or a line measured from an axis
// that starts at 40 makes a change of 10% look like a change of 100%.
//
// A range that crosses zero, such as -6 to +6, takes its step from its
// larger side, with about three gaps on that side. Both sides then read
// off one step and the axis stays close to the data: -6 to +6 steps by
// 2, where a step from the whole width would be 5 and draw -10 to +10.
type ValueAxis struct {
	// Min and Max are the ends of the axis, each a whole number of
	// steps.
	Min, Max float64
	// Ticks are the labels from the top of the axis down, each with
	// its position.
	Ticks []AxisTick
}

// NewValueAxis scales the range from lo to hi and labels it in f.
//
// A NaN or an infinite end counts as zero. A range of zero width
// returns the axis from 0 to 1, so a caller that draws anyway divides
// by no zero.
func NewValueAxis(lo, hi float64, f Format) ValueAxis {
	lo, hi = finiteOr0(lo), finiteOr0(hi)
	if lo > hi {
		lo, hi = hi, lo
	}
	lo, hi = min(lo, 0), max(hi, 0)
	if lo == hi {
		hi = 1
	}

	raw := (hi - lo) / tickTarget
	if lo < 0 && hi > 0 {
		raw = max(-lo, hi) / (tickTarget - 1)
	}
	step := niceStep(raw)
	// A count has no fraction of a thing to label. A step below one
	// prints the same label twice: 2, 2, 1, 1, 0.
	if (f == Count || f == Signed || f == Millis || (f == Percent && hi-lo >= 1)) && step < 1 {
		step = 1
	}

	a := ValueAxis{
		Min: math.Floor(lo/step) * step,
		Max: math.Ceil(hi/step) * step,
	}
	n := int(math.Round((a.Max - a.Min) / step))
	a.Ticks = make([]AxisTick, 0, n+1)
	for i := n; i >= 0; i-- {
		v := a.Min + float64(i)*step
		// Float error leaves a tick a hair off zero, and 0.0000001
		// prints as a value the axis does not have.
		if math.Abs(v) < step/2 {
			v = 0
		}
		a.Ticks = append(a.Ticks, AxisTick{Label: FormatValue(v, f), At: a.Up(v)})
	}
	return a
}

// Up is where v sits on the axis: 0 at Min and 1 at Max. A value
// outside the axis sits on its edge, and a NaN sits at zero.
func (a ValueAxis) Up(v float64) float64 {
	if math.IsNaN(v) {
		v = 0
	}
	at := (v - a.Min) / (a.Max - a.Min)
	return min(max(at, 0), 1)
}

// Labels are the tick labels from the top of the axis down. A view
// that spaces its labels evenly, such as the y axis of a bar chart,
// reads these and not the positions.
func (a ValueAxis) Labels() []string {
	out := make([]string, 0, len(a.Ticks))
	for _, t := range a.Ticks {
		out = append(out, t.Label)
	}
	return out
}

func (a ValueAxis) tickList() []AxisTick { return a.Ticks }

func (a ValueAxis) ends() (lo, hi float64) { return a.Min, a.Max }

// niceStep rounds a raw interval up to 1, 2, or 5 times a power of
// ten.
func niceStep(raw float64) float64 {
	mag := math.Pow(10, math.Floor(math.Log10(raw)))
	switch norm := raw / mag; {
	case norm <= 1:
		return mag
	case norm <= 2:
		return 2 * mag
	case norm <= 5:
		return 5 * mag
	default:
		return 10 * mag
	}
}

// finiteOr0 is v, or zero when v is NaN or infinite.
func finiteOr0(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}
