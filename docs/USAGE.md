# Usage

`opsgraph` is a single static binary. Build it once:

```bash
go build -o bin/opsgraph ./cmd/opsgraph   # or: make build
```

All commands accept `--format table|json` (default `table`).

## Commands

### `opsgraph demo`
Runs the full built-in `incident_checkout` scenario end to end. No config, no network.

```bash
opsgraph demo
opsgraph demo --format json
opsgraph demo --ai            # adds a local AI summary (offline fallback if Ollama absent)
```

### `opsgraph ask <service>`
The main command: evidence-backed context for a service.

```bash
# From a fixture pack (deterministic, offline):
opsgraph ask checkout --fixture fixtures/incident_checkout

# From live sources described in .opsgraph.yaml (local git + k8s snapshot):
opsgraph ask checkout --since 90m
opsgraph ask checkout --format json --ai
```

Flags: `--fixture`, `--config`, `--since <dur>`, `--runbook` (default true), `--ai`, `--format`.

Exit codes: `0` success, `1` service not found, `2` usage/config error.

### `opsgraph verify-runbook <service>`
Checks whether a runbook is still valid against current state.

```bash
opsgraph verify-runbook checkout --fixture fixtures/incident_checkout
```

Exit codes: `0` pass, `1` stale/fail, `2` missing/error.

### `opsgraph test <fixture-dir>`
Ingests a fixture pack and compares `ask`/`verify` output against the pack's
`expected/*.json` golden files. `--update` regenerates them.

```bash
opsgraph test ./fixtures/incident_checkout
opsgraph test ./fixtures/incident_checkout --update
```

### `opsgraph status`
Shows connector configuration and ingested data counts.

```bash
opsgraph status --fixture fixtures/incident_checkout
```

### `opsgraph version`
Prints version/build metadata.

## Validation

- Heavy validation runs in GitHub Actions (3-OS matrix, race, goldens).
- Locally, run the fast subset: `pwsh scripts/verify.ps1` (Windows) or
  `bash scripts/verify.sh` / `make quick` (Unix).

## Enabling AI (optional)

1. Install [Ollama](https://ollama.com) and pull models:
   ```bash
   ollama pull qwen2.5:7b
   ollama pull nomic-embed-text
   ```
2. Set `ai.enabled: true` in `.opsgraph.yaml` (or pass `--ai`).
3. If Ollama is unavailable, `--ai` degrades to a deterministic offline summary.
