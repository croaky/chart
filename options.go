package chart

import "time"

// Option configures a builder over time: [NewLines], [NewSparkline],
// [NewEventMarks], [NewHeatmap], and [TimeTicks].
//
// One option type serves them all. So WithBounds and WithLocation mean
// the same thing on each, and a page passes one list to each chart. Each
// builder's comment names the options it reads. A builder ignores the
// others.
type Option func(*options)

type options struct {
	log       bool
	from, to  time.Time
	bucket    time.Duration
	hasDomain bool
	lo, hi    float64
	loc       *time.Location
	clock12   bool
	endLabels bool
	events    []Event
}

// clock is how a builder prints a time: in which zone, and on which
// clock.
type clock struct {
	loc *time.Location
	h12 bool
}

func (o options) clock() clock {
	return clock{loc: orUTC(o.loc), h12: o.clock12}
}

func readOptions(opts []Option) options {
	var o options
	for _, opt := range opts {
		opt(&o)
	}
	return o
}

// WithLog scales the value axis logarithmically. See [LogAxis].
func WithLog() Option {
	return func(o *options) { o.log = true }
}

// WithBounds sets the ends of the time axis. A plot of the last hour
// then spans the hour when the first minutes have no data, and two
// plots on one page share one time axis. A zero time leaves that end
// to the data.
func WithBounds(from, to time.Time) Option {
	return func(o *options) { o.from, o.to = from, to }
}

// WithBuckets says that each point stands for the span that starts at
// its time and lasts step, as a bucketed series does. A hover band then
// covers that span, and the time axis ends where the last span ends.
// See [BucketColumns].
func WithBuckets(step time.Duration) Option {
	return func(o *options) { o.bucket = step }
}

// WithDomain fixes the value axis from lo to hi, rounded out to whole
// steps. A value outside it sits on the edge of the plot.
//
// Use it when the data does not set the scale: a probability from 0 to
// 100, or an anomaly that two plots on a page show on one scale. It
// also lets a series of zeros draw, which is otherwise no chart.
func WithDomain(lo, hi float64) Option {
	return func(o *options) { o.hasDomain, o.lo, o.hi = true, lo, hi }
}

// WithLocation sets the time zone of the time labels and the tip
// titles. Without it they are in UTC.
func WithLocation(loc *time.Location) Option {
	return func(o *options) { o.loc = loc }
}

// WithClock12 prints the time labels and the tip titles on a 12-hour
// clock, 3:04 PM, where a reader expects one. Without it they are on a
// 24-hour clock, 15:04.
func WithClock12() Option {
	return func(o *options) { o.clock12 = true }
}

// WithEndLabels labels the end of each line with its series and its
// last value, and leaves out the legend, because each line then has
// its name at its end. See [EndLabel].
func WithEndLabels() Option {
	return func(o *options) { o.endLabels = true }
}

// WithEvents marks the events over a line chart. See [EventMarks].
//
// [NewLines] places the marks on the time axis that it resolves from
// its bounds and its data. A caller that placed them with
// [NewEventMarks] would need both bounds to put a mark on the same
// time as the line.
func WithEvents(events []Event) Option {
	return func(o *options) { o.events = events }
}
