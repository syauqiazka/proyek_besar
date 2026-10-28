# Register Server Baru — Hermes

## Cara Pakai

1. Pastikan backend sudah jalan (`start.bat`)
2. Klik 2x `register-server.bat`
3. Ketik nama server (misal: `web-01`, `db-production`, `vm-staging`)
4. Tekan Enter untuk IP backend (default: `localhost:3001`)
5. Selesai! Folder `deploy/<nama-server>/` terbuat otomatis

## Yang Terjadi di Belakang Layar

```
register-server.bat
  ├── Register ke backend → POST /api/v1/agent/register
  │   └── Backend generate server_id + api_key (random 128 char hex)
  │
  ├── Generate .env → deploy/<nama-server>/.env
  │   └── Berisi: HERMES_SERVER_ID, HERMES_API_KEY, HERMES_BACKEND_URL, dll
  │
  └── Copy binary → deploy/<nama-server>/hermes-agent
      └── Binary Linux 10MB siap deploy
```

## Setelah Register

Folder `deploy/<nama-server>/` berisi 2 file:
- `hermes-agent` — binary untuk Linux
- `.env` — konfigurasi dengan kunci unik server ini

### Kirim ke server tujuan:
```bash
scp deploy/<nama-server>/* user@192.168.x.x:~/~
```

### Di server Linux:
```bash
chmod +x ~/hermes-agent
cd ~ && ./hermes-agent
```

### Kalau Permission denied (di VM/pod):
```bash
cd ~ && bash ~/hermes-agent
```

## Server Mana Saja yang Sudah Terdaftar

Cek dari terminal Windows:
```bash
curl http://localhost:3001/api/v1/servers
```

Atau buka dashboard: http://localhost:3000

## Berhentikan Agent di Server

```bash
pkill hermes-agent
```

## Mulai Ulang Agent

```bash
cd ~ && ./hermes-agent
```

## Background (tetap jalan walau terminal ditutup)

```bash
cd ~ && nohup ./hermes-agent > agent.log 2>&1 &
tail -f ~/agent.log
```