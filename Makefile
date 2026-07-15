.PHONY: all build test lint vet clean dev

BINARY := gitx
OUTPUT_DIR := bin

all: build

build:
	@mkdir -p $(OUTPUT_DIR)
	go build -o $(OUTPUT_DIR)/$(BINARY) ./cmd/gitx/

# Live-reload development server
dev:
	air

# Watch with custom args
dev-run:
	air -- gitx commit

test:
	go test -race -count=1 ./...

vet:
	go vet ./...

lint:
	golangci-lint run

clean:
	rm -rf $(OUTPUT_DIR)
