#!/bin/bash
echo "Running tests without TEST_DATABASE_URL (expected to SKIP)..."
go test -v ./internal/database ./internal/contactutil | grep -E "SKIP|PASS"

echo -e "\nRunning tests with incorrect TEST_DATABASE_URL (expected to FAIL)..."
export TEST_DATABASE_URL="postgres://test:test@127.0.0.1:5433/test?sslmode=disable"
go test -v ./internal/database ./internal/contactutil || true

echo -e "\nCoverage repro completed."
