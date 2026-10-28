#!/usr/bin/env bash
# Hermes Storage Monitor — Start Agent in Background (Nohup / Tanpa Sudo)
# Cocok untuk pengguna tanpa akses root, VPS, shared hosting, atau VM/pod

DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PID_FILE="$DIR/hermes-agent.pid"
LOG_FILE="$DIR/agent.log"
BINARY="$DIR/hermes-agent"

# Cek apakah file binary ada
if [ ! -f "$BINARY" ]; then
  echo "[ERROR] File binary $BINARY tidak ditemukan!"
  exit 1
fi

# Cek apakah file .env ada
if [ ! -f "$DIR/.env" ]; then
  echo "[WARNING] File .env tidak ditemukan di $DIR/. Pastikan konfigurasi HERMES_* sudah di-set!"
fi

# Cek apakah agent sudah berjalan
if [ -f "$PID_FILE" ]; then
  PID=$(cat "$PID_FILE")
  if kill -0 "$PID" 2>/dev/null; then
    echo "[INFO] Hermes Agent sudah berjalan di background (PID: $PID)."
    echo "Gunakan './status-agent.sh' untuk cek status atau './stop-agent.sh' untuk menghentikan."
    exit 0
  fi
fi

# Pastikan permission execute jika diizinkan
chmod +x "$BINARY" 2>/dev/null || true

echo "[INFO] Memulai Hermes Agent di background..."

# Coba jalankan secara langsung; jika gagal karena noexec, jalankan via bash
cd "$DIR"
if [ -x "$BINARY" ]; then
  nohup "$BINARY" >> "$LOG_FILE" 2>&1 &
  PID=$!
else
  echo "[INFO] Menjalankan via bash (dukungan filesystem noexec)..."
  nohup bash "$BINARY" >> "$LOG_FILE" 2>&1 &
  PID=$!
fi

echo "$PID" > "$PID_FILE"
sleep 1

# Cek apakah proses berhasil hidup
if kill -0 "$PID" 2>/dev/null; then
  echo "✓ Hermes Agent BERHASIL berjalan di background!"
  echo "  PID       : $PID"
  echo "  Log file  : $LOG_FILE"
  echo "  Pantau log: tail -f $LOG_FILE"
  echo "  Stop agent: ./stop-agent.sh"
else
  echo "[ERROR] Hermes Agent gagal berjalan. Periksa log:"
  tail -n 15 "$LOG_FILE"
  rm -f "$PID_FILE"
  exit 1
fi
