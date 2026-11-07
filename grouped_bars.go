package chart

import (
	"strconv"

	"github.com/croaky/hml"
)

// GroupedColumn is one category's numbers: its label and one count for
// each series, in the order the series were named.
//
// Axis, when set, is what the column prints under it, and Label is the
// title of its tip. The axis is as narrow as the column, and the tip
// has room for the whole name. A column with no Axis prints Label.
//
// The caller passes counts and not a map. So a series with nothing in
// this category is a zero the caller wrote, not a lookup that missed.
type GroupedColumn struct {
	Label  string
	Axis   string
	Counts []int
	// URLs, when set, holds a link for each series: the rows behind
	// that bar. A bar with a link draws as one.
	URLs []string
}

// GroupedBar is one bar: one series in one category. Style sets
// --chart-pct, as [Bar.Style] does.
type GroupedBar struct {
	Series string      `json:"series"`
	Class  string      `json:"class"`
	Count  int         `json:"count"`
	Value  string      `json:"value"`
	Style  hml.SafeCSS `json:"style"`
	URL    string      `json:"url"`
}

// GroupedCol is one category: its label, a bar for each series, and
// the tip that reads each series at that category.
//
// It has one tip for each column, not one for each bar. A reader reads
// the bars in a column together, which is the comparison the grouping
// is for.
type GroupedCol struct {
	Label string       `json:"label"`
	Axis  string       `json:"axis"`
	Bars  []GroupedBar `json:"bars"`
	Tip   Tip          `json:"tip"`
}

// GroupedBars compares named series across categories: a column for
// each category, and a bar for each series in it.
//
// Grouped and not stacked is a claim about the data, not a style. A
// stack says that the bars sum to the column. That is false where one
// thing can count in two series at once: the bars overlap, and their
// sum is not the category.
type GroupedBars struct {
	HasChart bool         `json:"has_chart"`
	Cols     []GroupedCol `json:"cols"`
	Ticks    []string     `json:"ticks"`
	Legend   []LegendKey  `json:"legend"`
	// LegendTitle, when set, is the chart's title, printed as the first
	// word of its legend. The legend names what the colors split by, so
	// a heading over the plot would say it a second time.
	LegendTitle string `json:"legend_title"`
}

// NewGroupedBars builds the chart from the series names and a column
// for each category. The series take [SeriesClass] colors.
//
// Each bar scales against one [AxisScale] top over every series. So a
// count is as tall in one column as in the next, and the chart reads
// off one axis.
//
// A column whose counts are all zero still draws, because the category
// is a place on the axis. A year with nothing in it is a gap, and
// leaving it out would put two years that are not neighbors side by
// side.
func NewGroupedBars(series []string, cols []GroupedColumn) GroupedBars {
	maxCount := 0
	for _, c := range cols {
		for _, n := range c.Counts {
			maxCount = max(maxCount, n)
		}
	}
	if len(cols) == 0 || maxCount <= 0 {
		return GroupedBars{}
	}

	maxY, ticks := AxisScale(maxCount)
	out := make([]GroupedCol, 0, len(cols))
	for i, c := range cols {
		bars := make([]GroupedBar, 0, len(series))
		tip := NewTip(c.Label, i, len(cols))
		for j, name := range series {
			count := at(c.Counts, j)
			bars = append(bars, GroupedBar{
				Series: name,
				Class:  SeriesClass(j),
				Count:  count,
				Value:  strconv.Itoa(count),
				Style:  pctStyle(float64(count) / float64(maxY)),
				URL:    at(c.URLs, j),
			})
			// Each series, zero included, so a reader learns the shape
			// of the panel once and does not look for a missing line.
			tip = tip.Count(name, count)
		}
		out = append(out, GroupedCol{Label: c.Label, Axis: axisLabel(c.Label, c.Axis), Bars: bars, Tip: tip})
	}
	return GroupedBars{HasChart: true, Cols: out, Ticks: ticks, Legend: SeriesLegend(series)}
}

// ValueGroupedColumn is one category of a [NewValueGroupedBars] chart:
// its label and one value for each series, in the order the series were
// named. A value at or below zero is no result.
type ValueGroupedColumn struct {
	Label  string
	Axis   string
	Values []float64
}

// NewValueGroupedBars is [NewGroupedBars] for values that are not small
// whole numbers. The caller names each series' class in its legend key,
// so one thing is one color here and in every other chart on the page.
func NewValueGroupedBars(f Format, series []LegendKey, cols []ValueGroupedColumn) GroupedBars {
	maxVal := 0.0
	for _, c := range cols {
		for _, v := range c.Values {
			maxVal = max(maxVal, finiteOr0(v))
		}
	}
	if len(cols) == 0 || maxVal <= 0 {
		return GroupedBars{}
	}

	axis := NewValueAxis(0, maxVal, f)
	out := make([]GroupedCol, 0, len(cols))
	for i, c := range cols {
		bars := make([]GroupedBar, 0, len(series))
		tip := NewTip(c.Label, i, len(cols))
		for j, s := range series {
			v := at(c.Values, j)
			value, shown := resultText(v, f)
			bars = append(bars, GroupedBar{
				Series: s.Label,
				Class:  s.Class,
				Value:  shown,
				Style:  pctStyle(axis.Up(max(finiteOr0(v), 0))),
			})
			tip = tip.Line(s.Label, value)
		}
		out = append(out, GroupedCol{Label: c.Label, Axis: axisLabel(c.Label, c.Axis), Bars: bars, Tip: tip})
	}
	return GroupedBars{HasChart: true, Cols: out, Ticks: axis.Labels(), Legend: series}
}

// axisLabel is what a column prints under it: its axis label, or its
// label when it has none.
func axisLabel(label, axis string) string {
	if axis != "" {
		return axis
	}
	return label
}

// at is s[i], or the zero value when s is too short. A caller that
// passes fewer counts than series means zero for the rest.
func at[T any](s []T, i int) T {
	if i < len(s) {
		return s[i]
	}
	var zero T
	return zero
}
