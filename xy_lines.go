package chart

import (
	"fmt"
	"math"

	"github.com/croaky/hml"
)

// XYLinePoint is one point on a [NewXYLines] series: its optional name
// (such as an effort tier), its two values, whether to highlight it
// with a ring, and extra lines for its tip.
type XYLinePoint struct {
	Label     string
	X, Y      float64
	Highlight bool
	Lines     []TipLine
}

// XYSeries is one series on a [NewXYLines] plot: its name, the class
// that colors it, an optional hex color, and its points in draw order.
//
// A series with two or more points draws a connecting polyline and a
// dot at each point. A series with one point draws its dot only.
type XYSeries struct {
	Label  string
	Class  string
	Color  string
	Points []XYLinePoint
}

// XYLinesArgs configures [NewXYLines].
//
// XMin/XMax and YMin/YMax fix an axis when Min < Max. Otherwise the
// axis scales from the data with [NewValueAxis], which holds zero.
//
// ReverseX maps Max to the left edge and Min to the right edge, so a
// lower cost and a higher score sit up and right.
//
// HasXMark draws a dashed vertical rule at XMark, such as a budget cap.
type XYLinesArgs struct {
	XFormat, YFormat Format
	XLabel, YLabel   string
	XMin, XMax       float64
	YMin, YMax       float64
	ReverseX         bool
	HasXMark         bool
	XMark            float64
	Series           []XYSeries
}

// XYLines is a multi-series XY line and scatter plot on linear axes.
//
// An SVG draws the connecting lines and stretches to the plot area.
// Colored HTML dots sit over the SVG and host each point's tip without
// JavaScript, so dots stay round and tips open on hover or focus.
type XYLines struct {
	HasChart   bool         `json:"has_chart"`
	ViewBox    string       `json:"view_box"`
	Shapes     []LineShape  `json:"shapes"`
	Dots       []XYDot      `json:"dots"`
	YTicks     []PlacedTick `json:"y_ticks"`
	XTicks     []PlacedTick `json:"x_ticks"`
	HasXMark   bool         `json:"has_x_mark"`
	XMarkStyle hml.SafeCSS  `json:"x_mark_style"`
	Legend     []LegendKey  `json:"legend"`
	HasLegend  bool         `json:"has_legend"`
}

// XYDot is one colored point on an [XYLines] plot and its tip. Style
// sets --chart-x, --chart-y, and an optional --chart-mark color.
type XYDot struct {
	Class     string      `json:"class"`
	Highlight bool        `json:"highlight"`
	Style     hml.SafeCSS `json:"style"`
	Tip       Tip         `json:"tip"`
}

// NewXYLines draws multi-series XY lines and points on linear axes.
//
// A point with a value that no axis can place is skipped. A series with
// no placeable points is omitted. When no series has a point,
// [XYLines.HasChart] stays false and the view renders nothing.
//
// Each dot prints at most [tipTextMax] bytes of its series label in its
// tip. The legend prints each label once, in full.
func NewXYLines(args XYLinesArgs) XYLines {
	cleaned := make([]XYSeries, 0, len(args.Series))
	xLo, xHi := math.Inf(1), math.Inf(-1)
	yLo, yHi := math.Inf(1), math.Inf(-1)
	totalPts := 0
	for _, s := range args.Series {
		pts := make([]XYLinePoint, 0, len(s.Points))
		for _, pt := range s.Points {
			if !placeable(pt.X) || !placeable(pt.Y) {
				continue
			}
			pts = append(pts, pt)
			xLo, xHi = min(xLo, pt.X), max(xHi, pt.X)
			yLo, yHi = min(yLo, pt.Y), max(yHi, pt.Y)
		}
		if len(pts) == 0 {
			continue
		}
		s.Points = pts
		if !hexColor.MatchString(s.Color) {
			s.Color = ""
		}
		cleaned = append(cleaned, s)
		totalPts += len(pts)
	}
	if len(cleaned) == 0 {
		return XYLines{}
	}

	xAxis := xyLinearAxis(xLo, xHi, args.XMin, args.XMax, args.XFormat)
	yAxis := xyLinearAxis(yLo, yHi, args.YMin, args.YMax, args.YFormat)

	across := func(v float64) float64 {
		at := xAxis.Up(v)
		if args.ReverseX {
			return 1 - at
		}
		return at
	}

	xTicks := make([]AxisTick, len(xAxis.Ticks))
	for i, t := range xAxis.Ticks {
		at := t.At
		if args.ReverseX {
			at = 1 - at
		}
		xTicks[i] = AxisTick{Label: t.Label, At: at}
	}

	shapes := make([]LineShape, 0, len(cleaned))
	dots := make([]XYDot, 0, totalPts)
	legend := make([]LegendKey, 0, len(cleaned))

	for _, s := range cleaned {
		legend = append(legend, LegendKey{Class: s.Class, Label: s.Label, Attrs: markAttrs(s.Color)})

		if len(s.Points) >= 2 {
			buf := make([]byte, 0, 13*(len(s.Points)+1))
			lastAt := 0
			for i, pt := range s.Points {
				if i > 0 {
					buf = append(buf, ' ')
				}
				lastAt = len(buf)
				x := linesWidth * across(pt.X)
				y := linesHeight * (1 - yAxis.Up(pt.Y))
				buf = appendCoord(buf, x, y)
			}
			points := string(buf)
			last := points[lastAt:]
			shapes = append(shapes, LineShape{
				Class:  s.Class,
				Points: points,
				End:    last + " " + last,
				Attrs:  markAttrs(s.Color),
			})
		}

		label := tipText(s.Label)
		for _, pt := range s.Points {
			ax := across(pt.X)
			ay := yAxis.Up(pt.Y)
			title := label
			if pt.Label != "" {
				if title != "" {
					title += " " + pt.Label
				} else {
					title = pt.Label
				}
			}
			tip := NewTipAt(title, ax)
			if args.YLabel != "" {
				tip = tip.Line(args.YLabel, FormatValue(pt.Y, args.YFormat))
			}
			if args.XLabel != "" {
				tip = tip.Line(args.XLabel, FormatValue(pt.X, args.XFormat))
			}
			tip.Lines = append(tip.Lines, pt.Lines...)

			style := DotStyle(ax, ay)
			if s.Color != "" {
				style += hml.SafeCSS("; --chart-mark: " + s.Color)
			}
			dots = append(dots, XYDot{
				Class:     s.Class,
				Highlight: pt.Highlight,
				Style:     style,
				Tip:       tip,
			})
		}
	}

	out := XYLines{
		HasChart:  true,
		ViewBox:   fmt.Sprintf("0 0 %.0f %.0f", linesWidth, linesHeight),
		Shapes:    shapes,
		Dots:      dots,
		YTicks:    PlacedTicks(yAxis.Ticks),
		XTicks:    PlacedTicks(xTicks),
		Legend:    legend,
		HasLegend: len(legend) > 0,
	}
	if args.HasXMark && placeable(args.XMark) {
		out.HasXMark = true
		out.XMarkStyle = atStyle(across(args.XMark))
	}
	return out
}

// xyLinearAxis returns a linear axis from an explicit [domLo, domHi]
// domain when domLo < domHi, or auto-scales from [dataLo, dataHi] with
// [NewValueAxis].
//
// An explicit domain lets a score plot span 40% to 70% without forcing
// zero into the axis, while an auto-scaled axis still holds zero.
func xyLinearAxis(dataLo, dataHi, domLo, domHi float64, f Format) ValueAxis {
	if placeable(domLo) && placeable(domHi) && domLo < domHi {
		return newRangeAxis(domLo, domHi, f)
	}
	return NewValueAxis(dataLo, dataHi, f)
}

// newRangeAxis builds a linear axis spanning [lo, hi] rounded out to
// whole 1/2/5 steps without forcing zero into the range.
func newRangeAxis(lo, hi float64, f Format) ValueAxis {
	step := niceStep((hi - lo) / tickTarget)
	if (f == Count || f == Signed || f == Millis || (f == Percent && hi-lo >= 1)) && step < 1 {
		step = 1
	}
	a := ValueAxis{
		Min: math.Floor(lo/step) * step,
		Max: math.Ceil(hi/step) * step,
	}
	gaps := math.Round((a.Max - a.Min) / step)
	if !(gaps >= 0 && gaps <= maxGaps) {
		gaps = 0
	}
	n := int(gaps)
	a.Ticks = make([]AxisTick, 0, n+1)
	for i := n; i >= 0; i-- {
		v := a.Min + float64(i)*step
		if math.Abs(v) < step/2 {
			v = 0
		}
		a.Ticks = append(a.Ticks, AxisTick{Label: FormatValue(v, f), At: a.Up(v)})
	}
	return a
}
