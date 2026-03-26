MODULE := github.com/shakestzd/htmlgraph
VERSION ?= dev
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test vet lint clean tidy

build:
	go build $(LDFLAGS) -o bin/htmlgraph ./cmd/htmlgraph

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

lint: vet
	@echo "lint passed (go vet)"

tidy:
	go mod tidy

clean:
	rm -rf bin/

all: tidy vet test build
