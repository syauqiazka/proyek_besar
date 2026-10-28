# Panduan Menjalankan Hermes Agent di Background (Linux Server)

Dokumen ini menjelaskan cara agar program `hermes-agent` terus berjalan di latar belakang (background) pada server Linux, sehingga pemantauan storage tetap aktif meski koneksi SSH ditutup ataupun server di-reboot.

---

## Ringkasan 2 Metode

| Metode | Keunggulan | Kapan Digunakan? |
|---|---|---|
| **Metode 1: Systemd Service** *(Rekomendasi Utama)* | Auto-start saat server boot, auto-restart saat crash, log tersimpan di `journalctl` | Server produksi, VPS (Ubuntu, Debian, CentOS, Rocky Linux) dengan akses `sudo` |
| **Metode 2: Skrip Nohup** | Tanpa butuh hak akses root/sudo, langsung jalan seketika | Pengguna biasa, VM/Pod terbatas, container, testing cepat |

---

## Persiapan File di Server Linux

Kirim file `hermes-agent` dan file konfigurasi `.env` ke folder server Linux Anda (misal di folder home `~/` atau `/opt/hermes`):

```bash
# Dari komputer lokal:
scp agent/hermes-agent user@IP_SERVER:~/
scp deploy/<nama-server>/.env user@IP_SERVER:~/
```

Contoh isi file `.env`:
```ini
HERMES_SERVER_ID=srv_xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
HERMES_API_KEY=xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
HERMES_BACKEND_URL=http://IP_BACKEND:3001/api/v1
HERMES_SCAN_INTERVAL=*/5 * * * *
HERMES_MOUNT_PATHS=/,/home
```

---

## METODE 1: Menggunakan Systemd Service (Rekomendasi Utama)

Metode ini memastikan agent tetap hidup 24/7 dan otomatis hidup kembali jika server restart.

### Cara Cepat (Menggunakan Skrip Otomatis):
```bash
sudo bash install-agent-service.sh
```

### Cara Manual:
1. Pindahkan binary dan `.env` ke direktori sistem:
   ```bash
   sudo mkdir -p /opt/hermes
   sudo cp hermes-agent /opt/hermes/
   sudo cp .env /opt/hermes/
   sudo chmod +x /opt/hermes/hermes-agent
   sudo chmod 600 /opt/hermes/.env
   ```

2. Buat file service `/etc/systemd/system/hermes-agent.service`:
   ```bash
   sudo nano /etc/systemd/system/hermes-agent.service
   ```
   Isi dengan:
   ```ini
   [Unit]
   Description=Hermes Storage Monitor Agent
   After=network.target
   Wants=network.target

   [Service]
   Type=simple
   User=root
   WorkingDirectory=/opt/hermes
   ExecStart=/opt/hermes/hermes-agent
   Restart=always
   RestartSec=5
   KillMode=process
   StandardOutput=journal
   StandardError=journal
   LimitNOFILE=65535

   [Install]
   WantedBy=multi-user.target
   ```

3. Aktifkan dan jalankan:
   ```bash
   sudo systemctl daemon-reload
   sudo systemctl enable hermes-agent
   sudo systemctl start hermes-agent
   ```

4. Perintah pengelolaan:
   - **Cek Status**: `sudo systemctl status hermes-agent`
   - **Pantau Log Real-time**: `sudo journalctl -u hermes-agent -f`
   - **Restart**: `sudo systemctl restart hermes-agent`
   - **Hentikan**: `sudo systemctl stop hermes-agent`

---

## METODE 2: Menggunakan Skrip Nohup (Tanpa Sudo/Root)

Jika Anda tidak memiliki hak akses `sudo`, gunakan skrip yang sudah kami sediakan:

### 1. Menjalankan di background:
```bash
chmod +x start-agent.sh stop-agent.sh status-agent.sh
./start-agent.sh
```
Atau cukup satu baris perintah langsung:
```bash
cd ~ && nohup ./hermes-agent > agent.log 2>&1 &
```

*(Jika muncul pesan error `Permission denied` akibat filesystem `noexec`, jalankan via bash:)*
```bash
cd ~ && nohup bash hermes-agent > agent.log 2>&1 &
```

### 2. Memeriksa status & log:
```bash
./status-agent.sh
# atau pantau log langsung:
tail -f agent.log
```

### 3. Menghentikan agent:
```bash
./stop-agent.sh
# atau perintah manual:
pkill -f hermes-agent
```

---

## METODE 3: Menggunakan Tmux / Screen (Alternatif)

Jika Anda menyukai terminal multiplexer:

```bash
# Menggunakan tmux
tmux new -s hermes -d "./hermes-agent"

# Untuk melihat tampilannya kembali:
tmux attach -t hermes

# Untuk keluar tanpa mematikan proses:
# Tekan Ctrl+B lalu tekan D (Detach)
```

---

## Troubleshooting

1. **Error: `Permission denied` saat eksekusi `./hermes-agent`**
   - Beberapa cloud VM atau pod me-mount `/home` atau `/tmp` dengan opsi `noexec`.
   - Solusi: Jalankan menggunakan bash langsung:
     ```bash
     bash hermes-agent
     # atau background:
     nohup bash hermes-agent > agent.log 2>&1 &
     ```

2. **Log menunjukkan `Failed to send metrics`**
   - Pastikan server Linux bisa mengakses alamat backend (`curl http://IP_BACKEND:3001/health`).
   - Periksa apakah firewall/port `3001` terbuka di sisi backend.

3. **Status server di Dashboard masih offline**
   - Agent akan mengirimkan metrik pertama kali saat dinyalakan, lalu berkala sesuai `HERMES_SCAN_INTERVAL`.
   - Refresh halaman Dashboard setelah agent berjalan.
