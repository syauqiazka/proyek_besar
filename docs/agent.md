# Archon — Agent Context

Dokumen ini adalah konteks kerja untuk AI coding agent (Claude Code atau sejenisnya) yang membantu development project Archon. Baca ini dulu sebelum membuat perubahan apa pun ke codebase.

## Dokumen Pendamping (wajib dibaca sesuai jenis pekerjaan)

| File | Baca kalau pekerjaannya |
|---|---|
| `docs/design.md` | Menyentuh UI: dashboard, komponen, warna, tipografi, copy tampilan |
| `docs/anti-slop.md` | Hampir semua pekerjaan. Wajib untuk UI dan copy, dianjurkan untuk kode dan scope |
| `docs/spesifikasi-teknis-fase1.md` | Menyentuh skema database atau endpoint API |

Aturan:
- Pekerjaan UI: baca `design.md` dan `anti-slop.md` dulu, lalu buat rencana singkat sebelum menulis kode.
- Jangan memasukkan aturan desain ke file ini. Aturan visual dan copy hidup di `design.md`.
- Kalau `anti-slop.md` atau `design.md` bertentangan dengan file ini atau permintaan eksplisit dari user, file ini dan permintaan user yang menang.
- Perubahan di file ini (scope, tabel, endpoint) harus disinkronkan ke `docs/spesifikasi-teknis-fase1.md`.

## Tentang Project

Archon adalah self-healing & predictive storage monitor — sistem yang memantau kapasitas storage server, memprediksi kapan server akan penuh, dan membantu membersihkan file secara aman. Target pasar: UMKM non-teknologi, digital agency (white-label), dan pemilik e-commerce lokal di Indonesia yang tidak punya DevOps engineer.

Status saat ini: **solo project, Fase 1 (MVP)**. Fokus ke storage monitor + backup status check dulu — jangan tambah scope di luar ini tanpa dikonfirmasi.

### Dua mode koneksi ke server klien

1. **Mode SSH tanpa agent (utama, default).** Backend terhubung ke server klien lewat SSH, menjalankan perintah allowlist secara berkala, dan mengumpulkan data. Onboarding lewat wizard "Hubungkan server" di dashboard, tanpa instalasi binary.
2. **Mode agent (opsional).** Agent Go dipasang di server untuk pemantauan lebih sering. Bukan syarat. Jangan membuat fitur baru yang hanya berjalan lewat agent.

Segmen yang belum terjangkau: shared hosting/cPanel tanpa akses SSH.

## Prinsip Desain Produk (non-negotiable)

Prinsip ini tentang perilaku produk, keselamatan data, dan keamanan akses. Desain visual ada di `docs/design.md`.

1. **Tidak ada aksi destruktif tanpa konfirmasi eksplisit.** Endpoint cleanup selalu butuh `confirmed: true` dari user, tidak ada auto-execute. Konfirmasi lewat WhatsApp berupa balasan eksplisit ("YA") setelah ringkasan aksi ditampilkan.
2. **Semua delete file lewat snapshot/trash dulu**, permanent delete baru terjadi setelah window waktu (default 48 jam), dengan opsi restore. Satu pengecualian terbatas: kategori "recipe reclaimable" (lihat bagian Cleanup), yang tidak lewat trash tetapi punya syarat lebih ketat.
3. **Prediksi selalu berupa range (`min`/`max`/`best_estimate`)**, bukan angka tunggal — supaya tidak menyesatkan klien. Kalau data kurang dari 7 hari, tampilkan status "data belum cukup", jangan memaksa angka.
4. **Audit trail wajib** untuk setiap aksi cleanup: siapa trigger, file apa saja, berapa bytes freed. Setiap perintah SSH yang dijalankan juga dicatat: server, perintah (template + parameter), waktu mulai dan selesai, exit code, siapa/apa pemicunya.
5. **Ringan di server klien**: target di bawah 1% CPU/RAM rata-rata. Berlaku untuk agent dan untuk perintah SSH (lihat Beban di Server Klien). Uji beban sebelum merge.
6. **MVP dulu, robust belakangan.** Untuk fase ini: backup verification hanya cek status (bukan sandbox test-restore penuh), satu channel notifikasi (WhatsApp), satu set policy filter default (belum per-tenant kustom). Jangan over-engineer di awal.
7. **Hanya perintah allowlist.** Archon tidak menjalankan perintah bebas di server klien. Tidak ada web terminal, tidak ada eksekusi string shell dari input user.
8. **Kredensial SSH tidak pernah tampil atau tercatat.** Private key dienkripsi at rest, tidak muncul di UI, log, response API, atau pesan error.
9. **Least privilege dan read-only sebagai default.** Server baru terhubung dalam mode read-only. Aksi cleanup baru bisa dipakai setelah pemilik server mengaktifkannya secara eksplisit per server.

## Keamanan Koneksi SSH

Menyimpan akses ke server klien menjadikan Archon target bernilai tinggi. Perlakukan area ini seperti cleanup: paling sensitif.

- **User khusus `archon`** di server klien, dibuat oleh skrip onboarding. Tanpa sudo penuh. Hak tambahan hanya untuk perintah spesifik yang dibutuhkan resep tertentu, lewat konfigurasi sudoers yang eksplisit dan sempit.
- **Autentikasi hanya dengan key** (tanpa password). Keypair per server, bukan satu key untuk semua klien.
- **Enkripsi private key at rest** dengan kunci master yang dipisahkan dari database (env/secret manager). Detail skema enkripsi harus dikonfirmasi sebelum diimplementasikan.
- **IP keluar backend yang statis**, agar klien bisa membatasi SSH hanya dari IP itu. Tampilkan IP ini di wizard onboarding.
- **Verifikasi host key.** Simpan fingerprint host saat koneksi pertama dan tolak koneksi kalau berubah (tampilkan peringatan, jangan diam-diam menerima). Jangan memakai `InsecureIgnoreHostKey` di luar test.
- **Allowlist perintah** di `backend/internal/ssh/`. Setiap perintah adalah template tetap dengan parameter yang divalidasi ketat:
  - path harus absolut, tanpa `..`, dan berada di dalam root yang diizinkan
  - parameter di-quote dan divalidasi, tidak ada interpolasi string mentah dari user
  - perintah di luar allowlist ditolak dan dicatat
- **Timeout dan batas output** untuk setiap perintah.
- **Rate limit** per server agar koneksi SSH tidak membanjiri server klien.
- Setiap fungsi di `internal/ssh/` (allowlist, validasi parameter, eksekusi) wajib punya unit test, termasuk kasus penolakan (path traversal, karakter shell, perintah di luar allowlist).

## Beban di Server Klien

Perintah yang berat (`du`, `find`) bisa membebani disk klien dan melanggar prinsip #5.
- Jalankan dengan `nice` dan `ionice`.
- Timeout dan batas kedalaman direktori.
- Interval minimum antar scan per server, dan jangan scan ulang kalau scan sebelumnya belum selesai.
- Scan berat tidak berjalan serentak untuk server yang sama.

## Cleanup: Dua Kategori

**A. File kandidat (default).** File yang terdeteksi lewat scan. Wajib: policy filter (whitelist/blacklist), dry-run, konfirmasi, pindah ke trash, window restore 48 jam, audit trail.

**B. Recipe reclaimable.** Resep pembersihan yang sudah dikenal aman dan didefinisikan di kode (bukan dari input user): cache apt, journal log lama, log yang sudah dirotasi, image Docker yang menggantung, cache aplikasi. Karena memindahkan ke trash di disk yang sama tidak membebaskan ruang, kategori ini tidak lewat trash. Syaratnya lebih ketat:
- Dry-run wajib dijalankan dulu dan menampilkan estimasi ruang yang dibebaskan.
- Konfirmasi eksplisit, dengan ringkasan yang menyebut bahwa aksi ini tidak bisa dikembalikan.
- Hanya resep yang terdaftar. Tidak ada resep dinamis.
- Tiap resep punya unit test dan dokumentasi: apa yang dijalankan, apa yang dihapus, kenapa aman.
- Audit trail sama lengkapnya dengan kategori A.

Strategi trash untuk kategori A (volume terpisah, kompresi, atau lainnya) belum diputuskan. Lihat Keputusan Terbuka.

## Tech Stack

| Layer | Teknologi |
|---|---|
| Koneksi ke server klien | SSH dari backend (Go, `golang.org/x/crypto/ssh`; verifikasi versi di `go.mod`) |
| Agent (opsional) | Go (single binary, cross-compile untuk Linux) |
| Backend API | Go (Fiber) |
| Database | PostgreSQL (via Supabase) |
| Dashboard | Next.js + Tailwind CSS |
| Notifikasi | WhatsApp via Wablas/Fonnte, di belakang interface tipis di `internal/notify/` |
| Background jobs | Go + robfig/cron |
| Hosting backend | VPS Singapore/Jakarta dengan IP statis |
| Hosting dashboard | Vercel |

## Struktur Repo (target)

```
archon/
├── agent/                  # Go agent OPSIONAL, di-deploy ke server klien
│   ├── cmd/agent/main.go
│   ├── internal/scanner/
│   ├── internal/reporter/
│   └── internal/config/
├── backend/                # Go API (Fiber)
│   ├── cmd/api/main.go
│   ├── internal/handlers/    # HTTP handlers per endpoint
│   ├── internal/ssh/         # koneksi SSH, allowlist perintah, validasi parameter
│   ├── internal/onboarding/  # generator skrip hubungkan server, deteksi stack
│   ├── internal/analysis/    # breakdown pemakaian disk, deteksi lonjakan
│   ├── internal/prediction/  # prediction engine (cron job)
│   ├── internal/cleanup/     # cleanup pipeline + safety net
│   ├── internal/recipes/     # resep cleanup kategori B
│   ├── internal/backup/      # backup status check
│   ├── internal/notify/      # notification service (WhatsApp)
│   └── internal/db/          # migrations, queries
├── dashboard/              # Next.js frontend
│   ├── app/
│   ├── components/
│   └── lib/
└── docs/
    ├── spesifikasi-teknis-fase1.md
    ├── agent.md            # file ini
    ├── design.md           # sistem desain: palet, tipografi, komponen, copy UI
    └── anti-slop.md        # aturan agar output AI tidak generik
```

## Fitur Fase 1

1. **Wizard "Hubungkan server".** Dashboard membuat keypair dan menampilkan satu skrip untuk disalin klien: membuat user `archon`, memasang public key, membatasi akses. Lalu tes koneksi dan deteksi otomatis (OS, web server, Docker, database, framework umum).
2. **Monitoring storage via SSH** (mode utama) dan via agent (opsional).
3. **Prediksi** berupa range, dengan status "data belum cukup" di minggu pertama.
4. **Analisis pemakaian disk:** folder dan file terbesar, serta deteksi lonjakan mendadak (misalnya kenaikan tajam dalam 24 jam terakhir).
5. **Cleanup kategori A** (file kandidat) dan **kategori B** (recipe reclaimable), sesuai aturan di atas.
6. **Cek status backup:** umur file backup, perubahan ukuran yang drastis dibanding backup sebelumnya, tes integritas arsip tanpa mengekstrak (misalnya `tar -tzf`, `gzip -t`). UI harus jujur bahwa ini bukan tes restore.
7. **WhatsApp:** notifikasi alert dengan deduplikasi dan ambang yang jelas (hindari spam), serta command `/cek-storage`, `/cek-backup`, `/bersihkan` dengan konfirmasi balasan "YA".

## Skema Database (ringkasan)

Referensi lengkap ada di `docs/spesifikasi-teknis-fase1.md` (perlu diperbarui untuk tabel baru). Tabel utama Fase 1 MVP:

- `tenants` — data klien (sudah siapkan `parent_agency_id` untuk white-label masa depan, tapi belum dipakai aktif di Fase 1)
- `servers` — server yang dimonitor, termasuk `connection_mode` (`ssh` / `agent`) dan `cleanup_enabled` (default false)
- `server_connections` — data koneksi SSH: host, port, username, private key terenkripsi, fingerprint host, status terakhir
- `ssh_command_logs` — audit setiap perintah SSH
- `agent_api_keys` — autentikasi agent (opsional)
- `storage_snapshots` — time-series data disk usage
- `disk_usage_breakdowns` — hasil analisis folder/file terbesar
- `file_candidates` — file yang terdeteksi bisa dibersihkan (kategori A)
- `cleanup_actions` — audit trail aksi cleanup, dengan kolom kategori (`file_candidate` / `recipe`) dan `recipe_id` bila relevan
- `backups` — metadata job backup klien
- `backup_verifications` — MVP: hanya status check dan tes integritas, belum sandbox restore
- `notifications` — log notifikasi terkirim
- `users` — user dashboard

## API Endpoints (ringkasan)

Referensi lengkap di `docs/spesifikasi-teknis-fase1.md`. Base path `/api/v1`.

Onboarding dan koneksi:
- `POST /servers` — buat server baru dan keypair, kembalikan skrip onboarding
- `POST /servers/{id}/connection-test` — tes koneksi SSH dan deteksi stack
- `PATCH /servers/{id}/settings` — aktifkan/nonaktifkan cleanup (default read-only)
- `GET /servers/{id}/command-logs` — riwayat perintah SSH

Agent (opsional):
- `POST /agent/register`
- `POST /agent/{server_id}/metrics`
- `POST /agent/{server_id}/file-candidates`
- `POST /agent/{server_id}/backup-status`

Monitoring dan analisis:
- `GET /servers/{id}/prediction` — range estimasi hari sampai penuh, atau status data belum cukup
- `GET /servers/{id}/disk-breakdown` — folder/file terbesar dan lonjakan

Cleanup:
- `GET /servers/{id}/cleanup-candidates` — kategori A
- `GET /servers/{id}/cleanup-recipes` — kategori B, beserta estimasi
- `POST /servers/{id}/cleanup` — trigger cleanup (butuh `confirmed: true`; body menentukan kategori A atau `recipe_id`; server harus `cleanup_enabled`)
- `POST /servers/{id}/cleanup?dry_run=true` — simulasi tanpa eksekusi
- `POST /cleanup-actions/{id}/restore` — undo kategori A dalam window snapshot (kategori B tidak bisa di-restore)

Notifikasi:
- `POST /notifications/whatsapp/webhook` — terima command masuk (`/cek-storage`, `/cek-backup`, `/bersihkan`, balasan konfirmasi)

## Konvensi Kode

- Go: ikuti `gofmt` standar, error handling eksplisit (jangan `_` buang error tanpa alasan jelas).
- Nama tabel dan kolom database: `snake_case`.
- Endpoint API: `kebab-case` untuk path, response JSON pakai `snake_case` untuk konsistensi dengan database.
- Setiap fungsi yang menyentuh operasi file (scan, delete, restore) dan setiap fungsi di `internal/ssh/` dan `internal/recipes/` wajib punya unit test, karena ini area paling berisiko di produk ini.
- Jangan pernah mencatat private key, passphrase, atau isi file klien di log aplikasi.
- Commit message: awali dengan modul yang diubah, misal `agent: fix disk scan timeout`, `backend: add cleanup confirmation check`, `ssh: reject paths outside allowed roots`.
- Dashboard: warna, font, dan copy UI mengikuti `docs/design.md`. Tidak ada hex yang di-hardcode di komponen, pakai token Tailwind.

## Yang Perlu Dicek Sebelum Merge Perubahan ke Fitur Cleanup

Karena ini area paling sensitif di produk:
- [ ] Ada dry-run mode yang bisa dites tanpa eksekusi nyata?
- [ ] Kategori A: file masuk snapshot/trash dulu, bukan langsung permanent delete?
- [ ] Kategori B: hanya resep terdaftar, dry-run wajib, konfirmasi menyebut aksi tidak bisa dikembalikan?
- [ ] Audit trail (siapa, apa, kapan, berapa bytes) tercatat dengan benar?
- [ ] Policy filter (whitelist/blacklist) dicek sebelum file masuk daftar kandidat?
- [ ] Server dicek `cleanup_enabled` sebelum aksi dijalankan?
- [ ] Kalau ada UI-nya: dialog konfirmasi mengikuti `docs/design.md` (tombol default-fokus "Batal", jumlah file dan ukuran disebut jelas, info window restore tampil untuk kategori A)?

## Yang Perlu Dicek Sebelum Merge Perubahan ke Modul SSH

- [ ] Semua perintah berasal dari allowlist, parameter divalidasi dan di-quote?
- [ ] Ada unit test untuk penolakan: path traversal, karakter shell, perintah di luar allowlist?
- [ ] Private key tidak muncul di log, response, atau pesan error?
- [ ] Host key diverifikasi, tidak ada `InsecureIgnoreHostKey` di kode non-test?
- [ ] Setiap perintah punya timeout dan batas output?
- [ ] Perintah berat memakai `nice`/`ionice` dan batas kedalaman?
- [ ] Setiap eksekusi tercatat di `ssh_command_logs`?

## Yang Perlu Dicek Sebelum Merge Perubahan UI Dashboard

- [ ] Semua warna dari token di `docs/design.md`, bukan hex langsung.
- [ ] Checklist di `docs/anti-slop.md` bagian 7 sudah dilewati.
- [ ] Prediksi ditampilkan sebagai range, atau status "data belum cukup".
- [ ] Copy dalam Bahasa Indonesia, tanpa istilah teknis yang tidak dijelaskan.
- [ ] Tampilan dicek di lebar 360px.

## Keputusan Terbuka (jangan diputuskan sendiri oleh agent)

1. **Strategi trash kategori A:** volume terpisah, kompresi, atau batasan lain, supaya cleanup benar-benar membebaskan ruang.
2. **Skema enkripsi private key** dan penyimpanan kunci master (env, secret manager, atau lainnya).
3. **IP keluar statis** untuk backend: penyedia dan cara menampilkannya di wizard.
4. **Interval polling SSH** per server dan batas beban yang diterima.
5. **Daftar resep awal kategori B** dan hak sudo spesifik yang dibutuhkan masing-masing.
6. **Abstraksi gateway WhatsApp:** cukup satu interface kecil agar bisa pindah ke WhatsApp Business API resmi nanti.

## Hal yang Sengaja Belum Dikerjakan di Fase 1 (jangan ditambah tanpa diskusi)

- Web terminal / akses shell interaktif dari dashboard
- Auto-fix atau auto-execute tanpa konfirmasi
- Sandbox restore otomatis untuk backup verification
- Integrasi Telegram (baru WhatsApp)
- Multi-tenant policy per klien (masih satu default policy)
- White-label branding di dashboard (token warna di `design.md` sudah disiapkan lewat CSS variable, tapi fiturnya belum dibangun)
- Laporan bulanan PDF untuk agency
- Dark mode
- Dukungan shared hosting/cPanel tanpa SSH
- Modul endpoint health, network health, security signal, serta monitoring CPU/RAM/SSL/uptime (rencana Fase 2 ke atas)