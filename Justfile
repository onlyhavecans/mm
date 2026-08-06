# List available recipes.
default:
    @just --list

# Build local binaries into ./bin/.
build:
    mkdir -p bin
    go build -trimpath -o bin/ ./...

# Run the test suite with the race detector.
test:
    go test -race ./...

# Lint.
lint:
    golangci-lint run

# Format all Go source.
fmt:
    golangci-lint fmt

# Lint with autofix.
fix:
    golangci-lint run --fix

# Lint + test.
check: lint test

# Tidy go.mod/go.sum.
tidy:
    go mod tidy

# Update all dependencies and tidy.
update:
    go get -u ./...
    go mod tidy

# Remove build artifacts.
clean:
    rm -rf bin/ dist/
