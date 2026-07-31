# Architecture

`oncallgraph` is a single static Go binary. Deterministic incident context is the core path; optional local AI enrichment never affects goldens/CI.

## Packages

- `cmd/oncallgraph` — cobra CLI (`ask`, fleet helpers, `demo`, `test`, `status`, …).
- `internal/model` — shared domain types (`AskResult`, services, alerts, evidence).
- `internal/config` — `.oncallgraph.yaml` loader (legacy `.opsgraph.yaml` accepted) with defaults.
- `internal/store` — pure-Go SQLite (`modernc.org/sqlite`), `PRAGMA user_version` gated.
- `internal/ingest` — fixtures, git, k8s snapshot, optional Prometheus/Alertmanager/Helm.
- `internal/ask` — blast radius, timeline, recommendations R1–R6.
- `internal/runbook` — Markdown parse + check catalog (`oncallgraph:check=` / legacy `opsgraph:check=`).
- `internal/ai` — optional Ollama + chromem-go RAG; stubbed in tests.
- `internal/{score,explain,report,graphviz,pathfind,impact,fingerprint}` — enterprise helpers.
- `fixtures/` — embedded `incident_checkout` pack for `demo`.

## Data flow

1. Resolve source: `--fixture` (ephemeral), `--data-dir` / auto persistent store, or live config connectors.
2. Upsert entities into SQLite.
3. `ask` assembles owner, changes, alerts, 1-hop blast, runbook verify, timeline, recommendations, evidence.
4. Render table or JSON via `internal/output` (`SetEscapeHTML(false)`, indent, trailing newline).

## Cross-platform

`CGO_ENABLED=0`, `filepath` for OS paths, `fs.FS` + forward slashes for fixtures, LF via `.gitattributes`. CI covers ubuntu/macOS/windows plus linux/darwin/windows × amd64/arm64 cross-builds.
