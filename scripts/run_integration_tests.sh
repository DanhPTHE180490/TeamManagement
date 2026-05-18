#!/usr/bin/env bash
set -euo pipefail

# Starts the DB service and runs integration tests against it.
# Usage: ./scripts/run_integration_tests.sh

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
cd "$ROOT_DIR"

echo "Starting database via docker compose..."
docker compose up -d db

echo "Waiting for DB to become available..."
sleep 8

echo "Running integration tests..."
go test ./internal/... -v

echo "Done."
