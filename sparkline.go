package chart

import "fmt"

// Sparkline is a small line chart with no hover and no style
// attribute. It is an SVG and a few labels.
//
// It is for a page that shows the shape of a series and names its
// ends, and for a page whose Content-Security-Policy refuses style
// attributes. [Lines] places its hover bands and tick labels with
// inline custom properties. A sparkline places nothing in HTML: its
// geometry is SVG attributes, and the view prints the labels in plain
// order around the SVG. So it renders under style-src 'self'.
type Sparkline struct {
	HasChart bool        `json:"has_chart"`
	ViewBox  string      `json:"view_box"`
	Shapes   []LineShape `json:"shapes"`
	// HasZero and ZeroY are the zero rule, as in [Lines].
	HasZero bool   `json:"has_zero"`
	ZeroY   string `json:"zero_y"`
	// Top and Bottom label the ends of the value axis.
	Top    string `json:"top"`
	Bottom string `json:"bottom"`
	// Start and End label the ends of the time axis.
	Start string `json:"start"`
	End   string `json:"end"`
}

// NewSparkline draws the series and prints the axis ends in f. It
// reads the options WithLog, WithDomain, WithBounds, WithBuckets, and
// WithLocation, with the same rules as [NewLines].
func NewSparkline(f Format, series []Series, opts ...Option) Sparkline {
	o := readOptions(opts)
	p, ok := newPlot(f, series, o)
	if !ok {
		return Sparkline{}
	}

	lo, hi := p.axis.ends()
	s := Sparkline{
		HasChart: true,
		ViewBox:  fmt.Sprintf("0 0 %.0f %.0f", linesWidth, linesHeight),
		Shapes:   p.shapes(),
		Top:      FormatValue(hi, f),
		Bottom:   FormatValue(lo, f),
	}
	s.HasZero, s.ZeroY = p.zero()
	// The ends take the labels of [Lines], so the two charts print one
	// time the same way.
	dates := TimeTicks(p.from, p.to, o.loc)
	s.Start, s.End = dates[0], dates[len(dates)-1]
	return s
}
