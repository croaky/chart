package chart

// SeriesLimit is how many series one plot colors.
//
// The palette sets the limit, and the palette is limited by how many
// hues a reader can tell apart at the width of a line. Past ten, the
// eleventh color is one a reader looks up in the legend every time.
const SeriesLimit = 10

// SeriesClass is the class that colors series i of a plot: the stroke
// of its line, the fill of its bar, and the swatch of its legend key.
//
// The palette is categorical, not a ramp. An ordered thing, such as a
// stage in a pipeline, wants neighbors that look related. Things
// compared on one plot have no order, so neighbors must look
// unrelated. An app keeps its own ramp for ordered things.
//
// Assign from a list the caller has ordered, once for each page. Then
// one thing is one color in every plot on the page, and a reader learns
// the colors once.
//
// Each class is written out, so a search for the selector finds the
// code that sets it. An index past the palette wraps, which repeats a
// color but leaves no line unpainted. A caller caps at [SeriesLimit].
func SeriesClass(i int) string {
	classes := []string{
		"chart-series chart-series--1",
		"chart-series chart-series--2",
		"chart-series chart-series--3",
		"chart-series chart-series--4",
		"chart-series chart-series--5",
		"chart-series chart-series--6",
		"chart-series chart-series--7",
		"chart-series chart-series--8",
		"chart-series chart-series--9",
		"chart-series chart-series--10",
	}
	if i < 0 {
		i = 0
	}
	return classes[i%len(classes)]
}

// LegendKey is one entry in a legend: a swatch in the class of the
// marks it names, and the name.
//
// The swatch wears the same class as its line or its bar, so a key
// cannot drift from its series. A legend goes below the plot, not
// beside it, so a narrow screen does not squeeze the plot to make room.
type LegendKey struct {
	Class string `json:"class"`
	Label string `json:"label"`
}

// SeriesLegend names each series in the order given, in its
// [SeriesClass] color. The order is the order of the bars in each
// group, so a reader matches the legend to a group from left to right.
func SeriesLegend(series []string) []LegendKey {
	keys := make([]LegendKey, 0, len(series))
	for i, name := range series {
		keys = append(keys, LegendKey{Class: SeriesClass(i), Label: name})
	}
	return keys
}
