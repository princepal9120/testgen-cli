.PHONY: build test clean install lint run help

# Binary name
BINARY_NAME=testgen

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOVET=$(GOCMD) vet
GOFMT=gofmt

# Build flags
LDFLAGS=-ldflags "-s -w"

## help: Show this help message
help:
	@echo "TestGen - AI-Powered Test Generation CLI"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'

## build: Build the binary
build:
	$(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME) .

## build-all: Build for all platforms
build-all:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-darwin-arm64 .
	GOOS=linux GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-amd64 .
	GOOS=linux GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-linux-arm64 .
	GOOS=windows GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BINARY_NAME)-windows-amd64.exe .

## install: Install the binary
install:
	$(GOCMD) install .

## test: Run tests
test:
	$(GOTEST) -v ./...

## test-coverage: Run tests with coverage
test-coverage:
	$(GOTEST) -v -cover -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

## lint: Run linters
lint:
	$(GOVET) ./...
	$(GOFMT) -l -s .

## fmt: Format code
fmt:
	$(GOFMT) -w -s .

## tidy: Tidy dependencies
tidy:
	$(GOMOD) tidy

## clean: Clean build artifacts
clean:
	rm -f $(BINARY_NAME)
	rm -f $(BINARY_NAME)-*
	rm -f coverage.out coverage.html

## run: Run the CLI
run: build
	./$(BINARY_NAME)

## demo: Run a demo (dry-run on examples)
demo: build
	./$(BINARY_NAME) generate --path=./examples --recursive --dry-run

## analyze: Analyze the examples directory
analyze: build
	./$(BINARY_NAME) analyze --path=./examples --cost-estimate --recursive
