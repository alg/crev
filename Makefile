.PHONY: build install clean run test

# Build the binary
build:
	go build -o crev ./cmd/crev

# Install to skill directory with symlinks in ~/.local/bin
SKILL_DIR := ~/.claude/skills/review

install: build
	mkdir -p ~/.local/bin $(SKILL_DIR)
	cp crev $(SKILL_DIR)/
	cp claude-skill/SKILL.md $(SKILL_DIR)/
	cp claude-skill/crev-popup $(SKILL_DIR)/
	chmod +x $(SKILL_DIR)/crev-popup
	ln -sf $(SKILL_DIR)/crev ~/.local/bin/crev
	ln -sf $(SKILL_DIR)/crev-popup ~/.local/bin/crev-popup
	@echo "Installed crev to $(SKILL_DIR)"

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
