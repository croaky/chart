# security: build NewLines tips in memory linear in the points

Findings:

- `find_01XiAq7SdfCrovKZ2SLG2XcZ` (low, CWE-407): `plot.columns` in
  `lines.go` reserves a `TipLine` for each series on each date, so
  memory grows as series times dates.

Commit `54f21af` added the reservation
`tip.Lines = make([]TipLine, 0, len(p.series))` to cut allocations.
It is right when the series share their dates. It is wrong when they
do not.

## Threat model

This plan applies the threat model in `README.md` to this finding.

- Attacker: a signed-in user of an app who stores many series whose
  dates do not match, such as one series for each thing the user
  creates. The app passes all of them to `NewLines`.
- Boundary: the `NewLines` builder. `NewSparkline` shares `newPlot`
  but does not build columns. `SeriesLimit` is a palette limit in a
  comment. No builder enforces it.
- Consequence: with N series and D dates, `plot.columns` holds
  N x D x 80 bytes until the view renders. 2,000 series of ten
  points each reserve about 3.2 GB for 20,000 points. A few page
  views push the server into an OOM kill.
- Second cost, not in the finding: the same loop does N x D map
  lookups. That is CPU that also grows as series times dates.

The finding meets the threat model. The data is linear, and the
builder turns it into quadratic work.

## Outcome: fix it

The fix replaces code and does not add a moving part. It removes the
N maps that index each series by date. It does not slow CI. Accepting
the risk does not fit: a few stored rows stop the server, and the fix
is smaller than a cap.

## Design

Build the tips from the points, not from the grid of series and
dates.

1. Count the lines of each column in one walk over the points:
   `counts[p.columnOf[pt.At.UnixNano()]]++`. `p.columnOf` exists.
2. Allocate one backing `[]TipLine` with the sum of the counts. Give
   each tip a slice of it with a full-slice cap:
   `lines[off:off:off+n]`. Then an append to one tip cannot write into
   the next.
3. Walk the series in order, and each series' points in order. Append
   a line to the tip of the point's column. The series order stays
   the order of the lines, so the markup does not change.
4. Keep the current rule for two points of one series on one date:
   the last value wins, as the `at` map does now. Keep the series
   index of the last line in each column. Overwrite that line when
   the index is the same, and count the column once in step 1.
5. Delete the `at` maps and the reservation.

Memory and CPU are then linear in the points. The aligned case of
`BenchmarkNewLines` makes one tip-line allocation for the chart, not
one for each tip.

## Options and costs

- A, recommended: the design above. Cost: two `[]int` of length D and
  about the same lines of code as now. No new moving part. Speed:
  likely faster than now, because it removes N maps and D allocations.
- B: delete the reservation line only. Cost: one line. Memory is then
  linear, but each tip grows by append, so the aligned benchmark loses
  what `54f21af` gained. The N x D lookups stay.
- C: count the hits for each date before the `make`. Cost: a second
  lookup for each series on each date. Memory is exact, but the
  N x D CPU stays and doubles.
- D: cap `NewLines` at `SeriesLimit` series. This drops a caller's
  data with no error, and it changes behavior for each app that passes
  more than ten series. It adds a rule that A makes needless. Not
  recommended.

## Contract

The finding also asks for a bound or a documented contract. After A,
each part of `NewLines` is linear in the points, so add no cap. Add
one paragraph to the `NewLines` doc comment: the work and the memory
of a chart are linear in the points that draw, and the caller bounds
how many points it passes from user data.

## Tests

Red first.

- A test in `lines_test.go` with three series whose dates do not
  match. Assert the exact tips: the dates, the order of the lines, and
  the values. Assert `cap(tip.Lines) == len(tip.Lines)` for each tip.
  Today the cap is 3, so the test fails.
- A test with two points of one series on one date. Assert one line
  with the last value.
- Add an `unaligned` case to `BenchmarkNewLines`: 200 series of ten
  points each on distinct dates. Compare `B/op` before and after.
- The view tests in `views_test.go` must pass with no change. They
  confirm the markup stays byte for byte the same.

## After the merge

Propose `claudesec resolve` on `find_01XiAq7SdfCrovKZ2SLG2XcZ` as
fixed, and name the merge commit.
