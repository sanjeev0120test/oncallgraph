# Architecture

```
              +-------------------+
  fixtures -->|                   |
  local git ->|  ingest (seed +   |---> SQLite store (modernc, pure Go)
  k8s snap  ->|  connectors)      |          |
              +-------------------+          v
                                        ask engine  --> output (table/json)
                                        runbook verify        |
                                        (deterministic)        v
                                                          [optional] ai
                                                     (Ollama + chromem-go RAG,
                                                      offline fallback)
```

## Packages

- `cmd/opsgraph` - cobra CLI: `ask`, `verify-runbook`, `demo`, `test`, `status`, `version`.
- `internal/model` - shared domain types.
- `internal/config` - `.opsgraph.yaml` loader with defaults.
- `internal/store` - pure-Go SQLite persistence (single connection; upserts + queries).
- `internal/ingest` - fixture parsing, k8s snapshot parser, local git connector, live seeding.
- `internal/runbook` - Markdown parsing + deterministic check verification.
- `internal/ask` - assembles the deterministic `AskResult` (timeline, blast radius, recommendations, evidence).
- `internal/output` - deterministic JSON and human table rendering.
- `internal/ai` - optional local summary (Ollama + RAG) with a deterministic offline fallback.
- `fixtures` - embedded incident packs (used by `demo`).

## Determinism

- A **fixture clock** (`meta.yaml: now`) drives all time-based logic in demo/test.
- All output slices are **stably sorted**; JSON is emitted with sorted map keys.
- Golden files are compared **byte-for-byte**; line endings are pinned to LF via `.gitattributes`.
- The `--ai` summary is **excluded** from goldens/tests (non-deterministic).

## Design decisions (see PLAN.md for the living record)

- **Kubernetes v1 = pure-Go snapshot** parser; `client-go` is intentionally not
  linked (guarded by a test). Live cluster support is a future opt-in.
- **AI is optional and local**; a character budget (not a downloaded tokenizer)
  keeps it fully offline.
- **CI-heavy, local-light**: comprehensive validation runs in GitHub Actions;
  local runs use a fast subset.
