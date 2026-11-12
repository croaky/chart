# Plans

Each plan is one change. A plan names the security findings it closes
by their finding ID.

No plan is open.

## Threat model

This package is a library. Internal apps on the public internet import
it and render its charts on server pages.

- Attacker: a signed-in user of an app who stores data that the app
  charts. The package cannot assume that each app clamps or checks
  that data.
- Boundary: the exported builders, the axis constructors, and the
  options that a caller passes to them.
- Asset: the CPU, the memory, and the request handlers of the app's
  server. The package holds no secrets and writes no data.
- Consequence that counts: a value that makes a builder panic, loop
  without end, or allocate without limit. A wrong chart for the
  attacker's own data does not count.
- Cost that the package owns: work linear in the data a caller passes.
  The caller owns how much data it passes. A builder must not turn
  linear data into work that grows faster than the data.

A finding needs a named attacker, a consequence, and a boundary. A
severity label alone is not enough.
