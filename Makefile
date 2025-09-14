# seqr - Hackathon build

BINARY_NAME=seqr

.PHONY: all build test clean dev

all: build

build:
	go build -o $(BINARY_NAME) ./cmd/seqr

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME)

dev:
	go run ./cmd/seqr