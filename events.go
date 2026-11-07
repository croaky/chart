package chart

import (
	"slices"
	"strconv"
	"time"

	"github.com/croaky/hml"
)

// Event is one thing that happened at a time, such as a deploy.
type Event struct {
	At    time.Time
	Label string
}

// EventMark is a mark on a time axis for one event, or for a cluster
// of events too close to tell apart. Style sets --chart-at, the mark's
// place across the plot as a percentage.
type EventMark struct {
	Count  int         `json:"count"`
	Style  hml.SafeCSS `json:"style"`
	Tip    Tip         `json:"tip"`
	Events []Event     `json:"events"`
}

// EventMarks are the marks for a run of events.
type EventMarks struct {
	HasMarks bool        `json:"has_marks"`
	Marks    []EventMark `json:"marks"`
}

// eventClusterGap is the least distance between two marks, as a
// percentage of the plot's width. Two events closer than this share a
// mark with a count, because two marks that touch cannot be told apart
// or pointed at one by one.
const eventClusterGap = 1.2

// NewEventMarks places the events on a time axis and clusters each run
// of events closer than eventClusterGap. It reads the options
// WithBounds, WithLocation, and WithClock12. A zero bound leaves that
// end to the events, and an event outside the bounds draws no mark.
//
// Use it for a chart that has no time axis of its own, such as
// [StackedColumns]. A line chart takes its events with [WithEvents],
// because only [NewLines] knows the ends that its data sets.
//
// A mark sits at the mean place of its events. Its tip lists the time
// and the label of each event, up to [TipNameLimit], and counts the
// rest.
func NewEventMarks(events []Event, opts ...Option) EventMarks {
	o := readOptions(opts)
	return eventMarks(events, o.from, o.to, o.clock())
}

// eventMarks places the events on the axis from from to to. A zero end
// takes the time of the first or the last event.
//
// An event outside the axis draws no mark. On the edge, a mark would
// name a time the event did not happen at. A list of events longer
// than the plot would also stack there as one false cluster.
func eventMarks(events []Event, from, to time.Time, c clock) EventMarks {
	if len(events) == 0 {
		return EventMarks{}
	}
	sorted := slices.Clone(events)
	slices.SortStableFunc(sorted, func(a, b Event) int { return a.At.Compare(b.At) })
	if from.IsZero() {
		from = sorted[0].At
	}
	if to.IsZero() {
		to = sorted[len(sorted)-1].At
	}
	span := to.Sub(from).Seconds()

	type cluster struct {
		events []Event
		sumX   float64
		lastX  float64
	}
	var clusters []cluster
	for _, e := range sorted {
		if !inBounds(e.At, from, to) {
			continue
		}
		// An axis of zero width puts each mark in the middle. TimeXs
		// would spread the marks apart, which is right for a line and
		// wrong for a mark that says when.
		x := 50.0
		if span > 0 {
			x = 100 * e.At.Sub(from).Seconds() / span
		}
		if n := len(clusters); n > 0 && x-clusters[n-1].lastX < eventClusterGap {
			c := &clusters[n-1]
			c.events = append(c.events, e)
			c.sumX += x
			c.lastX = x
			continue
		}
		clusters = append(clusters, cluster{events: []Event{e}, sumX: x, lastX: x})
	}
	if len(clusters) == 0 {
		return EventMarks{}
	}

	marks := make([]EventMark, 0, len(clusters))
	for _, cl := range clusters {
		pos := cl.sumX / float64(len(cl.events)) / 100
		title := c.point(cl.events[0].At, 0)
		if len(cl.events) > 1 {
			title = strconv.Itoa(len(cl.events)) + " events"
		}
		tip := NewTipAt(title, pos)
		for _, e := range cl.events[:min(len(cl.events), TipNameLimit)] {
			tip = tip.Line(c.time(e.At), e.Label)
		}
		if rest := len(cl.events) - TipNameLimit; rest > 0 {
			tip = tip.SubLine("", "+"+strconv.Itoa(rest)+" more")
		}
		marks = append(marks, EventMark{
			Count:  len(cl.events),
			Style:  atStyle(pos),
			Tip:    tip,
			Events: cl.events,
		})
	}
	return EventMarks{HasMarks: true, Marks: marks}
}
