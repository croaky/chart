package chart

import (
	"fmt"

	"github.com/croaky/hml"
)

// XYPoint is one mark on a [NewXYScatter] plot: its name, its two
// values, and the facts its tip adds after them.
type XYPoint struct {
	Label string
	X, Y  float64
	// Image, when set, is a picture for the tip, as [Tip.WithImage].
	Image string
	Lines []TipLine
}

// XYScatterArgs names the plot: what each axis measures and how to
// print it.
type XYScatterArgs struct {
	XFormat, YFormat Format
	XLabel, YLabel   string
	Points           []XYPoint
}

// Scatter is a plot of points on two axes, one dot for each point.
//
// It is divs and not an SVG. A scatter has no path to stroke, and a div
// placed by the builder is a tip host at no cost. An SVG stretched to
// its box with preserveAspectRatio="none" would draw each dot as an
// ellipse.
//
// The dots are one color. The information is position. Twenty colors
// for twenty points spend twenty hues on a fact that is in each tip.
type Scatter struct {
	HasChart bool         `json:"has_chart"`
	Dots     []ScatterDot `json:"dots"`
	// YTicks and XTicks each place their own labels, because a log axis
	// does not space its ticks evenly. The view also draws a grid rule
	// from each tick, so a rule cannot sit away from its label.
	YTicks []PlacedTick `json:"y_ticks"`
	XTicks []PlacedTick `json:"x_ticks"`
	// Omitted says how many points the plot left out and why. The
	// count of dots and the count in a table below then differ for a
	// stated reason.
	Omitted    string `json:"omitted"`
	HasOmitted bool   `json:"has_omitted"`
}

// ScatterDot is one dot and its tip. Style sets --chart-x and
// --chart-y, the dot's place across and up the plot as percentages.
type ScatterDot struct {
	Style hml.SafeCSS `json:"style"`
	Tip   Tip         `json:"tip"`
}

// NewXYScatter places each point on two log axes, for two measures
// that each span more than a decade.
//
// A log axis has no place for a value at or below zero. A point with
// one is left out, and Omitted says how many.
func NewXYScatter(args XYScatterArgs) Scatter {
	var s Scatter
	drawn := make([]XYPoint, 0, len(args.Points))
	var xr, yr Range
	for _, p := range args.Points {
		if !(finiteOr0(p.X) > 0 && finiteOr0(p.Y) > 0) {
			continue
		}
		drawn = append(drawn, p)
		xr.Add(p.X)
		yr.Add(p.Y)
	}
	if omitted := len(args.Points) - len(drawn); omitted > 0 {
		s.Omitted = pluralize(omitted, "point", "points") + " not plotted: no value on one axis."
		s.HasOmitted = true
	}
	if len(drawn) == 0 {
		return s
	}

	x := NewLogAxis(xr, args.XFormat)
	y := NewLogAxis(yr, args.YFormat)
	s.HasChart = true
	s.YTicks = PlacedTicks(y.Ticks)
	s.XTicks = PlacedTicks(x.Ticks)
	s.Dots = make([]ScatterDot, 0, len(drawn))
	for _, p := range drawn {
		across := x.Up(p.X)
		tip := NewTipAt(p.Label, across).
			WithImage(p.Image).
			Line(args.XLabel, FormatValue(p.X, args.XFormat)).
			Line(args.YLabel, FormatValue(p.Y, args.YFormat))
		tip.Lines = append(tip.Lines, p.Lines...)
		s.Dots = append(s.Dots, ScatterDot{Style: DotStyle(across, y.Up(p.Y)), Tip: tip})
	}
	return s
}

// DotStyle places a dot at x across and y up a plot, both fractions
// from 0 to 1. A caller that builds its own [Scatter], on axes of its
// own, places each dot with it.
func DotStyle(x, y float64) hml.SafeCSS {
	return hml.SafeCSS(fmt.Sprintf("--chart-x: %.2f%%; --chart-y: %.2f%%", 100*x, 100*y))
}
