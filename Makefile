# .PHONY ensures that the target runs even if a file with the same name exists.
.PHONY: all build run test clean

# Default target: runs 'make build' when you just type 'make'.
all: build

# Build the application and place the executable 'myapp' in the 'build/' directory.
# The 'go build' command automatically creates the 'build' directory if it doesn't exist.
build:
	@echo "Building executable..."
	@go build -o build/xnetperf ./cmd/xnetperf.go
	@echo "Executable created: build/xnetperf"

# Build and run the application directly for quick local testing.
run:
	@go run ./cmd/xnetperf.go

# Run all tests in the project.
test:
	@go test ./...

# Remove the entire build directory and all its contents.
clean:
	@echo "Cleaning up build artifacts..."
	@rm -rf build