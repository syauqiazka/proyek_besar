# Deploy Archon Agent (Panduan Lengkap — Termasuk Kasus Pod/Kubernetes)

Dokumen ini merangkum pengalaman nyata deploy — termasuk jebakan yang bikin gagal.

## Kasus 1: Server Linux Biasa (VPS / Ubuntu Server)

### Persiapan di Windows (kamu / admin)

1. **Build agent** (kalau belum):
   ```bash
   cd C:\Users\PC05\Documents\M.SyauqiAzka\proyek_besar\agent
   GOOS=linux GOARCH=amd64 go build -o archon-agent cmd/agent/main.go
   ```

2. **Register server di backend** — dapatkan `server_id` + `api_key`:
   ```bash
   curl -X POST http://localhost:3001/api/v1/agent/register \
     -H "Content-Type: application/json" \
     -d '{"hostname":"nama-server","tenant_id":"<TENANT_ID>"}'
   ```

3. **Buat file `.env`** di folder `deploy/nama-server/`:
   ```env
   ARCHON_SERVER_ID=<server_id>
   ARCHON_API_KEY=<api_key>
   ARCHON_BACKEND_URL=http://<IP-BACKEND>:3001/api/v1
   ARCHON_SCAN_INTERVAL=*/5 * * * *
   ARCHON_MOUNT_PATHS=/,/home
   ```

4. **Kirim 2 file ke server tujuan** (ganti user & IP):
   ```bash
   scp agent/archon-agent user@192.168.x.x:~/
   scp deploy/nama-server/.env user@192.168.x.x:~/
   ```

### Di server tujuan

```bash
chmod +x ~/archon-agent
cd ~ && ./archon-agent
```

Agent otomatis baca `.env` (sudah di-hook di main.go) → scan → kirim ke backend.

### Background (biar tetap jalan walau terminal ditutup)

```bash
cd ~ && nohup ./archon-agent > agent.log 2>&1 &
tail -f ~/agent.log    # lihat log
pkill archon-agent     # berhenti
```

---

## Kasus 2: Pod / Kubernetes (seperti `pod-uhammadsyauqi-node01`)

Home directory pod sering di-mount **noexec** — `chmod +x` DAN `./archon-agent` langsung ditolak dengan:
```
bash: ./archon-agent: Permission denied
```

### SOLUSI: jalankan lewat `bash` (tidak perlu execute bit)

```bash
bash ~/archon-agent
```

### Kenapa trik ini jalan?

| Cara | Hasil |
|---|---|
| `./archon-agent` | Gagal — kernel cek `noexec` pada filesystem |
| `bash archon-agent` | Jalan — `bash` baca file sebagai script/bytes dan eksekusi langsung, tidak butuh izin `x` |

### Konfigurasi di pod

File `.env` tetap dibaca normal (bukan executable, hanya data):
```bash
ls -la ~          # pastikan archon-agent + .env ada
bash ~/archon-agent
```

### Background di pod

```bash
cd ~ && nohup bash archon-agent > agent.log 2>&1 &
tail -f agent.log
```

---

## Verifikasi Sukses

1. Di **Windows** cek data masuk:
   ```bash
   curl http://localhost:3001/api/v1/servers
   ```
   Muncul server baru dengan `used_percent` (persen pemakaian disk REAL).

2. Di **dashboard**: refresh http://localhost:3000 — server baru tampil di tabel.

3. Di **pod/VM**: log agent menampilkan:
   ```
   Archon Agent starting...
   Running initial scan...
   Storage metrics sent successfully
   Agent running with scan interval: */5 * * * *
   ```

---

## Troubleshooting

| Gejala | Penyebab | Solusi |
|---|---|---|
| `Permission denied` di home | Filesystem `noexec` | `bash ~/archon-agent` |
| `ARCHON_SERVER_ID environment variable is required` | File `.env` tidak ketemu / beda folder | Pastikan `.env` di folder yang sama dengan binary |
| Agent jalan tapi dashboard kosong | Backend tidak bisa dijangkau pod | Cek `curl http://<IP-BACKEND>:3001/health` dari pod — harus `{"status":"ok"}` |
| Data tidak update | Interval scan belum lewat | Default `*/5 * * * *` = tiap 5 menit; tunggu atau restart agent |
| Firewall blok | Port 3001 tertutup | Buka port 3001 di Windows Firewall / security group |