package chart

import (
	"math"
	"strconv"
	"strings"
)

// Format is how an axis label and a tip print a number.
//
// The formats differ in more than a unit sign. A count under a million
// prints as digits, so 8,246 visitors reads as 8,246 and not as 8.2K.
// A reader would have to convert 8.2K back before comparing it with
// the axis. Money compacts at every size, because nobody reads
// $1,150,000,000 on an axis.
type Format int

const (
	// Count is a number of things: people, requests, stars. It is
	// delimited below a million and compact above it.
	Count Format = iota
	// USD is money, compact at every size: $500M, $1.2B.
	USD
	// Price is a unit price, such as dollars per million tokens. It
	// keeps three significant digits, so $0.00738 and $1.32 both read.
	// USD rounds the first to $0.
	Price
	// Percent is a share out of a hundred. It prints whole numbers,
	// and one decimal below 1%, so a small share does not print as 0%.
	Percent
	// Millis is a duration in milliseconds. It prints the unit that
	// keeps the number short: 850ms, 1.2s, 3m, 2h.
	Millis
	// Signed is a count read as a distance from zero, such as an
	// anomaly. It prints the sign of each value: +6, 0, −6. The minus
	// is U+2212, the width of the plus, so the digits line up.
	Signed
)

// FormatValue prints a number for an axis label or a tip.
//
// One function serves both, so a tip prints in the units of the axis
// beside it. A tip that printed 1,150,000,000 beside an axis labeled
// 1B would ask the reader to do the conversion the chart is for.
func FormatValue(v float64, f Format) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "–"
	}
	s := formatAbs(math.Abs(v), f)
	// A value that rounds to zero prints as zero, never as -0.
	if strings.Trim(s, "0.$%ms") == "" {
		return s
	}
	switch {
	case f == Signed && v < 0:
		return "−" + s
	case f == Signed:
		return "+" + s
	case v < 0:
		return "-" + s
	}
	return s
}

// formatAbs prints a value at or above zero.
func formatAbs(v float64, f Format) string {
	switch f {
	case USD:
		return "$" + compact(v)
	case Price:
		return "$" + significant(v)
	case Percent:
		if r := math.Round(v*10) / 10; r > 0 && r < 1 {
			return strconv.FormatFloat(r, 'f', 1, 64) + "%"
		}
		return delimited(int(math.Round(v))) + "%"
	case Millis:
		return millis(v)
	}
	// A count keeps its digits until they stop being countable.
	if v < 1_000_000 {
		return delimited(int(math.Round(v)))
	}
	return compact(v)
}

// millis prints a duration in the largest unit that keeps it at or
// above one. A value that rounds up to the next unit prints in that
// unit, so 59,990 ms is 1m and not 60s.
func millis(v float64) string {
	if math.Round(v) < 1000 {
		return delimited(int(math.Round(v))) + "ms"
	}
	units := []struct {
		size   float64
		suffix string
		next   string
		limit  float64
	}{
		{1000, "s", "1m", 60_000},
		{60_000, "m", "1h", 3_600_000},
	}
	for _, u := range units {
		if v < u.limit {
			s := oneDecimal(v / u.size)
			if s == "60" {
				return u.next
			}
			return s + u.suffix
		}
	}
	return oneDecimal(v/3_600_000) + "h"
}

// oneDecimal prints v with one decimal, and with none when the
// decimal is zero.
func oneDecimal(v float64) string {
	return strings.TrimSuffix(strconv.FormatFloat(v, 'f', 1, 64), ".0")
}

// significant prints a price to three significant digits: 0.00738,
// 0.129, 1.32, 12.5. From 100 up it prints whole numbers, which
// already carry as many digits as a reader compares.
func significant(v float64) string {
	if v == 0 {
		return "0"
	}
	if v >= 100 {
		return delimited(int(math.Round(v)))
	}
	scale := math.Pow(10, 2-math.Floor(math.Log10(v)))
	return strconv.FormatFloat(math.Round(v*scale)/scale, 'f', -1, 64)
}

// compactUnits is the suffix ladder, largest first.
var compactUnits = []struct {
	size   float64
	suffix string
}{
	{1_000_000_000_000, "T"},
	{1_000_000_000, "B"},
	{1_000_000, "M"},
	{1_000, "K"},
}

// compact prints a magnitude and its suffix: 1B, 1.2B, 500M, 8,246.
//
// It prints one decimal where the whole number would round away a
// difference, and none where the decimal is zero. So an axis of round
// steps reads as round numbers. Half a billion is 500M and not 0.5B,
// because the suffix that needs no decimal is the one to use. A value
// that rounds up to 1,000 of one unit moves to the next unit, so
// 999,960 prints as 1M and not 1000K.
func compact(v float64) string {
	if v < 1_000 {
		return delimited(int(math.Round(v)))
	}
	for i, u := range compactUnits {
		if v < u.size {
			continue
		}
		s := oneDecimal(v / u.size)
		if s == "1000" && i > 0 {
			return "1" + compactUnits[i-1].suffix
		}
		return s + u.suffix
	}
	return delimited(int(math.Round(v)))
}

// delimited prints a whole number with a comma between each group of
// three digits.
//
// It writes into one buffer of the final size, because an axis and
// each tip of a chart call it, and a growing builder allocated three
// times for each call.
func delimited(n int) string {
	// strconv.Itoa returns a small number without an allocation.
	if n > -1000 && n < 1000 {
		return strconv.Itoa(n)
	}
	var digits [24]byte
	s := strconv.AppendInt(digits[:0], int64(n), 10)
	start := 0
	if s[0] == '-' {
		start = 1
	}
	count := len(s) - start
	out := make([]byte, 0, len(s)+(count-1)/3)
	out = append(out, s[:start]...)
	first := start + count%3
	if first == start {
		first = start + 3
	}
	out = append(out, s[start:first]...)
	for i := first; i < len(s); i += 3 {
		out = append(out, ',')
		out = append(out, s[i:i+3]...)
	}
	return string(out)
}

// pluralize prints a count and the noun that agrees with it.
func pluralize(n int, singular, plural string) string {
	if n == 1 {
		return delimited(n) + " " + singular
	}
	return delimited(n) + " " + plural
}
