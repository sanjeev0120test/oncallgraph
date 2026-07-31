# Fixtures

Each subdirectory is a self-contained incident pack that `opsgraph` can ingest
with no external systems. Packs are used by `opsgraph demo`, `opsgraph test`,
and `opsgraph ask --fixture <pack>`.

## Layout

- `services.yaml`, `owners.yaml`, `dependencies.yaml`
- `changes.yaml`, `alerts.yaml`
- `k8s/deployments.yaml`, `k8s/events.yaml` (optional)
- `runbooks/*.md`
- `meta.yaml` (`now:` fixture clock)
- `expected/*.json` goldens for `opsgraph test`

Validate a pack: `opsgraph validate-fixture ./fixtures/incident_checkout`.
