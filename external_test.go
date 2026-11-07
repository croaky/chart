package chart_test

import (
	"strings"
	"testing"
	"time"

	"github.com/croaky/hml"
	"github.com/croaky/is"

	"github.com/croaky/chart"
)

// page is an app's own view that renders two charts by name.
const page = `%main
  = render "chart/lines", data: lines
  = render "chart/bars", data: bars
`

// partial is an app's partial writer: it passes each chart/ name to the
// library and keeps the rest.
func partial(name string, ctx *hml.Context, w *strings.Builder) error {
	if strings.HasPrefix(name, "chart/") {
		return chart.Partial(name, ctx, w)
	}
	w.WriteString("app partial ")
	w.WriteString(name)
	return nil
}

// A caller in another module builds charts from the public API and
// renders them through its own partial writer.
func TestAnAppRendersCharts(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	start := time.Date(2026, time.March, 2, 0, 0, 0, 0, time.UTC)
	lines := chart.NewLines(chart.USD, []chart.Series{{
		Label:  "Revenue",
		Class:  chart.SeriesClass(0),
		Points: []chart.Point{{At: start, Value: 1_000_000}, {At: start.AddDate(0, 1, 0), Value: 1_500_000}},
	}})
	bars := chart.NewSeriesBars("Deals", []chart.SeriesBarRow{{Label: "Won", Count: 4}, {Label: "Lost", Count: 6}})

	tmpl, err := hml.Parse(page, "page.hml", nil)
	is.NoErr(err)
	var out strings.Builder
	err = tmpl.RenderContextTo(&out, hml.NewContext(map[string]any{"lines": lines, "bars": bars}), partial)
	is.NoErr(err)

	html := out.String()
	is.True(strings.Contains(html, `class="chart-lines"`))
	is.True(strings.Contains(html, "$1.5M"))
	is.True(strings.Contains(html, `class="chart-bars"`))
	is.True(strings.Contains(html, "Deals"))
	is.True(strings.HasPrefix(chart.CSS, "/*"))
}
