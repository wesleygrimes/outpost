GOLANGCI_LINT ?= $(shell bash -c 'for p in $$(type -aP golangci-lint); do if "$$p" version 2>&1 | grep -q "^golangci-lint has version [2-9]"; then echo "$$p"; break; fi; done')
VERSION      ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS      := -ldflags "-s -w -X main.version=$(VERSION) -X github.com/wesgrimes/outpost/internal/grpcserver.Version=$(VERSION)"

.PHONY: all build build-linux build-release release proto lint fmt vet test check clean

all: check build

build:
	go build $(LDFLAGS) -o bin/outpost .

build-linux:
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o bin/outpost-linux-amd64 .

build-release:
	GOOS=linux  GOARCH=amd64 go build $(LDFLAGS) -o bin/outpost-linux-amd64 .
	GOOS=linux  GOARCH=arm64 go build $(LDFLAGS) -o bin/outpost-linux-arm64 .
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o bin/outpost-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o bin/outpost-darwin-arm64 .

release: build-release
	@if [ -z "$(GITEA_TOKEN)" ]; then echo "GITEA_TOKEN required"; exit 1; fi
	@echo "Uploading $(VERSION) to Gitea..."
	@for f in bin/outpost-*; do \
		echo "  $$f"; \
		curl -sS -X POST \
			-H "Authorization: token $(GITEA_TOKEN)" \
			-F "attachment=@$$f" \
			"https://git.grimes.pro/api/v1/repos/wesleygrimes/outpost/releases/$(VERSION)/assets"; \
	done

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

clean:
	rm -rf bin/
