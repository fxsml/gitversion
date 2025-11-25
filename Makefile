# gitversion Makefile

# Build configuration
BUILD_OUT_NAME := gitversion
BUILD_OUT_PATH := .
BIN_PATH := $(BUILD_OUT_PATH)/$(BUILD_OUT_NAME)

.PHONY: help
help: ## Show this help message
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build the gitversion binary
	@go build -o $(BIN_PATH)

.PHONY: install
install: ## Install gitversion to $GOPATH/bin
	@go install

.PHONY: test
test: ## Run tests
	@go test -v -race ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage
	@go test -v -cover -race ./...

.PHONY: cover
cover: ## Generate coverage report
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: clean
clean: ## Remove build artifacts and coverage files
	@rm -f $(BUILD_OUT_NAME) coverage.out coverage.html

.PHONY: run
run: build ## Build and run gitversion
	@./$(BUILD_OUT_NAME)

.PHONY: run-detailed
run-detailed: build ## Build and run gitversion with detailed output
	@./$(BUILD_OUT_NAME) -detailed

.PHONY: vet
vet: ## Run go vet
	@go vet ./...

.PHONY: fmt
fmt: ## Format code
	@go fmt ./...

.PHONY: lint
lint: vet ## Run linters
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
	fi

.PHONY: deps
deps: ## Download dependencies
	@go mod download

.PHONY: tidy
tidy: ## Tidy go.mod
	@go mod tidy

.PHONY: all
all: fmt vet test build ## Run fmt, vet, test and build

.DEFAULT_GOAL := help
