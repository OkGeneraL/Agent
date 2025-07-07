#!/bin/bash

echo "Building SuperAgent..."

# Clean up problematic modules
go clean -modcache || true

# Try to build with build constraints to skip problematic parts
echo "Attempting build..."
if ! go build -tags minimal -o superagent ./cmd/agent 2>/dev/null; then
    echo "Standard build failed, trying alternative approach..."
    
    # Remove problematic Docker functionality temporarily
    echo "Creating minimal build..."
    go mod edit -replace github.com/docker/docker=github.com/docker/docker@v20.10.21+incompatible
    go mod edit -replace github.com/docker/distribution=github.com/distribution/distribution@v2.7.1+incompatible
    go mod tidy
    
    if go build -o superagent ./cmd/agent; then
        echo "✅ SuperAgent built successfully!"
        ./superagent version
    else
        echo "❌ Build failed"
        exit 1
    fi
else
    echo "✅ SuperAgent built successfully with minimal tags!"
    ./superagent version
fi