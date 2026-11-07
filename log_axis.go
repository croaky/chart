package chart

import "math"

// Range is a low and a high value above zero, inclusive. A log axis
// takes its domain from one.
type Range struct {
	Min, Max float64
}

// Add widens the range to hold v.
//
// A value at or below zero, a NaN, and an infinity are no value: a log
// scale has no place for them. A zero Min means that the range is still
// empty, not that it reaches down to nothing.
func (r *Range) Add(v float64) {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return
	}
	r.Max = max(r.Max, v)
	if r.Min == 0 || v < r.Min {
		r.Min = v
	}
}

// LogAxis is a scale that gives each order of magnitude the same room.
//
// It is for a right-skewed metric. When nine values in ten sit under
// 200 and one is 1,600, a linear axis draws the nine in a line along
// the bottom and spends three quarters of its height on the one. On a
// log axis 40 against 400 reads as far apart as 400 against 4,000,
// which is the comparison a reader makes.
//
// The ends are round values just outside the data, not whole powers of
// ten. Values from 20 to 468 draw from 20 to 500. Rounding out to 10
// and 1,000 would leave the data in a third of the height.
type LogAxis struct {
	// Min and Max are the ends, both above zero.
	Min, Max float64
	// Ticks are the labels, smallest first, each with its position.
	Ticks []AxisTick
}

// logTickTarget is about how many ticks a log axis wants. The ladder is
// coarse, so the count lands within one or two of it.
//
// Five and not four: a reader cannot interpolate between two ticks of a
// log axis by eye, so it needs more of them than a linear axis.
const logTickTarget = 5

// mantissas is the 1/2/5 ladder. A round value on a log axis is one of
// these times a power of ten.
var mantissas = []float64{1, 2, 5}

// NewLogAxis scales a range, labels it in f, and returns an axis that
// holds every value in it.
//
// A range with nothing above zero, or with one value, still returns an
// axis with two ends. A caller that draws anyway divides by no zero.
func NewLogAxis(r Range, f Format) LogAxis {
	lo, hi := r.Min, r.Max
	if hi <= 0 || math.IsNaN(hi) || math.IsInf(hi, 0) {
		hi = 1
	}
	if lo <= 0 || lo > hi || math.IsNaN(lo) {
		lo = hi
	}

	a := LogAxis{Min: roundDown(lo), Max: roundUp(hi)}
	// One value, or two a hair apart, would divide by zero below. The
	// decade above it is an axis that holds the value.
	if a.Max <= a.Min {
		a.Max = a.Min * 10
	}
	a.Ticks = a.tickLadder(f)
	return a
}

// Up is where v sits: 0 at the bottom of the axis and 1 at the top. A
// value outside the axis sits on its edge. A log scale has no place for
// zero, so a value at or below zero sits at the bottom, with the
// smallest values.
func (a LogAxis) Up(v float64) float64 {
	if v <= 0 || math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	at := (math.Log10(v) - math.Log10(a.Min)) /
		(math.Log10(a.Max) - math.Log10(a.Min))
	return min(max(at, 0), 1)
}

func (a LogAxis) tickList() []AxisTick { return a.Ticks }

func (a LogAxis) ends() (lo, hi float64) { return a.Min, a.Max }

// tickLadder labels each round value the axis covers, then keeps fewer
// mantissas until the count is near logTickTarget.
//
// It thins by whole mantissas, not by single ticks. An axis of 1, 2, 5,
// 10, 20, 50 thinned to 1, 5, 20 reads as three unrelated numbers. An
// axis of 1, 10, 100 reads as a scale.
func (a LogAxis) tickLadder(f Format) []AxisTick {
	for _, keep := range [][]float64{mantissas, {1, 5}, {1}} {
		ticks := a.ticksFor(keep, f)
		if len(ticks) <= logTickTarget {
			return ticks
		}
	}
	// Past a thousandfold range even one tick per decade is too many,
	// so this keeps every other decade.
	return everyOther(a.ticksFor([]float64{1}, f))
}

// ticksFor labels each given mantissa in each decade the axis covers,
// smallest first.
func (a LogAxis) ticksFor(keep []float64, f Format) []AxisTick {
	var ticks []AxisTick
	lowExp := int(math.Floor(math.Log10(a.Min)))
	highExp := int(math.Ceil(math.Log10(a.Max)))

	for e := lowExp; e <= highExp; e++ {
		decade := math.Pow(10, float64(e))
		for _, m := range keep {
			v := m * decade
			// The ends are round values, so the comparison has to
			// admit them exactly. The slack is far below the gap
			// between two mantissas.
			if v < a.Min*0.999 || v > a.Max*1.001 {
				continue
			}
			// A count has no tenth of a thing to label. A step below
			// one prints the same label twice: 2, 2, 1, 1.
			if (f == Count || f == Signed || f == Millis) && v < 1 {
				continue
			}
			ticks = append(ticks, AxisTick{Label: FormatValue(v, f), At: a.Up(v)})
		}
	}
	return ticks
}

// everyOther keeps the first tick and every second one after it, so
// the bottom of the axis always has a label.
func everyOther(ticks []AxisTick) []AxisTick {
	out := make([]AxisTick, 0, (len(ticks)+1)/2)
	for i, t := range ticks {
		if i%2 == 0 {
			out = append(out, t)
		}
	}
	return out
}

// roundDown is the largest round value at or below v, and roundUp the
// smallest at or above it. Round is the 1/2/5 ladder: 468 rounds up to
// 500, and 20 stays at 20.
func roundDown(v float64) float64 {
	decade := math.Pow(10, math.Floor(math.Log10(v)))
	best := decade
	for _, m := range mantissas {
		if step := m * decade; step <= v*1.001 {
			best = step
		}
	}
	return best
}

func roundUp(v float64) float64 {
	decade := math.Pow(10, math.Floor(math.Log10(v)))
	for _, m := range mantissas {
		if step := m * decade; step >= v*0.999 {
			return step
		}
	}
	return 10 * decade
}
