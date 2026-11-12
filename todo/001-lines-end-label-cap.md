# Cap the NewLines end labels

Closes `find_01HazofbjUCujQ2ku5xhfvNR` (CWE-407, low).

## The finding

The `plot.endLabels` Go method in `lines.go` gives each end label one
CSS expression. The expression for label i holds one term for each
label at or above it. Thus n labels hold n(n+1)/2 + n terms of about 25
bytes each. `Lines.EndLabels` keeps them, and the `chart/lines` view
writes a second copy into the page.

- Attacker: a signed-in user who stores many series of one point each.
- Boundary: an app that passes those series to `NewLines` with
  `WithEndLabels()`, and does not cap the series count.
- Consequence: about 50 MB of style text at 2,000 series and 1.3 GB at
  10,000. A few page views can stop the server.

The `NewLines` doc comment also says that the work is linear in the
points. That is not true while end labels are on.

No app is open to this today. The one app that calls `WithEndLabels`
passes two series that its own code makes. The package contract is
still wrong, and the threat model says a builder must not turn linear
data into work that grows faster than the data.

## Outcome: fix it

Cap the end labels at a fixed count, `maxEndLabels`. When more series
draw than the cap, `NewLines` draws no end labels and sets
`HasLegend`, as it does without `WithEndLabels`.

The cap also fixes the chart. Each label needs one line of the plot.
At the default `--chart-height` of 12rem and a line height of 1.2,
about ten labels fit. Past that count, the bottom limit of the walk up
pushes the top labels above the plot, and they overlap the page. A
legend is the correct picture for many series.

Use 12 for the cap. It keeps a margin for a taller plot. At 12 labels
the expressions hold 90 terms, about 2 KB, on each chart.

Cost: about six lines of Go and one test. No new part, no new option,
and no change to the speed of a chart under the cap. No class, custom
property, or view argument changes, so no app breaks.

## Options not taken

- Remove `WithEndLabels`. One app uses it, so removal moves the work
  to that app.
- Build a linear CSS chain: each label reads a custom property of the
  label above it. Custom properties do not cross sibling elements, so
  the chain must sit on the parent. The browser then expands each
  `var()` again, which is quadratic in the browser and can hit its
  limit on expanded length. It moves the cost, and it adds code.
- Place the labels in Go. Go does not know the plot height in lines,
  which is why the browser does the last step. A new option for the
  height adds API for a case no app has.
- Only fix the doc comment and ask the caller to cap series. It is the
  smallest change, but each app must then know about a hidden rule of
  one option. The cap is a few lines more and makes the rule true for
  each app.

## Steps

1. Red: add `TestLinesEndLabelsCap` in `lines_test.go`. Give
   `maxEndLabels`+1 series of one point. Assert that `HasEndLabels` is
   false, `EndLabels` is empty, and `HasLegend` is true. Give
   `maxEndLabels` series and assert that each one has a label.
2. Green: in `NewLines`, set end labels only when
   `len(p.series) <= maxEndLabels`. Set `HasLegend` from the same
   test.
3. Say why in the doc comment of `WithEndLabels` and of `endLabels`:
   more labels than plot lines cannot fit, and the CSS of each label
   grows with the labels above it.
4. Fix the `NewLines` doc comment: the work is linear in the points,
   because the end labels stop at a fixed count.
5. Delete this plan and its line in `todo/README.md`.

Then resolve the finding as fixed, with the merge commit.
