package chart

import (
	"cmp"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/croaky/hml"
)

// The coordinate space of a line chart's SVG. CSS stretches the
// element to the width of its box with preserveAspectRatio="none", so
// these are ratios, not pixels. They only have to be large enough that
// rounding to a tenth is invisible.
const (
	linesWidth  = 1000.0
	linesHeight = 300.0
)

// Point is one observation: a time and what the metric read then.
type Point struct {
	At    time.Time
	Value float64
}

// Series is one line: its name, the class that colors it, and its
// observations, oldest first.
//
// A series brings its own class, not a color from its index. A page
// that compares the same things across several plots needs one color
// for each thing in all of them. See [SeriesClass].
//
// Color, when set, is a hex color, #rgb or #rrggbb, for a series whose
// color is data, such as a team's own color. The views write it as
// --chart-mark on the line, its end dot, its end label, and its legend
// swatch, and the class still names the series. A value that is not a
// hex color is dropped, because it goes into a style attribute.
type Series struct {
	Label  string
	Class  string
	Color  string
	Points []Point
}

// hexColor matches the colors that [Series.Color] takes.
var hexColor = regexp.MustCompile(`^#(?:[0-9a-fA-F]{3}|[0-9a-fA-F]{6})$`)

// markAttrs are the attributes that color one mark of a series, or
// nil for a series with no color, so the view writes no attribute.
func markAttrs(color string) map[string]any {
	if color == "" {
		return nil
	}
	return map[string]any{"style": hml.SafeCSS("--chart-mark: " + color)}
}

// Lines is a line chart over time, with one line for each series.
//
// The reader points at a date and reads every series at once. That
// needs no script, because the dates are known before the page is
// sent. The plot carries one hover band for each date, and the band's
// tip names each series that has a value on that date.
//
// An x position is a date's place in the range, not its index in the
// slice. A gap in reporting draws as a gap.
type Lines struct {
	HasChart bool `json:"has_chart"`
	// ViewBox is the SVG coordinate space, so the view does not repeat
	// it and drift from it.
	ViewBox string      `json:"view_box"`
	Shapes  []LineShape `json:"shapes"`
	// YTicks are the value labels, each placed by its own position.
	// A linear axis spaces them evenly and a log axis does not, so the
	// view places both the same way.
	YTicks []PlacedTick `json:"y_ticks"`
	// HasZero is true when the axis runs below zero. ZeroY is then the
	// SVG y of the zero rule, which a view draws under the lines.
	HasZero bool   `json:"has_zero"`
	ZeroY   string `json:"zero_y"`
	// Dates label the start, the middle, and the end of the time axis.
	// A chart always has one or more.
	Dates   []string     `json:"dates"`
	Columns []LineColumn `json:"columns"`
	// EndLabels name each line at its end, when the caller asked for
	// them with [WithEndLabels].
	EndLabels    []EndLabel  `json:"end_labels"`
	HasEndLabels bool        `json:"has_end_labels"`
	Legend       []LegendKey `json:"legend"`
	HasLegend    bool        `json:"has_legend"`
	// Events are the marks of [WithEvents], on the time axis of the
	// lines.
	Events EventMarks `json:"events"`
	// From and To are the ends of the time axis in RFC 3339, UTC. A
	// page script that links charts or selects a range reads them from
	// data attributes.
	From string `json:"from"`
	To   string `json:"to"`
}

// LineShape is one series as an SVG polyline.
//
// A series with one observation is a polyline of that point twice. A
// round line cap draws the zero-length line as a dot. A line through
// one point draws nothing at all.
//
// End is the last point, written twice in the same way, so a view can
// draw a dot at the end of the line. A circle would stretch into an
// ellipse with the SVG.
//
// Attrs holds the style of a [Series.Color], and is nil for a series
// with none. A view adds it to the line and the dot with **.
type LineShape struct {
	Class  string         `json:"class"`
	Points string         `json:"points"`
	End    string         `json:"end"`
	Attrs  map[string]any `json:"attrs"`
}

// LineColumn is one date's hover band across the plot, placed by
// [TimeColumns] or [BucketColumns], and its tip.
type LineColumn struct {
	Style     hml.SafeCSS `json:"style"`
	LineStyle hml.SafeCSS `json:"line_style"`
	Tip       Tip         `json:"tip"`
}

// EndLabel names one line at its right end: the series, its last
// value, and its color. Style sets --chart-at, the label's distance up
// the plot as a percentage.
//
// A label sits at the height of its line's last point. Two labels
// closer than endLabelGap move apart, and the larger value stays on
// top. Without that, two lines that end near each other print two
// labels on top of each other.
type EndLabel struct {
	Label string      `json:"label"`
	Value string      `json:"value"`
	Class string      `json:"class"`
	Style hml.SafeCSS `json:"style"`
}

// endLabelGap is the least distance between two end label centers, as a
// percentage of the plot's height. A label is one line of small text,
// about a tenth of a plot's height.
const endLabelGap = 10.0

// NewLines draws the series and prints each number in f. It reads the
// options WithLog, WithDomain, WithBounds, WithBuckets, WithLocation,
// WithClock12, WithEndLabels, and WithEvents.
//
// A point with a NaN or infinite value is a gap. A point outside
// WithBounds does not draw. A series with no other point does not
// draw.
//
// No series with a point means no chart: [Lines.HasChart] stays false,
// and the view renders nothing. A linear chart whose every value is
// zero is no chart either, unless WithDomain set the axis. An axis from
// 0 to 0 is a picture of nothing, and a box that says a thing has none
// of a metric is a box a reader reads to learn it is empty.
func NewLines(f Format, series []Series, opts ...Option) Lines {
	o := readOptions(opts)
	p, ok := newPlot(f, series, o)
	if !ok {
		return Lines{}
	}

	l := Lines{
		HasChart: true,
		ViewBox:  fmt.Sprintf("0 0 %.0f %.0f", linesWidth, linesHeight),
		Shapes:   p.shapes(),
		YTicks:   PlacedTicks(p.axis.tickList()),
		Dates:    TimeTicks(p.from, p.to, opts...),
		Columns:  p.columns(f, o),
		Legend:   seriesKeys(p.series),
		Events:   eventMarks(o.events, p.from, p.to, o.clock()),
		From:     p.from.UTC().Format(time.RFC3339),
		To:       p.to.UTC().Format(time.RFC3339),
	}
	l.HasZero, l.ZeroY = p.zero()
	// End labels name each line where it ends, so a legend below the
	// plot would name each one a second time.
	l.HasLegend = len(l.Legend) > 0 && !o.endLabels
	if o.endLabels {
		l.EndLabels = p.endLabels(f)
		l.HasEndLabels = len(l.EndLabels) > 0
	}
	return l
}

// plot is the geometry that a line chart and a sparkline share: the
// series that draw, the union of their dates, the x of each date, and
// the value axis.
type plot struct {
	series   []Series
	dates    []time.Time
	xs       []float64
	columnOf map[int64]int
	axis     scale
	from, to time.Time
}

// newPlot scales the series. It reports false when there is nothing to
// draw.
//
// It drops each point outside the bounds, as eventMarks drops an
// event. The SVG does not clip, because a round cap at the edge needs
// to overflow. So a point outside would draw over the y axis, its band
// would take the pointer outside the plot, and its value would set the
// scale of a plot that does not show it.
func newPlot(f Format, series []Series, o options) (plot, bool) {
	var p plot
	lo, hi := math.Inf(1), math.Inf(-1)
	var positive Range
	for _, s := range series {
		points := make([]Point, 0, len(s.Points))
		for _, pt := range s.Points {
			if math.IsNaN(pt.Value) || math.IsInf(pt.Value, 0) {
				continue
			}
			if !inBounds(pt.At, o.from, o.to) {
				continue
			}
			points = append(points, pt)
			lo, hi = min(lo, pt.Value), max(hi, pt.Value)
			positive.Add(pt.Value)
		}
		if len(points) == 0 {
			continue
		}
		s.Points = points
		if !hexColor.MatchString(s.Color) {
			s.Color = ""
		}
		p.series = append(p.series, s)
	}
	if len(p.series) == 0 {
		return plot{}, false
	}

	switch {
	case o.log && o.hasDomain:
		p.axis = NewLogAxis(Range{Min: o.lo, Max: o.hi}, f)
	case o.log:
		if positive.Max <= 0 {
			return plot{}, false
		}
		p.axis = NewLogAxis(positive, f)
	case o.hasDomain:
		p.axis = NewValueAxis(o.lo, o.hi, f)
	default:
		if lo == 0 && hi == 0 {
			return plot{}, false
		}
		p.axis = NewValueAxis(lo, hi, f)
	}

	p.dates = seriesDates(p.series)
	// A bucketed point stands for the span that starts at its time, so
	// the data ends where the last bucket ends. An axis that ended on
	// the last point would give its band no width.
	p.from, p.to = p.dates[0], p.dates[len(p.dates)-1].Add(max(o.bucket, 0))
	if !o.from.IsZero() {
		p.from = o.from
	}
	if !o.to.IsZero() {
		p.to = o.to
	}
	// The points read off the same ends as the date labels, so one bound
	// moves both. Ends of zero width fall back to TimeXs.
	p.xs = TimeXsWithBounds(p.dates, linesWidth, p.from, p.to)
	p.columnOf = make(map[int64]int, len(p.dates))
	for i, d := range p.dates {
		p.columnOf[d.UnixNano()] = i
	}
	return p, true
}

// seriesDates is every date any series reports, ascending and with no
// repeats. It is the union and not one series' dates, because series
// report on their own schedules, and a date only one of them has is
// still a date the reader can point at.
func seriesDates(series []Series) []time.Time {
	var dates []time.Time
	for _, s := range series {
		for _, p := range s.Points {
			dates = append(dates, p.At)
		}
	}
	slices.SortFunc(dates, func(a, b time.Time) int { return a.Compare(b) })
	return slices.CompactFunc(dates, func(a, b time.Time) bool { return a.Equal(b) })
}

// y is the SVG y of v, measured down from the top.
func (p plot) y(v float64) float64 {
	return linesHeight * (1 - p.axis.Up(v))
}

// shapes turns each series into its polyline.
func (p plot) shapes() []LineShape {
	shapes := make([]LineShape, 0, len(p.series))
	for _, s := range p.series {
		coords := make([]string, 0, len(s.Points)+1)
		for _, pt := range s.Points {
			x := p.xs[p.columnOf[pt.At.UnixNano()]]
			coords = append(coords, fmt.Sprintf("%.1f,%.1f", x, p.y(pt.Value)))
		}
		last := coords[len(coords)-1]
		if len(coords) == 1 {
			coords = append(coords, last)
		}
		shapes = append(shapes, LineShape{
			Class:  s.Class,
			Points: strings.Join(coords, " "),
			End:    last + " " + last,
			Attrs:  markAttrs(s.Color),
		})
	}
	return shapes
}

// zero reports whether the plot needs a zero rule, and its SVG y. A
// linear axis that starts at zero has its zero on the bottom edge
// already, and a log axis has no zero.
func (p plot) zero() (bool, string) {
	a, ok := p.axis.(ValueAxis)
	if !ok || a.Min >= 0 {
		return false, ""
	}
	return true, fmt.Sprintf("%.1f", p.y(0))
}

// columns builds one hover band for each date, with a tip that names
// each series that has a value on that date.
func (p plot) columns(f Format, o options) []LineColumn {
	placed := TimeColumns(p.xs, linesWidth)
	if o.bucket > 0 && p.to.After(p.from) {
		width := linesWidth * float64(o.bucket) / float64(p.to.Sub(p.from))
		placed = BucketColumns(p.xs, linesWidth, width)
	}

	// Each series is indexed by date once here, and not scanned for each
	// column: two years of weekly points across ten series is a
	// thousand points, and each column would walk all of them.
	at := make([]map[int64]float64, len(p.series))
	for i, s := range p.series {
		at[i] = make(map[int64]float64, len(s.Points))
		for _, pt := range s.Points {
			at[i][pt.At.UnixNano()] = pt.Value
		}
	}

	gap := minGap(p.dates)
	cols := make([]LineColumn, 0, len(p.dates))
	for i, d := range p.dates {
		tip := NewTipAt(o.clock().point(d, gap), p.xs[i]/linesWidth)
		for j, s := range p.series {
			if v, ok := at[j][d.UnixNano()]; ok {
				tip = tip.Line(s.Label, FormatValue(v, f))
			}
		}
		cols = append(cols, LineColumn{
			Style:     placed[i].Style,
			LineStyle: placed[i].LineStyle,
			Tip:       tip,
		})
	}
	return cols
}

// minGap is the smallest time between two neighboring dates, or zero
// for fewer than two.
func minGap(dates []time.Time) time.Duration {
	var gap time.Duration
	for i := 1; i < len(dates); i++ {
		if d := dates[i].Sub(dates[i-1]); gap == 0 || d < gap {
			gap = d
		}
	}
	return gap
}

// endLabels places a label at the end of each line and moves labels
// apart that sit closer than endLabelGap.
//
// It sorts the labels from the top of the plot down, then walks down
// and pushes each label below the one above it. A label pushed past the
// bottom pushes the labels above it back up. The larger value sorts
// first, so it stays on top when two lines end at the same height.
func (p plot) endLabels(f Format) []EndLabel {
	type label struct {
		end   EndLabel
		color string
		value float64
		top   float64
	}
	labels := make([]label, 0, len(p.series))
	for _, s := range p.series {
		v := s.Points[len(s.Points)-1].Value
		labels = append(labels, label{
			end:   EndLabel{Label: s.Label, Value: FormatValue(v, f), Class: s.Class},
			color: s.Color,
			value: v,
			top:   100 * (1 - p.axis.Up(v)),
		})
	}
	slices.SortStableFunc(labels, func(a, b label) int {
		return cmp.Or(cmp.Compare(a.top, b.top), cmp.Compare(b.value, a.value))
	})

	edge := endLabelGap / 2
	for i := range labels {
		labels[i].top = max(labels[i].top, edge)
		if i > 0 {
			labels[i].top = max(labels[i].top, labels[i-1].top+endLabelGap)
		}
	}
	for i := range slices.Backward(labels) {
		limit := 100 - edge
		if i < len(labels)-1 {
			limit = labels[i+1].top - endLabelGap
		}
		labels[i].top = min(labels[i].top, limit)
	}

	out := make([]EndLabel, 0, len(labels))
	for _, l := range labels {
		l.end.Style = atStyle(1 - l.top/100)
		if l.color != "" {
			l.end.Style += hml.SafeCSS("; --chart-mark: " + l.color)
		}
		out = append(out, l.end)
	}
	return out
}

// seriesKeys names each series in its own color, in the order given.
//
// It prints names only. Ten names with a number each is a paragraph,
// and each number is in a tip already.
func seriesKeys(series []Series) []LegendKey {
	keys := make([]LegendKey, 0, len(series))
	for _, s := range series {
		keys = append(keys, LegendKey{Class: s.Class, Label: s.Label, Attrs: markAttrs(s.Color)})
	}
	return keys
}
