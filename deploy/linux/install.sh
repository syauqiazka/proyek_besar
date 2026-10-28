#!/usr/bin/env bash
# Hermes Storage Monitor — Deploy & Start (Linux Server)
# Usage: ./install.sh [user]  (default user: hermes)

set -euo pipefail

USER_NAME="${1:-hermes}"
APP_ROOT="/opt/hermes"
SRC_ROOT="$(cd "$(dirname "$0")/.." && pwd)"

echo "============================================"
echo " Hermes Storage Monitor — Install Script"
echo "============================================"
echo

# 1. Create user if not exists
if ! id "$USER_NAME" &>/dev/null; then
  echo "[1/6] Creating user '$USER_NAME'..."
  useradd -m -s /bin/bash "$USER_NAME"
else
  echo "[1/6] User '$USER_NAME' already exists."
fi

# 2. Install dependencies
echo "[2/6] Installing dependencies (curl, Node.js, Go)..."
apt-get update -y
apt-get install -y curl nodejs npm golang-go

# 3. Create app directory
echo "[3/6] Creating $APP_ROOT..."
mkdir -p "$APP_ROOT/backend"
mkdir -p "$APP_ROOT/dashboard"

# 4. Copy source files
echo "[4/6] Copying source files..."
cp -r "$SRC_ROOT/backend"/* "$APP_ROOT/backend/" 2>/dev/null || true
cp -r "$SRC_ROOT/dashboard"/* "$APP_ROOT/dashboard/" 2>/dev/null || true

# 5. Build backend
echo "[5/6] Building backend..."
cd "$APP_ROOT/backend"
go build -o hermes-backend ./cmd/api

# 6. Install systemd service
echo "[6/6] Installing systemd service..."
cp "$SRC_ROOT/deploy/linux/hermes.service" /etc/systemd/system/hermes.service
systemctl daemon-reload
systemctl enable hermes.service
systemctl start hermes.service

echo
echo "============================================"
echo " Hermes installed & running in background!"
echo " URL: http://<server-ip>:3001/health"
echo " Check: systemctl status hermes"
echo "============================================"
echo