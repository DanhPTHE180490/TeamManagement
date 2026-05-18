#!/usr/bin/env bash
set -euo pipefail

# Starts the DB service and runs integration tests against it.
# Usage: ./scripts/run_integration_tests.sh

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

echo "Starting database via docker compose and waiting for it to become healthy..."
docker compose up -d --wait db

echo "Running integration tests..."
go test ./internal/... -v

echo "Done."
