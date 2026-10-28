#!/usr/bin/env bash
# Hermes Storage Monitor — Check Agent Status

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_FILE="$DIR/hermes-agent.pid"
LOG_FILE="$DIR/agent.log"

echo "========================================"
echo "      Hermes Agent Status Check         "
echo "========================================"

RUNNING=false

# Cek via systemd service jika ada
if command -v systemctl >/dev/null 2>&1 && systemctl is-active --quiet hermes-agent.service 2>/dev/null; then
  echo "Status: [AKTIF] Berjalan via Systemd Service"
  systemctl status hermes-agent.service --no-pager
  exit 0
fi

# Cek via PID file
if [ -f "$PID_FILE" ]; then
  PID=$(cat "$PID_FILE")
  if kill -0 "$PID" 2>/dev/null; then
    echo "Status : [AKTIF] Berjalan di Background"
    echo "PID    : $PID"
    RUNNING=true
  fi
fi

# Cek via pgrep jika PID file hilang tapi proses jalan
if [ "$RUNNING" = false ] && pgrep -f "hermes-agent" > /dev/null 2>&1; then
  PIDS=$(pgrep -f "hermes-agent")
  echo "Status : [AKTIF] Berjalan (PID: $PIDS)"
  RUNNING=true
fi

if [ "$RUNNING" = false ]; then
  echo "Status : [MATI / TIDAK AKTIF]"
  echo "Jalankan dengan: ./start-agent.sh (atau sudo systemctl start hermes-agent)"
fi

echo "----------------------------------------"
if [ -f "$LOG_FILE" ]; then
  echo "Log Terakhir ($LOG_FILE):"
  tail -n 10 "$LOG_FILE"
else
  echo "Belum ada file log ($LOG_FILE)."
fi
echo "========================================"
