# Deploy Hermes Agent (Panduan Lengkap — Termasuk Kasus Pod/Kubernetes)

Dokumen ini merangkum pengalaman nyata deploy — termasuk jebakan yang bikin gagal.

## Kasus 1: Server Linux Biasa (VPS / Ubuntu Server)

### Persiapan di Windows (kamu / admin)

1. **Build agent** (kalau belum):
   ```bash
   cd C:\Users\PC05\Documents\M.SyauqiAzka\proyek_besar\agent
   GOOS=linux GOARCH=amd64 go build -o hermes-agent cmd/agent/main.go
   ```

2. **Register server di backend** — dapatkan `server_id` + `api_key`:
   ```bash
   curl -X POST http://localhost:3001/api/v1/agent/register \
     -H "Content-Type: application/json" \
     -d '{"hostname":"nama-server","tenant_id":"<TENANT_ID>"}'
   ```

3. **Buat file `.env`** di folder `deploy/nama-server/`:
   ```env
   HERMES_SERVER_ID=<server_id>
   HERMES_API_KEY=<api_key>
   HERMES_BACKEND_URL=http://<IP-BACKEND>:3001/api/v1
   HERMES_SCAN_INTERVAL=*/5 * * * *
   HERMES_MOUNT_PATHS=/,/home
   ```

4. **Kirim 2 file ke server tujuan** (ganti user & IP):
   ```bash
   scp agent/hermes-agent user@192.168.x.x:~/
   scp deploy/nama-server/.env user@192.168.x.x:~/
   ```

### Di server tujuan

```bash
chmod +x ~/hermes-agent
cd ~ && ./hermes-agent
```

Agent otomatis baca `.env` (sudah di-hook di main.go) → scan → kirim ke backend.

### Background (biar tetap jalan walau terminal ditutup)

```bash
cd ~ && nohup ./hermes-agent > agent.log 2>&1 &
tail -f ~/agent.log    # lihat log
pkill hermes-agent     # berhenti
```

---

## Kasus 2: Pod / Kubernetes (seperti `pod-uhammadsyauqi-node01`)

Home directory pod sering di-mount **noexec** — `chmod +x` DAN `./hermes-agent` langsung ditolak dengan:
```
bash: ./hermes-agent: Permission denied
```

### SOLUSI: jalankan lewat `bash` (tidak perlu execute bit)

```bash
bash ~/hermes-agent
```

### Kenapa trik ini jalan?

| Cara | Hasil |
|---|---|
| `./hermes-agent` | Gagal — kernel cek `noexec` pada filesystem |
| `bash hermes-agent` | Jalan — `bash` baca file sebagai script/bytes dan eksekusi langsung, tidak butuh izin `x` |

### Konfigurasi di pod

File `.env` tetap dibaca normal (bukan executable, hanya data):
```bash
ls -la ~          # pastikan hermes-agent + .env ada
bash ~/hermes-agent
```

### Background di pod

```bash
cd ~ && nohup bash hermes-agent > agent.log 2>&1 &
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
   Hermes Agent starting...
   Running initial scan...
   Storage metrics sent successfully
   Agent running with scan interval: */5 * * * *
   ```

---

## Troubleshooting

| Gejala | Penyebab | Solusi |
|---|---|---|
| `Permission denied` di home | Filesystem `noexec` | `bash ~/hermes-agent` |
| `HERMES_SERVER_ID environment variable is required` | File `.env` tidak ketemu / beda folder | Pastikan `.env` di folder yang sama dengan binary |
| Agent jalan tapi dashboard kosong | Backend tidak bisa dijangkau pod | Cek `curl http://<IP-BACKEND>:3001/health` dari pod — harus `{"status":"ok"}` |
| Data tidak update | Interval scan belum lewat | Default `*/5 * * * *` = tiap 5 menit; tunggu atau restart agent |
| Firewall blok | Port 3001 tertutup | Buka port 3001 di Windows Firewall / security group |