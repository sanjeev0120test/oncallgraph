# Fixtures

Each subdirectory is a self-contained "incident pack" that opsgraph can ingest
with no external systems. Packs are used by `opsgraph demo`, `opsgraph test`,
and `opsgraph ask --fixture <pack>`.

## incident_checkout

A realistic paged incident:

- `checkout` is **degraded** (1/3 replicas ready) after a deploy **22 minutes ago**.
- Its upstream `auth` is **unhealthy** (0/2 ready).
- Alert `CheckoutErrorRateHigh` is **firing**.
- `order` is **downstream** (depends on checkout) and may be impacted.
- `redis` is a dependency with no service row - it is synthesized as `unknown`.
- The checkout runbook has one passing check, one stale check, and one manual step.

`meta.yaml` pins `now` so every evaluation is deterministic.

Files: `meta.yaml`, `services.yaml`, `owners.yaml`, `changes.yaml`,
`dependencies.yaml`, `alerts.yaml`, `k8s/deployments.yaml`, `k8s/events.yaml`,
`runbooks/*.md`, and `expected/*.json` (golden output).
