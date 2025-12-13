#!/bin/bash
# Calculate test coverage for library packages only (excluding examples and tools)
# Usage: ./scripts/coverage.sh [--html] [--verbose]

set -e

HTML=false
VERBOSE=false

for arg in "$@"; do
    case $arg in
        --html)
            HTML=true
            ;;
        --verbose|-v)
            VERBOSE=true
            ;;
    esac
done

# Get list of library packages (excluding examples and tools)
PACKAGES=$(go list ./... | grep -v '/examples/' | grep -v '/tools/')

if [ "$VERBOSE" = true ]; then
    echo "Testing packages:"
    echo "$PACKAGES" | tr ' ' '\n'
    echo ""
fi

# Run tests with coverage
if [ "$VERBOSE" = true ]; then
    echo "$PACKAGES" | xargs go test -coverprofile=coverage.out -covermode=atomic -v
else
    echo "$PACKAGES" | xargs go test -coverprofile=coverage.out -covermode=atomic
fi

# Show coverage summary
echo ""
echo "=== Coverage Summary ==="
go tool cover -func=coverage.out | tail -1

# Generate HTML report if requested
if [ "$HTML" = true ]; then
    go tool cover -html=coverage.out -o coverage.html
    echo "HTML report generated: coverage.html"
fi
