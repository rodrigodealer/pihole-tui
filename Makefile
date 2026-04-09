BINARY := pihole-tui
MODULE := github.com/rodrigodealer/pihole-tui
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-s -w -X main.version=$(VERSION)"

.PHONY: build run test lint fmt vet clean install tidy check

build:
	go build $(LDFLAGS) -o bin/$(BINARY) ./cmd/$(BINARY)

run: build
	./bin/$(BINARY)

install:
	go install $(LDFLAGS) ./cmd/$(BINARY)

test:
	go test -race -cover ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

check: fmt vet lint test

clean:
	rm -rf bin/
