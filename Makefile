# Variables
BINARY_NAME=xnetperf
BINARY_PATH=build/$(BINARY_NAME)
GOBUILD=go build
GOCLEAN=go clean
CMD_DIR=.

# .PHONY ensures that the target runs even if a file with the same name exists.
.PHONY: all build build-static build-dynamic build-portable run test clean help

# Default target: runs 'make build-static' for better compatibility across different GLIBC versions
all: build-static

# Build statically linked binary (default, recommended for cross-version compatibility)
# CGO_ENABLED=0: Disables CGO to allow full static linking
# -a: Force rebuilding of packages
# -ldflags: Linker flags
#   -s: Strip symbol table (reduces size)
#   -w: Strip DWARF debug info (reduces size)
#   -extldflags '-static': Force static linking (no GLIBC dependency)
build: build-static

build-static:
	@echo "🔨 Building statically linked executable (no GLIBC dependency)..."
	@mkdir -p build
	@CGO_ENABLED=0 GOOS=linux $(GOBUILD) -a -ldflags="-s -w -extldflags '-static'" -o $(BINARY_PATH) $(CMD_DIR)
	@echo "✅ Static executable created: $(BINARY_PATH)"
	@echo "📊 Verifying static build..."
	@if command -v ldd > /dev/null 2>&1; then \
		if ldd $(BINARY_PATH) 2>&1 | grep -q "not a dynamic executable"; then \
			echo "✅ Successfully built static binary (no dynamic dependencies)"; \
		else \
			echo "⚠️  Warning: Binary may have dynamic dependencies:"; \
			ldd $(BINARY_PATH) 2>&1 || true; \
		fi \
	else \
		file $(BINARY_PATH); \
	fi
	@du -h $(BINARY_PATH)

# Build dynamically linked binary (may have GLIBC version issues)
build-dynamic:
	@echo "🔨 Building dynamically linked executable..."
	@mkdir -p build
	@$(GOBUILD) -o $(BINARY_PATH) $(CMD_DIR)
	@echo "✅ Dynamic executable created: $(BINARY_PATH)"
	@if command -v ldd > /dev/null 2>&1; then \
		echo "📊 Dynamic dependencies:"; \
		ldd $(BINARY_PATH) || true; \
	fi
	@du -h $(BINARY_PATH)

# Build portable binary (CGO disabled, but not necessarily static)
build-portable:
	@echo "🔨 Building portable executable (CGO disabled)..."
	@mkdir -p build
	@CGO_ENABLED=0 $(GOBUILD) -o $(BINARY_PATH) $(CMD_DIR)
	@echo "✅ Portable executable created: $(BINARY_PATH)"
	@du -h $(BINARY_PATH)

# Build and run the application directly for quick local testing (development mode)
run:
	@echo "🚀 Running in development mode..."
	@go run $(CMD_DIR)

# Run all tests in the project
test:
	@echo "🧪 Running all tests..."
	@go test ./...

# Run tests with verbose output
test-verbose:
	@echo "🧪 Running all tests (verbose)..."
	@go test -v ./...

# Run tests with coverage
test-coverage:
	@echo "🧪 Running tests with coverage..."
	@go test -cover ./...

# Remove the entire build directory and all its contents
clean:
	@echo "🧹 Cleaning up build artifacts..."
	@$(GOCLEAN)
	@rm -rf $(BINARY_PATH)
	@rm -rf build/$(BINARY_PATH)
	@echo "✅ Clean complete"

# Display help information
help:
	@echo "Available targets:"
	@echo "  make                 - Build static binary (default)"
	@echo "  make build           - Build static binary (alias)"
	@echo "  make build-static    - Build statically linked binary (recommended)"
	@echo "  make build-dynamic   - Build dynamically linked binary"
	@echo "  make build-portable  - Build portable binary (CGO disabled)"
	@echo "  make run             - Run in development mode"
	@echo "  make test            - Run all tests"
	@echo "  make test-verbose    - Run tests with verbose output"
	@echo "  make test-coverage   - Run tests with coverage"
	@echo "  make clean           - Clean build artifacts"
	@echo "  make help            - Display this help message"
	@echo ""
	@echo "Recommended for production: make build-static"
	@echo "This creates a fully static binary with no GLIBC dependencies"