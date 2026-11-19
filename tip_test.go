package chart

import (
	"strings"
	"testing"

	"github.com/croaky/is"
)

// A panel opens toward the side of the plot with room for it, split by
// thirds.
func TestTipPlacement(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(NewTip("a", 0, 14).Class, "chart-tip--left")
	is.Eq(NewTip("a", 1, 14).Class, "chart-tip--left")
	is.Eq(NewTip("a", 7, 14).Class, "chart-tip--center")
	is.Eq(NewTip("a", 13, 14).Class, "chart-tip--right")
	is.Eq(NewTip("a", 0, 1).Class, "chart-tip--center")
	is.Eq(NewTipAt("a", 0.9).Class, "chart-tip--right")
	is.Eq(NewStackedTip("a").Class, "chart-tip--left")
}

func TestTipTitle(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	tip := NewTip("  Acme  ", 0, 3)
	is.Eq(tip.Title, "Acme")
	is.True(tip.HasTitle)
	is.True(!NewStackedTip(" ").HasTitle)

	is.Eq(tip.WithImage(" /a.png ").Image, "/a.png")
	is.True(tip.WithImage("/a.png").HasImage)
	is.True(!tip.WithImage("  ").HasImage)
}

func TestTipLines(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	tip := NewTip("Q1", 0, 1).
		Count("Total", 1234).
		SubLine("Met", "12").
		Line("", "alone").
		CountWithNames("Won", 2, []string{"Acme", " ", "Beta"})

	is.Eq(tip.Lines, []TipLine{
		{Label: "Total", HasLabel: true, Value: "1,234"},
		{Label: "Met", HasLabel: true, Value: "12", Class: "chart-tip-line--sub"},
		{Value: "alone"},
		{Label: "Won", HasLabel: true, Value: "2", Names: "Acme, Beta", HasNames: true},
	})
}

// A tip prints at most tipTextMax bytes of a name, cut on a rune
// boundary, and "…" says that it cut.
func TestTipText(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	a := strings.Repeat("a", tipTextMax)
	is.Eq(tipText("  Acme  "), "Acme")
	is.Eq(tipText(a), a)
	is.Eq(tipText(a+"b"), a+"…")
	// The cut falls inside the two bytes of é, so the é goes.
	is.Eq(tipText(a[1:]+"é"), a[1:]+"…")
	// A name that is mostly white space reads only the first bytes.
	is.Eq(tipText(strings.Repeat(" ", 1<<20)+"a"), "")
	is.Eq(tipText("a"+strings.Repeat(" ", 1<<20)), "a…")

	long := a + strings.Repeat("b", 1<<20)
	tip := NewTip(long, 0, 1).Line(long, long).CountWithNames(long, 1, []string{long}).WithNames([]string{long})
	is.Eq(tip.Title, a+"…")
	is.Eq(tip.Lines[0].Label, a+"…")
	is.Eq(tip.Lines[0].Value, a+"…")
	is.Eq(tip.Lines[1].Label, a+"…")
	is.Eq(tip.Lines[1].Names, a+"…")
	is.Eq(tip.Names, a+"…")
}

// A tip lists eight names and counts the rest. A blank name is no name.
func TestTipNamesAreCapped(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	names := []string{"a", "b", "", "c", "d", "e", "f", "g", "h", "i", "j"}
	tip := NewTip("x", 0, 1).WithNames(names)
	is.Eq(tip.Names, "a, b, c, d, e, f, g, h, +2 more")
	is.True(tip.HasNames)

	is.True(!NewTip("x", 0, 1).WithNames([]string{" ", ""}).HasNames)
	is.Eq(capNames([]string{"a", "b"}), "a, b")
}

func TestSeriesClass(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(SeriesClass(0), "chart-series chart-series--1")
	is.Eq(SeriesClass(9), "chart-series chart-series--10")
	is.Eq(SeriesClass(SeriesLimit), "chart-series chart-series--1")
	is.Eq(SeriesClass(-3), "chart-series chart-series--1")
}

func TestSeriesLegend(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.Eq(SeriesLegend([]string{"Met", "Won"}), []LegendKey{
		{Class: "chart-series chart-series--1", Label: "Met"},
		{Class: "chart-series chart-series--2", Label: "Won"},
	})
}
