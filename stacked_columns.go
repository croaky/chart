package chart

import (
	"slices"

	"github.com/croaky/hml"
)

// StackedColumn is one period's numbers: its label, one count for each
// series in the order the series were named, and the names behind them
// for the tip.
//
// Label names the column: the dates under a long run, and the title of
// its tip. Title, when set, is the tip's title in place of Label, for a
// column whose short label needs more words. Axis, when set, is what a
// labeled column prints under it, as [GroupedColumn.Axis] is.
type StackedColumn struct {
	Label  string
	Title  string
	Axis   string
	Counts []int
	// Names lists the things behind the whole column, for the tip.
	Names []string
	// NamesBySeries lists the things behind each series, in series
	// order. The tip prints each list below its count.
	NamesBySeries [][]string
	// URLs, when set, holds a link for each series: the rows behind that
	// segment. A segment with a link draws as one.
	URLs []string
	// Selected marks the column whose rows the page lists, after a click
	// on one of its segments. The other columns fade.
	Selected bool
}

// StackedSegment is one series' share of one column. Style sets
// --chart-pct, as [Bar.Style] does.
//
// LinkAttrs holds the attributes of [StackedColumns.WithLinkAttr], and
// is nil for a segment with no URL.
type StackedSegment struct {
	Series    string         `json:"series"`
	Class     string         `json:"class"`
	Count     int            `json:"count"`
	Style     hml.SafeCSS    `json:"style"`
	URL       string         `json:"url"`
	LinkAttrs map[string]any `json:"link_attrs"`
}

// StackedCol is one column: its label, its total, a segment for each
// series with a count above zero, and the tip that reads each series.
//
// It has one tip for each column, not one for each segment. A reader
// reads a stack as a whole, and eight panels on one column would each
// say the same date.
type StackedCol struct {
	Label    string           `json:"label"`
	Axis     string           `json:"axis"`
	Total    int              `json:"total"`
	Segments []StackedSegment `json:"segments"`
	Tip      Tip              `json:"tip"`
	// Class is "chart-bars-col--selected" on the selected column and
	// empty on each other column.
	Class string `json:"class"`
}

// stackedLabelMax is the most columns that print a label each. Eight
// quarters and nine years fit under a plot. A year of weeks does not.
const stackedLabelMax = 12

// StackedColumns is a column for each period, divided by series: how
// many things there were, and what kind they were.
//
// Stacked and not grouped is a claim about the data. A stack says that
// the segments sum to the column, so it is right only where each thing
// counts in exactly one series, such as the stage a deal stood at on a
// day. [GroupedBars] is the other case.
//
// It is divs and not an SVG. The caller passes every period, the empty
// ones included, so the columns are evenly spaced and the axis is a
// list of categories. A chart whose marks sit at uneven places on a
// continuous axis is [Lines].
type StackedColumns struct {
	HasChart bool         `json:"has_chart"`
	Cols     []StackedCol `json:"cols"`
	Ticks    []string     `json:"ticks"`
	Legend   []LegendKey  `json:"legend"`
	// LegendTitle, when set, is the chart's title, printed as the first
	// word of its legend, as [GroupedBars.LegendTitle] is.
	LegendTitle string `json:"legend_title"`
	// Dates name the first period, the middle one, and the last. Only a
	// long run has them, because it prints no label under each column.
	Dates    []string `json:"dates"`
	HasDates bool     `json:"has_dates"`
	// Labeled is true for a run of stackedLabelMax columns or fewer. It
	// prints each column's label under it and lays out like a bar
	// chart: columns on a wide screen and rows on a narrow one. Three
	// dates under four quarters would name two of them, and put neither
	// under its column.
	Labeled bool `json:"labeled"`
	// Class is "chart-bars--long" on a long run, which keeps its
	// columns at every width. It is "chart-bars--columns" on a labeled
	// run after [StackedColumns.KeepColumns], and empty on another
	// labeled run.
	Class string `json:"class"`
	// Events are marks a caller sets from [NewEventMarks]. The columns
	// have no time axis, so the caller passes WithBounds that span the
	// first and the last period.
	Events EventMarks `json:"events"`
}

// NewStackedColumns builds the chart from the series names, the class
// that colors each series, and a column for each period.
//
// classes holds one class for each series. A nil list, or a blank
// entry, takes [SeriesClass] by index. A caller names a class where a
// series means the same thing in two charts, so a reader matches them.
//
// Either way the class comes from the series' index in the series
// list, not from its place in a column. So a series is one color in
// each column, and a column that lacks the first series does not paint
// its second series in the first color.
//
// Each segment scales against one [AxisScale] top over the column
// totals, so a count is as tall in one column as in the next.
//
// A column whose counts are all zero still draws, because the period is
// a place on the axis. Leaving it out would put two periods that are not
// neighbors side by side and read as a run.
func NewStackedColumns(series, classes []string, cols []StackedColumn) StackedColumns {
	classOf := func(i int) string {
		if c := at(classes, i); c != "" {
			return c
		}
		return SeriesClass(i)
	}
	totals := make([]int, len(cols))
	maxTotal := 0
	for i, c := range cols {
		for _, n := range c.Counts {
			totals[i] += n
		}
		maxTotal = max(maxTotal, totals[i])
	}
	if len(cols) == 0 || maxTotal <= 0 {
		return StackedColumns{}
	}

	maxY, ticks := AxisScale(maxTotal)
	out := make([]StackedCol, 0, len(cols))
	for i, c := range cols {
		segments := make([]StackedSegment, 0, len(series))
		title := c.Title
		if title == "" {
			title = c.Label
		}
		tip := NewTip(title, i, len(cols)).Count("Total", totals[i])
		for j, name := range series {
			count := at(c.Counts, j)
			// Each series, zero included, so a reader learns the shape
			// of the panel once and does not look for a missing line.
			tip = tip.CountWithNames(name, count, at(c.NamesBySeries, j))
			if count <= 0 {
				continue
			}
			segments = append(segments, StackedSegment{
				Series: name,
				Class:  classOf(j),
				Count:  count,
				Style:  pctStyle(float64(count) / float64(maxY)),
				URL:    at(c.URLs, j),
			})
		}
		out = append(out, StackedCol{
			Label:    c.Label,
			Axis:     axisLabel(c.Label, c.Axis),
			Total:    totals[i],
			Segments: segments,
			Tip:      tip.WithNames(c.Names),
			Class:    selectedClass(c.Selected),
		})
	}

	legend := make([]LegendKey, 0, len(series))
	for i, name := range series {
		legend = append(legend, LegendKey{Class: classOf(i), Label: name})
	}
	sc := StackedColumns{
		HasChart: true,
		Cols:     out,
		Ticks:    ticks,
		Legend:   legend,
		Labeled:  len(cols) <= stackedLabelMax,
	}
	if !sc.Labeled {
		labels := make([]string, 0, len(cols))
		for _, c := range cols {
			labels = append(labels, c.Label)
		}
		sc.Dates = endsAndMiddle(labels)
		sc.HasDates = true
		sc.Class = "chart-bars--long"
	}
	return sc
}

// WithLinkAttr returns the chart with one more attribute on the link of
// each segment that has a URL, as [GroupedBars.WithLinkAttr] does. The
// chart it is called on stays as it was.
func (s StackedColumns) WithLinkAttr(name string) StackedColumns {
	cols := slices.Clone(s.Cols)
	for i := range cols {
		segs := slices.Clone(cols[i].Segments)
		for j := range segs {
			segs[j].LinkAttrs = withLinkAttr(segs[j].LinkAttrs, name, segs[j].URL)
		}
		cols[i].Segments = segs
	}
	s.Cols = cols
	return s
}

// KeepColumns returns the chart with its columns at every width. A
// labeled run otherwise turns into rows on a narrow screen.
//
// Use it for a run of periods that sits under a time axis, such as a
// line chart over the same hours. Rows would turn time from across to
// down, and the reader could not match a column to the line above it.
// A long run keeps its columns already, so it stays as it was. The
// chart it is called on stays as it was too.
func (s StackedColumns) KeepColumns() StackedColumns {
	if s.Labeled {
		s.Class = "chart-bars--columns"
	}
	return s
}

// selectedClass is the class a column wears: the selected one, or none.
func selectedClass(selected bool) string {
	if selected {
		return "chart-bars-col--selected"
	}
	return ""
}

// endsAndMiddle keeps the first label, the middle one, and the last.
// The columns are one period each with none missing, so the middle
// column is the middle date.
func endsAndMiddle(labels []string) []string {
	switch len(labels) {
	case 0, 1, 2:
		return labels
	default:
		return []string{labels[0], labels[len(labels)/2], labels[len(labels)-1]}
	}
}
