package chart

import (
	"math"
	"testing"

	"github.com/croaky/is"
)

func TestNewXYLinesEmptyAndUnplaceable(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.Eq(NewXYLines(XYLinesArgs{}).HasChart, false)

	bad := NewXYLines(XYLinesArgs{
		Series: []XYSeries{{
			Label: "Bad",
			Points: []XYLinePoint{
				{X: math.NaN(), Y: 50},
				{X: 2, Y: math.Inf(1)},
			},
		}},
	})
	is.Eq(bad.HasChart, false)
}

func TestNewXYLinesReversedAxisAndDomain(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	got := NewXYLines(XYLinesArgs{
		XFormat:  Price,
		YFormat:  Percent,
		XLabel:   "Cost",
		YLabel:   "Score",
		XMin:     0,
		XMax:     10,
		YMin:     40,
		YMax:     70,
		ReverseX: true,
		HasXMark: true,
		XMark:    2.5,
		Series: []XYSeries{
			{
				Label: "Alpha",
				Class: SeriesClass(0),
				Color: "#123456",
				Points: []XYLinePoint{
					{Label: "High", X: 10, Y: 70},
					{Label: "Low", X: 0, Y: 40, Highlight: true},
				},
			},
			{
				Label: "Beta",
				Class: SeriesClass(1),
				Color: "not-a-color",
				Points: []XYLinePoint{
					{Label: "High", X: 5, Y: 55},
				},
			},
		},
	})

	is.Eq(got.HasChart, true)
	is.Eq(got.ViewBox, "0 0 1000 300")
	// Multi-point series draws a polyline; single-point series draws a dot only.
	is.Eq(len(got.Shapes), 1)
	is.Eq(got.Shapes[0].Points, "0.0,0.0 1000.0,300.0")
	is.Eq(len(got.Dots), 3)
	is.Eq(string(got.Dots[0].Style), "--chart-x: 0.00%; --chart-y: 100.00%; --chart-mark: #123456")
	is.Eq(got.Dots[0].Highlight, false)
	is.Eq(got.Dots[0].Tip.Title, "Alpha High")
	is.Eq(string(got.Dots[1].Style), "--chart-x: 100.00%; --chart-y: 0.00%; --chart-mark: #123456")
	is.Eq(got.Dots[1].Highlight, true)
	// Invalid hex color is dropped.
	is.Eq(string(got.Dots[2].Style), "--chart-x: 50.00%; --chart-y: 50.00%")
	// XMark at 2.5 on reversed [0, 10] sits at 75%.
	is.Eq(got.HasXMark, true)
	is.Eq(string(got.XMarkStyle), "--chart-at: 75.00%")
	is.Eq(got.HasLegend, true)
	is.Eq(len(got.Legend), 2)
}

func TestNewXYLinesAutoScale(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	got := NewXYLines(XYLinesArgs{
		XFormat: Count,
		YFormat: Count,
		XLabel:  "Steps",
		YLabel:  "Tasks",
		Series: []XYSeries{{
			Label: "Run",
			Class: SeriesClass(0),
			Points: []XYLinePoint{
				{X: 0, Y: 0},
				{X: 10, Y: 20},
			},
		}},
	})
	is.Eq(got.HasChart, true)
	is.Eq(got.Shapes[0].Points, "0.0,300.0 1000.0,0.0")
	is.Eq(got.HasXMark, false)
}
