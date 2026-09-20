.PHONY: all build run test test-race lint docker-build docker-run clean

IMAGE_NAME ?= ghcr.io/ckchessmaster/zitadel-actions-api:local
PORT ?= 8080

all: test build

build:
	CGO_ENABLED=0 go build -v -ldflags="-s -w" -o bin/zitadel-actions-api ./cmd/server

run:
	PORT=$(PORT) LOG_LEVEL=debug go run ./cmd/server

test:
	go test -v ./...

test-race:
	go test -v -race ./...

lint:
	go vet ./...
	@test -z "$$(gofmt -l .)" || (echo "Unformatted files found:" && gofmt -l . && exit 1)

docker-build:
	docker build -t $(IMAGE_NAME) .

docker-run:
	docker run --rm -p $(PORT):8080 $(IMAGE_NAME)

clean:
	rm -rf bin/
