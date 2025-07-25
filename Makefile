# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GORUN=$(GOCMD) run

# Binary name
BINARY_NAME=k8s-resource-adjuster

.PHONY: all build run test clean deps get-repos

all: build

# Build the main application
build:
	@echo "Building the application..."
	$(GOBUILD) -o $(BINARY_NAME) ./cmd/main.go

# Run the tests
test:
	@echo "Running tests..."
	$(GOTEST) -v ./...

# Clean up build artifacts
clean:
	@echo "Cleaning up..."
	$(GOCLEAN)
	rm -f $(BINARY_NAME)

# Run the main application
run:
	@echo "Running the application..."
	$(GORUN) ./cmd/main.go

# Tidy and download dependencies
deps:
	@echo "Tidying and downloading dependencies..."
	go mod tidy

# Run the script to get GitLab repositories
get-repos: deps
	@echo "Fetching GitLab repositories..."
	$(GORUN) ./scripts/get_gitlab_repos.go

help:
	@echo "Available commands:"
	@echo "  build      - Build the main application"
	@echo "  test       - Run all tests"
	@echo "  clean      - Clean up build artifacts"
	@echo "  run        - Run the main application"
	@echo "  deps       - Install dependencies"
	@echo "  get-repos  - Fetch GitLab repositories and update .env file"
