#!/usr/bin/env bash
# ──────────────────────────────────────────────────────────────────────────────
# scripts/perf-test.sh — run k6 performance tests via Docker
#
# Usage:
#   ./scripts/perf-test.sh [BASE_URL]
#
# Examples:
#   ./scripts/perf-test.sh
#   ./scripts/perf-test.sh http://host.docker.internal:8080
# ──────────────────────────────────────────────────────────────────────────────
set -euo pipefail

BASE_URL="${1:-http://host.docker.internal:8080}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
K6_SCRIPT="$SCRIPT_DIR/k6/perf-test.js"

# ── preflight ─────────────────────────────────────────────────────────────────
if ! command -v docker &>/dev/null; then
  echo "Docker is not installed or not in PATH."
  exit 1
fi

if ! docker info &>/dev/null; then
  echo "Docker daemon is not running."
  exit 1
fi

HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" \
  "${BASE_URL/host.docker.internal/localhost}/api/users" 2>/dev/null || echo "000")

if [[ "$HTTP_CODE" != "200" ]]; then
  echo "Server not reachable at ${BASE_URL/host.docker.internal/localhost}/api/users (HTTP $HTTP_CODE)"
  echo "Run: make migrate && make run"
  exit 1
fi

echo "Running k6 via Docker against $BASE_URL ..."
echo ""

# ── run k6 ───────────────────────────────────────────────────────────────────
docker run --rm \
  -v "$SCRIPT_DIR/k6:/scripts" \
  -e "BASE_URL=$BASE_URL" \
  --add-host "host.docker.internal:host-gateway" \
  grafana/k6:latest run /scripts/perf-test.js
