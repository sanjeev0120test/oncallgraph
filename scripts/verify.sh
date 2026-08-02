#!/usr/bin/env bash
# Fast local validation for Unix - keeps things quick.
# Heavy checks (race, 3-OS matrix, cross-compile) run in GitHub Actions instead.
set -euo pipefail
cd "$(dirname "$0")/.."
export CGO_ENABLED=0

echo "==> gofmt"
if [ -n "$(gofmt -l .)" ]; then echo "unformatted files:"; gofmt -l .; exit 1; fi

echo "==> go vet"
go vet ./...

if command -v go >/dev/null 2>&1 && go tool staticcheck -h >/dev/null 2>&1; then
  echo "==> go tool staticcheck (optional)"
  go tool staticcheck ./...
fi

echo "==> go mod tidy (check)"
go mod tidy
git diff --exit-code go.mod go.sum

echo "==> go build"
go build -trimpath -ldflags="-s -w -buildid=" -o bin/opsgraph ./cmd/opsgraph

echo "==> go test (no race)"
go test ./...

if [ -d ./fixtures/incident_checkout ]; then
  echo "==> validate-fixture"
  ./bin/opsgraph validate-fixture ./fixtures/incident_checkout
  echo "==> fixture/golden test"
  ./bin/opsgraph test ./fixtures/incident_checkout
  echo "==> demo"
  ./bin/opsgraph demo --format json >/dev/null
fi

if [ -f ./fixtures/ci_live_k8s/.opsgraph.yaml ]; then
  echo "==> live k8s smoke"
  ./bin/opsgraph ask checkout --config ./fixtures/ci_live_k8s/.opsgraph.yaml --format json >/dev/null
fi

echo "==> graph mermaid stdout"
mermaid_out="$(mktemp)"
mermaid_err="$(mktemp)"
./bin/opsgraph graph --fixture ./fixtures/incident_checkout --format mermaid >"$mermaid_out" 2>"$mermaid_err"
test -s "$mermaid_out"
test ! -s "$mermaid_err"
grep -F flowchart "$mermaid_out" >/dev/null

echo "OK - local validation passed"
