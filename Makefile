GO ?= go
BINARY := bin/eventmodeling-hcl
EXAMPLES := $(wildcard examples/*.em.hcl)

.DEFAULT_GOAL := help

.PHONY: help build test test-race vet fmt-check tidy-check staticcheck exhaustive vulncheck validate-examples release-tag-check release-version-check verify clean

help: ## List available commands.
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z0-9_-]+:.*##/ {printf "%-18s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: $(BINARY) ## Build the validator into bin/.

$(BINARY): $(shell find cmd internal -type f -name '*.go') go.mod go.sum
	@mkdir -p $(@D)
	$(GO) build -o $@ ./cmd/eventmodeling-hcl

test: ## Run unit and fixture tests.
	$(GO) test ./...

test-race: ## Run tests with the race detector.
	$(GO) test -race ./...

vet: ## Run go vet.
	$(GO) vet ./...

fmt-check: ## Fail if tracked Go files are not gofmt-formatted.
	@test -z "$$($(GO)fmt -l $$(git ls-files '*.go'))"

tidy-check: ## Fail if go.mod or go.sum need tidying.
	$(GO) mod tidy
	git diff --exit-code -- go.mod go.sum

staticcheck: ## Run Staticcheck (downloads its CI-pinned version if needed).
	$(GO) run honnef.co/go/tools/cmd/staticcheck@2026.1 ./...

exhaustive: ## Check switch-statement exhaustiveness over local enum types.
	$(GO) tool exhaustive -default-signifies-exhaustive=false ./...

vulncheck: ## Run the Go vulnerability checker.
	$(GO) run golang.org/x/vuln/cmd/govulncheck@v1.1.4 ./...

validate-examples: build ## Validate every shipped Event Modeling example.
	@for model in $(EXAMPLES); do \
		$(BINARY) validate "$$model" || exit $$?; \
	done

release-tag-check: ## Test the stable semantic-version tag validator.
	sh scripts/test-require-stable-tag.sh

release-version-check: ## Verify linker-injected release versions are reported by the binary.
	@temporary=$$(mktemp -d); \
	trap 'rm -rf "$$temporary"' EXIT; \
	$(GO) build -ldflags '-X main.version=v0.1.0' -o "$$temporary/eventmodeling-hcl" ./cmd/eventmodeling-hcl; \
	test "$$($$temporary/eventmodeling-hcl version)" = 'eventmodeling-hcl v0.1.0'

verify: fmt-check tidy-check vet test test-race staticcheck exhaustive vulncheck validate-examples release-tag-check release-version-check ## Run the complete local verification suite.

clean: ## Remove locally built artifacts.
	rm -rf bin dist
