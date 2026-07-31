# Usage

`opsgraph` is a single static binary. Build it once:

```bash
go build -o bin/opsgraph ./cmd/opsgraph   # or: make build
```

Most inspection commands accept `--format table|json` (default `table`).
Exceptions: `graph` (`ascii|table|mermaid`), `export` (`json|markdown`), `watch`/`why`/`doctor` (text).

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

# From live sources described in .opsgraph.yaml (local git + k8s snapshot):
opsgraph ask checkout --since 90m
opsgraph ask checkout --format json --ai
```

Flags: `--fixture`, `--config`, `--data-dir`, `--since`, `--runbook` (default true), `--ai`, `--format`.

Exit codes: `0` success, `1` service not found / empty store, `2` usage/config error.

### `opsgraph verify-runbook <service>`
Checks whether a runbook is still valid against current state.

```bash
opsgraph verify-runbook checkout --fixture fixtures/incident_checkout
```

Exit codes: `0` pass, `1` stale/fail/missing, `2` usage/error.

### `opsgraph test <fixture-dir>`
Compares `ask`/`verify` output against the pack's `expected/*.json` goldens.

```bash
opsgraph test ./fixtures/incident_checkout
opsgraph test ./fixtures/incident_checkout --update
```

### `opsgraph status` / `opsgraph doctor` / `opsgraph version`
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

In `.opsgraph.yaml` (see `.opsgraph.example.yaml`; legacy `.oncallgraph.yaml` still accepted):

- `connectors.git` — local repo scan
- `connectors.kubernetes.snapshot` — `deployments.yaml` / `events.yaml` / optional Helm `releases.yaml`
- `connectors.prometheus` / `connectors.alertmanager` — disabled by default

Optional cluster demo: `bash hack/kind-demo.sh`.

## Install

Precompiled binaries are published only in **GitHub Releases** (never committed
to the git source tree). Each `v*` tag builds linux/darwin/windows × amd64/arm64
archives, `SHA256SUMS`, provenance attestation, and install scripts.

Prefer the **attested release copies** of the installers (same tag as the binary):

```bash
# Linux/macOS — pin VERSION to a release tag (example: v0.1.1)
VERSION=v0.1.1
curl -fsSL "https://github.com/sanjeev0120test/opsgraph/releases/download/${VERSION}/install.sh" -o install.sh
chmod +x install.sh
OPSGRAPH_VERSION="$VERSION" ./install.sh

# Windows PowerShell
$Version = "v0.1.1"
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
