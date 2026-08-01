# Usage

`opsgraph` is a single static binary. Build it once:

```bash
go build -o bin/opsgraph ./cmd/opsgraph   # or: make build
```

Most inspection commands accept `--format table|json` (default `table`).
Exceptions: `graph` (`ascii|table|mermaid`), `export` (`json|markdown`), `watch`/`why`/`handoff`/`doctor` (text).

## Core commands

### `opsgraph demo`
Runs the built-in `incident_checkout` scenario end to end. No config, no network.

```bash
opsgraph demo
opsgraph demo --format json
opsgraph demo --ai            # local AI summary (offline fallback if Ollama absent)
```

### `opsgraph ask <service>`
Evidence-backed context for a service.

```bash
# From a fixture pack (deterministic, offline):
opsgraph ask checkout --fixture fixtures/incident_checkout

# From a persistent store (explicit; no live re-scrape):
opsgraph ask checkout --data-dir .opsgraph/data

# Live k8s/prom/AM connectors from .opsgraph.yaml (preferred over stale state.db):
opsgraph ask checkout --since 90m
opsgraph ask checkout --format json --ai
```

Flags: `--fixture`, `--config`, `--data-dir`, `--since`, `--runbook` (default true), `--ai`, `--format`.

Notes:
- Active (`firing`/`pending`) alerts are always shown; `--since` filters resolved/historical alerts only.
- Changes use at least a 60m lookback (covers the 30m prime-suspect window and change→alert correlation) even when `--since` is narrower. The reported window shows both when they differ (e.g. `10m (changes 60m)`).
- If kubernetes / Prometheus / Alertmanager are enabled in `.opsgraph.yaml`, `ask`/`status`/`why`/`watch` re-scrape them (a prior `state.db` alone will not freeze the view). Git alone does not displace a populated store. Use `--data-dir` to force the persistent store.

Exit codes: `0` success, `1` service not found / empty store, `2` usage/config error.

### `opsgraph ingest`
Load a fixture pack (or live config sources) into a persistent data dir.

```bash
opsgraph ingest --fixture fixtures/incident_checkout --data-dir .opsgraph/data
opsgraph ingest --since 2h          # live connectors; change lookback (default: config default_since)
opsgraph ask checkout --data-dir .opsgraph/data
```

### `opsgraph verify-runbook <service>`
Checks whether a runbook is still valid against current state.

```bash
opsgraph verify-runbook checkout --fixture fixtures/incident_checkout
```

Exit codes: `0` pass, `1` stale/fail/missing, `2` usage/error.

### `opsgraph handoff <service>`
Short evidence-backed handoff note (health, score, fingerprint, linked alerts, next steps).

```bash
opsgraph handoff checkout --fixture fixtures/incident_checkout
```

### `opsgraph test <fixture-dir>`
Compares `ask`/`verify` output against the pack's `expected/*.json` goldens.

```bash
opsgraph test ./fixtures/incident_checkout
opsgraph test ./fixtures/incident_checkout --update
```

### `opsgraph status` / `opsgraph doctor` / `opsgraph version`
Store counts, environment checks, and build metadata. `status` uses the same source selection as `ask`. `doctor` verifies git repo and (when enabled) the kubernetes snapshot path.

## Fleet / incident helpers

| Command | Purpose |
|---------|---------|
| `services`, `owners`, `health`, `top` | Inventory and ranking |
| `blast` | 1-hop upstream/downstream |
| `impact` | Recursive downstream impact |
| `changes`, `alerts`, `timeline`, `evidence` | Signal browsers |
| `explain`, `why`, `score`, `fingerprint` | Hypotheses / severity |
| `path`, `graph`, `compare`, `who`, `resolve` | Topology / ownership |
| `report`, `export`, `handoff` | Markdown/JSON/text handoff |
| `watch` | Poll until healthy (live/persistent sources) |
| `validate-fixture`, `completion` | Pack checks / shell completion |

## Validation

- Heavy validation runs in GitHub Actions (3-OS matrix, race on ubuntu, cross-compile 6 targets, install smoke).
- Locally: `pwsh scripts/verify.ps1` (Windows) or `bash scripts/verify.sh` / `make quick` (Unix).

## Optional live connectors

In `.opsgraph.yaml` (see `.opsgraph.example.yaml`):

- `connectors.git` — local repo scan
- `connectors.kubernetes.snapshot` — `deployments.yaml` / `events.yaml` / optional Helm `releases.yaml`
- `connectors.prometheus` / `connectors.alertmanager` — disabled by default

Optional cluster demo: `bash hack/kind-demo.sh`.

## Install

Precompiled binaries are published only in **GitHub Releases** (never committed
to the git source tree). Pushing a `v*` tag automatically runs the release
workflow: build 6 targets, package archives, write `SHA256SUMS`, attest
provenance, and publish the Release.

Prefer the **attested release copies** of the installers (same tag as the binary):

```bash
# Linux/macOS — pin VERSION to a release tag
VERSION=v0.1.8
curl -fsSL "https://github.com/sanjeev0120test/opsgraph/releases/download/${VERSION}/install.sh" -o install.sh
chmod +x install.sh
OPSGRAPH_VERSION="$VERSION" ./install.sh

# Windows PowerShell
$Version = "v0.1.8"
Invoke-WebRequest "https://github.com/sanjeev0120test/opsgraph/releases/download/$Version/install.ps1" -OutFile install.ps1
$env:OPSGRAPH_VERSION = $Version
./install.ps1

# From source (no release binary):
go install github.com/sanjeev0120test/opsgraph/cmd/opsgraph@latest
```

Release page: https://github.com/sanjeev0120test/opsgraph/releases

## Enabling AI (optional)

1. Install [Ollama](https://ollama.com) and pull models (`qwen2.5:7b`, `nomic-embed-text`).
2. Set `ai.enabled: true` in `.opsgraph.yaml` (or pass `--ai`).
3. If Ollama is unavailable, `--ai` degrades to a deterministic offline summary.
