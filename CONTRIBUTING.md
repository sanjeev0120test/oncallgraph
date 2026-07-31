# Contributing

Thanks for your interest in opsgraph!

## Prerequisites

- Go 1.25+
- (optional) [Ollama](https://ollama.com) for the `--ai` path

## Local workflow (fast, laptop-friendly)

Heavy validation runs in GitHub Actions. Locally, use the fast subset:

```bash
pwsh scripts/verify.ps1     # Windows
bash scripts/verify.sh      # Unix
make quick                  # Unix (make)
```

This runs gofmt, `go vet`, build, `go test` (no race), the fixture/golden test,
and the demo.

## Before opening a PR

- Keep the engine **deterministic**: golden output must be byte-identical across
  runs and OSes. If you intentionally change output, regenerate goldens:
  ```bash
  go run ./cmd/opsgraph test ./fixtures/incident_checkout --update
  ```
- Do **not** add `k8s.io/*` to the default build (a test enforces this).
- Run `go mod tidy` and commit the result.
- Add tests for new behavior. Avoid network in tests (the AI path is stub/fallback tested).

## Commit style

Small, focused commits with simple, lowercase, imperative messages
(e.g. `add prometheus connector`).

## Code layout

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md).
