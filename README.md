# opsgraph

**opsgraph** is a free, offline-first command-line tool that gives an on-call engineer evidence-backed incident context in seconds: what changed, what's affected, what depends on it, who owns it, and whether the runbook is still valid.

- **Who it's for:** on-call SREs / DevOps / backend engineers who just got paged.
- **The problem:** when an incident starts, context is scattered across git history, Kubernetes, alerts, and runbooks. You waste minutes hunting for it.
- **What it does:** one command returns a clear timeline + blast radius + runbook validity, every claim backed by an evidence ID.

```
opsgraph ask checkout
```

It runs from a single static Go binary with **no accounts, no API tokens, no paid services**. The core works entirely from checked-in fixtures, and optional AI runs 100% locally via [Ollama](https://ollama.com).

## Quickstart

```bash
# See a full simulated incident, end to end:
go run ./cmd/opsgraph demo

# Run the golden fixture tests (deterministic, offline):
go run ./cmd/opsgraph test ./fixtures/incident_checkout

# Ask about a service from a fixture pack:
go run ./cmd/opsgraph ask checkout --fixture fixtures/incident_checkout
```

Build a binary:

```bash
go build -o bin/opsgraph ./cmd/opsgraph      # or: make build
```

## Example (abbreviated)

```
SERVICE   checkout (degraded)   owner: Payments Team <payments@example.com>
CHANGES   deploy 22m ago  rev abc123  "bump checkout to v1.4.2"        [ev-change-1]
ALERTS    CheckoutErrorRateHigh (critical, firing)                     [ev-alert-1]
BLAST     upstream: auth (unhealthy), redis (unknown)   downstream: order
RUNBOOK   runbooks/checkout.md  -> STALE (1 pass, 1 stale, 1 manual)
NEXT      1. Inspect recent change/deploy abc123 first.
          2. Check upstream service auth health before changing checkout.
          ...
```

## Free stack

| Concern      | Choice                                          | Cost |
|--------------|-------------------------------------------------|------|
| CLI          | cobra                                           | free |
| Storage      | modernc.org/sqlite (pure Go, no CGO)            | free |
| Git          | go-git                                          | free |
| Kubernetes   | checked-in YAML snapshot (pure Go)              | free |
| AI (optional)| langchaingo + chromem-go + local Ollama         | free |
| CI           | GitHub Actions (no secrets)                     | free |

No cloud LLM, no SaaS, no MCP. See [docs/FREE_STACK.md](docs/FREE_STACK.md).

## How CI validates it

GitHub Actions runs the heavy validation on every push/PR with **zero secrets**: gofmt, `go vet`, `go mod tidy` check, unit tests (`-race` on Linux/macOS), a 3-OS build matrix, the golden fixture test, and the demo. Locally you can run the fast subset:

```bash
pwsh scripts/verify.ps1     # Windows
bash scripts/verify.sh      # Unix   (or: make quick)
```

## Configuration

`opsgraph` needs no config to run the demo or tests. For real use, copy [`.opsgraph.example.yaml`](.opsgraph.example.yaml) to `.opsgraph.yaml` and describe your services, owners, dependencies, and runbooks. See [docs/USAGE.md](docs/USAGE.md).

## Runbook annotations

Runbooks are plain Markdown with YAML front matter and step checks like:

```
1. Confirm a recent deploy is the likely cause.
<!-- opsgraph:check=deploy_age_lt:60m -->
```

`opsgraph verify-runbook checkout` evaluates each check against current state and reports `pass` / `fail` / `stale` / `manual`. See [docs/RUNBOOK_FORMAT.md](docs/RUNBOOK_FORMAT.md).

## Roadmap

- Phase-2 (all still free/local): Prometheus + Alertmanager connectors (mock-tested), Helm release enrichment, a scripted `kind` demo.
- Future opt-in: live Kubernetes connector via `client-go` behind a build tag.

## License

MIT - see [LICENSE](LICENSE).
