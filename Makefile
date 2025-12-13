.PHONY: all build test lint clean install

# Default target
all: lint test build install

# Build the binary
build:
	go build -o resume-tailor .

# Run tests
test:
	go test -v ./...

# Run golangci-lint with namedreturns
lint:
	@echo "Running namedreturns linter..."
	namedreturns ./...
	@echo "Running golangci-lint..."
	golangci-lint run

# Clean build artifacts
clean:
	rm -f resume-tailor
	go clean

# Install the binary to $GOPATH/bin
install:
	go install .

# Download dependencies
deps:
	go mod download
	go mod tidy

# Run the tool (example)
run:
	go run . --help
