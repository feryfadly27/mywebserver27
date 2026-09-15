# ⚡ MyLokalWebserver

**MyLokalWebserver** adalah paket web server lokal yang ringan, modern, dan portabel untuk Windows. Dibuat menggunakan **Go**, aplikasi ini menggabungkan **Apache 2.4**, **PHP 8.4 (Thread-Safe)**, **MariaDB 11.4**, dan **phpMyAdmin 5.2** dengan antarmuka web yang bersih dan terminal interaktif terintegrasi.

---

## ✨ Fitur Utama

- **🚀 Sangat Ringan & Portabel**: Dibangun dengan Go (single `.exe`, tanpa butuh runtime Node.js/Python), konsumsi RAM sangat hemat (~10-20 MB).
- **📥 Auto-Download**: Otomatis mengunduh dan mengekstrak Apache, PHP, MariaDB, dan phpMyAdmin langsung dari UI pada peluncuran pertama.
- **⚙️ Port Dinamis**: Port Apache (default `8080`) dan MariaDB (default `3306`) dapat diubah langsung dari menu Pengaturan di Web UI.
- **💻 Web Terminal Terintegrasi**: Terminal interaktif berbasis web yang langsung terbuka di direktori proyek `www/htdocs`, dengan `php`, `mysql`, dan `httpd` otomatis terdaftar di PATH sesi.
- **📁 Folder Proyek Mandiri**: Tempatkan website Anda di folder `www/htdocs/`.
- **📊 phpMyAdmin Siap Pakai**: Terintegrasi langsung untuk memudahkan pengelolaan basis data MySQL/MariaDB.
- **🔒 Bebas Registry & Service**: Tidak mengubah registry Windows, tidak memasang Windows Service, data database di folder `data/mysql/` aman dan terisolasi.

---

## 📂 Struktur Direktori

```
webserver/
├── mylokalwebserver.exe      # Aplikasi utama (Control Panel & Service Manager)
├── settings.json             # Konfigurasi port & preferensi
├── start.bat                 # Script peluncur cepat
├── build.bat                 # Script kompilasi ulang dari source
├── bin/
│   ├── apache/               # Apache HTTP Server 2.4
│   ├── php/                  # PHP 8.4 TS & ekstensi
│   └── mariadb/              # MariaDB 11.4 Engine
├── data/
│   └── mysql/                # Data database MariaDB
├── www/
│   └── htdocs/               # Tempat file website lokal Anda
│       ├── index.php         # Halaman utama bawaan
│       └── phpmyadmin/       # phpMyAdmin GUI
├── logs/                     # Log error Apache, PHP, MariaDB
└── tmp/                      # Temporary files & session PHP
```

---

## 🚀 Cara Menjalankan

### Cara 1: Menggunakan Launcher
Cukup klik ganda pada `start.bat` atau `mylokalwebserver.exe`.

### Cara 2: Build dari Source
Jika Anda memiliki compiler Go:
```bash
build.bat
```
lalu jalankan `mylokalwebserver.exe`.

---

## ⚙️ Konfigurasi Default

| Layanan | Port Default | URL Akses |
|---|---|---|
| **Control Panel** | `3000` | [http://localhost:3000](http://localhost:3000) |
| **Apache (Website)** | `8080` | [http://localhost:8080](http://localhost:8080) |
| **phpMyAdmin** | `8080` | [http://localhost:8080/phpmyadmin](http://localhost:8080/phpmyadmin) |
| **MariaDB (MySQL)** | `3306` | Host: `127.0.0.1`, User: `root`, Pass: *(kosong)* |

---

## 📌 Catatan & Prasyarat

- Pastikan komputer Anda telah terpasang **Microsoft Visual C++ Redistributable (2015–2022)** yang merupakan dependensi standar Apache & PHP di Windows.
