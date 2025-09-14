BINARY_NAME=seqr

.PHONY: all build test clean dev install

all: build

build:
	go build -o $(BINARY_NAME) ./cmd/seqr

install: build
	sudo mv $(BINARY_NAME) /usr/local/bin/

test:
	go test ./...

clean:
	rm -f $(BINARY_NAME)

dev:
	go run ./cmd/seqr