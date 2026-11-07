# chart

Charts for a server-rendered Go page, with no chart library and, for
most charts, no JavaScript. A builder computes the geometry in Go, an
[hml](https://github.com/croaky/hml) view writes divs or an SVG, and
one CSS file paints them.

```go
lines := chart.NewLines(chart.USD, []chart.Series{{
	Label:  "Revenue",
	Class:  chart.SeriesClass(0),
	Points: points,
}}, chart.WithLocation(loc))
```

```hml
= render "chart/lines", data: lines
```

Pass each `chart/` partial name to the package from the app's own
partial writer:

```go
func partial(name string, ctx *hml.Context, w *strings.Builder) error {
	if strings.HasPrefix(name, "chart/") {
		return chart.Partial(name, ctx, w)
	}
	// the app's own views
}
```

Serve `chart.CSS`, or add it to the app's sheet. Its rules sit in
`@layer components`, and every class starts with `chart-`. Set the
custom properties in its header to theme it.

The charts:

- `NewLines`: a line for each series over time, with a tip for each
  date. Options set a log axis, a fixed or signed domain, time bounds,
  buckets, a time zone, labels at the end of each line, and event marks.
- `NewSparkline`: a small line chart with no hover and no style
  attribute.
- `NewSeriesBars`, `NewValueBars`: a bar for each category.
- `NewGroupedBars`, `NewValueGroupedBars`: a bar for each series,
  grouped by category.
- `NewStackedColumns`: a column for each period, divided by series.
- `NewXYScatter`: a dot for each point on two log axes.
- `NewEventMarks`: marks over a time axis, clustered when close.
- `NewHeatmap`: a grid of counts on a color ramp.

Each view with a tip, a bar, or a placed label writes style attributes
that set custom properties only. Under a policy of `style-src 'self'`,
add `style-src-attr 'unsafe-inline'`, or use `NewSparkline`, which
needs neither. See `doc.go`.

## GitHub repo is a mirror

Development happens on [sockeye](https://sockeye.sh), a
self-hosted review and CI server, which holds in progress branches.
GitHub receives `main` and the tags so `go get` works.

## License

MIT
