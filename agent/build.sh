#!/bin/bash
# File: agent/build.sh

set -e

echo "Building Hermes Agent..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Go not found. Please install Go first."
    exit 1
fi

# Create build directory if it doesn't exist
mkdir -p build

# Build the agent
echo "Compiling agent..."
cd "$(dirname "$0")"
go build -o build/hermes-agent cmd/agent/main.go

echo "Build complete!"
echo "Binary is available at: $(pwd)/build/hermes-agent"