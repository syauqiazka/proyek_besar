# Archon — Anti-Slop Guide

"Slop" adalah output generik yang terasa dibuat otomatis: desain template, copy klise, kode berlebihan, dan fitur yang tidak diminta. Dokumen ini berisi aturan agar agent tidak menghasilkan slop di Archon.

Baca bersama `agent.md` (konteks project) dan `design.md` (sistem desain). Kalau ada konflik, `agent.md` dan permintaan eksplisit dari user menang.

## 1. Proses Wajib untuk Pekerjaan UI

Jangan langsung menulis kode UI. Urutannya:

1. **Rencana singkat.** Tulis token yang dipakai (dari `design.md`), layout dalam satu atau dua kalimat, dan komponen apa yang dibutuhkan. Untuk layout yang tidak jelas, gambar wireframe ASCII.
2. **Review terhadap rencana sendiri.** Tanyakan: "Apakah ini yang akan saya hasilkan untuk dashboard apa pun?" Kalau iya, itu default, bukan keputusan. Revisi dan sebutkan apa yang diubah serta alasannya.
3. **Bangun.** Pakai token dari `design.md`. Jangan hardcode hex.
4. **Kritik hasilnya.** Periksa dengan checklist di bagian 7. Kalau environment mendukung, ambil screenshot dan lihat hasilnya sendiri.

Kalau kebutuhan desain tidak ada di `design.md`, tanya. Jangan mengarang warna, font, atau pola baru.

## 2. Slop Visual (dilarang)

**Warna dan permukaan**
- Gradient sebagai dekorasi (hero gradient, kartu gradient, teks gradient, blob blur di background).
- Warna di luar palet `design.md`. Tidak ada ungu-biru "AI" generik, tidak ada neon, tidak ada krem plus terakosta.
- Putih murni atau hitam murni sebagai base.
- Shadow abu-abu lembut yang sama di setiap kartu.
- Glassmorphism dan efek blur tanpa alasan fungsional.

**Layout**
- Konten dipotong jadi deretan kartu identik dengan radius sama. Kalau datanya tabel atau daftar, buat tabel atau daftar.
- Grid tiga kolom "fitur" dengan ikon di atas judul di atas satu kalimat.
- Hero dengan angka besar, label kecil, dan statistik pendukung sebagai tampilan default.
- Semua teks di-center.
- Satu `border-radius` untuk semua elemen.

**Tipografi**
- Label ALL CAPS dengan letter-spacing lebar di atas judul (eyebrow).
- Menonjolkan satu kata dalam judul dengan italic atau warna lain.
- Menambah label kecil di atas konten yang tidak butuh label.
- Font monospace untuk label data kecil hanya supaya terlihat "teknis".
- Mengganti font dari yang ditetapkan di `design.md`.

**Struktur dan dekorasi**
- Penomoran "01 / 02 / 03" pada konten yang bukan urutan sebenarnya.
- String meta dengan titik tengah ("A · B · C") dan label gaya "KATA — fragmen".
- Tanda panah `→` di setiap tombol dan link.
- Ikon dekoratif di samping setiap judul.
- Emoji sebagai ikon UI atau hiasan copy.
- Badge "Baru", "Pro", "AI-powered" yang tidak diminta.
- Ilustrasi stok generik (orang dengan laptop, roket, awan 3D).

**Motion**
- Fade-and-slide-up di setiap section saat scroll.
- Hover transition (naik, membesar, glow) di setiap kartu.
- Animasi loading dekoratif yang lebih menonjol daripada kontennya.

## 3. Slop Copy (dilarang)

- Kalimat promosi kosong: "Revolusi pengelolaan server Anda", "Solusi canggih berbasis AI", "Seamless", "Powerful", "Tingkatkan produktivitas".
- Sapaan basa-basi: "Selamat datang kembali! 🎉", "Mari kita mulai petualangan!".
- Error yang bernada manusia atau meminta maaf: "Oops!", "Maaf, ada yang salah ya".
- Error atau status tanpa penjelasan: "Terjadi kesalahan", "Gagal". Selalu jelaskan penyebab dan langkah berikutnya.
- Tombol generik: "Submit", "OK", "Klik di sini". Gunakan kata kerja spesifik: "Simpan perubahan", "Pindahkan 12 file".
- Nama aksi yang berubah di tengah alur (tombol "Hapus", toast "Dibuang").
- Istilah teknis yang tidak dijelaskan untuk pengguna UMKM (inode, mount point, volume, partition).
- Campur Indonesia dan Inggris tanpa pola. Istilah yang sudah umum boleh (server, backup, dashboard). Sisanya Bahasa Indonesia.
- Data contoh palsu: "Lorem ipsum", "John Doe", "Acme Inc", "Server 1". Pakai data realistis lokal (lihat `design.md` bagian 6).
- Testimoni, angka statistik, atau logo klien karangan.
- Kata yang menjanjikan lebih dari fakta. Archon memberi estimasi, bukan kepastian: jangan menulis "Dijamin aman" atau "Tepat 14 hari".

## 4. Slop Kode (dilarang)

- Komentar yang hanya mengulang kode (`// loop through files`). Komentar hanya untuk alasan, bukan apa.
- Abstraksi prematur: interface, factory, atau layer yang hanya punya satu implementasi dan tidak dibutuhkan sekarang.
- Membuang error dengan `_` atau `catch {}` kosong, terutama di operasi file.
- Import package atau memanggil API yang tidak diverifikasi ada. Cek `go.mod`, `package.json`, dan dokumentasi sebelum memakai. Jangan menebak nama fungsi atau field.
- Menambah dependency baru untuk pekerjaan yang bisa diselesaikan dengan standard library atau dependency yang sudah ada. Kalau memang perlu, jelaskan alasannya.
- Hex warna, ukuran, atau string copy yang di-hardcode di dalam komponen padahal ada token.
- Data mock yang terlihat nyata dan bisa terbawa ke production. Beri penanda jelas dan simpan di tempat khusus mock.
- `TODO` tanpa konteks. Tulis apa, kenapa, dan siapa yang memutuskan.
- Fungsi yang menyentuh file (scan, delete, restore) tanpa unit test.
- Mengubah banyak file yang tidak berhubungan dalam satu perubahan.
- Mengklaim sesuatu sudah berjalan atau teruji padahal tidak dijalankan. Katakan apa yang diuji dan apa yang belum.

## 5. Slop Scope (dilarang)

Ini paling sering terjadi pada solo project.

- Menambah fitur di luar Fase 1: sandbox restore, Telegram, kebijakan per tenant, branding white-label, modul endpoint/network/security.
- Menambah halaman, tab, atau pengaturan yang tidak diminta supaya dashboard "terlihat lengkap".
- Membuat endpoint, tabel, atau kolom baru tanpa konfirmasi, apalagi yang bertentangan dengan `docs/spesifikasi-teknis-fase1.md`.
- Melonggarkan prinsip desain non-negotiable di `agent.md` (konfirmasi eksplisit, trash dulu, prediksi berupa range, audit trail, agent ringan) demi UI yang lebih simpel.
- Refactor besar yang tidak diminta.
- Membuat auto-execute, tombol "hapus semua", atau checkbox destruktif yang tercentang otomatis.

Kalau menemukan sesuatu yang perlu di luar scope, catat dan tanyakan. Jangan langsung dibuat.

## 6. Cara Menghindari Slop Secara Aktif

- **Mulai dari konten nyata.** Layout dibuat dari data yang sebenarnya ditampilkan (storage, prediksi, kandidat file, status backup), bukan dari template dashboard.
- **Satu hal yang menonjol.** Di setiap halaman, satu elemen jadi fokus (misalnya kalimat prediksi di halaman server). Sisanya tenang dan rapi.
- **Setiap elemen punya satu tugas.** Kalau garis, label, ikon, atau badge tidak membawa informasi, hapus.
- **Tanya diri sendiri sebelum selesai:** apa yang bisa dibuang? Sebelum menyerahkan hasil, hapus satu hal yang tidak diperlukan.
- **Ikuti tema pengguna.** Pengguna adalah pemilik toko, bukan engineer. Kalau sebuah kata atau elemen hanya dimengerti DevOps, ganti.

## 7. Checklist Sebelum Menyerahkan Hasil

**Visual**
- [ ] Semua warna berasal dari token di `design.md`, tidak ada hex di komponen.
- [ ] Tidak ada gradient dekoratif, glassmorphism, atau shadow generik.
- [ ] Radius punya hierarki (bukan satu nilai untuk semua).
- [ ] Tidak ada eyebrow ALL CAPS, penomoran palsu, titik tengah di meta, atau `→` di semua tombol.
- [ ] Teks rata kiri, sentence case, minimal 14px.
- [ ] Tidak ada animasi yang tidak merespons aksi pengguna.
- [ ] Responsif di 360px, fokus keyboard terlihat, kontras teks 4.5:1.
- [ ] Status tidak hanya dibedakan dengan warna.

**Copy**
- [ ] Bahasa Indonesia, sapaan "Anda", istilah teknis dihindari atau dijelaskan.
- [ ] Tombol memakai kata kerja spesifik dan nama aksi konsisten di seluruh alur.
- [ ] Error menjelaskan penyebab dan langkah berikutnya. Kondisi kosong mengajak bertindak.
- [ ] Tidak ada klaim berlebihan atau angka karangan. Prediksi selalu range.
- [ ] Data contoh realistis dan lokal.

**Kode dan scope**
- [ ] Tidak ada fitur, tabel, atau endpoint di luar Fase 1 tanpa konfirmasi.
- [ ] Tidak ada dependency baru tanpa alasan.
- [ ] Error ditangani eksplisit. Fungsi operasi file punya unit test.
- [ ] Semua import dan API sudah diverifikasi ada.
- [ ] Prinsip desain di `agent.md` tidak dilanggar.
- [ ] Saya menyebutkan apa yang sudah dijalankan atau diuji dan apa yang belum.

## 8. Kalau Ragu

Tanya satu pertanyaan spesifik kepada user. Jangan menebak lalu membangun banyak hal di atas tebakan itu.