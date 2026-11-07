package chart

import (
	"testing"
	"time"

	"github.com/croaky/is"
)

var day0 = time.Date(2026, time.January, 5, 0, 0, 0, 0, time.UTC)

func days(n ...int) []time.Time {
	ts := make([]time.Time, len(n))
	for i, d := range n {
		ts[i] = day0.AddDate(0, 0, d)
	}
	return ts
}

// A time sits where it falls in the range, not where it sits in the
// slice, so a gap draws as a gap.
func TestTimeXsPlacesByTime(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.Eq(TimeXs(days(0, 1, 3), 300), []float64{0, 100, 300})
}

func TestTimeXsDegenerate(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(TimeXs(nil, 300), []float64{})
	is.Eq(TimeXs(days(2), 300), []float64{150})
	is.Eq(TimeXs(days(1, 1, 1), 300), []float64{0, 150, 300})
}

// Bounds set the ends of the plot, so a time sits where it falls
// between them, and bounds of zero width fall back to the data.
func TestTimeXsWithBounds(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	is.Eq(TimeXsWithBounds(days(0, 2), 400, day0.AddDate(0, 0, -1), day0.AddDate(0, 0, 3)), []float64{100, 300})
	is.Eq(TimeXsWithBounds(days(0, 2), 400, day0, day0), []float64{0, 400})
}

// The label fits the span, in the location the caller names.
func TestTimeTicks(t *testing.T) {
	t.Parallel()
	is := is.NewRelaxed(t)

	noon := day0.Add(12 * time.Hour)
	est := time.FixedZone("EST", -5*60*60)
	is.Eq(TimeTicks(noon, noon.Add(6*time.Hour), nil), []string{"12:00", "15:00", "18:00"})
	is.Eq(TimeTicks(noon, noon.Add(6*time.Hour), est), []string{"07:00", "10:00", "13:00"})
	is.Eq(TimeTicks(noon, noon.Add(36*time.Hour), nil), []string{"Jan 5 12:00", "Jan 6 06:00", "Jan 7 00:00"})
	is.Eq(TimeTicks(day0, day0.AddDate(0, 0, 4), nil), []string{"Jan 5", "Jan 7", "Jan 9"})
	is.Eq(TimeTicks(day0, day0.AddDate(1, 0, 0), nil), []string{"Jan 2026", "Jul 2026", "Jan 2027"})
	is.Eq(TimeTicks(noon, noon, nil), []string{"Jan 5 12:00"})
}

// A band runs halfway to each neighbor, and its crosshair sits at its
// own point inside it.
func TestTimeColumns(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.Eq(TimeColumns([]float64{0, 100, 300}, 300), []Column{
		{Style: "--chart-left: 0.00%; --chart-width: 16.67%", LineStyle: "--chart-at: 0.00%"},
		{Style: "--chart-left: 16.67%; --chart-width: 50.00%", LineStyle: "--chart-at: 33.33%"},
		{Style: "--chart-left: 66.67%; --chart-width: 33.33%", LineStyle: "--chart-at: 100.00%"},
	})
}

// One point spans the plot, and its crosshair sits at the point.
func TestTimeColumnsSinglePoint(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.Eq(TimeColumns([]float64{150}, 300), []Column{
		{Style: "--chart-left: 0.00%; --chart-width: 100.00%", LineStyle: "--chart-at: 50.00%"},
	})
}

// A bucket's band starts at its point and runs one bucket, to the edge
// of the plot at most.
func TestBucketColumns(t *testing.T) {
	t.Parallel()
	is := is.New(t)

	is.Eq(BucketColumns([]float64{0, 250, 900}, 1000, 200), []Column{
		{Style: "--chart-left: 0.00%; --chart-width: 20.00%", LineStyle: "--chart-at: 0.00%"},
		{Style: "--chart-left: 25.00%; --chart-width: 20.00%", LineStyle: "--chart-at: 0.00%"},
		{Style: "--chart-left: 90.00%; --chart-width: 10.00%", LineStyle: "--chart-at: 0.00%"},
	})
}
