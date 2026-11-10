package chart

import (
	"math"
	"testing"
	"time"

	"github.com/croaky/hml"
	"github.com/croaky/is"
)

func pt(day int, v float64) Point {
	return Point{At: day0.AddDate(0, 0, day), Value: v}
}

func twoSeries() []Series {
	return []Series{
		{Label: "Acme", Class: SeriesClass(0), Points: []Point{pt(0, 10), pt(1, 20), pt(3, 40)}},
		{Label: "Beta", Class: SeriesClass(1), Points: []Point{pt(1, 5)}},
	}
}

// Each date sits at its place in the range, and a series with one
// point draws as a dot.
func TestLinesShapes(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	l := NewLines(Count, twoSeries())

	is.True(l.HasChart)
	is.Eq(l.ViewBox, "0 0 1000 300")
	is.Eq(l.Shapes, []LineShape{
		{Class: SeriesClass(0), Points: "0.0,225.0 333.3,150.0 1000.0,0.0", End: "1000.0,0.0 1000.0,0.0"},
		{Class: SeriesClass(1), Points: "333.3,262.5 333.3,262.5", End: "333.3,262.5 333.3,262.5"},
	})
	is.Eq(l.YTicks[0], PlacedTick{Label: "40", Style: "--chart-at: 100.00%"})
	is.Eq(len(l.YTicks), 5)
	is.True(!l.HasZero)
}

// Each date has a band, and its tip names each series with a value on
// that date.
func TestLinesColumns(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	l := NewLines(Count, twoSeries())

	is.Eq(len(l.Columns), 3)
	is.Eq(l.Columns[1].Tip.Title, "Jan 6, 2026")
	is.Eq(l.Columns[1].Tip.Lines, []TipLine{
		{Label: "Acme", HasLabel: true, Value: "20"},
		{Label: "Beta", HasLabel: true, Value: "5"},
	})
	is.Eq(len(l.Columns[2].Tip.Lines), 1)
	is.Eq(l.Columns[2].Tip.Class, "chart-tip--right")
}

func TestLinesLabelsAndLegend(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	l := NewLines(Count, twoSeries())

	is.Eq(l.Dates, []string{"Jan 5", "Jan 6", "Jan 8"})
	is.Eq(l.Legend, []LegendKey{{Class: SeriesClass(0), Label: "Acme"}, {Class: SeriesClass(1), Label: "Beta"}})
	is.True(l.HasLegend)
	is.Eq(l.From, "2026-01-05T00:00:00Z")
	is.Eq(l.To, "2026-01-08T00:00:00Z")
	is.True(!l.HasEndLabels)
}

// No point, or nothing but zeros, is no chart. A domain lets zeros
// draw.
func TestLinesWithNothingToPlot(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	zeros := []Series{{Label: "a", Points: []Point{pt(0, 0), pt(1, 0)}}}
	is.True(!NewLines(Count, nil).HasChart)
	is.True(!NewLines(Count, []Series{{Label: "a"}}).HasChart)
	is.True(!NewLines(Count, zeros).HasChart)
	is.True(NewLines(Count, zeros, WithDomain(0, 10)).HasChart)
	is.True(!NewLines(Count, zeros, WithLog()).HasChart)
}

// A point that is not a number is a gap, and a series of nothing else
// does not draw.
func TestLinesSkipsValuesThatAreNotNumbers(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	l := NewLines(Count, []Series{
		{Label: "a", Points: []Point{pt(0, 10), pt(1, math.NaN()), pt(2, 20)}},
		{Label: "b", Points: []Point{pt(1, math.Inf(1))}},
	})

	is.Eq(len(l.Shapes), 1)
	is.Eq(l.Shapes[0].Points, "0.0,150.0 1000.0,0.0")
	is.Eq(len(l.Columns), 2)
}

// A value too large or too small for the axis is a gap, as a NaN is.
// It does not set the axis, on a linear axis or on a log axis.
func TestLinesSkipsExtremeValues(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	l := NewLines(Count, []Series{
		{Label: "a", Points: []Point{pt(0, 10), pt(1, math.MaxFloat64), pt(2, 20)}},
		{Label: "b", Points: []Point{pt(1, -math.MaxFloat64)}},
	})
	is.Eq(len(l.Shapes), 1)
	is.Eq(l.Shapes[0].Points, "0.0,150.0 1000.0,0.0")
	is.Eq(len(l.Columns), 2)

	l = NewLines(Count, []Series{{Label: "a", Points: []Point{pt(0, 20), pt(1, 1e-320), pt(2, 500)}}}, WithLog())
	is.Eq(l.Shapes[0].Points, "0.0,300.0 1000.0,0.0")
	is.Eq(len(l.YTicks), 5)

	// A domain end with no place on the axis is no end.
	series := []Series{{Label: "a", Points: []Point{pt(0, 20), pt(1, 500)}}}
	l = NewLines(USD, series, WithLog(), WithDomain(math.SmallestNonzeroFloat64, math.MaxFloat64))
	is.True(l.HasChart)
	is.True(len(l.YTicks) <= 9)
	l = NewLines(USD, series, WithDomain(-math.MaxFloat64, math.MaxFloat64))
	is.True(l.HasChart)
	is.True(len(l.YTicks) <= 9)
}

// A signed domain draws a zero rule where the axis crosses zero.
func TestLinesSignedDomain(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	l := NewLines(Count, []Series{{Label: "a", Points: []Point{pt(0, -1), pt(1, 2)}}}, WithDomain(-6, 6))

	is.True(l.HasZero)
	is.Eq(l.ZeroY, "150.0")
	is.Eq(l.Shapes[0].Points, "0.0,175.0 1000.0,100.0")
	is.Eq(l.YTicks[0], PlacedTick{Label: "6", Style: "--chart-at: 100.00%"})
	is.Eq(l.YTicks[len(l.YTicks)-1], PlacedTick{Label: "-6", Style: "--chart-at: 0.00%"})
}

// A log axis places each value by its order of magnitude.
func TestLinesLog(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	l := NewLines(Count, []Series{{Label: "a", Points: []Point{pt(0, 20), pt(1, 500)}}}, WithLog())

	is.Eq(l.Shapes[0].Points, "0.0,300.0 1000.0,0.0")
	is.Eq(len(l.YTicks), 5)
	is.True(!l.HasZero)
}

// Bounds set the ends of the time axis, and buckets set each band to
// the span its point stands for.
func TestLinesBoundsAndBuckets(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	from, to := day0, day0.Add(4*time.Hour)
	series := []Series{{Label: "a", Points: []Point{
		{At: day0.Add(time.Hour), Value: 1},
		{At: day0.Add(2 * time.Hour), Value: 2},
	}}}
	l := NewLines(Count, series, WithBounds(from, to), WithBuckets(time.Hour))

	is.Eq(l.Shapes[0].Points, "250.0,150.0 500.0,0.0")
	is.Eq(string(l.Columns[0].Style), "--chart-left: 25.00%; --chart-width: 25.00%")
	is.Eq(l.Dates, []string{"00:00", "02:00", "04:00"})
	is.Eq(l.To, "2026-01-05T04:00:00Z")
}

// With no end bound, the axis ends where the last bucket ends. The last
// band then spans its bucket, not zero width at the right edge.
func TestLinesBucketsWithNoEndBound(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	series := []Series{{Label: "a", Points: []Point{
		{At: day0, Value: 1},
		{At: day0.Add(time.Hour), Value: 2},
		{At: day0.Add(2 * time.Hour), Value: 3},
	}}}
	l := NewLines(Count, series, WithBuckets(time.Hour))

	is.Eq(string(l.Columns[2].Style), "--chart-left: 66.67%; --chart-width: 33.33%")
	is.Eq(l.Dates, []string{"00:00", "01:30", "03:00"})
	is.Eq(l.To, "2026-01-05T03:00:00Z")
}

// One bound sets one end, and the data sets the other. The points and
// the date labels read off the same ends.
func TestLinesOneBound(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	series := []Series{{Label: "a", Points: []Point{
		{At: day0.Add(time.Hour), Value: 1},
		{At: day0.Add(2 * time.Hour), Value: 2},
	}}}
	l := NewLines(Count, series, WithBounds(day0, time.Time{}))

	is.Eq(l.Shapes[0].Points, "500.0,150.0 1000.0,0.0")
	is.Eq(l.Dates, []string{"00:00", "01:00", "02:00"})
	is.Eq(l.From, "2026-01-05T00:00:00Z")
}

// A point outside the bounds draws nothing, has no band, and does not
// set the value axis.
func TestLinesDropsPointsOutsideTheBounds(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	series := []Series{{Label: "a", Points: []Point{pt(0, 100), pt(1, 10), pt(3, 20), pt(5, 100)}}}
	l := NewLines(Count, series, WithBounds(day0.AddDate(0, 0, 1), day0.AddDate(0, 0, 3)))

	is.Eq(l.Shapes[0].Points, "0.0,150.0 1000.0,0.0")
	is.Eq(len(l.Columns), 2)
	is.Eq(l.YTicks[0].Label, "20")

	l = NewLines(Count, series, WithBounds(day0.AddDate(0, 0, 1), time.Time{}))
	is.Eq(len(l.Columns), 3)
	is.Eq(l.To, "2026-01-10T00:00:00Z")

	is.True(!NewLines(Count, series, WithBounds(day0.AddDate(0, 0, 6), time.Time{})).HasChart)
}

// Points less than a day apart name their time in the tip, in the
// caller's location.
func TestLinesLocation(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	est := time.FixedZone("EST", -5*60*60)
	series := []Series{{Label: "a", Points: []Point{
		{At: day0.Add(12 * time.Hour), Value: 1},
		{At: day0.Add(13 * time.Hour), Value: 2},
	}}}
	l := NewLines(Count, series, WithLocation(est))

	is.Eq(l.Columns[0].Tip.Title, "Jan 5 07:00")
	is.Eq(l.Dates[0], "07:00")
}

// A series color reaches each mark of the series, and a value that is
// not a hex color goes nowhere near a style attribute.
func TestLinesSeriesColor(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	series := []Series{
		{Label: "a", Class: "a", Color: "#0af", Points: []Point{pt(0, 0), pt(1, 50)}},
		{Label: "b", Class: "b", Color: "red; background: url(x)", Points: []Point{pt(0, 0), pt(1, 10)}},
	}
	l := NewLines(Percent, series, WithDomain(0, 100), WithEndLabels())

	is.Eq(l.Shapes[0].Attrs, map[string]any{"style": hml.SafeCSS("--chart-mark: #0af")})
	is.Nil(l.Shapes[1].Attrs)
	is.Eq(string(l.EndLabels[0].Style), "--chart-top: min(max(calc(0.5 * 1lh), 50.00%), calc(100% - 1.5 * 1lh)); --chart-mark: #0af")
	is.Eq(string(l.EndLabels[1].Style), "--chart-top: min(max(calc(1.5 * 1lh), calc(50.00% + 1 * 1lh), 90.00%), calc(100% - 0.5 * 1lh))")
	is.Eq(l.Legend[0].Attrs, map[string]any{"style": hml.SafeCSS("--chart-mark: #0af")})
	is.Nil(l.Legend[1].Attrs)
	// The end labels name each line, so the view draws no legend.
	is.True(!l.HasLegend)
}

// WithClock12 prints the axis and the tip times on a 12-hour clock.
func TestLinesClock12(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	est := time.FixedZone("EST", -5*60*60)
	series := []Series{{Label: "a", Points: []Point{
		{At: day0.Add(17 * time.Hour), Value: 1},
		{At: day0.Add(18 * time.Hour), Value: 2},
	}}}
	l := NewLines(Count, series, WithLocation(est), WithClock12())

	is.Eq(l.Dates, []string{"12:00 PM", "12:30 PM", "1:00 PM"})
	is.Eq(l.Columns[0].Tip.Title, "Jan 5 12:00 PM")
	is.Eq(TimeTicks(day0, day0.Add(36*time.Hour), WithClock12()), []string{"Jan 5 12:00 AM", "Jan 5 6:00 PM", "Jan 6 12:00 PM"})
}

func endAt(label string, v float64) Series {
	return Series{Label: label, Class: label, Points: []Point{pt(0, 0), pt(1, v)}}
}

// Labels sort from the top of the plot down, and the larger value stays
// on top when two lines end at one height. Each label sits at least one
// label line below the one above it: the label in row i is the largest
// of each label k above it, moved down i-k lines.
func TestLinesEndLabelsMoveApart(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	l := NewLines(Percent, []Series{endAt("a", 50), endAt("b", 52), endAt("c", 52)}, WithDomain(0, 100), WithEndLabels())

	is.True(l.HasEndLabels)
	is.Eq(l.EndLabels, []EndLabel{
		{Label: "b", Value: "52%", Class: "b", Style: "--chart-top: min(max(calc(0.5 * 1lh), 48.00%), calc(100% - 2.5 * 1lh))"},
		{Label: "c", Value: "52%", Class: "c", Style: "--chart-top: min(max(calc(1.5 * 1lh), calc(48.00% + 1 * 1lh), 48.00%), calc(100% - 1.5 * 1lh))"},
		{Label: "a", Value: "50%", Class: "a", Style: "--chart-top: min(max(calc(2.5 * 1lh), calc(48.00% + 2 * 1lh), calc(48.00% + 1 * 1lh), 50.00%), calc(100% - 0.5 * 1lh))"},
	})
}

// The gap is one line of the label's own text, not a share of the plot,
// so a short plot does not print two labels over each other. Each
// label stays half a line inside the top and the bottom of the plot.
func TestLinesEndLabelsStayInThePlot(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	l := NewLines(Percent, []Series{endAt("top", 100)}, WithDomain(0, 100), WithEndLabels())
	is.Eq(string(l.EndLabels[0].Style), "--chart-top: min(max(calc(0.5 * 1lh), 0.00%), calc(100% - 0.5 * 1lh))")
}

// A sparkline names the ends of both axes and needs no hover.
func TestSparkline(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	s := NewSparkline(Count, []Series{{Label: "a", Points: []Point{pt(0, -2), pt(3, 3)}}}, WithDomain(-6, 6))

	is.True(s.HasChart)
	is.Eq(s.Top, "6")
	is.Eq(s.Bottom, "-6")
	is.True(s.HasZero)
	is.Eq(s.ZeroY, "150.0")
	is.Eq(s.Start, "Jan 5")
	is.Eq(s.End, "Jan 8")
	is.Eq(s.Shapes[0].Points, "0.0,200.0 1000.0,75.0")
}

func TestSparklineDegenerate(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.True(!NewSparkline(Count, nil).HasChart)

	s := NewSparkline(Count, []Series{{Label: "a", Points: []Point{pt(0, 5)}}})
	is.Eq(s.Start, "Jan 5, 2026")
	is.Eq(s.End, "Jan 5, 2026")
	is.Eq(s.Top, "6")
	is.Eq(s.Bottom, "0")
	is.True(!s.HasZero)

	// A value too large for the axis is a gap, so one point is left.
	s = NewSparkline(Count, []Series{{Label: "a", Points: []Point{pt(0, 5), pt(1, -math.MaxFloat64)}}})
	is.Eq(s.Start, "Jan 5, 2026")
	is.Eq(s.Top, "6")
	is.Eq(s.Bottom, "0")
}
