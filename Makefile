# Makefile for DDNSwitch

BINARY_NAME=ddnswitch
VERSION=1.0.0
BUILD_DIR=build
LDFLAGS=-ldflags "-X main.version=${VERSION}"

# Default target
.DEFAULT_GOAL := build

# Build for current platform
.PHONY: build
build:
	@echo "Building ${BINARY_NAME} v${VERSION}..."
	@mkdir -p ${BUILD_DIR}
	go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME} .

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-darwin build-windows

.PHONY: build-linux
build-linux:
	@echo "Building for Linux..."
	@mkdir -p ${BUILD_DIR}
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-amd64 .
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-linux-arm64 .

.PHONY: build-darwin
build-darwin:
	@echo "Building for macOS..."
	@mkdir -p ${BUILD_DIR}
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-amd64 .
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-darwin-arm64 .

.PHONY: build-windows
build-windows:
	@echo "Building for Windows..."
	@mkdir -p ${BUILD_DIR}
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-amd64.exe .
	GOOS=windows GOARCH=arm64 go build ${LDFLAGS} -o ${BUILD_DIR}/${BINARY_NAME}-windows-arm64.exe .

# Install locally
.PHONY: install
install: build
	@echo "Installing ${BINARY_NAME}..."
	cp ${BUILD_DIR}/${BINARY_NAME} /usr/local/bin/
	@echo "${BINARY_NAME} installed to /usr/local/bin/"

# Install to user's local bin
.PHONY: install-user
install-user: build
	@echo "Installing ${BINARY_NAME} to ~/bin..."
	@mkdir -p ~/bin
	cp ${BUILD_DIR}/${BINARY_NAME} ~/bin/
	@echo "${BINARY_NAME} installed to ~/bin/"
	@echo "Make sure ~/bin is in your PATH"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf ${BUILD_DIR}

# Run tests
.PHONY: test
test:
	go test -v ./...

# Format code
.PHONY: fmt
fmt:
	go fmt ./...

# Vet code
.PHONY: vet
vet:
	go vet ./...

# Run all checks
.PHONY: check
check: fmt vet test

# Initialize go modules
.PHONY: deps
deps:
	go mod tidy
	go mod download

# Add a debug target for testing with debug mode
.PHONY: debug
debug: build
	@echo "Running ${BINARY_NAME} in debug mode..."
	${BUILD_DIR}/${BINARY_NAME} --debug version

# Show help
.PHONY: help
help:
	@echo "Available targets:"
	@echo "  build        - Build binary for current platform"
	@echo "  build-all    - Build binaries for all platforms"
	@echo "  build-linux  - Build binaries for Linux"
	@echo "  build-darwin - Build binaries for macOS"
	@echo "  build-windows- Build binaries for Windows"
	@echo "  install      - Install binary to /usr/local/bin"
	@echo "  install-user - Install binary to ~/bin"
	@echo "  clean        - Remove build artifacts"
	@echo "  test         - Run tests"
	@echo "  debug        - Run in debug mode"
	@echo "  fmt          - Format code"
	@echo "  vet          - Vet code"
	@echo "  check        - Run fmt, vet, and test"
	@echo "  deps         - Download dependencies"
	@echo "  help         - Show this help message"
