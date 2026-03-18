GOLANGCI_LINT := /opt/homebrew/bin/golangci-lint

.PHONY: all build lint fmt vet check clean

all: check build

build:
	go build -o bin/outpost .

lint:
	$(GOLANGCI_LINT) run ./...

fmt:
	$(GOLANGCI_LINT) fmt ./...

vet:
	go vet ./...

check: vet lint

clean:
	rm -rf bin/
