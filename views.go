package chart

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"sync"

	"github.com/croaky/hml"
)

//go:embed views/*.hml
var viewFS embed.FS

// CSS is the stylesheet for the views, chart.css. An app serves it as a
// file or adds it to its own sheet. Its rules sit in
// @layer components, so the app's @layer order line places them.
//
// Colors, fonts, and sizes are custom properties with fallbacks, so an
// app sets its own values in :root and does not edit the sheet. The
// sheet's header lists them.
//
//go:embed chart.css
var CSS string

// Partial renders the view that name names, such as "chart/lines",
// into w. Its signature is [hml.PartialWriter], so an app's own
// partial writer passes each name that starts with "chart/" to it:
//
//	func partial(name string, ctx *hml.Context, w *strings.Builder) error {
//		if strings.HasPrefix(name, "chart/") {
//			return chart.Partial(name, ctx, w)
//		}
//		// the app's own views
//	}
//
// A page renders a chart with `= render "chart/lines", data: lines`.
// Each view reads data, the struct its builder returns, except
// "chart/legend", which reads keys and title.
func Partial(name string, ctx *hml.Context, w *strings.Builder) error {
	views, err := parseViews()
	if err != nil {
		return err
	}
	v, ok := views[name]
	if !ok {
		return fmt.Errorf("chart: no view %q", name)
	}
	return v.RenderContextTo(w, ctx, Partial)
}

// parseViews parses each embedded view once, on first use. A view that
// does not parse is an error on every render, not a panic at init, and
// a test parses each one.
var parseViews = sync.OnceValues(func() (map[string]*hml.Template, error) {
	files, err := fs.Glob(viewFS, "views/*.hml")
	if err != nil {
		return nil, err
	}
	views := make(map[string]*hml.Template, len(files))
	for _, file := range files {
		src, err := viewFS.ReadFile(file)
		if err != nil {
			return nil, err
		}
		t, err := hml.Parse(string(src), file, nil)
		if err != nil {
			return nil, err
		}
		views["chart/"+strings.TrimSuffix(path.Base(file), ".hml")] = t
	}
	return views, nil
})
