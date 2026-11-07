package chart

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/croaky/hml"
	"github.com/croaky/is"
)

func render(t *testing.T, name string, locals map[string]any) string {
	t.Helper()
	var b strings.Builder
	if err := Partial(name, hml.NewContext(locals), &b); err != nil {
		t.Fatalf("render %s: %v", name, err)
	}
	return b.String()
}

// mustContain reports each part that out does not hold, so one run
// names every missing piece of markup.
func mustContain(t *testing.T, out string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(out, p) {
			t.Errorf("output lacks %q\n%s", p, out)
		}
	}
}

func TestViewsParse(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	views, err := parseViews()
	is.NoErr(err)
	names := make([]string, 0, len(views))
	for name := range views {
		names = append(names, name)
	}
	slices.Sort(names)
	is.Eq(names, []string{
		"chart/bars", "chart/events", "chart/grouped_bars", "chart/heatmap", "chart/legend",
		"chart/lines", "chart/scatter", "chart/sparkline", "chart/stacked_columns", "chart/tip",
	})
}

// Each view reads data and nothing else, except the legend, so a page
// renders a chart with one argument.
func TestViewsReadOnlyTheirArguments(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	views, err := parseViews()
	is.NoErr(err)
	for name, v := range views {
		want := []string{"data"}
		if name == "chart/legend" {
			want = []string{"keys", "title"}
		}
		is.Eq(v.Names(), want)
	}
}

func TestPartialRefusesAnUnknownView(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	var b strings.Builder
	err := Partial("chart/nope", hml.NewContext(nil), &b)
	is.HasErr(err)
	is.Eq(err.Error(), `chart: no view "chart/nope"`)
}

// A chart with nothing to draw renders nothing, not an empty axis.
func TestViewsRenderNothingWithNoChart(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	for name, data := range map[string]any{
		"chart/lines":           Lines{},
		"chart/sparkline":       Sparkline{},
		"chart/bars":            Bars{},
		"chart/grouped_bars":    GroupedBars{},
		"chart/stacked_columns": StackedColumns{},
		"chart/scatter":         Scatter{},
		"chart/events":          EventMarks{},
		"chart/heatmap":         Heatmap{},
	} {
		is.Eq(strings.TrimSpace(render(t, name, map[string]any{"data": data})), "")
	}
}

func TestLinesView(t *testing.T) {
	t.Parallel()

	l := NewLines(Count, twoSeries(), WithEndLabels(),
		WithEvents([]Event{{At: day0.AddDate(0, 0, 1), Label: "deploy"}}))
	out := render(t, "chart/lines", map[string]any{"data": l})

	mustContain(t, out,
		`data-chart-from="2026-01-05T00:00:00Z"`,
		`viewBox="0 0 1000 300"`,
		`preserveAspectRatio="none"`,
		`points="0.0,225.0 333.3,150.0 1000.0,0.0"`,
		`style="--chart-at: 100.00%"`,
		`style="--chart-left: 0.00%; --chart-width: 16.67%"`,
		`tabindex="0"`,
		`Jan 6, 2026`,
		`chart-lines-end`,
		`chart-lines-label`,
		`chart-event`,
		`deploy`,
		`chart-legend-swatch`,
		`Jan 8`,
	)
}

// A signed axis draws its zero rule.
func TestLinesViewZeroRule(t *testing.T) {
	t.Parallel()

	l := NewLines(Count, []Series{{Label: "a", Points: []Point{pt(0, -1), pt(1, 2)}}}, WithDomain(-6, 6))
	out := render(t, "chart/lines", map[string]any{"data": l})

	mustContain(t, out, `chart-plot-zero`, `y1="150.0"`)
}

// A sparkline writes no style attribute, so it renders under a
// Content-Security-Policy of style-src 'self'.
func TestSparklineViewWritesNoStyle(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	s := NewSparkline(Count, []Series{{Label: "a", Class: SeriesClass(0), Points: []Point{pt(0, -2), pt(3, 3)}}}, WithDomain(-6, 6))
	out := render(t, "chart/sparkline", map[string]any{"data": s})

	mustContain(t, out, `chart-sparkline-svg`, `points="0.0,200.0 1000.0,75.0"`, `chart-plot-zero`, `Jan 8`, `-6`)
	is.True(!strings.Contains(out, "style="))
}

func TestTipView(t *testing.T) {
	t.Parallel()

	tip := NewTipAt("Acme", 0.9).
		WithImage("/acme.png").
		CountWithNames("Deals", 2, []string{"One", "Two"}).
		SubLine("Open", "1").
		WithNames([]string{"Ann", "Bo"})
	out := render(t, "chart/tip", map[string]any{"data": tip})

	mustContain(t, out,
		`chart-tip--right`,
		`role="tooltip"`,
		`src="/acme.png"`,
		`Acme`,
		`chart-tip-label`,
		`chart-tip-line-names`,
		`One, Two`,
		`chart-tip-line--sub`,
		`chart-tip-names`,
		`Ann, Bo`,
	)
}

func TestBarsView(t *testing.T) {
	t.Parallel()

	b := NewSeriesBars("Companies", []SeriesBarRow{{Label: "A", Count: 3}, {Label: "B", Count: 7}})
	out := render(t, "chart/bars", map[string]any{"data": b})

	mustContain(t, out, `chart-axis-y--even`, `style="--chart-pct: 37.5%"`, `chart-bars-count`, `chart-tip--left`)
}

func TestGroupedBarsView(t *testing.T) {
	t.Parallel()

	g := NewGroupedBars([]string{"Met", "Won"}, []GroupedColumn{{Label: "Q1", Counts: []int{3, 1}, URLs: []string{"/q1"}}})
	g.LegendTitle = "Outcome:"
	out := render(t, "chart/grouped_bars", map[string]any{"data": g})

	mustContain(t, out, `href="/q1"`, `aria-label="Met: 3"`, `chart-bars-group`, `chart-legend-title`, `Outcome:`)
}

func TestStackedColumnsView(t *testing.T) {
	t.Parallel()

	s := NewStackedColumns([]string{"Met"}, nil, []StackedColumn{
		{Label: "Q1", Counts: []int{2}, URLs: []string{"/q1"}, Selected: true},
		{Label: "Q2", Counts: []int{1}},
	})
	out := render(t, "chart/stacked_columns", map[string]any{"data": s})
	mustContain(t, out, `href="/q1"`, `aria-label="Met: 2"`, `chart-bars-col--selected`, `chart-axis-x`)

	cols := make([]StackedColumn, 13)
	for i := range cols {
		cols[i] = StackedColumn{Label: "W" + delimited(i+1), Counts: []int{1}}
	}
	long := NewStackedColumns([]string{"Met"}, nil, cols)
	long.Events = NewEventMarks([]Event{{At: day0, Label: "launch"}})
	out = render(t, "chart/stacked_columns", map[string]any{"data": long})
	mustContain(t, out, `chart-bars--long`, `chart-axis-dates--bars`, `W13`, `launch`)
}

func TestScatterView(t *testing.T) {
	t.Parallel()

	s := NewXYScatter(XYScatterArgs{Points: []XYPoint{{Label: "A", X: 10, Y: 100}, {Label: "B", X: 1000, Y: 10}, {Label: "C"}}})
	out := render(t, "chart/scatter", map[string]any{"data": s})

	mustContain(t, out, `chart-rule--down`, `chart-axis-x-tick`, `style="--chart-x: 0.00%; --chart-y: 100.00%"`, `1 point not plotted`)
}

func TestHeatmapView(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	h := NewHeatmap(HeatmapArgs{XLabels: []string{"a", "b"}, YLabels: []string{"fast"}, Counts: [][]int{{3}, {0}}},
		WithBounds(day0, day0.Add(time.Hour)))
	out := render(t, "chart/heatmap", map[string]any{"data": h})

	mustContain(t, out, `chart-ramp--8`, `data-chart-to="2026-01-05T01:00:00Z"`, `fast`)
	// A cell is not a stop in the tab order.
	is.True(!strings.Contains(out, `tabindex`))
}

var (
	cssComment = regexp.MustCompile(`(?s)/\*.*?\*/`)
	classInCSS = regexp.MustCompile(`\.([a-z][a-z0-9_-]*)`)
	// classInView reads the classes of a tag line: %tag.a.b or .a.b.
	classInView = regexp.MustCompile(`^\s*(?:%[a-z0-9]+)?((?:\.[a-z][a-z0-9_-]*)+)`)
	// classInGo reads a string literal of classes that Go sets.
	classInGo = regexp.MustCompile(`"(chart-[a-z0-9_ -]+)"`)
	// componentName is the rule of blog/web/css for a component class:
	// kebab-case, an optional --modifier, and no __. Each class here
	// also starts with chart-, so it cannot collide with an app's.
	componentName = regexp.MustCompile(`^chart-[a-z0-9]+(?:-[a-z0-9]+)*(?:--[a-z0-9-]+)?$`)
)

// usedClasses is each class a view writes or a Go file sets.
func usedClasses(t *testing.T) map[string]bool {
	t.Helper()
	used := map[string]bool{}
	views, err := filepath.Glob("views/*.hml")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range views {
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for line := range strings.Lines(string(src)) {
			m := classInView.FindStringSubmatch(line)
			if m == nil {
				continue
			}
			for c := range strings.SplitSeq(strings.TrimPrefix(m[1], "."), ".") {
				used[c] = true
			}
		}
	}
	goFiles, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range goFiles {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		src, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, m := range classInGo.FindAllStringSubmatch(string(src), -1) {
			for c := range strings.FieldsSeq(m[1]) {
				used[c] = true
			}
		}
	}
	return used
}

// cssClasses is each class a selector in chart.css names.
func cssClasses() map[string]bool {
	classes := map[string]bool{}
	for _, m := range classInCSS.FindAllStringSubmatch(cssComment.ReplaceAllString(CSS, ""), -1) {
		classes[m[1]] = true
	}
	return classes
}

// The three layers agree: each class the views or the builders write
// has a rule, and each rule's class is one something writes. A rule
// nothing writes is a rule nobody can test, and a class with no rule
// is a mark that draws unstyled.
func TestCSSCoversEachClass(t *testing.T) {
	t.Parallel()

	used, styled := usedClasses(t), cssClasses()
	for c := range used {
		if !styled[c] {
			t.Errorf("class %q has no rule in chart.css", c)
		}
	}
	for c := range styled {
		if !used[c] {
			t.Errorf("chart.css styles %q, which no view or builder writes", c)
		}
	}
	if len(used) < 50 {
		t.Errorf("found %d classes, want more: the scan of views and Go files is broken", len(used))
	}
}

func TestClassNamesFollowTheComponentRule(t *testing.T) {
	t.Parallel()

	for c := range usedClasses(t) {
		if !componentName.MatchString(c) {
			t.Errorf("class %q is not chart- and kebab-case with an optional --modifier", c)
		}
	}
}

// Each rule sits in @layer components, so an app's layer order places
// it. A rule outside a layer would beat each layered rule in the app,
// and !important would beat the app's utilities.
func TestCSSSitsInTheComponentsLayer(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	css := strings.TrimSpace(cssComment.ReplaceAllString(CSS, ""))
	is.True(strings.HasPrefix(css, "@layer components {"))
	is.True(strings.HasSuffix(css, "}"))
	is.Eq(strings.Count(css, "@layer"), 1)
	is.True(!strings.Contains(css, "!important"))
	is.Eq(strings.Count(css, "{"), strings.Count(css, "}"))
}

// A breakpoint is a strict range, width < 70em. max-width: 70em also
// matches at 70em, where a min-width: 70em rule applies, so a screen
// that wide would get both layouts.
func TestCSSMediaQueriesUseRanges(t *testing.T) {
	t.Parallel()

	for _, m := range regexp.MustCompile(`@media[^{]*`).FindAllString(CSS, -1) {
		if strings.Contains(m, "max-width") || strings.Contains(m, "min-width") {
			t.Errorf("%q: write the breakpoint as a range, such as (width < 70em)", m)
		}
	}
}

// Spacing, font size, and weight spend a token from the app's scale,
// so a chart is the same size as the page around it. A hairline in px
// and a zero are not steps of the scale and keep a literal.
func TestCSSSpendsTokens(t *testing.T) {
	t.Parallel()

	decl := regexp.MustCompile(`(?m)^\s*((?:margin|padding|gap|column-gap|row-gap|font-size|font-weight)[a-z-]*):\s*([^;]+);`)
	literal := regexp.MustCompile(`^(?:0|[1-2]px)(?: (?:0|[1-2]px))*$`)
	for _, m := range decl.FindAllStringSubmatch(cssComment.ReplaceAllString(CSS, ""), -1) {
		prop, value := m[1], strings.TrimSpace(m[2])
		if !strings.Contains(value, "var(--") && !literal.MatchString(value) {
			t.Errorf("%s: %s spends no token", prop, value)
		}
	}
}
