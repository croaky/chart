# Plans

Each plan is one change. A plan names the security findings it closes
by their finding ID.

- [001-axis-extreme-values.md](001-axis-extreme-values.md): drop chart
  values that an axis cannot place. It closes
  `find_01JxSA6Fsi2GgKU1oKvtuD2J` and `find_012nS3hRBUKWZm2FggG54pyb`.

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

A finding needs a named attacker, a consequence, and a boundary. A
severity label alone is not enough.
