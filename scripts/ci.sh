#!/usr/bin/env bash
set -euo pipefail

echo "Running full test suite..."
go test ./...
