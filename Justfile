# Show all tasks by default
list:
  just --list --unsorted

fmt:
  golangci-lint fmt

lint:
  golangci-lint run

# Runs lint and fmt first
test: fmt lint
  go test ./...

# Requires tests passing
build: test
  go build
