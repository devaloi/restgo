.PHONY: build test lint run clean cover docker-up docker-down migrate

BINARY=restgo
BUILD_DIR=bin

build:
	go build -o $(BUILD_DIR)/$(BINARY) ./cmd/restgo

test:
	go test ./... -race -count=1

lint:
	golangci-lint run ./...

run: build
	./$(BUILD_DIR)/$(BINARY)

clean:
	rm -rf $(BUILD_DIR)
	go clean -testcache

cover:
	go test ./... -race -coverprofile=coverage.out
	go tool cover -html=coverage.out -o coverage.html

docker-up:
	docker compose up -d

docker-down:
	docker compose down -v

migrate:
	go run ./cmd/restgo -migrate
