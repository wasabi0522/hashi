.PHONY: all fmt fix fmt-check lint test cover cover-html build generate generate-check clean help

all: fmt-check lint cover generate-check build

fmt:
	go fmt ./...

fix:
	go fix ./...

fmt-check:
	@go fmt ./... > /dev/null
	@go fix ./... > /dev/null
	@if [ -n "$$(git diff --name-only -- '*.go')" ]; then \
		echo "The following files need formatting:"; \
		git diff --name-only -- '*.go'; \
		exit 1; \
	fi

lint:
	golangci-lint run ./...

test:
	go test -race ./...

cover:
	@go test -coverprofile=coverage.raw.out -race ./... 2>&1 | grep -v "^ok\|^?"
	@grep -v '_mock.go' coverage.raw.out > coverage.out
	@echo "=== Per-package coverage (mock excluded) ==="
	@for pkg in cmd internal/config internal/context internal/exec internal/git internal/tmux internal/resource internal/ui; do \
		grep "github.com/wasabi0522/hashi/$$pkg/" coverage.out \
		| awk -F'[ \t]+' '{stmts+=$$2; if($$3>0) covered+=$$2} END {if(stmts>0) printf "  %-30s %4d/%-4d  %.1f%%\n", pkg, covered, stmts, covered/stmts*100}' pkg="$$pkg"; \
	done
	@echo "=== Total ==="
	@go tool cover -func=coverage.out | tail -1
	@total=$$(go tool cover -func=coverage.out | tail -1 | awk '{print $$NF}' | tr -d '%'); \
	threshold=90; \
	if awk "BEGIN{exit(!($$total < $$threshold))}"; then \
		echo "FAIL: total coverage $${total}% is below $${threshold}% threshold"; \
		exit 1; \
	fi

cover-html:
	go test -coverprofile=coverage.raw.out ./...
	grep -v '_mock.go' coverage.raw.out > coverage.out
	go tool cover -html=coverage.out -o coverage.html

build:
	go build -ldflags "-s -w -X github.com/wasabi0522/hashi/cmd.version=$$(git describe --tags --always --dirty 2>/dev/null || echo dev)" -o bin/hashi .

generate:
	PATH="$$(go env GOPATH)/bin:$$PATH" go generate ./internal/...

generate-check: generate
	@if [ -n "$$(git diff --name-only -- '*_mock.go')" ]; then \
		echo "Generated mocks are out of date. Run 'make generate' and commit the changes:"; \
		git diff --name-only -- '*_mock.go'; \
		exit 1; \
	fi

clean:
	rm -rf bin/ coverage.out coverage.raw.out coverage.html

help: ## Show available targets
	@echo "Usage: make [target]"
	@echo ""
	@echo "Targets:"
	@echo "  all              Run fmt-check, lint, cover, generate-check, and build"
	@echo "  fmt              Format Go source files"
	@echo "  fix              Apply Go fix rewrites"
	@echo "  fmt-check        Check formatting (CI)"
	@echo "  lint             Run golangci-lint"
	@echo "  test             Run tests with race detector"
	@echo "  cover            Run tests with coverage report (90% threshold)"
	@echo "  cover-html       Generate HTML coverage report"
	@echo "  build            Build binary to bin/hashi"
	@echo "  generate         Regenerate moq mocks"
	@echo "  generate-check   Verify generated mocks are up to date (CI)"
	@echo "  clean            Remove build artifacts"
	@echo "  help             Show this help"
