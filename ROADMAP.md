# 🗺️ Roadmap Pengembangan & Ide Fitur MyLokalWebserver

Dokumen ini berisi daftar fitur yang telah selesai dibangun serta daftar ide fitur yang direncanakan untuk pengembangan MyLokalWebserver selanjutnya.

---

## 🌟 Status Saat Ini (Versi 1.1.0)

- [x] **Web Server Apache 2.4 & PHP 8.4** (Portabel tanpa install registry)
- [x] **Database MariaDB 11.4 & phpMyAdmin 5.2** (Portabel + auto database init)
- [x] **Dashboard Control Panel Modern** (Light UI berbasis Web di port 3000)
- [x] **Web-based Interactive Terminal** (xterm.js + PTY dengan auto PATH php, mysql, httpd)
- [x] **Virtual Host Manager** (Domain kustom .test + otomatis konfigurasi file hosts)
- [x] **Project Auto-Detector** (Deteksi otomatis PHP Native, Laravel, CI4, WordPress di www/htdocs)
- [x] **Multi-Platform Support**:
  - Windows x64 (mylokalwebserver.exe)
  - macOS Apple Silicon M1/M2/M3/M4 (MyLokalWebserver.app)
  - macOS Intel (MyLokalWebserver.app)
  - Linux x64 (mylokalwebserver)
- [x] **Self-Updater Terintegrasi** (Cek dan perbarui binary via GitHub Release / Branch Main)
- [x] **Komponen Downloader Otomatis** dengan resume & penanganan SSL/TLS.
- [x] **📱 Share Proyek ke Smartphone via QR Code WiFi Lokal** (Deteksi otomatis IP LAN + QR Code Canvas offline)
- [x] **🗄️ One-Click Database Backup & Restore (.sql)** (Export/import database cepat, download file .sql, dan riwayat backup di `data/backups/`)

---

## 💡 Rencana Fitur Baru (Future Feature Roadmap)

### 1. 🗄️ Database & Quick Data Tools
- [x] **One-Click Database Backup & Restore (.sql)** (Sudah selesai di v1.2.0)
- [x ] **Quick Database Creator**
  - Modal formulir ringkas untuk membuat database baru dan user MySQL langsung dalam 1 klik.
- [x ] **Database Seed Helper**
  - Generator data dummy otomatis untuk pengujian aplikasi lokal.

---

### 2. 📧 Fake SMTP Mail Catcher (Mailpit / In-App Mailbox)
- [ ] **Penangkap Email Lokal Terintegrasi**
  - Menangkap semua email yang dikirim lewat fungsi PHP mail() (seperti email verifikasi akun, token OTP, atau reset password).
  - Dilengkapi tab inbox visual di Dashboard untuk melihat isi email, layout HTML, dan lampiran secara instan tanpa perlu koneksi internet.

---

### 3. 📱 Mobile & Network Sharing (QR Code LAN)
- [x] **Share ke Jaringan WiFi Lokal dengan QR Code** (Sudah selesai di v1.2.0)

---

### 4. ⚙️ Manajemen Ekstensi PHP Visual (GUI Toggle)
- [ ] **Checklist Ekstensi PHP di Dashboard**
  - Halaman pengaturan dengan saklar (toggle on/off) untuk mengaktifkan/menonaktifkan ekstensi populer (curl, gd, intl, mbstring, zip, openssl, pdo_mysql, xdebug, dll.).
  - Otomatis memperbarui php.ini dan me-restart Apache tanpa perlu edit teks manual.

---

### 5. 🔒 Otomatis SSL / HTTPS Lokal (https://*.test)
- [ ] **Local HTTPS Generator**
  - Pembuatan sertifikat SSL lokal yang valid dan terpercaya di browser untuk Virtual Host domain .test (contoh: https://toko.test).
  - Memudahkan pengujian fitur yang mewajibkan HTTPS (PWA, OAuth Login Google/Facebook, Webhook lokal, Clipboard API).

---

### 6. 🚀 1-Click Project Starter (Framework & CMS)
- [ ] **Pembuat Proyek Baru Siap Pakai**
  - Pilihan template saat membuat proyek baru:
    - **Laravel Starter**: Otomatis setup struktur folder, file .env, dan key app.
    - **CodeIgniter 4 Starter**: Siap pakai dengan file env & base URL.
    - **WordPress Starter**: Otomatis download & konfigurasi wp-config.php.
    - **PHP Native CRUD Starter**: Template siap pakai dengan Bootstrap 5 & koneksi database.

---

### 7. 🌐 Tunneling Akses Publik (Cloudflare Tunnel / Ngrok)
- [ ] **Share Website ke Klien dari Jarak Jauh**
  - 1-Klik untuk mengaktifkan URL publik sementara (contoh: https://kasir-fery.trycloudflare.com).
  - Memungkinkan klien atau rekan kerja mengakses website lokal Anda dari internet tanpa perlu sewa VPS/hosting.

---

### 8. 🛠️ Troubleshooting & Port Conflict Auto-Fix
- [ ] **Pembersih Port Bentrok Otomatis**
  - Tombol diagnosa untuk mendeteksi aplikasi yang mengunci Port 80, 8080, atau 3306 (seperti IIS atau Skype) dan membebaskan port tersebut dalam 1 klik.
- [ ] **Auto-Installer Visual C++ Runtime**
  - Tombol download & install otomatis Microsoft Visual C++ Redistributable langsung dari dalam aplikasi jika mendeteksi Windows baru.

---

## 📌 Rekomendasi Prioritas Implementasi

| Fase | Target Fitur | Estimasi Kompleksitas |
|:---:|---|:---:|
| **Fase 1** | Share Proyek ke HP via QR Code LAN + One-Click Database Backup/Restore | Ringan & Sangat Berguna |
| **Fase 2** | Manajemen Ekstensi PHP Visual + Auto-Installer VC++ Runtime | Sedang |
| **Fase 3** | Fake SMTP Mail Catcher Lokal | Sedang |
| **Fase 4** | 1-Click Project Starter Generator (Laravel, CI4, WP) | Sedang |
| **Fase 5** | Local HTTPS / SSL Generator (https://*.test) + Cloudflare Tunnel | Lanjutan |

---
*Dokumen ini diperbarui secara berkala seiring berjalannya pengembangan MyLokalWebserver.*
