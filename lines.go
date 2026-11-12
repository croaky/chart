package chart

import (
	"cmp"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strconv"
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
// value, and its color. Style sets --chart-top, the distance of the
// label's center down from the top of the plot.
//
// A label sits at the height of its line's last point. Two labels
// closer than one label line move apart, and the larger value stays on
// top. Without that, two lines that end near each other print two
// labels on top of each other.
type EndLabel struct {
	Label string      `json:"label"`
	Value string      `json:"value"`
	Class string      `json:"class"`
	Style hml.SafeCSS `json:"style"`
}

// NewLines draws the series and prints each number in f. It reads the
// options WithLog, WithDomain, WithBounds, WithBuckets, WithLocation,
// WithClock12, WithEndLabels, and WithEvents.
//
// A point with a value that no axis can place is a gap: a NaN, an
// infinity, or a nonzero magnitude above 1e300 or below 1e-300. A
// point outside WithBounds does not draw. A series with no other point
// does not draw.
//
// The work and the memory of a chart are linear in the points that
// draw. The end labels stop at a fixed count, so they do not change
// that bound. The package sets no limit on the series or the points.
// An app that charts user data limits how many points it passes.
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
	// plot would name each one a second time. Past maxEndLabels the
	// labels do not fit, and the legend names the lines.
	endLabels := o.endLabels && len(p.series) <= maxEndLabels
	l.HasLegend = len(l.Legend) > 0 && !endLabels
	if endLabels {
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
			if !placeable(pt.Value) {
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
		// One buffer for each polyline, and not a string for each point:
		// a chart of ten series of two years of weekly points has a
		// thousand points. A point in the plot is at most 13 bytes:
		// "1000.0,300.0 ".
		buf := make([]byte, 0, 13*(len(s.Points)+1))
		lastAt := 0
		for i, pt := range s.Points {
			if i > 0 {
				buf = append(buf, ' ')
			}
			lastAt = len(buf)
			buf = appendCoord(buf, p.xs[p.columnOf[pt.At.UnixNano()]], p.y(pt.Value))
		}
		points := string(buf)
		last := points[lastAt:]
		if len(s.Points) == 1 {
			points = last + " " + last
		}
		shapes = append(shapes, LineShape{
			Class:  s.Class,
			Points: points,
			End:    last + " " + last,
			Attrs:  markAttrs(s.Color),
		})
	}
	return shapes
}

// appendCoord appends an SVG point, x and y to a tenth, as "%.1f,%.1f"
// prints it.
func appendCoord(buf []byte, x, y float64) []byte {
	buf = strconv.AppendFloat(buf, x, 'f', 1, 64)
	buf = append(buf, ',')
	return strconv.AppendFloat(buf, y, 'f', 1, 64)
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

	// The tip lines come from one walk over the points, and not from each
	// series on each date. Series whose dates do not match have about as
	// many dates as points, so a grid of series and dates grows as the
	// square of the data. One backing slice holds the lines of all the
	// tips, so memory is linear in the points.
	//
	// last is the series of the newest line in each column. Two points
	// of one series on one date make one line, and the last value wins.
	counts := make([]int, len(p.dates))
	last := make([]int, len(p.dates))
	for i := range last {
		last[i] = -1
	}
	total := 0
	for j, s := range p.series {
		for _, pt := range s.Points {
			if c := p.columnOf[pt.At.UnixNano()]; last[c] != j {
				last[c] = j
				counts[c]++
				total++
			}
		}
	}

	gap := minGap(p.dates)
	lines := make([]TipLine, total)
	cols := make([]LineColumn, len(p.dates))
	off := 0
	for i, d := range p.dates {
		tip := NewTipAt(o.clock().point(d, gap), p.xs[i]/linesWidth)
		// The cap stops an append to one tip at the start of the next.
		tip.Lines = lines[off : off : off+counts[i]]
		off += counts[i]
		last[i] = -1
		cols[i] = LineColumn{
			Style:     placed[i].Style,
			LineStyle: placed[i].LineStyle,
			Tip:       tip,
		}
	}

	for j, s := range p.series {
		for _, pt := range s.Points {
			c := p.columnOf[pt.At.UnixNano()]
			tip := &cols[c].Tip
			if last[c] == j {
				tip.Lines = tip.Lines[:len(tip.Lines)-1]
			}
			last[c] = j
			*tip = tip.Line(s.Label, FormatValue(pt.Value, f))
		}
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

// maxEndLabels is the most end labels a chart draws. Past it, NewLines
// draws the legend.
//
// Each label needs one line of the plot. At the default --chart-height
// of 12rem and a line height of 1.2, about ten fit. Past that count the
// labels leave the plot and cover the page. The CSS of each label also
// holds one term for each label above it, so n labels hold about n²/2
// terms. At 12 that is 90 terms, about 2 KB. At 10,000 series from user
// data it would be more than a gigabyte.
const maxEndLabels = 12

// endLabels places a label at the end of each line and moves labels
// apart that sit closer than one label line.
//
// The gap is a line of the label's text, 1lh, and not a share of the
// plot. Go knows each label's place as a share of the plot but not the
// plot's height in lines, so the browser does the last step. Each label
// gets one CSS expression.
//
// The labels sort from the top of the plot down, and the larger value
// sorts first, so it stays on top when two lines end at one height.
// Then a walk down pushes each label at least one line below the one
// above it, and starts half a line inside the top. Label i lands at the
// largest of i+0.5 lines, and of each label k at or above it moved down
// i-k lines. A walk back up then holds each label above the bottom,
// with room for the labels under it: n-i-0.5 lines inside the bottom
// edge. The walk down leaves each label a line or more below the one
// above it, so in the walk up only the bottom edge can move a label.
// One min of one max is then the result of both walks.
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

	out := make([]EndLabel, 0, len(labels))
	for i, l := range labels {
		lowest := []string{fmt.Sprintf("calc(%g * 1lh)", float64(i)+0.5)}
		for k := range i + 1 {
			top := fmt.Sprintf("%.2f%%", labels[k].top)
			if k < i {
				top = fmt.Sprintf("calc(%s + %d * 1lh)", top, i-k)
			}
			lowest = append(lowest, top)
		}
		l.end.Style = hml.SafeCSS(fmt.Sprintf("--chart-top: min(max(%s), calc(100%% - %g * 1lh))",
			strings.Join(lowest, ", "), float64(len(labels)-i)-0.5))
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
