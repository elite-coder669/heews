#!/usr/bin/env bash
set -e
MONOREPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
cd "$MONOREPO_DIR"

# Defaults
APP_MODE="${APP_MODE:-MOCK}"
PORT="${PORT:-8080}"
FRONTEND_PORT="${FRONTEND_PORT:-5173}"

echo "==> HEEWS Local Runner"
echo "   APP_MODE=$APP_MODE  Backend port=$PORT  Frontend port=$FRONTEND_PORT"

# ──────────────────────────────────────────────
# 1) Environment
# ──────────────────────────────────────────────
if [ ! -f .env ]; then
  if [ "$APP_MODE" = "MOCK" ]; then
    cp .env.example .env || true
    echo "⚠  No .env found — created from .env.example (MOCK mode)"
  else
    echo "❌  No .env file found for LIVE mode. Copy .env.example and set:" >&2
    echo "   OPEN_METEO_BASE=https://api.open-meteo.com/v1/forecast" >&2
    echo "   OPEN_METEO_LAT=17.3850" >&2
    echo "   OPEN_METEO_LON=78.4867" >&2
    exit 1
  fi
fi
# shellcheck source=/dev/null
set -a; source .env; set +a

# ──────────────────────────────────────────────
# 2) Backend (Go)
# ──────────────────────────────────────────────
start_backend() {
  echo "==> Starting Go backend (LIVE=$APP_MODE) on :$PORT"
  cd backend
  if [ "$APP_MODE" = "MOCK" ]; then
    go run ./cmd/server
  else
    OPEN_METEO_BASE="${OPEN_METEO_BASE:-https://api.open-meteo.com/v1/forecast}" \
    OPEN_METEO_LAT="${OPEN_METEO_LAT:-17.3850}" \
    OPEN_METEO_LON="${OPEN_METEO_LON:-78.4867}" \
    go run ./cmd/server
  fi
}

# ──────────────────────────────────────────────
# 3) Frontend (Vite/React)
# ──────────────────────────────────────────────
start_frontend() {
  echo "==> Starting React frontend on :$FRONTEND_PORT"
  cd frontend
  # Vite dev server (HMR + proxies /api and /data to the Go backend on :8080)
  npm install 2>&1 | tail -1
  npm run dev -- --port "$FRONTEND_PORT" &
  cd "$MONOREPO_DIR"
}

# ──────────────────────────────────────────────
# 4) ML / Agent (Python)
# ──────────────────────────────────────────────
start_ml() {
  echo "==> Ensuring ML deps are available"
  pip install -q -r ml/requirements.txt 2>&1 | tail -1
}

# ──────────────────────────────────────────────
# 5) Main — launch what's requested
# ──────────────────────────────────────────────
case "${1:-full}" in
  backend)
    start_backend
    ;;
  frontend)
    start_frontend
    ;;
  ml)
    start_ml
    ;;
  full)
    echo "---"
    echo "HEEWS Local Environment"
    echo "  Monorepo: $MONOREPO_DIR"
    echo "  Mode:     $APP_MODE"
    echo ""

    # Start backend in background
    start_backend &
    BACKEND_PID=$!

    # Start frontend in background
    start_frontend
    FRONTEND_PID=$!

    echo ""
    echo "==> Services running:"
    echo "  Backend:  http://localhost:$PORT  (press Ctrl+C to stop)"
    echo "  Frontend: http://localhost:$FRONTEND_PORT"
    echo "  API:      http://localhost:$PORT/api/health"
    echo ""

    # Wait for both; on interrupt, kill both
    trap 'kill $BACKEND_PID $FRONTEND_PID 2>/dev/null; trap - EXIT; exit 0' INT TERM
    wait $BACKEND_PID $FRONTEND_PID 2>/dev/null
    ;;
  *)
    echo "Usage: $0 {backend|frontend|ml|full}"
    echo "  backend – only Go backend"
    echo "  frontend – only React dev server"
    echo "  ml – ensure Python ML deps"
    echo "  full   – start both backend + frontend (default)"
    exit 1
    ;;
esac