#!/bin/bash
# Update coverage badge on the 'badges' branch
# Parses coverage from coverage.out, generates a shields.io JSON badge,
# and force-pushes to the badges branch.
# Usage: bash scripts/update-coverage-badge.sh

set -e

# Parse coverage percentage from coverage.out
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}' | sed 's/%//')

# Determine badge color based on coverage thresholds
if (( $(echo "$COVERAGE >= 90" | bc -l) )); then COLOR="brightgreen"
elif (( $(echo "$COVERAGE >= 80" | bc -l) )); then COLOR="green"
elif (( $(echo "$COVERAGE >= 70" | bc -l) )); then COLOR="yellowgreen"
elif (( $(echo "$COVERAGE >= 60" | bc -l) )); then COLOR="yellow"
else COLOR="red"; fi

# Generate shields.io endpoint badge JSON
BADGE="{\"schemaVersion\":1,\"label\":\"coverage\",\"message\":\"${COVERAGE}%\",\"color\":\"${COLOR}\"}"

# Configure git for automated commits
git config user.email "github-actions[bot]@users.noreply.github.com"
git config user.name "github-actions[bot]"

# Create an orphan branch, replace contents with badge JSON, and force-push
git checkout --orphan badges-temp
git rm -rf . > /dev/null 2>&1 || true
echo "$BADGE" > coverage.json
git add coverage.json
git commit -m "Update coverage to ${COVERAGE}%"
git push origin badges-temp:badges --force
