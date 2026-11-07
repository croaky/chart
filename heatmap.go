package chart

import (
	"math"
	"time"
)

// HeatmapArgs names a heatmap's grid: a label for each column, a label
// for each row, and a count for each cell, Counts[col][row].
//
// RowLabel and CountLabel name the two facts in each cell's tip, such
// as "Latency" and "Requests".
type HeatmapArgs struct {
	XLabels    []string
	YLabels    []string
	Counts     [][]int
	RowLabel   string
	CountLabel string
}

// HeatmapCell is one cell: its count, the class of its step on the
// ramp, and its tip. A cell with a count of zero has no class, and the
// view draws it empty.
type HeatmapCell struct {
	Count int    `json:"count"`
	Class string `json:"class"`
	Tip   Tip    `json:"tip"`
}

// HeatmapCol is one column of cells, from the first row, at the
// bottom, to the last row, at the top. A y axis reads up, so a caller
// passes the smallest band first.
type HeatmapCol struct {
	Label string        `json:"label"`
	Cells []HeatmapCell `json:"cells"`
}

// Heatmap is a grid of counts on a color ramp, such as requests by
// minute and by latency band. It shows where a distribution sits and
// how it moves, which a line of one percentile hides.
type Heatmap struct {
	HasChart bool         `json:"has_chart"`
	XLabels  []string     `json:"x_labels"`
	YLabels  []string     `json:"y_labels"`
	Cols     []HeatmapCol `json:"cols"`
	MaxCount int          `json:"max_count"`
	// Dates name the first column, the middle one, and the last, for
	// the axis under the grid.
	Dates []string `json:"dates"`
	// From and To are the ends of the time axis in RFC 3339, UTC, when
	// the caller set them with WithBounds.
	From string `json:"from"`
	To   string `json:"to"`
}

// rampSteps is how many steps the ramp has. Eight is about as many
// shades of one hue as a reader can tell apart.
const rampSteps = 8

// NewHeatmap builds the grid. It reads the option WithBounds.
//
// A cell's step is its count's share of the largest count, rounded up,
// so any count above zero is at least step 1 and draws. No labels on
// either axis, or no count above zero, means no chart.
func NewHeatmap(args HeatmapArgs, opts ...Option) Heatmap {
	if len(args.XLabels) == 0 || len(args.YLabels) == 0 {
		return Heatmap{}
	}
	maxCount := 0
	for _, col := range args.Counts {
		for _, c := range col {
			maxCount = max(maxCount, c)
		}
	}
	if maxCount <= 0 {
		return Heatmap{}
	}

	cols := make([]HeatmapCol, 0, len(args.XLabels))
	for i, x := range args.XLabels {
		pos := float64(i) / float64(max(len(args.XLabels)-1, 1))
		counts := at(args.Counts, i)
		cells := make([]HeatmapCell, 0, len(args.YLabels))
		for j, y := range args.YLabels {
			count := at(counts, j)
			cells = append(cells, HeatmapCell{
				Count: count,
				Class: rampClass(count, maxCount),
				Tip: NewTipAt(x, pos).
					Line(args.RowLabel, y).
					Count(args.CountLabel, count),
			})
		}
		cols = append(cols, HeatmapCol{Label: x, Cells: cells})
	}

	hm := Heatmap{
		HasChart: true,
		XLabels:  args.XLabels,
		YLabels:  args.YLabels,
		Cols:     cols,
		MaxCount: maxCount,
		Dates:    endsAndMiddle(args.XLabels),
	}
	o := readOptions(opts)
	if !o.from.IsZero() {
		hm.From = o.from.UTC().Format(time.RFC3339)
	}
	if !o.to.IsZero() {
		hm.To = o.to.UTC().Format(time.RFC3339)
	}
	return hm
}

// rampClass is the class of a count's step on the ramp, or empty for a
// count at or below zero. Each class is written out, so a search for
// the selector finds the code that sets it.
func rampClass(count, maxCount int) string {
	if count <= 0 || maxCount <= 0 {
		return ""
	}
	classes := [rampSteps]string{
		"chart-ramp chart-ramp--1",
		"chart-ramp chart-ramp--2",
		"chart-ramp chart-ramp--3",
		"chart-ramp chart-ramp--4",
		"chart-ramp chart-ramp--5",
		"chart-ramp chart-ramp--6",
		"chart-ramp chart-ramp--7",
		"chart-ramp chart-ramp--8",
	}
	step := int(math.Ceil(float64(count) / float64(maxCount) * rampSteps))
	return classes[min(max(step, 1), rampSteps)-1]
}
