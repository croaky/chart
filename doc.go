// Package chart draws charts on a server-rendered page with no chart
// library and, for most charts, no JavaScript.
//
// A builder computes the geometry in Go. An hml view writes divs or an
// SVG from it, and chart.css paints them. The package ships all three,
// because the part of a chart that drifts between pages is the part
// around the plot: the tip, the legend, the axis labels, and the
// classes. One shape for each, tested together, keeps two pages from
// reading as two apps.
//
// # Rules
//
//   - Go computes each position. A percentage, a polyline point, and
//     the side a tip opens on are data, so a test can call the function
//     that makes them. A view runs no arithmetic.
//   - A categorical chart is divs. A chart with a continuous axis is an
//     SVG for the marks and HTML for the text, because a glyph in an
//     SVG that stretches to its box would distort.
//   - A time sits where it falls in the range, not where it sits in the
//     slice. A gap in reporting draws as a gap. See [TimeXs].
//   - A bar scales against the top of its axis, not its largest value,
//     so the tallest bar stops at a number the axis names. Use
//     [AxisScale] for small counts and [NewValueAxis] for anything else.
//   - The axis and the tip print each number through [FormatValue], in
//     one [Format], so a tip reads in the units of the axis beside it.
//   - A tip is an element that CSS shows on hover and on focus. It is
//     not a title attribute. See [Tip].
//   - Things compared on one plot take [SeriesClass] colors. Assign them
//     once for each page, so one thing is one color in every plot. A
//     legend swatch wears the class of its marks, so it cannot drift.
//   - A chart with nothing to draw is no chart. Each builder sets
//     HasChart to false, and its view renders nothing.
//
// # Views and CSS
//
// [Partial] renders a view by name, such as "chart/lines", and an app's
// partial writer passes each chart/ name to it. [CSS] is the
// stylesheet. Its rules sit in @layer components, and each class
// starts with chart-, so the sheet sits beside an app's own sheet. An
// app themes it with custom properties that the sheet's header lists.
//
// A view sets an inline style that holds custom properties only, such
// as --chart-pct and --chart-at, never a layout property. The sheet
// decides how to spend the number. A bar's share is a height in a
// column on a wide screen and a width in a row on a narrow one, from
// the same markup, and CSS cannot reread an inline height as a width.
//
// # Content-Security-Policy
//
// A view with hover bands, bars, or placed labels writes style
// attributes. A policy that refuses them, such as style-src 'self',
// needs style-src-attr 'unsafe-inline' for those views. That keeps
// style-src-elem 'self' and script-src 'self'. A style attribute can
// run no script.
//
// [Sparkline] writes no style attribute. Its geometry is SVG attributes,
// and its labels sit in plain order around the SVG. It renders under
// style-src 'self' with no exception.
package chart
