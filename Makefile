# opsgraph Makefile
# Philosophy: CI does the heavy validation (race, matrix, coverage). Locally,
# use `make quick` (fast) so your machine stays responsive.

BINARY      := opsgraph
PKG         := ./cmd/opsgraph
BIN_DIR     := bin
FIXTURE     := ./fixtures/incident_checkout
ifeq ($(OS),Windows_NT)
EXE := .exe
else
EXE :=
endif
BIN := $(BIN_DIR)/$(BINARY)$(EXE)

VERSION     ?= dev
COMMIT      ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE        ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS     := -s -w \
	-X github.com/sanjeev0120test/opsgraph/internal/version.Version=$(VERSION) \
	-X github.com/sanjeev0120test/opsgraph/internal/version.Commit=$(COMMIT) \
	-X github.com/sanjeev0120test/opsgraph/internal/version.Date=$(DATE)

export CGO_ENABLED := 0

.PHONY: build test fixture-test demo quick lint fmt vet tidy-check race ci cross clean

build: ## Build the static binary
	go build -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

test: ## Run unit tests (no race - fast, local-friendly)
	go test ./...

race: ## Run unit tests with the race detector (heavy - prefer CI)
	CGO_ENABLED=1 go test -race ./...

fixture-test: build ## Run the golden fixture test
	$(BIN) test $(FIXTURE)

demo: build ## Run the built-in incident demo
	$(BIN) demo

fmt: ## Check formatting
	@test -z "$$(gofmt -l .)" || (echo "unformatted files:"; gofmt -l .; exit 1)

vet: ## go vet
	go vet ./...

tidy-check: ## Ensure go.mod/go.sum are tidy
	go mod tidy
	git diff --exit-code go.mod go.sum

quick: fmt vet build test fixture-test demo ## Fast local validation (recommended)

ci: fmt vet tidy-check race build fixture-test demo ## Heavy validation (mirrors GitHub Actions)

cross: ## Cross-compile release binaries (linux/darwin/windows × amd64/arm64)
	@mkdir -p dist
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-amd64   $(PKG)
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-linux-arm64   $(PKG)
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-amd64  $(PKG)
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-darwin-arm64  $(PKG)
	GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-amd64.exe $(PKG)
	GOOS=windows GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o dist/$(BINARY)-windows-arm64.exe $(PKG)

clean: ## Remove build artifacts
	go clean
	-rm -rf $(BIN_DIR) dist cover.out
