#!/usr/bin/env bash
# Hermes Storage Monitor — Stop Background Agent

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_FILE="$DIR/hermes-agent.pid"

STOPPED=false

if [ -f "$PID_FILE" ]; then
  PID=$(cat "$PID_FILE")
  if kill -0 "$PID" 2>/dev/null; then
    echo "[INFO] Menghentikan Hermes Agent (PID: $PID)..."
    kill "$PID" 2>/dev/null || true
    sleep 1
    if kill -0 "$PID" 2>/dev/null; then
      kill -9 "$PID" 2>/dev/null || true
    fi
    echo "✓ Hermes Agent (PID: $PID) berhasil dihentikan."
    STOPPED=true
  fi
  rm -f "$PID_FILE"
fi

# Fallback: cari dan bunuh proses hermes-agent lain jika masih ada
if pgrep -f "hermes-agent" > /dev/null 2>&1; then
  echo "[INFO] Menghentikan sisa proses hermes-agent..."
  pkill -f "hermes-agent" 2>/dev/null || true
  STOPPED=true
  echo "✓ Sisa proses hermes-agent dihentikan."
fi

if [ "$STOPPED" = false ]; then
  echo "[INFO] Tidak ada Hermes Agent yang sedang berjalan."
fi
