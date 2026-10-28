# Hermes — Monitoring Storage yang Menjaga Server Kamu

Hermes adalah sistem yang **mengawasi kapasitas penyimpanan (storage) server** secara otomatis. Dia memberi tahu kamu sebelum server penuh, memperkirakan kapan server akan penuh, dan membantu membersihkan file sampah dengan aman.

Dibuat khusus untuk UMKM, digital agency, dan pemilik toko online Indonesia yang **tidak punya tim IT khusus**.

> **Baru pertama kali pakai?** Baca **[TUTORIAL.md](TUTORIAL.md)** — panduan lengkap langkah demi langkah dari nol sampai bisa.

---

## 1. Apa Itu Hermes? (Penjelasan Sederhana)

Bayangkan server seperti **gudang penyimpanan**. Kalau gudang penuh, barang baru tidak bisa masuk, website bisa down, dan bisnis berhenti.

Hermes bekerja seperti **satpam gudang yang pintar**:
- 🛎️ **Memantau** — tahu berapa banyak ruang gudang yang terpakai setiap saat
- 🔮 **Memprediksi** — memperkirakan "gudang ini bakal penuh dalam 15 hari"
- 🧹 **Membantu bersih-bersih** — menemukan file sampah seperti log lama, file backup ganda, file sementara
- 💾 **Mengawasi backup** — memastikan data penting ter-backup dengan benar
- 📱 **Memberi tahu via WhatsApp** — muncul notifikasi sebelum terjadi masalah

---

## 2. Bagian-Bagian Hermes

| Bagian | Namanya | Fungsinya | Analogi |
|---|---|---|---|
| **Agent** | `hermes-agent` | Dipasang DI DALAM server yang dipantau. Tugasnya scan disk dan lapor. | Satpam yang stand by di dalam gudang |
| **Backend** | `hermes-backend.exe` | Pusat data. Menerima laporan semua agent, menyimpan ke database, menghitung prediksi. | Kantor pusat pengelolaan |
| **Dashboard** | Website (Next.js) | Tampilan di browser untuk melihat semua server dalam satu layar. | Layar monitor CCTV |
| **Database** | Supabase (PostgreSQL) | Tempat penyimpanan semua data, di cloud (online). | Buku catatan besar di cloud |

---

## 3. Persiapan Sebelum Mulai

Kamu butuh 3 hal ini di komputer kamu (Windows):

| Kebutuhan | Cara Dapat | Untuk Apa |
|---|---|---|
| **Go** (bahasa pemrograman) | Download di https://go.dev/dl/ lalu install | Menjalankan backend & agent (sudah terinstall di komputer ini) |
| **Node.js** | Download di https://nodejs.org/ (pilih versi LTS) | Menjalankan dashboard |
| **Akun Supabase gratis** | Daftar di https://supabase.com/ | Database cloud untuk simpan semua data |

> 💡 **Jangan khawatir** — kalau kamu ikuti panduan di README ini urut dari atas ke bawah, semuanya akan berjalan.

---

## 4. Satu Kali Install (Setup Awal)

Buka **Command Prompt** atau **Terminal** di folder project:

```bash
cd C:\Users\PC05\Documents\M.SyauqiAzka\proyek_besar
```

### 4a. Install Backend (Go)

```bash
cd backend
go mod tidy
go build -o hermes-backend.exe cmd/api/main.go
cd ..
```

### 4b. Install Dashboard (Node.js)

```bash
cd dashboard
npm install
cd ..
```

### 4c. Buat Akun Supabase (Database Cloud)

1. Buka https://supabase.com/ → klik **Start your project** → daftar gratis
2. Buat project baru, beri nama apa saja (misal `hermes`)
3. Ingat **password database** yang kamu buat (simpan baik-baik!)
4. Tunggu sampai project jadi (~2 menit)

---

## 5. Mengisi File Konfigurasi (.env)

File `.env` adalah **file konfigurasi rahasia** project ini — berisi kunci akses database dan API. Jangan pernah bagikan ke orang lain!

File ini lokasinya di: `C:\Users\PC05\Documents\M.SyauqiAzka\proyek_besar\.env`

Isi file-nya seperti ini (ganti bagian yang ada `<...>` dengan data milik kamu):

```env
PORT=3001
DATABASE_URL=postgresql://postgres.<ref>:<password-db>@aws-0-<region>.pooler.supabase.com:6543/postgres
NEXT_PUBLIC_SUPABASE_URL=https://<project-ref>.supabase.co
NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY=sb_publishable_xxxxxxxxxxxxx
WHATSAPP_API_URL=https://api.wablas.com/v125
WHATSAPP_SECRET=<secret-wablas-kamu>
ENVIRONMENT=development
```

### Cara mengambil DATABASE_URL dari Supabase (langkah demi langkah):

1. Masuk ke https://supabase.com/dashboard
2. Klik project kamu
3. Di menu kiri, klik **Settings** (ikon gigi) → **Database**
4. Cari bagian **Connection string** → pilih **URI**
5. Di dropdown, pilih **Transaction pooler**
6. Klik tombol **copy** pada connection string
7. Tempel ke file `.env` — ganti `<password>` dengan password database yang kamu buat di langkah 4c

### Cara mengambil URL + kunci publik Supabase:

1. Masih di dashboard Supabase, klik menu **Settings** → **API**
2. Salin **Project URL** → masukkan sebagai `NEXT_PUBLIC_SUPABASE_URL`
3. Salin **anon / public key** (awalannya `sb_publishable_...`) → masukkan sebagai `NEXT_PUBLIC_SUPABASE_PUBLISHABLE_KEY`

---

## 6. Menjalankan Hermes

### Cara Paling Mudah (disarankan untuk awam)

Ada 2 file bantuan di folder project:

```
proyek_besar/
├── start.bat     ← klik 2x untuk MENJALANKAN semua
├── stop.bat      ← klik 2x untuk MEMATIKAN semua
└── README.md     ← file ini
```

1. **Klik 2x `start.bat`**
2. Tunggu 10 detik — browser akan terbuka otomatis
3. Dashboard muncul di: **http://localhost:3000**

Untuk mematikan semua: **klik 2x `stop.bat`**

### Cara Manual (jika mau kontrol penuh)

**Terminal 1 — Backend** (wajib dijalankan dari folder project ROOT):

```bash
cd C:\Users\PC05\Documents\M.SyauqiAzka\proyek_besar
backend\hermes-backend.exe
```

**Terminal 2 — Dashboard:**

```bash
cd C:\Users\PC05\Documents\M.SyauqiAzka\proyek_besar\dashboard
npm run dev
```

---

> ⚠️ **PENTING — Backend harus dijalankan dari folder ROOT (proyek_besar), BUKAN dari folder backend.** Kalau dijalankan dari folder backend, file `.env` tidak ketemu dan program langsung berhenti dengan pesan `DATABASE_URL environment variable is required`.

---

## 7. Memahami Dashboard (Layar Monitor)

Saat dashboard terbuka, ini yang kamu lihat:

### Kartu Statistik (4 kartu di atas)
- **Server Aktif** — berapa server yang sedang dipantau
- **Total Terpakai** — total penyimpanan yang terpakai semua server
- **Rata-rata Pemakaian** — persentase pemakaian rata-rata
- **Server Berisiko** — berapa server yang hampir penuh (≥85%) — kartu ini merah kalau ada

### Tabel Server (daftar semua server)
| Kolom | Artinya |
|---|---|
| Server | Nama server (misal `web-01`) |
| Status | `active` = sehat, `inactive` = bermasalah |
| Pemakaian | Persen terpakai + bar warna: 🟢 aman, 🟡 waspada (≥75%), 🔴 bahaya (≥90%) |
| Terpakai / Tersedia | Ukuran dalam GB/TB |
| Prediksi Penuh | Estimasi berapa hari lagi server penuh (best estimate + range min/max) |
| Last Seen | Terakhir kali server lapor |

### Tabel Kandidat Cleanup
Daftar file sampah yang ditemukan (log lama, file sementara, dsb.) — beserta ukuran dan kategori. Ini **hanya rekomendasi** — Hermes TIDAK menghapus apa pun tanpa persetujuan kamu.

---

## 9. Menambah Server Baru

### Cara Mudah (1 langkah)

Klik 2x `register-server.bat` → ketik nama server → selesai.

```
Masukkan nama server (misal: web-01): production-db
IP backend (Enter untuk localhost:3001): [Enter]

[1/3] Register "production-db" ke backend...
[2/3] Generate file konfigurasi...
[3/3] Siap deploy...

 SERVER "production-db" BERHASIL DIDAFTARKAN!
 File siap deploy di: deploy\production-db\
```

Folder `deploy\production-db\` berisi:
- `hermes-agent` (binary Linux)
- `.env` (konfigurasi dengan server_id + api_key otomatis)

Tinggal kirim ke server tujuan.

### Cara Manual (kalau mau lebih kontrol)

Buka browser, buka halaman ini (ganti `<NAMA-SERVER>` misal `web-02`):

```
http://localhost:3001/api/v1/agent/register
```

Atau via terminal:

```bash
curl -X POST http://localhost:3001/api/v1/agent/register \
  -H "Content-Type: application/json" \
  -d '{"hostname":"web-02","tenant_id":"<TENANT_ID>"}'
```

Response berisi 2 kunci penting: **`server_id`** dan **`api_key`**. Simpan keduanya.

**Langkah 2 — Salin agent ke server**

Copy file `hermes-agent` (dari folder `agent/`) ke server Linux tujuan.

**Langkah 3 — Buat file konfigurasi agent di server tujuan**

Buat file `.env` di sebelah binary agent, isinya:

```env
# ID server & kunci dari Langkah 1
HERMES_SERVER_ID=<server_id>
HERMES_API_KEY=<api_key>

# Alamat backend — ganti dengan IP/server tempat backend berjalan
HERMES_BACKEND_URL=http://<IP-SERVER-BACKEND>:3001/api/v1

# Berapa sering scan (format cron): "0 * * * *" = setiap jam
HERMES_SCAN_INTERVAL=0 * * * *

# Folder yang dipantau (pisahkan dengan koma)
HERMES_MOUNT_PATHS=/,/var,/home
```

**Langkah 4 — Jalankan agent**

```bash
chmod +x hermes-agent
./hermes-agent
```

> ⚠️ **Kalau muncul `Permission denied`** padahal sudah `chmod +x` (biasanya di VM/pod milik penyedia hosting):
> ```bash
> bash ~/hermes-agent
> ```
> Itu trik untuk filesystem yang melarang eksekusi langsung (`noexec`). Agent tetap jalan normal lewat bash.

**Langkah 5 — Cek di dashboard**

Refresh http://localhost:3000 — server baru muncul di tabel.

> 💡 Semakin banyak server, tinggal ulangi Langkah 1–4 dengan nama berbeda.
> Satu backend + satu dashboard bisa menangani banyak server sekaligus.
> Panduan deploy lengkap dengan troubleshooting ada di `deploy/ubuntu-vm/DEPLOY.md`.

### Menjalankan agent di background (tetap jalan walau terminal ditutup)

```bash
cd ~ && nohup ./hermes-agent > agent.log 2>&1 &
tail -f ~/agent.log    # lihat log
pkill hermes-agent     # berhenti
```

### Verifikasi data masuk

```bash
curl http://localhost:3001/api/v1/servers
```
Server baru muncul dengan `used_percent` (persen pemakaian disk asli).

---

## 9. Menghubungkan Notifikasi WhatsApp (Opsional)

Hermes bisa kirim notifikasi ke WhatsApp kamu kalau ada yang bermasalah (misal storage hampir penuh).

1. Daftar di https://wablas.com/ (atau provider lain seperti Fonnte)
2. Setelah dapat API token, isi di file `.env`:
   ```env
   WHATSAPP_API_URL=https://api.wablas.com/v125
   WHATSAPP_SECRET=<token-dari-wablas>
   ```
3. Restart backend (`stop.bat` lalu `start.bat`)

Command WhatsApp yang bisa dipakai (kirim ke nomor bot):
- `/cek-storage` — status storage semua server
- `/cek-backup` — status backup
- `/help` — daftar command

---

## 10. Struktur Folder Project

```
proyek_besar/
├── agent/                  # Program Agent (dipasang di server yang dipantau)
│   ├── hermes-agent        # Binary agent untuk Linux
│   └── cmd/agent/main.go   # Kode sumber agent
├── backend/                # Program Backend (pusat data)
│   ├── hermes-backend.exe  # Binary backend untuk Windows
│   └── internal/           # Kode sumber backend (handler, db, dsb)
├── dashboard/              # Website Dashboard
│   └── app/page.tsx        # Halaman utama dashboard
├── docs/agent.md           # Dokumentasi teknis untuk developer
├── .env                    # File konfigurasi rahasia (JANGAN dibagikan!)
├── start.bat               # Jalankan semua
├── stop.bat                # Matikan semua
└── README.md               # File ini
```

---

## 11. Troubleshooting (Masalah Umum & Solusinya)

### "Gagal terhubung ke backend" di dashboard
- Pastikan `hermes-backend.exe` jalan. Cek dengan klik 2x `start.bat`
- Cek di browser: buka http://localhost:3001/health — harus muncul `{"status":"ok"}`
- Kalau tidak, tekan **Ctrl+F5** di dashboard (refresh paksa)

### Backend langsung keluar dengan error `DATABASE_URL environment variable is required`
- Backend dijalankan dari folder yang salah. Wajib dari folder `proyek_besar` (ROOT), bukan `backend`
- Atau file `.env` belum dibuat

### Pesan `password authentication failed`
- Password database di file `.env` salah
- Ganti password database di Supabase: masukkan password yang benar dari situs Supabase (Settings → Database → Connection string → ganti `<password>`)
- Password database Supabase BEDA dari password akun login Supabase

### Dashboard tampak lama "Memuat data…"
- Tekan **Ctrl+F5** untuk refresh paksa
- Pastikan backend jalan (lihat troubleshooting pertama)

### Port 3000 atau 3001 sudah dipakai (error "address already in use")
- Artinya ada program lain yang pakai port tersebut
- Jalankan `stop.bat` dulu, lalu `start.bat` lagi

### Pesan "Port 3000 is in use, using available port 3001 instead"
- Ada dashboard yang sudah jalan. Buka saja http://localhost:3000 — yang lama tetap melayani
- Atau jalankan `stop.bat` lalu `start.bat` untuk memulai bersih

### Agent tidak bisa jalan di Windows
- Wajar — agent dibuat untuk **Linux** (server). Windows hanya untuk backend + dashboard
- Deploy agent di server Linux

### Cara tahu semua server hidup?
- Di dashboard, kolom **Last Seen** menunjukkan kapan terakhir server lapor
- Server yang sudah lama tidak lapor kemungkinan mati atau agent-nya berhenti

---

## 12. Pertanyaan Umum (FAQ)

**Q: Apakah Hermes bisa menghapus file sendiri?**
A: Tidak. Hermes hanya memberikan rekomendasi file mana yang aman dibersihkan. Penghapusan selalu butuh persetujuan kamu, dan file masuk ke tempat sampah (snapshot) dulu — masih bisa dipulihkan dalam 48 jam.

**Q: Berapa biaya untuk pakai ini?**
A: Biaya utama adalah Supabase (gratis untuk pemakaian kecil) dan server tempat backend berjalan. Satu akun Supabase gratis sudah cukup untuk project ini.

**Q: Apakah aman?**
A: Ya. Koneksi ke database pakai enkripsi (SSL). File `.env` berisi kunci rahasia — jangan pernah bagikan atau commit ke GitHub.

**Q: Server saya pakai Windows, bisa dipantau?**
A: Saat ini agent baru untuk Linux. Versi Windows menyusul.

**Q: Perlu beli domain / hosting khusus?**
A: Untuk dicoba di rumah, cukup komputer kamu sebagai backend. Untuk produksi, sebaiknya backend ditaruh di VPS (server cloud) supaya menyala 24 jam.

---

## 13. Dukungan & Pengembangan

Project ini dikembangkan bertahap (fase MVP dulu). Fitur yang sedang dikembangkan:
- [x] Monitoring storage + prediksi penuh
- [x] Kandidat file cleanup (rekomendasi)
- [x] Dashboard multi-server
- [x] Koneksi WhatsApp webhook (kerangka)
- [ ] Notifikasi WhatsApp aktif (Wablas terhubung)
- [ ] Aksi cleanup dengan konfirmasi
- [ ] Versi agent Windows

Untuk developer: dokumentasi teknis lengkap ada di `docs/agent.md`.