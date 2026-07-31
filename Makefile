# opsgraph Makefile
# Philosophy: CI does the heavy validation (race, matrix, coverage). Locally,
# use `make quick` (fast) so your machine stays responsive.

BINARY      := opsgraph
PKG         := ./cmd/opsgraph
BIN_DIR     := bin
FIXTURE     := ./fixtures/incident_checkout

VERSION     ?= dev
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
	-X github.com/opsgraph/opsgraph/internal/version.Version=$(VERSION) \
	-X github.com/opsgraph/opsgraph/internal/version.Commit=$(COMMIT) \
	-X github.com/opsgraph/opsgraph/internal/version.Date=$(DATE)

export CGO_ENABLED := 0

.PHONY: build test fixture-test demo quick lint fmt vet tidy-check race ci cross clean

build: ## Build the static binary
	go build -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$(BINARY) $(PKG)

test: ## Run unit tests (no race - fast, local-friendly)
	go test ./...

race: ## Run unit tests with the race detector (heavy - prefer CI)
	CGO_ENABLED=1 go test -race ./...

fixture-test: build ## Run the golden fixture test
	$(BIN_DIR)/$(BINARY) test $(FIXTURE)

demo: build ## Run the built-in incident demo
	$(BIN_DIR)/$(BINARY) demo

fmt: ## Check formatting
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

vet: ## go vet
	go vet ./...

tidy-check: ## Ensure go.mod/go.sum are tidy
	go mod tidy
	git diff --exit-code go.mod go.sum

quick: fmt vet build test fixture-test demo ## Fast local validation (recommended)

ci: fmt vet tidy-check race build fixture-test demo ## Heavy validation (mirrors GitHub Actions)

cross: ## Cross-compile release binaries
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64   $(PKG)
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64  $(PKG)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64  $(PKG)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe $(PKG)

clean: ## Remove build artifacts
	rm -rf $(BIN_DIR) dist cover.out
