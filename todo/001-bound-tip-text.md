# Bound the label text each chart mark prints

Closes `find_01HMJxkQ5vKSHdV1jtLdSVwt` (CWE-770, medium),
`find_01BRfRsA5bPTNggSus9egwZi` (CWE-770, medium), and
`find_01KkW5ru74ZN2rDVvte8WAhe` (CWE-770, low).

## The findings

A caller passes a name once, and a builder puts it in each mark:

- `NewLines`: `plot.columns` puts `Series.Label` in the tip of each
  point.
- `NewGroupedBars`, `NewValueGroupedBars`, and `NewStackedColumns`
  put each series name in the tip of each column. They also put it in
  `GroupedBar.Series` and `StackedSegment.Series`, which the views
  write into the `aria-label` of each linked bar or segment.
- `NewHeatmap` puts each X label in the tip title of each cell in its
  column, and each Y label in a tip line of each cell in its row.

Nothing bounds the bytes of the name. The `Tip` keeps a substring, so
builder memory stays linear. But the view prints each copy in full, so
the page grows as name bytes times marks. `strings.TrimSpace` also
scans the white space of the full name for each mark.

- Attacker: a signed-in user who stores a long series, tag, or project
  name.
- Boundary: an app that charts that name and does not cap its length.
- Consequence: a 100 KB name on 1,000 line points renders 100 MB. On
  365 linked columns of grouped bars it renders 73 MB. A few page views
  can stop the server.

The same path, with no finding yet, is in `NewXYScatter` (`XLabel` and
`YLabel` on each point), `NewSeriesBars` (`countLabel` on each row),
`NewValueBars` (`valueLabel` on each row), and `NewHeatmap`
(`RowLabel` and `CountLabel` on each cell).

The `NewLines` doc comment says that the work is linear in the points.
That is not true for the label length.

## Outcome: fix it

Bound the text at the `Tip` constructors, which each builder uses.

- Add `tipText`. It reads at most `tipTextMax` bytes of a string, cuts
  on a rune boundary, trims the white space, and adds "…" when it cut.
  It never scans past the limit.
- `newTip` passes the title through it. `newTipLine` passes the label
  and the value. `capNames` passes each name.
- `NewGroupedBars`, `NewValueGroupedBars`, and `NewStackedColumns`
  bound each series name once, before the column loop. The bounded
  name goes into the tip and into `Series`, so the `aria-label` is
  bounded too.

Use 200 bytes for `tipTextMax`. The tip panel is at most 20rem wide,
about 40 characters on each line, so 200 bytes is five lines. A name
longer than that is not a glance, and the full name belongs in a table
under the chart.

Each mark then prints at most a fixed number of bytes of each name.
The page is linear in the name bytes plus the marks. The legend, the
end labels, and the axis labels print a name once, so they keep it in
full.

Cost: one helper and its calls. No class, custom property, or view
argument changes. No app reads `GroupedBar.Series` or
`StackedSegment.Series`. A tip title over 200 bytes now ends in "…",
so the `Tip` doc comment changes from "in full".

## Options not taken

- Trim each name once per series in each builder. It removes the scan,
  but each mark still prints the full name, so the page still grows as
  name bytes times marks.
- Limit the bytes in `Partial`. `Partial` writes into the page builder
  of the app, so this package cannot limit the page. It needs an hml
  change and a setting in each app.
- Ask each app to cap name length. It keeps the cost contract hidden,
  and the threat model says that the package cannot assume a cap.

## Steps

1. Red: add `TestTipText` in `tip_test.go`. Assert a short name, a
   name over the limit, a cut inside a rune, and a name that is mostly
   white space.
2. Red: add `TestViewsPrintEachNameABoundedNumberOfTimes` in
   `views_test.go`. For each builder and each name it takes, render
   once with a 1,000 byte name and once with a 101,000 byte name. The
   growth divided by 100,000 is the count of full copies. Assert that
   count for each case.
3. Green: add `tipText` and call it in `newTip`, `newTipLine`, and
   `capNames`. Bound the series names in the three column builders.
4. Update the doc comments of `Tip`, `NewLines`, and the three column
   builders: each mark prints at most `tipTextMax` bytes of a name.
5. Delete this plan and its line in `todo/README.md`. Add the rule to
   the threat model: a string a caller passes once does not print in
   full once for each mark.

Then resolve each finding as fixed, with the merge commit.
