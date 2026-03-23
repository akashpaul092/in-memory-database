#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."
go build -o bin/server ./cmd/server/
echo "Built: bin/server"
