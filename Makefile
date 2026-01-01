.PHONY: build install clean run test

# Build the binary
build:
	go build -o crev ./cmd/crev

# Install to ~/.local/bin
install: build
	mkdir -p ~/.local/bin
	cp crev ~/.local/bin/
	@echo "Installed crev to ~/.local/bin/crev"

# Clean build artifacts
clean:
	rm -f crev

# Run in development
run: build
	./crev

# Run tests
test:
	go test -v ./...

# Format code
fmt:
	go fmt ./...

# Lint code
lint:
	golangci-lint run

# Build for all platforms
build-all:
	GOOS=darwin GOARCH=amd64 go build -o crev-darwin-amd64 ./cmd/crev
	GOOS=darwin GOARCH=arm64 go build -o crev-darwin-arm64 ./cmd/crev
	GOOS=linux GOARCH=amd64 go build -o crev-linux-amd64 ./cmd/crev
