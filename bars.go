package chart

import (
	"fmt"
	"strconv"

	"github.com/croaky/hml"
)

// Bar is one bar: a label, and a share of the axis as long as its
// value's share of the top of the axis.
//
// Style sets --chart-pct and not a length, because the view draws
// along whichever axis it has room for: columns on a wide screen and
// rows on a narrow one. A percentage is the same number either way, and
// CSS cannot read an inline height as a width. So the share is a custom
// property that the two layouts spend as a height and as a width.
//
// Value is the number a row prints beside its bar, where the axis has
// no room: the count for a count, and a formatted value for
// [NewValueBars].
type Bar struct {
	Label string      `json:"label"`
	Class string      `json:"class"`
	Count int         `json:"count"`
	Value string      `json:"value"`
	Style hml.SafeCSS `json:"style"`
	Tip   Tip         `json:"tip"`
}

// Bars is a bar for each category.
type Bars struct {
	HasChart bool     `json:"has_chart"`
	Bars     []Bar    `json:"bars"`
	Ticks    []string `json:"ticks"`
	// Total is the sum of the counts. Unknown is a count the caller
	// knows of and does not draw, such as rows with no category. The
	// page prints it beside the chart, so the shares a reader takes off
	// the bars still add up.
	Total      int  `json:"total"`
	Unknown    int  `json:"unknown"`
	HasUnknown bool `json:"has_unknown"`
}

// SeriesBarRow is one category of a [NewSeriesBars] chart: its label,
// its count, and the names of the things counted.
type SeriesBarRow struct {
	Label string
	Count int
	Names []string
}

// NewSeriesBars draws a bar for each category, each in the
// [SeriesClass] color of its index. countLabel names the count in each
// tip, such as "Companies".
//
// It is the cut with one bar for each series of data that a grouped
// chart draws over time. A page that draws both passes the series in
// the same order, so one series is one color in both plots.
//
// A category with nothing in it keeps its place and draws an empty
// bar. The empty bar is an answer, and a missing one reads as a gap.
func NewSeriesBars(countLabel string, rows []SeriesBarRow) Bars {
	maxCount, total := 0, 0
	for _, r := range rows {
		maxCount = max(maxCount, r.Count)
		total += r.Count
	}
	if maxCount <= 0 {
		return Bars{}
	}

	maxY, ticks := AxisScale(maxCount)
	bars := make([]Bar, 0, len(rows))
	for i, r := range rows {
		bars = append(bars, Bar{
			Label: r.Label,
			Class: SeriesClass(i),
			Count: r.Count,
			Value: strconv.Itoa(r.Count),
			Style: pctStyle(float64(r.Count) / float64(maxY)),
			Tip: NewTip(r.Label, i, len(rows)).
				Count(countLabel, r.Count).
				WithNames(r.Names),
		})
	}
	return Bars{HasChart: true, Bars: bars, Ticks: ticks, Total: total}
}

// ValueBarRow is one category of a [NewValueBars] chart: its label, its
// class, its value, and the facts its tip adds after the value.
type ValueBarRow struct {
	Label string
	Class string
	Value float64
	Lines []TipLine
}

// NewValueBars draws a bar for each category for a value that is not a
// small whole number, such as a price, a throughput, or a percentage.
// valueLabel names the value in each tip.
//
// It scales against [NewValueAxis] and not [AxisScale]. The step table
// of AxisScale is for small counts, and a throughput of 1,500 would get
// a step of 500 and a top of 2,000 for no reason. The caller names each
// bar's class, so one thing is one color in every chart on a page.
//
// A value at or below zero, or a value that no axis can place, is no
// result. Its bar keeps its place and draws empty, and its tip says so,
// because a thing missing from a comparison is a fact the reader needs.
func NewValueBars(f Format, valueLabel string, rows []ValueBarRow) Bars {
	maxVal := 0.0
	for _, r := range rows {
		maxVal = max(maxVal, placeableOr0(r.Value))
	}
	if maxVal <= 0 {
		return Bars{}
	}

	axis := NewValueAxis(0, maxVal, f)
	bars := make([]Bar, 0, len(rows))
	for i, r := range rows {
		value, shown := resultText(r.Value, f)
		tip := NewTip(r.Label, i, len(rows)).Line(valueLabel, value)
		tip.Lines = append(tip.Lines, r.Lines...)
		bars = append(bars, Bar{
			Label: r.Label,
			Class: r.Class,
			Value: shown,
			Style: pctStyle(axis.Up(max(placeableOr0(r.Value), 0))),
			Tip:   tip,
		})
	}
	return Bars{HasChart: true, Bars: bars, Ticks: axis.Labels()}
}

// noResult is what the tip of a bar with no value says. The bar prints
// noResultMark beside it, because a row on a phone has no room for the
// words.
const (
	noResult     = "No result"
	noResultMark = "–"
)

// resultText is the text of a value for its tip and for its bar.
func resultText(v float64, f Format) (tip, bar string) {
	if !(placeableOr0(v) > 0) {
		return noResult, noResultMark
	}
	s := FormatValue(v, f)
	return s, s
}

// pctStyle sets a bar's share of the axis, a fraction from 0 to 1.
func pctStyle(share float64) hml.SafeCSS {
	return hml.SafeCSS(fmt.Sprintf("--chart-pct: %.1f%%", 100*min(max(share, 0), 1)))
}
