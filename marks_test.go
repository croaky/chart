package chart

import (
	"math"
	"testing"
	"time"

	"github.com/croaky/is"
)

// Each point sits on two log axes, and a point with no place on one is
// left out and counted.
func TestXYScatter(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	extra := TipLine{Label: "Year", HasLabel: true, Value: "2024"}
	s := NewXYScatter(XYScatterArgs{
		XFormat: Count, YFormat: USD,
		XLabel: "Tokens", YLabel: "Cost",
		Points: []XYPoint{
			{Label: "A", X: 10, Y: 100, Image: "/a.png", Lines: []TipLine{extra}},
			{Label: "B", X: 1000, Y: 10},
			{Label: "C", X: 0, Y: 5},
			{Label: "D", X: math.NaN(), Y: 5},
		},
	})

	is.True(s.HasChart)
	is.Eq(s.Omitted, "2 points not plotted: no value on one axis.")
	is.True(s.HasOmitted)
	is.Eq(len(s.Dots), 2)
	is.Eq(string(s.Dots[0].Style), "--chart-x: 0.00%; --chart-y: 100.00%")
	is.Eq(string(s.Dots[1].Style), "--chart-x: 100.00%; --chart-y: 0.00%")
	is.Eq(s.Dots[0].Tip.Class, "chart-tip--left")
	is.Eq(s.Dots[0].Tip.Image, "/a.png")
	is.Eq(s.Dots[0].Tip.Lines, []TipLine{
		{Label: "Tokens", HasLabel: true, Value: "10"},
		{Label: "Cost", HasLabel: true, Value: "$100"},
		extra,
	})
	is.Eq(s.XTicks[0], PlacedTick{Label: "10", Style: "--chart-at: 0.00%"})
	is.Eq(s.YTicks[len(s.YTicks)-1], PlacedTick{Label: "$100", Style: "--chart-at: 100.00%"})
}

func TestXYScatterWithNothingToPlot(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	s := NewXYScatter(XYScatterArgs{Points: []XYPoint{{Label: "C", X: 0, Y: 5}}})
	is.True(!s.HasChart)
	is.Eq(s.Omitted, "1 point not plotted: no value on one axis.")
}

func TestDotStyle(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.Eq(string(DotStyle(0.25, 0.5)), "--chart-x: 25.00%; --chart-y: 50.00%")
}

// Events closer than the cluster gap share one mark with a count.
func TestEventMarks(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	events := []Event{
		{At: day0.Add(2 * time.Hour), Label: "deploy b"},
		{At: day0, Label: "deploy a"},
		{At: day0.Add(time.Minute), Label: "restart"},
	}
	m := NewEventMarks(events, WithBounds(day0, day0.Add(4*time.Hour)))

	is.True(m.HasMarks)
	is.Eq(len(m.Marks), 2)
	is.Eq(m.Marks[0].Count, 2)
	is.Eq(string(m.Marks[0].Style), "--chart-at: 0.21%")
	is.Eq(m.Marks[0].Tip.Title, "2 events")
	is.Eq(m.Marks[0].Tip.Lines, []TipLine{
		{Label: "00:00", HasLabel: true, Value: "deploy a"},
		{Label: "00:01", HasLabel: true, Value: "restart"},
	})
	is.Eq(string(m.Marks[1].Style), "--chart-at: 50.00%")
	is.Eq(m.Marks[1].Tip.Title, "Jan 5, 2026")
	is.Eq(m.Marks[1].Tip.Class, "chart-tip--center")
}

// A cluster lists eight events and counts the rest, at times in the
// caller's location.
func TestEventMarksCapAndLocation(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	events := make([]Event, 10)
	for i := range events {
		events[i] = Event{At: day0.Add(12 * time.Hour), Label: "e"}
	}
	m := NewEventMarks(events, WithLocation(time.FixedZone("EST", -5*60*60)))

	is.Eq(len(m.Marks), 1)
	lines := m.Marks[0].Tip.Lines
	is.Eq(len(lines), TipNameLimit+1)
	is.Eq(lines[0].Label, "07:00")
	is.Eq(lines[TipNameLimit], TipLine{Value: "+2 more", Class: "chart-tip-line--sub"})
}

// One bound sets one end, and the events set the other.
func TestEventMarksOneBound(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	m := NewEventMarks([]Event{{At: day0.Add(2 * time.Hour), Label: "deploy"}}, WithBounds(day0, time.Time{}))

	is.Eq(string(m.Marks[0].Style), "--chart-at: 100.00%")
}

// NewLines places its marks on the time axis of its lines, which its
// data sets when no bound does.
func TestLinesEvents(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	series := []Series{{Label: "a", Points: []Point{pt(0, 1), pt(3, 2)}}}
	l := NewLines(Count, series, WithEvents([]Event{{At: day0.AddDate(0, 0, 1), Label: "deploy"}}))

	is.True(l.Events.HasMarks)
	is.Eq(string(l.Events.Marks[0].Style), "--chart-at: 33.33%")
}

// An event outside the time axis draws no mark. On the edge, it would
// name a time it did not happen at.
func TestLinesEventsOutsideAxis(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	series := []Series{{Label: "a", Points: []Point{pt(0, 1), pt(3, 2)}}}
	l := NewLines(Count, series, WithEvents([]Event{
		{At: day0.AddDate(0, 0, -5), Label: "before"},
		{At: day0.AddDate(0, 0, 1), Label: "deploy"},
		{At: day0.AddDate(0, 0, 9), Label: "after"},
	}))

	is.Eq(len(l.Events.Marks), 1)
	is.Eq(l.Events.Marks[0].Events[0].Label, "deploy")

	l = NewLines(Count, series, WithEvents([]Event{{At: day0.AddDate(0, 0, 9), Label: "after"}}))
	is.True(!l.Events.HasMarks)
}

func TestEventMarksWithNoEvents(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.True(!NewEventMarks(nil).HasMarks)
}

// A cell's step is its share of the largest count, rounded up, and a
// cell with no count has no step.
func TestHeatmap(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	h := NewHeatmap(HeatmapArgs{
		XLabels:    []string{"a", "b", "c"},
		YLabels:    []string{"fast", "slow"},
		Counts:     [][]int{{0, 8}, {4}, {1, 2}},
		RowLabel:   "Latency",
		CountLabel: "Requests",
	}, WithBounds(day0, day0.Add(time.Hour)))

	is.True(h.HasChart)
	is.Eq(h.MaxCount, 8)
	is.Eq(h.Cols[0].Cells[0].Class, "")
	is.Eq(h.Cols[0].Cells[1].Class, "chart-ramp chart-ramp--8")
	is.Eq(h.Cols[1].Cells[0].Class, "chart-ramp chart-ramp--4")
	is.Eq(h.Cols[1].Cells[1].Class, "")
	is.Eq(h.Cols[2].Cells[0].Class, "chart-ramp chart-ramp--1")
	is.Eq(h.Cols[2].Cells[1].Class, "chart-ramp chart-ramp--2")
	is.Eq(h.Cols[0].Cells[1].Tip.Lines, []TipLine{
		{Label: "Latency", HasLabel: true, Value: "slow"},
		{Label: "Requests", HasLabel: true, Value: "8"},
	})
	is.Eq(h.Dates, []string{"a", "b", "c"})
	is.Eq(h.From, "2026-01-05T00:00:00Z")
	is.Eq(h.To, "2026-01-05T01:00:00Z")
}

func TestHeatmapWithNothingToPlot(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.True(!NewHeatmap(HeatmapArgs{}).HasChart)
	is.True(!NewHeatmap(HeatmapArgs{XLabels: []string{"a"}, YLabels: []string{"b"}, Counts: [][]int{{0}}}).HasChart)
}
