.PHONY: build generate run clean

BINARY_NAME := homma
BINARY_DIR := bin
CMD_DIR := cmd

build:
	@echo "Building $(BINARY_NAME)..."
	go build -o $(BINARY_DIR)/$(BINARY_NAME) ./$(CMD_DIR)
	@echo "Build complete. Binary: $(BINARY_DIR)/$(BINARY_NAME)"

generate:
	go mod tidy
	@echo "Ensuring easyjson is installed..."
	@if ! command -v easyjson >/dev/null 2>&1; then \
		echo "easyjson not found, installing..."; \
		go get github.com/mailru/easyjson && go install github.com/mailru/easyjson/...@latest; \
	else \
		echo "easyjson already installed."; \
	fi
	@echo "Running go generate..."
	find . -type f -name "*.go" -exec dirname {} \; | sort -u | xargs -L 1 go generate
	@echo "Go generate complete."

run: generate build
	@echo "Running $(BINARY_NAME)..."
	./$(BINARY_DIR)/$(BINARY_NAME)

clean:
	@echo "Cleaning up..."
	rm -f $(BINARY_DIR)/$(BINARY_NAME)
	@echo "Clean complete."
