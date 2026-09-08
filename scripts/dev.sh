#!/usr/bin/env bash
set -e
cd "$(dirname "$0")/.."

echo "==> MOCK mode run"
docker compose up --build