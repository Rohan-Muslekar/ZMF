# Makefile for ZMultiField

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
BINARY_NAME=zmultifield
GOFILES=$(wildcard *.go)

# Build flags
LDFLAGS=-ldflags "-s -w"

.PHONY: all build clean test coverage lint tidy download update help bench

all: test build

build: 
	$(GOBUILD) -o $(BINARY_NAME) -v

clean: 
	$(GOCLEAN)
	rm -f $(BINARY_NAME)
	rm -f coverage.out

test: 
	$(GOTEST) -v ./...

coverage: 
	$(GOTEST) -coverprofile=coverage.out ./...
	$(GOCMD) tool cover -html=coverage.out

lint:
	@if command -v golangci-lint > /dev/null; then \
		golangci-lint run ./...; \
	else \
		echo "golangci-lint not installed"; \
		exit 1; \
	fi

tidy:
	$(GOMOD) tidy

download:
	$(GOMOD) download

update:
	$(GOMOD) download all

bench:
	$(GOTEST) -bench=. -benchmem ./...

# Docker targets
docker-build:
	docker build -t $(BINARY_NAME) .

docker-run:
	docker run --rm $(BINARY_NAME)

# Example to build for multiple platforms
# Adjust PLATFORMS as needed
PLATFORMS=linux/amd64 darwin/amd64 windows/amd64
$(PLATFORMS):
	GOOS=$(word 1,$(subst /, ,$@)) GOARCH=$(word 2,$(subst /, ,$@)) $(GOBUILD) -o $(BINARY_NAME)-$(word 1,$(subst /, ,$@))-$(word 2,$(subst /, ,$@)) -v

release: $(PLATFORMS)

help:
	@echo "Make targets:"
	@echo "  all        - Run tests and build binary"
	@echo "  build      - Build binary"
	@echo "  clean      - Clean build artifacts"
	@echo "  test       - Run tests"
	@echo "  coverage   - Generate test coverage report"
	@echo "  lint       - Run linter"
	@echo "  tidy       - Run go mod tidy"
	@echo "  download   - Download dependencies"
	@echo "  update     - Update dependencies"
	@echo "  bench      - Run benchmarks"
	@echo "  docker-build - Build Docker image"
	@echo "  docker-run   - Run Docker container"
	@echo "  release    - Build for multiple platforms"
	@echo "  help       - Show this help"
