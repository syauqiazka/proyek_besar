# Archon — Design System

Panduan visual dan copywriting untuk dashboard Archon (`dashboard/`: Next.js + Tailwind). Baca file ini sebelum membuat atau mengubah UI apa pun. Baca juga `anti-slop.md` — file itu berisi daftar hal yang dilarang dan proses self-review.

Kalau sebuah keputusan desain tidak ada di file ini, jangan mengarang sendiri. Tanya dulu.

## 1. Konteks Desain

- **Pengguna:** pemilik UMKM, staf digital agency, pemilik toko online di Indonesia. Mereka bukan engineer. Mereka tidak tahu apa itu inode, mount point, atau partition.
- **Tugas utama dashboard:** dalam 5 detik, pengguna tahu apakah storage servernya aman, kapan kira-kira penuh, dan apa yang bisa dilakukan.
- **Nada:** tenang, jelas, meyakinkan. Produk ini menyentuh file klien, jadi desainnya harus terasa hati-hati, bukan ramai atau main-main.
- **Bahasa UI:** Bahasa Indonesia. Istilah teknis hanya muncul kalau memang tidak ada padanan yang dimengerti pengguna, dan selalu disertai penjelasan singkat.

## 2. Palet Warna

Base palette dari Color Hunt: https://colorhunt.co/palette/f6e5f5fbf4f9f6e7e6b9cced

| Token | Hex | Peran |
|---|---|---|
| `lilac-100` | `#F6E5F5` | Surface sekunder, area yang di-highlight lembut, hover baris tabel |
| `blush-50` | `#FBF4F9` | Background halaman utama |
| `rose-100` | `#F6E7E6` | Surface kartu, panel samping |
| `periwinkle-300` | `#B9CCED` | Aksen lembut: bar progress, chip, garis fokus, ilustrasi |

Keempat warna ini semuanya terang. **Jangan dipakai sebagai warna teks atau sebagai latar tombol berteks putih**, kontrasnya tidak cukup. Karena itu palet diperluas dengan token turunan di bawah. Token turunan dibuat dari hue yang sama dengan base palette supaya tetap harmonis.

### Token turunan (dibuat untuk kebutuhan kontras)

| Token | Hex | Peran |
|---|---|---|
| `ink` | `#2A2640` | Teks utama, judul. Indigo-plum gelap, bukan hitam murni |
| `ink-muted` | `#5E5A78` | Teks sekunder, label, keterangan |
| `line` | `#E3CFE2` | Border dan pemisah. Turunan lebih gelap dari `lilac-100` |
| `accent` | `#3A5A9B` | Tombol utama, link, elemen interaktif. Turunan gelap dari `periwinkle-300` |
| `accent-hover` | `#2F4A82` | Hover/pressed pada `accent` |

### Token status (khusus monitoring storage)

Status storage harus terbaca tanpa bergantung pada warna saja: selalu sertakan ikon dan teks ("Aman", "Perlu perhatian", "Kritis").

| Status | Teks | Latar | Dipakai untuk |
|---|---|---|---|
| `ok` | `#2F6B58` | `#DDEFE8` | Storage aman, backup berhasil |
| `warn` | `#8A5A0F` | `#F8EBCB` | Perkiraan penuh dalam waktu dekat, backup terlambat |
| `danger` | `#A8324D` | `#F9DDE3` | Hampir penuh, backup gagal, aksi permanen |

Nilai kontras di atas dihitung kasar (target minimal 4.5:1 untuk teks biasa). Verifikasi dengan contrast checker setiap kali menambah kombinasi warna baru.

### Aturan pemakaian warna

1. Background halaman `blush-50`. Kartu dan panel `rose-100` atau `lilac-100`. Jangan putih murni (`#FFFFFF`) dan jangan hitam murni.
2. Satu warna aksen interaktif saja: `accent`. Jangan menambah warna brand baru.
3. `periwinkle-300` untuk visualisasi (bar storage, chip, outline fokus), bukan untuk teks.
4. Warna status hanya untuk status. Jangan dipakai sebagai dekorasi.
5. Tidak ada gradient sebagai dekorasi. Gradient boleh hanya kalau membawa informasi (misalnya skala risiko), dan harus diputuskan lewat diskusi.
6. Dark mode belum masuk Fase 1.

### Implementasi: semua lewat token

Jangan menulis hex langsung di komponen. Semua warna lewat CSS variable supaya white-label untuk agency (Fase berikutnya, lihat `parent_agency_id`) cukup mengganti token, bukan mengubah komponen.

```css
/* dashboard/app/globals.css */
:root {
  --lilac-100: #F6E5F5;
  --blush-50: #FBF4F9;
  --rose-100: #F6E7E6;
  --periwinkle-300: #B9CCED;

  --ink: #2A2640;
  --ink-muted: #5E5A78;
  --line: #E3CFE2;
  --accent: #3A5A9B;
  --accent-hover: #2F4A82;

  --ok-fg: #2F6B58;     --ok-bg: #DDEFE8;
  --warn-fg: #8A5A0F;   --warn-bg: #F8EBCB;
  --danger-fg: #A8324D; --danger-bg: #F9DDE3;
}
```

```js
// dashboard/tailwind.config.js (potongan)
theme: {
  extend: {
    colors: {
      page: 'var(--blush-50)',
      surface: 'var(--rose-100)',
      tint: 'var(--lilac-100)',
      soft: 'var(--periwinkle-300)',
      ink: 'var(--ink)',
      'ink-muted': 'var(--ink-muted)',
      line: 'var(--line)',
      accent: 'var(--accent)',
      'accent-hover': 'var(--accent-hover)',
      ok: { fg: 'var(--ok-fg)', bg: 'var(--ok-bg)' },
      warn: { fg: 'var(--warn-fg)', bg: 'var(--warn-bg)' },
      danger: { fg: 'var(--danger-fg)', bg: 'var(--danger-bg)' },
    },
  },
}
```

## 3. Tipografi

- **Typeface:** Plus Jakarta Sans (Google Fonts, dibuat desainer Indonesia). Satu family saja untuk semua teks. Fallback: `system-ui, sans-serif`.
- **Angka:** aktifkan `font-variant-numeric: tabular-nums` pada angka di tabel dan metrik supaya kolom rata.
- **Hierarki lewat ukuran dan weight**, bukan lewat warna-warni atau huruf kapital.

| Peran | Ukuran | Weight |
|---|---|---|
| Judul halaman | 28px | 700 |
| Judul section | 20px | 600 |
| Body | 16px | 400 |
| Keterangan | 14px | 400 |

- Line-height body 1.6. Lebar baris teks maksimal sekitar 70 karakter.
- Sentence case di semua tempat. Tidak ada label ALL CAPS.
- Tidak ada teks di bawah 14px.
- Jangan menonjolkan satu kata di dalam judul dengan italic atau warna lain.

## 4. Layout dan Bentuk

- **Alignment:** rata kiri. Teks panjang tidak pernah center atau justify.
- **Spacing:** skala kelipatan 4px (4, 8, 12, 16, 24, 32, 48). Jangan pakai angka acak.
- **Radius punya hierarki:** chip dan badge `999px`, tombol dan input `10px`, kartu `16px`, dialog `20px`. Jangan satu radius untuk semuanya.
- **Pemisah:** border 1px `line` pada surface berwarna. Tidak ada shadow abu-abu generik di tiap kartu. Shadow hanya untuk elemen yang melayang (dialog, dropdown).
- **Jangan memotong semua konten menjadi kartu yang identik.** Kalau informasi bisa jadi daftar atau tabel, pakai daftar atau tabel.
- Responsif sampai lebar 360px. Banyak pengguna membuka dashboard dari HP.

## 5. Komponen Khusus Archon

### Prediksi storage
Prinsip desain #3: prediksi selalu range, tidak pernah angka tunggal.
- Tampilkan sebagai kalimat: "Storage diperkirakan penuh dalam 12–19 hari." Estimasi terbaik muncul sebagai keterangan: "Perkiraan paling mungkin: 15 hari."
- Visualisasi: bar yang menunjukkan rentang (`min` sampai `max`) dengan penanda `best_estimate`.
- Jangan membuat angka besar tunggal dengan label kecil dan gradient.

### Bar pemakaian storage
- Track `lilac-100`, isi `periwinkle-300`. Berubah ke `warn` dan `danger` hanya saat melewati ambang.
- Selalu tampilkan angka nyata di sebelahnya (misal "182 GB dari 250 GB"), bukan hanya persen.

### Daftar kandidat cleanup
- Tabel: nama file, lokasi, ukuran, terakhir diubah, alasan direkomendasikan (dalam bahasa awam, misal "File log lama, tidak diakses 90 hari").
- Pengguna bisa memilih per file. Tidak ada pilihan "hapus semua" yang tercentang otomatis.
- Ringkasan di bawah daftar: "12 file dipilih, 3,2 GB akan dibebaskan".

### Dialog konfirmasi cleanup
Aksi destruktif (prinsip #1 dan #2) harus terasa serius dan jelas.
- Judul menyebut apa yang terjadi: "Pindahkan 12 file ke tempat sampah?"
- Isi menjelaskan: jumlah file, total ukuran, dan bahwa file bisa dikembalikan selama 48 jam sebelum dihapus permanen.
- Tombol utama memakai kata kerja spesifik: "Pindahkan 12 file". Tombol sekunder: "Batal". Tombol default-fokus adalah "Batal", bukan aksi destruktif.
- Mode dry-run ("Lihat simulasi") tersedia sebagai tombol terpisah dan tidak mengubah apa pun.

### Tempat sampah dan restore
- Tampilkan sisa waktu sebelum permanen: "Dihapus permanen dalam 31 jam".
- Tombol "Kembalikan" selalu terlihat selama window berlaku.

### Riwayat cleanup (audit trail)
- Kolom: waktu, dipicu oleh siapa, jumlah file, bytes dibebaskan, status (selesai, dikembalikan, dihapus permanen).
- Ini tabel biasa. Jangan didekorasi.

### Status backup
- Satu baris per server: nama, status backup terakhir (`ok`/`warn`/`danger`), waktu backup terakhir, keterangan singkat.
- Fase 1 hanya status check. Jangan membuat UI yang mengesankan restore sudah diuji otomatis.

## 6. Copywriting

- Gunakan "Anda". Konsisten di seluruh UI. Sentence case. Kalimat aktif.
- Bahasa awam. Pengguna "mengelola notifikasi", bukan "konfigurasi webhook". "Ruang penyimpanan" boleh, "inode" tidak.
- Tombol menyebut hasil: "Simpan perubahan", bukan "Submit". Nama aksi sama di seluruh alur: tombol "Pindahkan 12 file" menghasilkan notifikasi "12 file dipindahkan ke tempat sampah".
- **Error:** jelaskan apa yang terjadi dan apa yang bisa dilakukan. Jangan minta maaf berlebihan, jangan "Oops!", jangan "Terjadi kesalahan" tanpa penjelasan.
  - Buruk: "Oops! Ada yang salah."
  - Baik: "Agent di server toko-sepatu tidak mengirim data sejak 2 jam lalu. Periksa apakah server menyala, lalu muat ulang halaman."
- **Kondisi kosong** adalah ajakan bertindak: "Belum ada server. Tambahkan server pertama Anda untuk mulai memantau storage."
- Jangan menjual. Deskripsikan apa yang dilakukan sesuatu, bukan betapa hebatnya.
- Data contoh harus realistis dan lokal: nama server seperti `toko-sepatu-prod`, path seperti `/var/www/uploads`, ukuran dalam GB. Bukan "Lorem ipsum", "John Doe", atau "Acme Inc".

## 7. Motion

- Tidak ada animasi masuk di setiap section dan tidak ada hover transition di setiap kartu.
- Motion yang boleh: respons terhadap aksi pengguna (dialog terbuka, baris terpilih, tombol ditekan, toast muncul), durasi 150–200ms.
- Hormati `prefers-reduced-motion`.

## 8. Batas Kualitas (tidak perlu diumumkan, harus terpenuhi)

- Responsif sampai 360px.
- Fokus keyboard terlihat jelas (outline `periwinkle-300` 2px dengan offset, atau `accent`).
- Kontras teks minimal 4.5:1.
- Semua ikon yang membawa makna punya label teks atau `aria-label`.
- Status tidak hanya dibedakan dengan warna.