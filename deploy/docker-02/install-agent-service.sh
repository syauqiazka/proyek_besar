#!/usr/bin/env bash
# Hermes Storage Monitor Agent — Systemd Installer
# Menjalankan hermes-agent secara otomatis di background saat boot
#
# Penggunaan:
#   sudo bash install-agent-service.sh
#   sudo bash install-agent-service.sh /path/ke/folder-agent    # jika file di direktori berbeda

set -euo pipefail

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}====================================================${NC}"
echo -e "${BLUE}   Hermes Agent — Linux Background Service Setup   ${NC}"
echo -e "${BLUE}====================================================${NC}"

# Cek hak akses root/sudo
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}[ERROR] Skrip ini membutuhkan hak akses root atau sudo.${NC}"
  echo -e "Silakan jalankan ulang dengan: ${YELLOW}sudo bash $0${NC}"
  exit 1
fi

INSTALL_DIR="/opt/hermes"

# Direktori sumber: ambil dari argumen pertama, atau dari lokasi skrip, atau dari ~
if [ -n "${1:-}" ]; then
  SOURCE_DIR="$1"
  echo -e "${BLUE}[INFO] Menggunakan direktori sumber dari argumen: $SOURCE_DIR${NC}"
elif [ -f "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/hermes-agent" ]; then
  SOURCE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  echo -e "${BLUE}[INFO] Menggunakan direktori skrip: $SOURCE_DIR${NC}"
elif [ -f "$HOME/hermes-agent" ]; then
  SOURCE_DIR="$HOME"
  echo -e "${BLUE}[INFO] Menggunakan home directory: $SOURCE_DIR${NC}"
else
  SOURCE_DIR="$(pwd)"
  echo -e "${YELLOW}[INFO] Menggunakan direktori saat ini: $SOURCE_DIR${NC}"
fi

# Cari binary hermes-agent
BINARY_SRC=""
if [ -f "$SOURCE_DIR/hermes-agent" ]; then
  BINARY_SRC="$SOURCE_DIR/hermes-agent"
fi

# Cari file .env
ENV_SRC=""
if [ -f "$SOURCE_DIR/.env" ]; then
  ENV_SRC="$SOURCE_DIR/.env"
elif [ -f "$SOURCE_DIR/hermes-agent.env" ]; then
  ENV_SRC="$SOURCE_DIR/hermes-agent.env"
fi

# 1. Buat direktori instalasi
echo -e "\n${YELLOW}[1/4] Menyiapkan direktori $INSTALL_DIR...${NC}"
mkdir -p "$INSTALL_DIR"

# 2. Hentikan service dulu jika sedang berjalan (menghindari "Text file busy")
echo -e "${YELLOW}[2/4] Memasang file hermes-agent dan konfigurasi...${NC}"
if systemctl is-active --quiet hermes-agent 2>/dev/null; then
  echo -e "${YELLOW}  -> Menghentikan service hermes-agent yang sedang berjalan...${NC}"
  systemctl stop hermes-agent || true
  sleep 1
fi

# Salin binary
if [ -n "$BINARY_SRC" ]; then
  cp -f "$BINARY_SRC" "$INSTALL_DIR/hermes-agent"
  chmod +x "$INSTALL_DIR/hermes-agent"
  echo -e "${GREEN}✓ Binary hermes-agent berhasil disalin ke $INSTALL_DIR/hermes-agent${NC}"
else
  if [ -f "$INSTALL_DIR/hermes-agent" ]; then
    chmod +x "$INSTALL_DIR/hermes-agent"
    echo -e "${GREEN}✓ Menggunakan binary yang sudah ada di $INSTALL_DIR/hermes-agent${NC}"
  else
    echo -e "${RED}[PERINGATAN] Binary hermes-agent tidak ditemukan!${NC}"
    echo -e "  Letakkan hermes-agent di salah satu lokasi berikut:"
    echo -e "  - $SOURCE_DIR/hermes-agent"
    echo -e "  - $INSTALL_DIR/hermes-agent"
    echo -e "  Atau jalankan: sudo bash install-agent-service.sh /path/ke/folder/"
  fi
fi

# Salin .env
if [ -n "$ENV_SRC" ]; then
  cp -f "$ENV_SRC" "$INSTALL_DIR/.env"
  chmod 600 "$INSTALL_DIR/.env"
  echo -e "${GREEN}✓ File .env berhasil disalin ke $INSTALL_DIR/.env${NC}"
elif [ -f "$INSTALL_DIR/.env" ]; then
  echo -e "${GREEN}✓ Menggunakan file .env yang sudah ada di $INSTALL_DIR/.env${NC}"
else
  echo -e "${YELLOW}! File .env belum ada di $INSTALL_DIR/.env. Jangan lupa buat file .env sebelum menjalankan agent!${NC}"
fi

# 3. Pasang Systemd Service
echo -e "${YELLOW}[3/4] Mendaftarkan systemd service hermes-agent...${NC}"
SERVICE_FILE="/etc/systemd/system/hermes-agent.service"

cat > "$SERVICE_FILE" <<EOF
[Unit]
Description=Hermes Storage Monitor Agent
After=network.target
Wants=network.target

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$INSTALL_DIR/.env
ExecStart=$INSTALL_DIR/hermes-agent
Restart=always
RestartSec=5
KillMode=process
StandardOutput=journal
StandardError=journal
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

# 4. Reload, Enable, dan Start
echo -e "${YELLOW}[4/4] Mengaktifkan dan menyalakan service...${NC}"
systemctl daemon-reload
systemctl enable hermes-agent.service
systemctl restart hermes-agent.service

echo -e "\n${GREEN}====================================================${NC}"
echo -e "${GREEN}  ✓ Hermes Agent Berhasil Berjalan di Background!  ${NC}"
echo -e "${GREEN}====================================================${NC}"
echo -e "Status Service:  ${YELLOW}systemctl status hermes-agent${NC}"
echo -e "Pantau Log Live: ${YELLOW}journalctl -u hermes-agent -f${NC}"
echo -e "Restart Service: ${YELLOW}systemctl restart hermes-agent${NC}"
echo -e "Stop Service:    ${YELLOW}systemctl stop hermes-agent${NC}"
echo -e "====================================================\n"
