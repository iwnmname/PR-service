.PHONY: help lint lint-fix test build run docker-up docker-down clean load-test

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

test-e2e:
	@echo "Starting E2E tests..."
	@docker-compose -f docker-compose.e2e.yml up --build -d
	@echo "Waiting for tests to complete..."
	@docker wait reviewer_e2e_tests
	@echo "\n=== E2E Test Results ==="
	@docker logs reviewer_e2e_tests
	@docker-compose -f docker-compose.e2e.yml down -v

load-test:
	@echo "Starting load test..."
	@cd load-test && go run main.go

.DEFAULT_GOAL := help