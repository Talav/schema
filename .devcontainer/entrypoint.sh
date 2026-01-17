#!/bin/bash
set -e  # Exit on any error

# Change to workspace directory
cd /workspace

# Download and tidy Go modules if go.mod exists
# This ensures dependencies are up to date
if [ -f go.mod ]; then
    echo "Downloading Go modules..."
    go mod download
    echo "Tidying Go modules..."
    go mod tidy
fi

# Execute the original command passed to the container
exec "$@"
