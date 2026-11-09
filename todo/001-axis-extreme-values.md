# security: drop chart values that an axis cannot place

Findings:

- `find_01JxSA6Fsi2GgKU1oKvtuD2J` (medium, CWE-835): the `LogAxis` tick
  loop runs without end when a rounded end is zero or infinite.
- `find_012nS3hRBUKWZm2FggG54pyb` (low, CWE-681): `NewValueAxis`
  panics in `make` when a tick count from an extreme value is not
  finite.

Both findings have one cause, so one change fixes both. The two axes
accept every finite float. The 1/2/5 rounding then overflows to `+Inf`
above about `1e308`, or underflows to `0` below about `1e-323`.

## Threat model

This plan applies the threat model in `README.md` to these findings.

- Attacker: a signed-in user of an app who stores a number that the
  app charts. The app passes the float to a builder without a clamp.
  Honest data can also do it: an underflowed ratio is a subnormal.
- Boundary: the package's exported builders and axis constructors.
  The package cannot assume that each app clamps its input.
- Consequence for the log axis: each render pins a CPU core or grows
  memory until the host kills the process. `recover` does not stop a
  loop that never returns. A stored value hits each page view, so a
  few views stop the server.
- Consequence for the value axis: the request panics. A recovering
  server fails the request for each user until someone removes the
  value. A goroutine with no `recover` stops the process.

## Outcome: fix it

The fix is small, adds no dependency, and costs no measurable speed.
It adds a few comparisons for each value and does not slow CI.
Accepting the risk does not fit. The log case stops a server through
data that a user can store, and no `recover` contains it.

## Design

1. Replace `finiteOr0` in `value_axis.go` with one rule for a value
   that an axis can place: finite, and zero or a magnitude from
   `1e-300` to `1e300`. A value outside that range counts as no value,
   the same as NaN. The margin keeps `2 * hi`, `10 * decade`, and
   `math.Pow(10, e)` finite and normal.
2. `NewValueBars`, `NewValueGroupedBars`, `resultText`, and
   `NewXYScatter` already call `finiteOr0`, so they get the rule with
   no new code. A bar with such a value draws empty with "No result".
   A scatter point counts in `Omitted`.
3. `newPlot` in `lines.go` tests `math.IsNaN || math.IsInf` inline.
   Make it call the helper, so `NewLines` and `NewSparkline` drop the
   point.
4. `Range.Add` uses the same rule. `NewLogAxis` checks `r.Min` and
   `r.Max` against it again, because `WithDomain` and a direct caller
   pass a `Range` that `Add` did not build.
5. Add a hard limit to each loop as a second layer, at about four
   lines. `ticksFor` clamps `lowExp` and `highExp` to -308 and 308.
   `NewValueAxis` sets `n` to 0 when the quotient is not finite or is
   above 64. The doc comments of `finiteOr0` and `NewLogAxis` state
   why.

## Options and costs

- A, recommended: drop the value (steps 1 to 4) and add the loop
  limits (step 5). Cost: about 15 lines plus tests. No new moving
  part.
- B: clamp an extreme value to `±1e300` and do not drop it. The point
  stays visible, but the axis prints `1e300` for a value of `1e305`,
  and the tip prints the true value. This is more code in each builder
  and less true output. Not recommended.
- C: add only the loop limits (step 5). This is the smallest change,
  but `NewValueAxis` then returns NaN ends and a chart with no correct
  positions. Not recommended.
- D: remove the log axis (`WithLog`, `NewXYScatter`). This fixes
  only the first finding. The apps use the log axis for right-skewed
  metrics, so the removal moves the work to the apps. Not recommended.

## Tests

Red first. Add table cases to `axis_test.go` for `NewValueAxis` and
`NewLogAxis`. Use `math.SmallestNonzeroFloat64`, `1e-320`,
`math.MaxFloat64`, `-math.MaxFloat64` with `math.MaxFloat64`, and each
mixed with an ordinary value, for each `Format`. Assert finite ends,
`Min < Max`, and at most 9 ticks. Add one builder case each for
`NewValueBars`, `NewValueGroupedBars`, `NewLines` with `WithLog()`,
`NewSparkline`, and `NewXYScatter`. Assert the exact output with the
extreme value left out.

## After the merge

Propose `claudesec resolve` on both findings as fixed, and name the
merge commit.
