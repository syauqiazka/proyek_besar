# Hermes — Agent Context

Dokumen ini adalah konteks kerja untuk AI coding agent (Claude Code atau sejenisnya) yang membantu development project Hermes. Baca ini dulu sebelum membuat perubahan apa pun ke codebase.

## Tentang Project

Hermes adalah self-healing & predictive storage monitor — sistem yang memantau kapasitas storage server, memprediksi kapan server akan penuh, dan membantu membersihkan file secara aman. Target pasar: UMKM non-teknologi, digital agency (white-label), dan pemilik e-commerce lokal di Indonesia yang tidak punya DevOps engineer.

Status saat ini: **solo project, Fase 1 (MVP)**. Fokus ke storage monitor + backup status check dulu — jangan tambah scope di luar ini tanpa dikonfirmasi.

## Prinsip Desain (non-negotiable)

1. **Tidak ada aksi destruktif tanpa konfirmasi eksplisit.** Endpoint cleanup selalu butuh `confirmed: true` dari user, tidak ada auto-execute.
2. **Semua delete file lewat snapshot/trash dulu**, permanent delete baru terjadi setelah window waktu (default 48 jam), dengan opsi restore.
3. **Prediksi selalu berupa range (`min`/`max`/`best_estimate`)**, bukan angka tunggal — supaya tidak menyesatkan klien.
4. **Audit trail wajib** untuk setiap aksi cleanup: siapa trigger, file apa saja, berapa bytes freed.
5. **Agent harus ringan**: target di bawah 1% CPU/RAM rata-rata. Uji beban sebelum merge perubahan ke agent.
6. **MVP dulu, robust belakangan.** Untuk fase ini: backup verification cukup status check (bukan sandbox test-restore penuh), satu channel notifikasi (WhatsApp), satu set policy filter default (belum per-tenant kustom). Jangan over-engineer di awal.

## Tech Stack

| Layer | Teknologi |
|---|---|
| Agent | Go (single binary, cross-compile untuk Linux) |
| Backend API | Go (Fiber) |
| Database | PostgreSQL (via Supabase) |
| Dashboard | Next.js + Tailwind CSS |
| Notifikasi | WhatsApp via Wablas/Fonnte |
| Background jobs | Go + robfig/cron |
| Hosting backend | VPS Singapore/Jakarta |
| Hosting dashboard | Vercel |

## Struktur Repo (target)

```
hermes/
├── agent/                 # Go agent, di-deploy ke server klien
│   ├── cmd/agent/main.go
│   ├── internal/scanner/  # scan disk usage & file candidates
│   ├── internal/reporter/ # kirim data ke backend
│   └── internal/config/
├── backend/                # Go API (Fiber)
│   ├── cmd/api/main.go
│   ├── internal/handlers/  # HTTP handlers per endpoint
│   ├── internal/prediction/# prediction engine (cron job)
│   ├── internal/cleanup/   # cleanup pipeline + safety net logic
│   ├── internal/backup/    # backup status check
│   ├── internal/notify/    # notification service (WhatsApp)
│   └── internal/db/        # migrations, queries
├── dashboard/               # Next.js frontend
│   ├── app/
│   ├── components/
│   └── lib/
└── docs/
    ├── spesifikasi-teknis-fase1.md
    └── agent.md            # file ini
```

## Skema Database (ringkasan)

Referensi lengkap ada di `docs/spesifikasi-teknis-fase1.md`. Tabel utama Fase 1 MVP:

- `tenants` — data klien (sudah siapkan `parent_agency_id` untuk white-label masa depan, tapi belum dipakai aktif di Fase 1)
- `servers` — server yang dimonitor
- `agent_api_keys` — autentikasi agent
- `storage_snapshots` — time-series data disk usage
- `file_candidates` — file yang terdeteksi bisa dibersihkan
- `cleanup_actions` — audit trail aksi cleanup
- `backups` — metadata job backup klien
- `backup_verifications` — MVP: hanya status check, belum sandbox restore
- `notifications` — log notifikasi terkirim
- `users` — user dashboard

## API Endpoints (ringkasan)

Referensi lengkap di `docs/spesifikasi-teknis-fase1.md`. Base path `/api/v1`.

- `POST /agent/register` — registrasi agent baru
- `POST /agent/{server_id}/metrics` — agent kirim data storage
- `POST /agent/{server_id}/file-candidates` — agent kirim hasil scan file
- `POST /agent/{server_id}/backup-status` — update status backup
- `GET /servers/{id}/prediction` — estimasi hari sampai penuh
- `GET /servers/{id}/cleanup-candidates` — daftar file rekomendasi cleanup
- `POST /servers/{id}/cleanup` — trigger cleanup (butuh `confirmed: true`)
- `POST /cleanup-actions/{id}/restore` — undo cleanup dalam window snapshot
- `POST /notifications/whatsapp/webhook` — terima command masuk (`/cek-storage`, `/cek-backup`)

## Konvensi Kode

- Go: ikuti `gofmt` standar, error handling eksplisit (jangan `_` buang error tanpa alasan jelas).
- Nama tabel dan kolom database: `snake_case`.
- Endpoint API: `kebab-case` untuk path, response JSON pakai `snake_case` untuk konsistensi dengan database.
- Setiap fungsi yang menyentuh operasi file (scan, delete, restore) wajib punya unit test, karena ini area paling berisiko di produk ini.
- Commit message: awali dengan modul yang diubah, misal `agent: fix disk scan timeout`, `backend: add cleanup confirmation check`.

## Yang Perlu Dicek Sebelum Merge Perubahan ke Fitur Cleanup

Karena ini area paling sensitif di produk:
- [ ] Ada dry-run mode yang bisa dites tanpa eksekusi nyata?
- [ ] File yang dihapus masuk snapshot/trash dulu, bukan langsung permanent delete?
- [ ] Audit trail (siapa, apa, kapan, berapa bytes) tercatat dengan benar?
- [ ] Policy filter (whitelist/blacklist) dicek sebelum file masuk daftar kandidat?

## Hal yang Sengaja Belum Dikerjakan di Fase 1 (jangan ditambah tanpa diskusi)

- Sandbox restore otomatis untuk backup verification (masih status check manual)
- Integrasi Telegram (baru WhatsApp)
- Multi-tenant policy per klien (masih satu default policy)
- White-label branding di dashboard
- Modul endpoint health, network health, security signal (rencana Fase 2 ke atas)
