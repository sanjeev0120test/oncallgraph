# Usage

`oncallgraph` is a single static binary. Build it once:

```bash
go build -o bin/oncallgraph ./cmd/oncallgraph   # or: make build
```

Most inspection commands accept `--format table|json` (default `table`).
Exceptions: `graph` (`ascii|table|mermaid`), `export` (`json|markdown`), `watch`/`why`/`doctor` (text).

## Core commands

### `oncallgraph demo`
Runs the built-in `incident_checkout` scenario end to end. No config, no network.

```bash
oncallgraph demo
oncallgraph demo --format json
oncallgraph demo --ai            # local AI summary (offline fallback if Ollama absent)
```

### `oncallgraph ask <service>`
Evidence-backed context for a service.

```bash
# From a fixture pack (deterministic, offline):
oncallgraph ask checkout --fixture fixtures/incident_checkout

# From live sources described in .oncallgraph.yaml (local git + k8s snapshot):
oncallgraph ask checkout --since 90m
oncallgraph ask checkout --format json --ai
```

Flags: `--fixture`, `--config`, `--data-dir`, `--since`, `--runbook` (default true), `--ai`, `--format`.

Exit codes: `0` success, `1` service not found / empty store, `2` usage/config error.

### `oncallgraph verify-runbook <service>`
Checks whether a runbook is still valid against current state.

```bash
oncallgraph verify-runbook checkout --fixture fixtures/incident_checkout
```

Exit codes: `0` pass, `1` stale/fail/missing, `2` usage/error.

### `oncallgraph test <fixture-dir>`
Compares `ask`/`verify` output against the pack's `expected/*.json` goldens.

```bash
oncallgraph test ./fixtures/incident_checkout
oncallgraph test ./fixtures/incident_checkout --update
```

### `oncallgraph status` / `oncallgraph doctor` / `oncallgraph version`
Store counts, environment checks, and build metadata.

## Fleet / incident helpers

| Command | Purpose |
|---------|---------|
| `services`, `owners`, `health`, `top` | Inventory and ranking |
| `blast` | 1-hop upstream/downstream |
| `impact` | Recursive downstream impact |
| `changes`, `alerts`, `timeline`, `evidence` | Signal browsers |
| `explain`, `why`, `score`, `fingerprint` | Hypotheses / severity |
| `path`, `graph`, `compare`, `who`, `resolve` | Topology / ownership |
| `report`, `export` | Markdown/JSON handoff |
| `watch` | Poll until healthy (live/persistent sources) |
| `validate-fixture`, `completion` | Pack checks / shell completion |

## Validation

- Heavy validation runs in GitHub Actions (3-OS matrix, race on ubuntu, cross-compile 6 targets, install smoke).
- Locally: `pwsh scripts/verify.ps1` (Windows) or `bash scripts/verify.sh` / `make quick` (Unix).

## Optional live connectors

In `.oncallgraph.yaml` (see `.oncallgraph.example.yaml`; legacy `.opsgraph.yaml` still accepted):

- `connectors.git` — local repo scan
- `connectors.kubernetes.snapshot` — `deployments.yaml` / `events.yaml` / optional Helm `releases.yaml`
- `connectors.prometheus` / `connectors.alertmanager` — disabled by default

Optional cluster demo: `bash hack/kind-demo.sh`.

## Install

```bash
# From a GitHub Release (Linux/macOS; Windows zip supported):
curl -fsSL https://raw.githubusercontent.com/sanjeev0120test/oncallgraph/main/scripts/install.sh | bash

# Windows PowerShell:
irm https://raw.githubusercontent.com/sanjeev0120test/oncallgraph/main/scripts/install.ps1 | iex

# Or: go install github.com/sanjeev0120test/oncallgraph/cmd/oncallgraph@latest
```

## Enabling AI (optional)

1. Install [Ollama](https://ollama.com) and pull models (`qwen2.5:7b`, `nomic-embed-text`).
2. Set `ai.enabled: true` in `.oncallgraph.yaml` (or pass `--ai`).
3. If Ollama is unavailable, `--ai` degrades to a deterministic offline summary.
