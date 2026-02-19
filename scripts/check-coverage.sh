#!/bin/bash
# Check that test coverage meets the minimum threshold.
# Usage: ./scripts/check-coverage.sh [threshold]
# Default threshold: 75

set -euo pipefail

THRESHOLD="${1:-75}"
COVERPROFILE=$(mktemp)

echo "Running tests with coverage..."
go test -coverprofile="$COVERPROFILE" ./... > /dev/null 2>&1

# Extract the total coverage percentage
COVERAGE=$(go tool cover -func="$COVERPROFILE" | grep ^total: | awk '{print $3}' | tr -d '%')

rm -f "$COVERPROFILE"

if [ -z "$COVERAGE" ]; then
    echo "FAIL: could not determine coverage"
    exit 1
fi

# Compare using awk for float comparison
PASS=$(awk "BEGIN {print ($COVERAGE >= $THRESHOLD) ? 1 : 0}")

if [ "$PASS" -eq 1 ]; then
    echo "PASS: coverage ${COVERAGE}% >= ${THRESHOLD}% threshold"
    exit 0
else
    echo "FAIL: coverage ${COVERAGE}% < ${THRESHOLD}% threshold"
    exit 1
fi
