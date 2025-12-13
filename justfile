# List available commands
default:
	@just --list

# Build the application
build:
	go build -o bin/media-organizer ./cmd/media-organizer

# Run the application
run:
	go run ./cmd/media-organizer

# Run all tests
test:
	go test ./...

# Run tests with coverage
test-coverage:
	go test -cover ./...

# Run a single test (usage: just test-one ./pkg/foo TestName)
test-one package test:
	go test {{package}} -run {{test}}

# Lint the code
lint:
	golangci-lint run

# Format the code
fmt:
	gofmt -s -w .
	goimports -w .

# Clean build artifacts
clean:
	rm -rf bin/
	rm -rf dist/

# Install dependencies
deps:
	go mod download
	go mod tidy
