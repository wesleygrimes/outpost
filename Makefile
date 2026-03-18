GOLANGCI_LINT := /opt/homebrew/bin/golangci-lint

.PHONY: all build build-linux proto lint fmt vet check clean

all: check build

build:
	go build -o bin/outpost .

build-linux:
	GOOS=linux GOARCH=amd64 go build -o bin/outpost-linux .

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
