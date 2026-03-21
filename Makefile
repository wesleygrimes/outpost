GOLANGCI_LINT ?= $(shell bash -c 'for p in $$(type -aP golangci-lint); do if "$$p" version 2>&1 | grep -q "^golangci-lint has version [2-9]"; then echo "$$p"; break; fi; done')
GORELEASER   ?= $(shell command -v goreleaser 2>/dev/null)
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS      := -ldflags "-s -w -X main.version=$(VERSION) -X github.com/wesleygrimes/outpost/internal/grpcserver.Version=$(VERSION)"

.PHONY: all build build-linux build-cross release proto lint fmt vet test check ci clean setup \
       wt-new wt-list wt-remove wt-prune

all: check build

setup:
	brew install golangci-lint goreleaser bufbuild/buf/buf

build:
	go build $(LDFLAGS) -o bin/outpost .

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/outpost-linux-amd64 .

build-cross:
	GOOS=linux  GOARCH=amd64 go build $(LDFLAGS) -o bin/outpost-linux-amd64 .
	GOOS=linux  GOARCH=arm64 go build $(LDFLAGS) -o bin/outpost-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/outpost-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/outpost-darwin-arm64 .

release:
	@if [ -z "$(GITEA_TOKEN)" ]; then echo "GITEA_TOKEN required"; exit 1; fi
	@if [ -z "$(GORELEASER)" ]; then echo "goreleaser not found; brew install goreleaser/tap/goreleaser"; exit 1; fi
	@set -e; \
	LATEST=$$(git tag -l 'v*' --sort=-v:refname | head -1); \
	if [ -z "$$LATEST" ]; then NEXT="v0.1.0"; \
	else \
		MAJOR=$$(echo $$LATEST | cut -d. -f1); \
		MINOR=$$(echo $$LATEST | cut -d. -f2); \
		PATCH=$$(echo $$LATEST | cut -d. -f3); \
		NEXT="$$MAJOR.$$MINOR.$$((PATCH + 1))"; \
	fi; \
	SEMVER=$${NEXT#v}; \
	echo "==> Stamping plugin version $$SEMVER"; \
	sed "s/\"version\": \"[^\"]*\"/\"version\": \"$$SEMVER\"/" \
		.claude-plugin/marketplace.json > .claude-plugin/marketplace.json.tmp \
		&& mv .claude-plugin/marketplace.json.tmp .claude-plugin/marketplace.json; \
	git add .claude-plugin/marketplace.json; \
	git diff --cached --quiet || git commit -m "chore: bump plugin version to $$SEMVER"; \
	echo "==> Tagging $$NEXT"; \
	git tag -a -m "Release $$NEXT" "$$NEXT"; \
	git push origin main "$$NEXT"; \
	echo "==> Running GoReleaser"; \
	$(GORELEASER) release --clean

proto:
	cd proto && buf lint && buf generate

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	$(GOLANGCI_LINT) fmt ./...

vet:
	go vet ./...

test:
	go test -race -count=1 ./...

check: vet lint test

ci: check
	goreleaser check

clean:
	rm -rf bin/ dist/

# --- Worktrees -----------------------------------------------------------
# Convention: worktrees live at ../outpost-<name> on branch <name>.
# Usage:
#   make wt-new name=my-feature
#   make wt-list
#   make wt-remove name=my-feature
#   make wt-prune

REPO_PARENT := $(shell cd .. && pwd)
WT_DIR       = $(REPO_PARENT)/outpost-$(name)

wt-new:
	@if [ -z "$(name)" ]; then echo "usage: make wt-new name=<branch>"; exit 1; fi
	git worktree add "$(WT_DIR)" -b "$(name)"
	@echo "Worktree ready: $(WT_DIR)"

wt-list:
	git worktree list

wt-remove:
	@if [ -z "$(name)" ]; then echo "usage: make wt-remove name=<branch>"; exit 1; fi
	git worktree remove "$(WT_DIR)"
	@echo "Removed worktree: $(WT_DIR)"

wt-prune:
	git worktree prune -v
