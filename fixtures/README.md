# Fixtures

Each subdirectory is a self-contained incident pack that `oncallgraph` can ingest
with no external systems. Packs are used by `oncallgraph demo`, `oncallgraph test`,
and `oncallgraph ask --fixture <pack>`.

## Layout

- `services.yaml`, `owners.yaml`, `dependencies.yaml`
- `changes.yaml`, `alerts.yaml`
- `k8s/deployments.yaml`, `k8s/events.yaml` (optional)
- `runbooks/*.md`
- `meta.yaml` (`now:` fixture clock)
- `expected/*.json` goldens for `oncallgraph test`

Validate a pack: `oncallgraph validate-fixture ./fixtures/incident_checkout`.
