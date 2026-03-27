# Define variables
GO_BIN = go
GO_BUILD_FLAGS = -v
BINARY_NAME = gh-install
MAIN_PACKAGE = .
OUTPUT_DIR = .

.PHONY: all build clean fmt lint tidy

all: build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p $(OUTPUT_DIR)
	$(GO_BIN) build $(GO_BUILD_FLAGS) -o $(OUTPUT_DIR)/$(BINARY_NAME) $(MAIN_PACKAGE)
	@echo "Build complete. Binary located at $(OUTPUT_DIR)/$(BINARY_NAME)"

clean:
	@echo "Cleaning up..."
	@rm -rf $(OUTPUT_DIR)
	@$(GO_BIN) clean
	@echo "Cleanup complete."

fmt:
	@echo "Formatting Go code..."
	$(GO_BIN) fmt ./...

lint:
	@echo "Running linters"
	golangci-lint run ./...

tidy:
	@echo "Tidying Go modules..."
	$(GO_BIN) mod tidy