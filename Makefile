# Makefile for project setup

.PHONY: setup clean build agent backend dashboard

setup:
	@echo "Setting up environment variables..."
	@cp .env.example .env
	@cp agent/.env.example agent/.env
	@echo "Setup complete. Edit .env files with your values."

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf agent/hermes-agent backend/hermes-backend
	@echo "Clean complete."

build:
	@echo "Building agent binary..."
	@cd agent && go build -o hermes-agent cmd/agent/main.go
	@echo "Building backend..."
	@cd backend && go build -o hermes-backend cmd/api/main.go
	@echo "Build complete."

dashboard:
	@echo "Starting dashboard (requires Node.js)..."
	@cd dashboard && npm install && npm run dev

test:
	@echo "Running tests..."
	@echo "No tests implemented yet"

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  setup      - Setup environment variables"
	@echo "  clean      - Clean build artifacts"
	@echo "  build      - Build binaries for agent and backend"
	@echo "  dashboard  - Start dashboard (npm dev)"
	@echo "  test       - Run tests (not implemented)"
	@echo "  help       - Show this help"