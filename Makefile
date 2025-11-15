.PHONY: help lint lint-fix test build run docker-up docker-down clean

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

lint:
	golangci-lint run ./...

lint-fix:
	golangci-lint run --fix ./...

test:
	go test -v -race -coverprofile=coverage.out ./...

build:
	go build -o bin/app cmd/app/main.go

run:
	go run cmd/app/main.go

docker-up:
	docker-compose up --build

docker-down:
	docker-compose down -v

clean:
	rm -rf bin/
	go clean -cache -modcache -testcache

.DEFAULT_GOAL := help