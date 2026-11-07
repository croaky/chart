package chart

import (
	"math"
	"testing"

	"github.com/croaky/is"
)

// The top of a small-integer axis is the largest value rounded up to a
// step that widens with the range.
func TestAxisScale(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	cases := []struct {
		maxVal, maxY int
	}{
		{1, 2},
		{7, 8},
		{10, 10},
		{39, 40},
		{100, 100},
		{180, 200},
		{460, 500},
		{3_800, 4_000},
		{12_345, 15_000},
	}
	for _, c := range cases {
		maxY, ticks := AxisScale(c.maxVal)
		is.Eq(maxY, c.maxY)
		is.Eq(len(ticks), 2)
		is.Eq(ticks[1], "0")
	}
}

// A caller that draws anyway gets the smallest axis, not a division by
// zero.
func TestAxisScaleWithNothingToPlot(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	maxY, ticks := AxisScale(0)
	is.Eq(maxY, 10)
	is.Eq(ticks, []string{"10", "0"})
}

func TestPlacedTicks(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	got := PlacedTicks([]AxisTick{{Label: "0", At: 0}, {Label: "50", At: 0.5}})
	is.Eq(got, []PlacedTick{
		{Label: "0", Style: "--chart-at: 0.00%"},
		{Label: "50", Style: "--chart-at: 50.00%"},
	})
}

// The step is the 1/2/5 sequence at the range's own magnitude, so a
// valuation axis reads as three numbers.
func TestValueAxisScalesToTheMagnitude(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	a := NewValueAxis(0, 1_000_000_000, USD)
	is.Eq(a.Max, 1_000_000_000.0)
	is.Eq(a.Labels(), []string{"$1B", "$500M", "$0"})

	a = NewValueAxis(0, 1_150_000_000, USD)
	is.Eq(a.Max, 1_500_000_000.0)
	is.Eq(a.Labels(), []string{"$1.5B", "$1B", "$500M", "$0"})

	a = NewValueAxis(0, 12_000, Count)
	is.Eq(a.Labels(), []string{"15,000", "10,000", "5,000", "0"})

	a = NewValueAxis(0, 250, Count)
	is.Eq(a.Labels(), []string{"300", "200", "100", "0"})
}

// However many steps the axis has, its last tick is zero and not the
// float error that a walk down leaves.
func TestValueAxisEndsAtZero(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	for _, hi := range []float64{0.3, 3, 47, 640, 8_246, 92_500, 2_400_000, 36_000_000_000} {
		a := NewValueAxis(0, hi, Price)
		labels := a.Labels()
		is.True(a.Max >= hi)
		is.True(len(labels) >= 3 && len(labels) <= 9)
		is.Eq(labels[len(labels)-1], "$0")
		is.Eq(a.Ticks[len(a.Ticks)-1].At, 0.0)
		is.Eq(a.Ticks[0].At, 1.0)
	}
}

// A count axis never steps below 1, so a small count labels each tick
// once.
func TestValueAxisStepsByWholeCounts(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	cases := []struct {
		hi     float64
		labels []string
	}{
		{1, []string{"1", "0"}},
		{2, []string{"2", "1", "0"}},
		{3, []string{"3", "2", "1", "0"}},
		{5, []string{"6", "4", "2", "0"}},
	}
	for _, c := range cases {
		is.Eq(NewValueAxis(0, c.hi, Count).Labels(), c.labels)
	}

	// A share under one point keeps its fractions.
	is.Eq(NewValueAxis(0, 0.5, Percent).Labels(), []string{"0.6%", "0.4%", "0.2%", "0%"})
}

// A range that crosses zero steps from its larger side, so -6 to +6
// draws as -6 to +6 and not as -10 to +10.
func TestValueAxisCrossesZero(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	a := NewValueAxis(-6, 6, Count)
	is.Eq(a.Min, -6.0)
	is.Eq(a.Max, 6.0)
	is.Eq(a.Labels(), []string{"6", "4", "2", "0", "-2", "-4", "-6"})
	is.Eq(a.Up(0), 0.5)

	a = NewValueAxis(10, -2, Count)
	is.Eq(a.Labels(), []string{"10", "5", "0", "-5"})

	a = NewValueAxis(-30, -10, Count)
	is.Eq(a.Max, 0.0)
	is.Eq(a.Labels(), []string{"0", "-10", "-20", "-30"})
}

// An empty or broken range is the axis from 0 to 1, not a division by
// zero.
func TestValueAxisWithNothingToPlot(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	for _, a := range []ValueAxis{
		NewValueAxis(0, 0, Count),
		NewValueAxis(math.NaN(), math.Inf(1), Count),
	} {
		is.Eq(a.Min, 0.0)
		is.Eq(a.Max, 1.0)
		is.Eq(a.Labels(), []string{"1", "0"})
	}
}

// A value outside the axis sits on its edge, and a NaN sits at zero.
func TestValueAxisUpClamps(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	a := NewValueAxis(0, 100, Count)
	is.Eq(a.Up(50), 0.5)
	is.Eq(a.Up(-10), 0.0)
	is.Eq(a.Up(500), 1.0)
	is.Eq(a.Up(math.NaN()), 0.0)
}

// The ends are round values just outside the data, and the ticks keep
// whole mantissas.
func TestLogAxisRoundsOutToTheLadder(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	a := NewLogAxis(Range{Min: 20, Max: 468}, Count)
	is.Eq(a.Min, 20.0)
	is.Eq(a.Max, 500.0)
	labels := make([]string, 0, len(a.Ticks))
	for _, t := range a.Ticks {
		labels = append(labels, t.Label)
	}
	is.Eq(labels, []string{"20", "50", "100", "200", "500"})
	is.Eq(a.Ticks[0].At, 0.0)
	is.Eq(a.Ticks[len(a.Ticks)-1].At, 1.0)
	is.Eq(a.Up(100), (math.Log10(100)-math.Log10(20))/(math.Log10(500)-math.Log10(20)))
}

// Past a thousandfold range the axis labels every other decade.
func TestLogAxisThinsWideRanges(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	a := NewLogAxis(Range{Min: 1, Max: 1_000_000}, Count)
	labels := make([]string, 0, len(a.Ticks))
	for _, t := range a.Ticks {
		labels = append(labels, t.Label)
	}
	is.Eq(labels, []string{"1", "100", "10,000", "1M"})
}

// A count has no tick below one.
func TestLogAxisSkipsFractionsOfACount(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	a := NewLogAxis(Range{Min: 0.2, Max: 3}, Count)
	labels := make([]string, 0, len(a.Ticks))
	for _, t := range a.Ticks {
		labels = append(labels, t.Label)
	}
	is.Eq(labels, []string{"1", "2", "5"})
}

// One value, or none, still gets an axis with two ends.
func TestLogAxisDegenerate(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	a := NewLogAxis(Range{Min: 7, Max: 7}, Count)
	is.Eq(a.Min, 5.0)
	is.Eq(a.Max, 10.0)

	a = NewLogAxis(Range{}, Count)
	is.Eq(a.Min, 1.0)
	is.Eq(a.Max, 10.0)
	is.Eq(a.Up(0), 0.0)
	is.Eq(a.Up(-5), 0.0)
	is.Eq(a.Up(math.NaN()), 0.0)
	is.Eq(a.Up(1_000), 1.0)
}

// A value a log scale cannot place does not widen a range.
func TestRangeAddSkipsValuesWithNoPlace(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	var r Range
	for _, v := range []float64{0, -3, math.NaN(), math.Inf(1), 40, 4, 400} {
		r.Add(v)
	}
	is.Eq(r, Range{Min: 4, Max: 400})
}
