# Makefile for VPN Config Scanner

.PHONY: test test-verbose test-coverage build run-scanner clean help

# Default target
all: test build

# Run tests
test:
	@echo "🧪 Running tests..."
	go test -v ./...

# Run tests with verbose output
test-verbose:
	@echo "🧪 Running tests with verbose output..."
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "📊 Coverage report generated: coverage.html"

# Build the scanner
build:
	@echo "🔨 Building scanner..."
	go build -o scanner scanner_main.go scanner.go

# Run the scanner on test data
run-scanner:
	@echo "🚀 Running scanner on test data..."
	go run scanner_main.go -dir=test_data -timeout=2s

# Run the scanner on current directory
run-scanner-current:
	@echo "🚀 Running scanner on current directory..."
	go run scanner_main.go -dir=. -timeout=2s

# Clean generated files
clean:
	@echo "🧹 Cleaning up..."
	rm -f scanner
	rm -f coverage.out coverage.html
	rm -f *.txt
	rm -f fast_*.txt

# Show help
help:
	@echo "Available targets:"
	@echo "  test              - Run tests"
	@echo "  test-verbose      - Run tests with verbose output and race detection"
	@echo "  test-coverage     - Run tests with coverage report"
	@echo "  build             - Build the scanner binary"
	@echo "  run-scanner       - Run scanner on test data"
	@echo "  run-scanner-current - Run scanner on current directory"
	@echo "  clean             - Clean generated files"
	@echo "  help              - Show this help message"
