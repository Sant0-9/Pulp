.PHONY: build install clean test lint release run

VERSION ?= dev
COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.date=$(DATE)

build:
	go build -ldflags "$(LDFLAGS)" -o bin/pulp ./cmd/pulp

run: build
	./bin/pulp

install: build
	cp bin/pulp /usr/local/bin/
	mkdir -p ~/.local/share/pulp/python
	cp -r python/* ~/.local/share/pulp/python/

clean:
	rm -rf bin/
	rm -rf dist/

test:
	go test ./...

lint:
	golangci-lint run

release:
	goreleaser release --snapshot --clean

release-dry:
	goreleaser release --snapshot --clean --skip=publish
