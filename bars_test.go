package chart

import (
	"math"
	"strconv"
	"testing"

	"github.com/croaky/is"
)

// Each bar scales against the axis top, not the largest count, and an
// empty category keeps its place.
func TestSeriesBars(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	b := NewSeriesBars("Companies", []SeriesBarRow{
		{Label: "A", Count: 3, Names: []string{"Acme"}},
		{Label: "B", Count: 0},
		{Label: "C", Count: 7},
	})

	is.True(b.HasChart)
	is.Eq(b.Ticks, []string{"8", "0"})
	is.Eq(b.Total, 10)
	is.Eq(len(b.Bars), 3)
	is.Eq(string(b.Bars[0].Style), "--chart-pct: 37.5%")
	is.Eq(b.Bars[0].Class, SeriesClass(0))
	is.Eq(b.Bars[0].Value, "3")
	is.Eq(string(b.Bars[1].Style), "--chart-pct: 0.0%")
	is.Eq(b.Bars[0].Tip.Lines, []TipLine{{Label: "Companies", HasLabel: true, Value: "3"}})
	is.Eq(b.Bars[0].Tip.Names, "Acme")
	is.Eq(b.Bars[2].Tip.Class, "chart-tip--right")
}

func TestSeriesBarsWithNothingToPlot(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.True(!NewSeriesBars("Companies", []SeriesBarRow{{Label: "A"}}).HasChart)
}

// A value bar scales against a value axis, and a value at or below zero
// is no result.
func TestValueBars(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	extra := TipLine{Label: "Chip", HasLabel: true, Value: "H100"}
	b := NewValueBars(Price, "Price", []ValueBarRow{
		{Label: "A", Class: "a", Value: 1.32, Lines: []TipLine{extra}},
		{Label: "B", Class: "b", Value: 0},
		{Label: "C", Class: "c", Value: 3.96},
	})

	is.Eq(b.Ticks, []string{"$4", "$3", "$2", "$1", "$0"})
	is.Eq(string(b.Bars[0].Style), "--chart-pct: 33.0%")
	is.Eq(b.Bars[0].Value, "$1.32")
	is.Eq(b.Bars[0].Tip.Lines, []TipLine{{Label: "Price", HasLabel: true, Value: "$1.32"}, extra})
	is.Eq(b.Bars[1].Value, "–")
	is.Eq(string(b.Bars[1].Style), "--chart-pct: 0.0%")
	is.Eq(b.Bars[1].Tip.Lines[0].Value, "No result")
}

// A value too large or too small for the axis is no result, as a NaN
// is. It does not set the axis.
func TestValueBarsDropExtremeValues(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	b := NewValueBars(Price, "Price", []ValueBarRow{
		{Label: "A", Class: "a", Value: 1.32},
		{Label: "B", Class: "b", Value: math.MaxFloat64},
		{Label: "C", Class: "c", Value: 3.96},
		{Label: "D", Class: "d", Value: 1e-320},
	})

	is.Eq(b.Ticks, []string{"$4", "$3", "$2", "$1", "$0"})
	for _, i := range []int{1, 3} {
		is.Eq(b.Bars[i].Value, "–")
		is.Eq(string(b.Bars[i].Style), "--chart-pct: 0.0%")
		is.Eq(b.Bars[i].Tip.Lines[0].Value, "No result")
	}
	is.True(!NewValueBars(USD, "Cost", []ValueBarRow{{Label: "A", Value: 1e-320}}).HasChart)
}

func TestValueBarsWithNothingToPlot(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.True(!NewValueBars(Price, "Price", []ValueBarRow{{Label: "A", Value: -1}}).HasChart)
}

// Each bar scales against one axis over every series, and a short list
// of counts means zero for the rest.
func TestGroupedBars(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	g := NewGroupedBars([]string{"Met", "Won"}, []GroupedColumn{
		{Label: "Q1 2026", Axis: "Q1", Counts: []int{3}, URLs: []string{"/q1/met"}},
		{Label: "Q2 2026", Counts: []int{4, 9}},
	})

	is.Eq(g.Ticks, []string{"10", "0"})
	is.Eq(g.Legend, SeriesLegend([]string{"Met", "Won"}))
	is.Eq(g.Cols[0].Axis, "Q1")
	is.Eq(g.Cols[1].Axis, "Q2 2026")
	is.Eq(g.Cols[0].Bars, []GroupedBar{
		{Series: "Met", Class: SeriesClass(0), Count: 3, Value: "3", Style: "--chart-pct: 30.0%", URL: "/q1/met"},
		{Series: "Won", Class: SeriesClass(1), Count: 0, Value: "0", Style: "--chart-pct: 0.0%"},
	})
	is.Eq(string(g.Cols[1].Bars[1].Style), "--chart-pct: 90.0%")
	is.Eq(g.Cols[0].Tip.Title, "Q1 2026")
	is.Eq(len(g.Cols[0].Tip.Lines), 2)
}

func TestGroupedBarsWithNothingToPlot(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.True(!NewGroupedBars([]string{"a"}, nil).HasChart)
	is.True(!NewGroupedBars([]string{"a"}, []GroupedColumn{{Label: "x", Counts: []int{0}}}).HasChart)
}

// The caller's legend keys color the series.
func TestValueGroupedBars(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	series := []LegendKey{{Class: "x", Label: "A"}, {Class: "y", Label: "B"}}
	g := NewValueGroupedBars(USD, series, []ValueGroupedColumn{{Label: "L", Values: []float64{10}}})

	is.Eq(g.Ticks, []string{"$10", "$5", "$0"})
	is.Eq(g.Legend, series)
	is.Eq(g.Cols[0].Bars, []GroupedBar{
		{Series: "A", Class: "x", Value: "$10", Style: "--chart-pct: 100.0%"},
		{Series: "B", Class: "y", Value: "–", Style: "--chart-pct: 0.0%"},
	})
	is.Eq(g.Cols[0].Tip.Lines[1].Value, "No result")
	is.True(!NewValueGroupedBars(USD, series, nil).HasChart)

	// A value too large for the axis is no result, as a NaN is.
	g = NewValueGroupedBars(USD, series, []ValueGroupedColumn{{Label: "L", Values: []float64{10, math.MaxFloat64}}})
	is.Eq(g.Ticks, []string{"$10", "$5", "$0"})
	is.Eq(g.Cols[0].Bars[1], GroupedBar{Series: "B", Class: "y", Value: "–", Style: "--chart-pct: 0.0%"})
}

// A series keeps its class in each column, even in a column that lacks
// the series before it.
func TestStackedColumns(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	s := NewStackedColumns([]string{"Met", "Reached"}, []string{"", "custom"}, []StackedColumn{
		{Label: "Q1", Counts: []int{2, 3}, NamesBySeries: [][]string{{"a", "b"}, {"c"}}, URLs: []string{"/met"}},
		{Label: "Q2", Title: "Q2 2026", Counts: []int{0, 1}, Selected: true},
	})

	is.Eq(s.Ticks, []string{"6", "0"})
	is.Eq(s.Cols[0].Segments, []StackedSegment{
		{Series: "Met", Class: SeriesClass(0), Count: 2, Style: "--chart-pct: 33.3%", URL: "/met"},
		{Series: "Reached", Class: "custom", Count: 3, Style: "--chart-pct: 50.0%"},
	})
	is.Eq(s.Cols[1].Segments, []StackedSegment{
		{Series: "Reached", Class: "custom", Count: 1, Style: "--chart-pct: 16.7%"},
	})
	is.Eq(s.Cols[1].Class, "chart-bars-col--selected")
	is.Eq(s.Cols[0].Class, "")
	is.Eq(s.Cols[1].Tip.Title, "Q2 2026")
	is.Eq(s.Cols[1].Tip.Lines, []TipLine{
		{Label: "Total", HasLabel: true, Value: "1"},
		{Label: "Met", HasLabel: true, Value: "0"},
		{Label: "Reached", HasLabel: true, Value: "1"},
	})
	is.Eq(s.Cols[0].Tip.Lines[1].Names, "a, b")
	is.Eq(s.Legend, []LegendKey{{Class: SeriesClass(0), Label: "Met"}, {Class: "custom", Label: "Reached"}})
	is.True(s.Labeled)
	is.True(!s.HasDates)
	is.Eq(s.Class, "")
}

// A long run prints the dates of its ends and its middle, and keeps its
// columns at every width.
func TestStackedColumnsLongRun(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	cols := make([]StackedColumn, 13)
	for i := range cols {
		cols[i] = StackedColumn{Label: "W" + delimited(i+1), Counts: []int{1}}
	}
	s := NewStackedColumns([]string{"Met"}, nil, cols)

	is.True(!s.Labeled)
	is.Eq(s.Dates, []string{"W1", "W7", "W13"})
	is.True(s.HasDates)
	is.Eq(s.Class, "chart-bars--long")
	// A long run keeps its columns already.
	is.Eq(s.KeepColumns().Class, "chart-bars--long")
}

// KeepColumns keeps a labeled run in columns at every width, for a run
// of periods under a time axis. It leaves the chart it is called on as
// it was.
func TestStackedColumnsKeepColumns(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	s := NewStackedColumns([]string{"Met"}, nil, []StackedColumn{{Label: "Q1", Counts: []int{1}}})
	kept := s.KeepColumns()

	is.Eq(kept.Class, "chart-bars--columns")
	is.True(kept.Labeled)
	is.Eq(s.Class, "")
}

func TestStackedColumnsWithNothingToPlot(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.True(!NewStackedColumns([]string{"a"}, nil, nil).HasChart)
	is.True(!NewStackedColumns([]string{"a"}, nil, []StackedColumn{{Label: "x"}}).HasChart)
	is.Eq(endsAndMiddle([]string{"a", "b"}), []string{"a", "b"})
}

func BenchmarkNewValueBars(b *testing.B) {
	rows := make([]ValueBarRow, 20)
	for i := range rows {
		rows[i] = ValueBarRow{Label: "Row " + strconv.Itoa(i), Class: SeriesClass(i), Value: 1.5 * float64(i*i)}
	}
	b.ReportAllocs()
	for b.Loop() {
		NewValueBars(USD, "Revenue", rows)
	}
}

func BenchmarkNewSeriesBars(b *testing.B) {
	names := make([]string, 30)
	for i := range names {
		names[i] = "Name " + strconv.Itoa(i)
	}
	rows := make([]SeriesBarRow, 10)
	for i := range rows {
		rows[i] = SeriesBarRow{Label: "Row " + strconv.Itoa(i), Count: 40 * i, Names: names}
	}
	b.ReportAllocs()
	for b.Loop() {
		NewSeriesBars("Companies", rows)
	}
}
