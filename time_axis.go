package chart

import (
	"fmt"
	"time"

	"github.com/croaky/hml"
)

// TimeXs places each of an ascending run of times across a plot of the
// given width. A time sits where it falls in the range, not where it
// sits in the slice. So a year with two observations and a year with
// none look different, and a gap in reporting draws as a gap.
//
// The degenerate cases:
//
//   - No times: no positions.
//   - One time has no range to sit in, so it sits in the middle.
//   - Equal times still have no range, so they spread evenly. That is
//     the one answer that is not a division by zero.
func TimeXs(ts []time.Time, width float64) []float64 {
	xs := make([]float64, len(ts))
	if len(ts) == 0 {
		return xs
	}
	if len(ts) == 1 {
		xs[0] = width / 2
		return xs
	}
	first := ts[0]
	span := ts[len(ts)-1].Sub(first).Seconds()
	for i, t := range ts {
		if span <= 0 {
			xs[i] = width * float64(i) / float64(len(ts)-1)
			continue
		}
		xs[i] = width * t.Sub(first).Seconds() / span
	}
	return xs
}

// TimeXsWithBounds places times across a plot whose ends are from and
// to, not the first and the last time. A plot of the last hour then
// keeps its width when the first minutes have no data. A time outside
// the bounds sits outside the plot, so a caller drops it first.
//
// Bounds of zero width fall back to [TimeXs].
func TimeXsWithBounds(ts []time.Time, width float64, from, to time.Time) []float64 {
	span := to.Sub(from).Seconds()
	if span <= 0 {
		return TimeXs(ts, width)
	}
	xs := make([]float64, len(ts))
	for i, t := range ts {
		xs[i] = width * t.Sub(from).Seconds() / span
	}
	return xs
}

// inBounds reports whether t is on the axis from from to to, ends
// included. A zero end is open, as a zero time in [WithBounds] is.
func inBounds(t, from, to time.Time) bool {
	return (from.IsZero() || !t.Before(from)) && (to.IsZero() || !t.After(to))
}

// TimeTicks labels the start, the middle, and the end of a range, in
// loc. A nil loc is UTC.
//
// The label fits the span, not the data: a time of day for a few hours,
// a date and a time for a day or two, a date for a few months, and a
// month and a year past that. A label that repeats across the axis
// says nothing.
//
// The middle is the time halfway through the range, not the middle
// observation. The two are the same only when the data has no gaps.
func TimeTicks(from, to time.Time, loc *time.Location) []string {
	if !to.After(from) {
		return []string{spanLabel(from, 24*time.Hour, loc)}
	}
	span := to.Sub(from)
	return []string{
		spanLabel(from, span, loc),
		spanLabel(from.Add(span/2), span, loc),
		spanLabel(to, span, loc),
	}
}

// spanLabel prints t at the precision a range of the given span needs.
func spanLabel(t time.Time, span time.Duration, loc *time.Location) string {
	t = t.In(orUTC(loc))
	switch {
	case span <= 12*time.Hour:
		return t.Format("15:04")
	case span <= 48*time.Hour:
		return t.Format("Jan 2 15:04")
	case span <= 120*24*time.Hour:
		return t.Format("Jan 2")
	default:
		return t.Format("Jan 2006")
	}
}

// pointLabel names one point in the title of its tip. Points less than
// a day apart need their time, not only their date.
func pointLabel(t time.Time, gap time.Duration, loc *time.Location) string {
	t = t.In(orUTC(loc))
	if gap > 0 && gap < 24*time.Hour {
		return t.Format("Jan 2 15:04")
	}
	return t.Format("Jan 2, 2006")
}

// orUTC is loc, or UTC when loc is nil. A server's time.Local is
// usually UTC too, but a library that reads it labels a chart in the
// zone of whatever machine runs it, so the caller names the zone.
func orUTC(loc *time.Location) *time.Location {
	if loc == nil {
		return time.UTC
	}
	return loc
}

// Column places one hover band across a plot and the crosshair inside
// it. Style sets --chart-left and --chart-width on the band, and
// LineStyle sets --chart-at on the crosshair, each as a percentage.
//
// The crosshair sits at the point's own position in the band. That is
// not the band's center when the neighbors are not evenly spaced.
type Column struct {
	Style     hml.SafeCSS `json:"style"`
	LineStyle hml.SafeCSS `json:"line_style"`
}

// TimeColumns places a band for each x across a plot of the given
// width. A band runs to the midpoint between its x and each neighbor.
// That makes a sparse chart easy to point at: two observations four
// years apart get two bands half the plot wide, not two slivers with
// dead space between them.
//
// A chart that places its marks by time places its bands by time too.
// A band laid out as an equal flex item names a different date than
// the point under the pointer as soon as one date is missing.
func TimeColumns(xs []float64, width float64) []Column {
	cols := make([]Column, 0, len(xs))
	for i, x := range xs {
		left, right := 0.0, width
		if i > 0 {
			left = (xs[i-1] + x) / 2
		}
		if i < len(xs)-1 {
			right = (x + xs[i+1]) / 2
		}
		bandWidth := right - left
		// A single x spans the whole plot, so its crosshair sits at
		// its own position, not at the band's center.
		lineAt := 0.5
		if bandWidth > 0 {
			lineAt = (x - left) / bandWidth
		}
		cols = append(cols, Column{
			Style:     bandStyle(left/width, bandWidth/width),
			LineStyle: atStyle(lineAt),
		})
	}
	return cols
}

// BucketColumns places a band for each point of a bucketed series. A
// band starts at its point and runs one bucket, to the plot edge at
// most, and its crosshair sits on its left edge.
//
// Each point stands for the span that starts at its time. So the band
// under the pointer is the bucket the pointer is in. A band that split
// the gap to each neighbor would show the next bucket's values when the
// pointer is in the right half of a bucket.
func BucketColumns(xs []float64, width, bucket float64) []Column {
	cols := make([]Column, 0, len(xs))
	for _, x := range xs {
		right := min(width, x+bucket)
		cols = append(cols, Column{
			Style:     bandStyle(x/width, max(right-x, 0)/width),
			LineStyle: atStyle(0),
		})
	}
	return cols
}

// bandStyle places a band from left across width, both fractions of
// the plot.
func bandStyle(left, width float64) hml.SafeCSS {
	return hml.SafeCSS(fmt.Sprintf("--chart-left: %.2f%%; --chart-width: %.2f%%", 100*left, 100*width))
}
