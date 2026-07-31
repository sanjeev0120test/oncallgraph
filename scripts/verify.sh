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

echo "==> go build"
go build -o bin/opsgraph ./cmd/opsgraph

echo "==> go test (no race)"
go test ./...

if [ -d ./fixtures/incident_checkout ]; then
  echo "==> fixture/golden test"
  ./bin/opsgraph test ./fixtures/incident_checkout
  echo "==> demo"
  ./bin/opsgraph demo --format json >/dev/null
fi

echo "OK - local validation passed"
