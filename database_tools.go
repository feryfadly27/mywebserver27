package main

import (
	"bytes"
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

type BackupFileInfo struct {
	Filename      string    `json:"filename"`
	DatabaseName  string    `json:"database_name"`
	Size          int64     `json:"size"`
	SizeFormatted string    `json:"size_formatted"`
	CreatedAt     time.Time `json:"created_at"`
	CreatedAtStr  string    `json:"created_at_str"`
}

func getMariaDBClientBin() string {
	exeExt := GetExecutableExt()
	mariadbDir := filepath.Join(AppRootDir, "bin", "mariadb", "bin")
	for _, name := range []string{"mariadb", "mysql"} {
		path := filepath.Join(mariadbDir, name+exeExt)
		if fileExists(path) {
			return path
		}
		pathNoExt := filepath.Join(mariadbDir, name)
		if fileExists(pathNoExt) {
			return pathNoExt
		}
	}
	return ""
}

func getMariaDBDumpBin() string {
	exeExt := GetExecutableExt()
	mariadbDir := filepath.Join(AppRootDir, "bin", "mariadb", "bin")
	for _, name := range []string{"mariadb-dump", "mysqldump"} {
		path := filepath.Join(mariadbDir, name+exeExt)
		if fileExists(path) {
			return path
		}
		pathNoExt := filepath.Join(mariadbDir, name)
		if fileExists(pathNoExt) {
			return pathNoExt
		}
	}
	return ""
}

func ListDatabases() ([]string, error) {
	settings := GetCurrentSettings()
	clientBin := getMariaDBClientBin()

	// If MariaDB is running, query via client binary
	if clientBin != "" && Manager.IsPortOpen(settings.MariaDBPort) {
		cmd := exec.Command(clientBin, "-u", "root", "-P", fmt.Sprintf("%d", settings.MariaDBPort), "-e", "SHOW DATABASES;")
		SetCmdHideWindow(cmd)
		out, err := cmd.CombinedOutput()
		if err == nil {
			var dbs []string
			lines := strings.Split(string(out), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.EqualFold(line, "Database") {
					continue
				}
				lower := strings.ToLower(line)
				if lower == "information_schema" || lower == "performance_schema" || lower == "mysql" || lower == "sys" {
					continue
				}
				dbs = append(dbs, line)
			}
			sort.Strings(dbs)
			return dbs, nil
		}
	}

	// Fallback: Scan data/mysql directories on disk
	dataDir := filepath.Join(AppRootDir, "data", "mysql")
	entries, err := os.ReadDir(dataDir)
	if err != nil {
		return []string{}, nil
	}

	var dbs []string
	for _, entry := range entries {
		if entry.IsDir() {
			name := entry.Name()
			lower := strings.ToLower(name)
			if lower == "information_schema" || lower == "performance_schema" || lower == "mysql" || lower == "sys" {
				continue
			}
			dbs = append(dbs, name)
		}
	}
	sort.Strings(dbs)
	return dbs, nil
}

func BackupDatabase(dbName string) (*BackupFileInfo, error) {
	if dbName == "" {
		return nil, fmt.Errorf("nama database tidak boleh kosong")
	}

	settings := GetCurrentSettings()
	if !Manager.IsPortOpen(settings.MariaDBPort) {
		// Auto start MariaDB if not running
		if err := Manager.StartMariaDB(); err != nil {
			return nil, fmt.Errorf("MariaDB tidak aktif dan gagal dinyalakan: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	dumpBin := getMariaDBDumpBin()
	if dumpBin == "" {
		return nil, fmt.Errorf("mariadb-dump / mysqldump binary tidak ditemukan")
	}

	backupsDir := filepath.Join(AppRootDir, "data", "backups")
	if err := os.MkdirAll(backupsDir, 0755); err != nil {
		return nil, fmt.Errorf("gagal membuat folder backups: %w", err)
	}

	timestamp := time.Now().Format("2006-01-02_150405")
	filename := fmt.Sprintf("%s_%s.sql", sanitizeFilename(dbName), timestamp)
	targetPath := filepath.Join(backupsDir, filename)

	outFile, err := os.Create(targetPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat file backup: %w", err)
	}
	defer outFile.Close()

	cmd := exec.Command(dumpBin, "-u", "root", "-P", fmt.Sprintf("%d", settings.MariaDBPort), "--databases", dbName)
	cmd.Stdout = outFile
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	SetCmdHideWindow(cmd)

	if err := cmd.Run(); err != nil {
		_ = os.Remove(targetPath)
		return nil, fmt.Errorf("gagal melakukan backup: %s (err: %w)", strings.TrimSpace(errBuf.String()), err)
	}

	fileInfo, err := os.Stat(targetPath)
	if err != nil {
		return nil, err
	}

	return &BackupFileInfo{
		Filename:      filename,
		DatabaseName:  dbName,
		Size:          fileInfo.Size(),
		SizeFormatted: formatBytes(fileInfo.Size()),
		CreatedAt:     fileInfo.ModTime(),
		CreatedAtStr:  fileInfo.ModTime().Format("02 Jan 2006 15:04"),
	}, nil
}

func RestoreDatabaseFromSQL(dbName string, sqlReader io.Reader) error {
	settings := GetCurrentSettings()
	if !Manager.IsPortOpen(settings.MariaDBPort) {
		if err := Manager.StartMariaDB(); err != nil {
			return fmt.Errorf("MariaDB tidak aktif dan gagal dinyalakan: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	clientBin := getMariaDBClientBin()
	if clientBin == "" {
		return fmt.Errorf("mariadb / mysql client binary tidak ditemukan")
	}

	// First ensure database exists if dbName is specified
	if dbName != "" {
		createCmd := exec.Command(clientBin, "-u", "root", "-P", fmt.Sprintf("%d", settings.MariaDBPort), "-e", fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;", dbName))
		SetCmdHideWindow(createCmd)
		_ = createCmd.Run()
	}

	var args []string
	args = append(args, "-u", "root", "-P", fmt.Sprintf("%d", settings.MariaDBPort))
	if dbName != "" {
		args = append(args, dbName)
	}

	cmd := exec.Command(clientBin, args...)
	cmd.Stdin = sqlReader
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	SetCmdHideWindow(cmd)

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(errBuf.String())
		if errStr != "" {
			return fmt.Errorf("gagal me-restore database: %s", errStr)
		}
		return fmt.Errorf("restore error: %w", err)
	}

	return nil
}

func ListBackups() ([]BackupFileInfo, error) {
	backupsDir := filepath.Join(AppRootDir, "data", "backups")
	_ = os.MkdirAll(backupsDir, 0755)

	entries, err := os.ReadDir(backupsDir)
	if err != nil {
		return []BackupFileInfo{}, nil
	}

	var backups []BackupFileInfo
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(strings.ToLower(entry.Name()), ".sql") {
			info, err := entry.Info()
			if err != nil {
				continue
			}

			// Extract DB name from prefix (e.g. kasir_db_2026-09-15_143000.sql)
			rawName := strings.TrimSuffix(entry.Name(), ".sql")
			parts := strings.Split(rawName, "_")
			dbName := rawName
			if len(parts) >= 3 {
				dbName = strings.Join(parts[:len(parts)-2], "_")
			}

			backups = append(backups, BackupFileInfo{
				Filename:      entry.Name(),
				DatabaseName:  dbName,
				Size:          info.Size(),
				SizeFormatted: formatBytes(info.Size()),
				CreatedAt:     info.ModTime(),
				CreatedAtStr:  info.ModTime().Format("02 Jan 2006 15:04"),
			})
		}
	}

	// Sort newest first
	sort.Slice(backups, func(i, j int) bool {
		return backups[i].CreatedAt.After(backups[j].CreatedAt)
	})

	return backups, nil
}

func DeleteBackupFile(filename string) error {
	filename = filepath.Base(filename)
	if !strings.HasSuffix(strings.ToLower(filename), ".sql") {
		return fmt.Errorf("invalid backup filename")
	}
	target := filepath.Join(AppRootDir, "data", "backups", filename)
	return os.Remove(target)
}

// TableColumnInfo describes a column in a MariaDB table
type TableColumnInfo struct {
	Field         string `json:"field"`
	Type          string `json:"type"`
	Null          string `json:"null"`
	Key           string `json:"key"`
	Default       string `json:"default"`
	Extra         string `json:"extra"`
	IsAutoIncr    bool   `json:"is_auto_incr"`
	DetectedType  string `json:"detected_type"`
	DetectedLabel string `json:"detected_label"`
}

// ListTables returns all tables inside a specific database
func ListTables(dbName string) ([]string, error) {
	if dbName == "" {
		return nil, fmt.Errorf("nama database tidak boleh kosong")
	}

	settings := GetCurrentSettings()
	clientBin := getMariaDBClientBin()
	if clientBin == "" {
		return nil, fmt.Errorf("mariadb client binary tidak ditemukan")
	}

	if !Manager.IsPortOpen(settings.MariaDBPort) {
		if err := Manager.StartMariaDB(); err != nil {
			return nil, fmt.Errorf("MariaDB tidak aktif: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	cmd := exec.Command(clientBin, "-u", "root", "-P", fmt.Sprintf("%d", settings.MariaDBPort), dbName, "-e", "SHOW TABLES;")
	SetCmdHideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gagal membaca tabel: %s", strings.TrimSpace(string(out)))
	}

	var tables []string
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || i == 0 { // skip header
			continue
		}
		tables = append(tables, line)
	}
	sort.Strings(tables)
	return tables, nil
}

// DescribeTable returns schema information and detected generator types for all columns
func DescribeTable(dbName, tableName string) ([]TableColumnInfo, error) {
	if dbName == "" || tableName == "" {
		return nil, fmt.Errorf("database dan nama tabel tidak boleh kosong")
	}

	settings := GetCurrentSettings()
	clientBin := getMariaDBClientBin()
	if clientBin == "" {
		return nil, fmt.Errorf("mariadb client binary tidak ditemukan")
	}

	if !Manager.IsPortOpen(settings.MariaDBPort) {
		if err := Manager.StartMariaDB(); err != nil {
			return nil, fmt.Errorf("MariaDB tidak aktif: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	query := fmt.Sprintf("DESCRIBE `%s`;", tableName)
	cmd := exec.Command(clientBin, "-u", "root", "-P", fmt.Sprintf("%d", settings.MariaDBPort), dbName, "-e", query)
	SetCmdHideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("gagal mendeskripsikan tabel '%s': %s", tableName, strings.TrimSpace(string(out)))
	}

	var columns []TableColumnInfo
	lines := strings.Split(string(out), "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, "\r\n")
		if line == "" || i == 0 { // skip header
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 2 {
			field := parts[0]
			colType := parts[1]
			nullVal := ""
			if len(parts) > 2 {
				nullVal = parts[2]
			}
			keyVal := ""
			if len(parts) > 3 {
				keyVal = parts[3]
			}
			defVal := ""
			if len(parts) > 4 {
				defVal = parts[4]
			}
			extraVal := ""
			if len(parts) > 5 {
				extraVal = parts[5]
			}
			isAuto := strings.Contains(strings.ToLower(extraVal), "auto_increment")

			col := TableColumnInfo{
				Field:      field,
				Type:       colType,
				Null:       nullVal,
				Key:        keyVal,
				Default:    defVal,
				Extra:      extraVal,
				IsAutoIncr: isAuto,
			}

			genType, genLabel := DetectColumnGenerator(col)
			col.DetectedType = genType
			col.DetectedLabel = genLabel
			columns = append(columns, col)
		}
	}

	return columns, nil
}

// DetectColumnGenerator analyzes column name and data type to choose best data generator
func DetectColumnGenerator(col TableColumnInfo) (genType string, label string) {
	if col.IsAutoIncr {
		return "auto_increment", "Auto Increment ID (Dilewati)"
	}

	name := strings.ToLower(col.Field)
	cType := strings.ToLower(col.Type)

	// Explicit Primary ID check if not auto_increment
	if name == "id" || strings.HasSuffix(name, "_id") && strings.Contains(cType, "int") && col.Key == "PRI" {
		return "auto_increment", "Primary Key ID (Dilewati)"
	}

	// 1. Nama Orang (Pasien, Dokter, Siswa, Karyawan, User, Pelanggan)
	if strings.Contains(name, "pasien") || strings.Contains(name, "nama") || strings.Contains(name, "name") ||
		strings.Contains(name, "customer") || strings.Contains(name, "pelanggan") || strings.Contains(name, "dokter") ||
		strings.Contains(name, "pegawai") || strings.Contains(name, "karyawan") || strings.Contains(name, "siswa") ||
		strings.Contains(name, "user") && !strings.Contains(name, "username") {
		return "nama_lengkap", "Nama Orang (Indonesia)"
	}

	// 2. NIK / KTP / No Identitas / NISN / NIP
	if strings.Contains(name, "nik") || strings.Contains(name, "ktp") || strings.Contains(name, "identitas") ||
		strings.Contains(name, "nisn") || strings.Contains(name, "nip") || strings.Contains(name, "no_rek") {
		return "nik", "Nomor Identitas / NIK (16 Digit)"
	}

	// 3. No HP / Telepon / WA
	if strings.Contains(name, "hp") || strings.Contains(name, "telp") || strings.Contains(name, "phone") ||
		strings.Contains(name, "wa") || strings.Contains(name, "kontak") {
		return "phone", "No. Telepon / HP (0812...)"
	}

	// 4. Email
	if strings.Contains(name, "email") || strings.Contains(name, "mail") || strings.Contains(name, "surel") {
		return "email", "Alamat Email"
	}

	// 5. Alamat / Domisili / Kota
	if strings.Contains(name, "alamat") || strings.Contains(name, "address") || strings.Contains(name, "domisili") ||
		strings.Contains(name, "lokasi") || strings.Contains(name, "kota") {
		return "alamat", "Alamat Jalan & Kota (Indonesia)"
	}

	// 6. Jenis Kelamin / Gender
	if strings.Contains(name, "jk") || strings.Contains(name, "gender") || strings.Contains(name, "sex") ||
		strings.Contains(name, "jenis_kelamin") {
		return "gender", "Jenis Kelamin (Laki-laki / Perempuan)"
	}

	// 7. Keluhan / Diagnosa / Penyakit / Catatan Medis (Sangat cocok untuk data pasien)
	if strings.Contains(name, "keluhan") || strings.Contains(name, "diagnosa") || strings.Contains(name, "penyakit") ||
		strings.Contains(name, "gejala") || strings.Contains(name, "anamnesa") || strings.Contains(name, "terapi") ||
		strings.Contains(name, "resep") {
		return "keluhan_medis", "Keluhan Medis / Gejala Pasien"
	}

	// 8. Tanggal / Tanggal Lahir / Created At
	if strings.Contains(name, "tgl_lahir") || strings.Contains(name, "birth") || strings.Contains(name, "lahir") {
		return "birth_date", "Tanggal Lahir (Usia 18-65 Tahun)"
	}
	if strings.Contains(name, "tgl") || strings.Contains(name, "tanggal") || strings.Contains(name, "date") ||
		strings.Contains(name, "waktu") || strings.Contains(name, "created") || strings.Contains(name, "updated") ||
		cType == "date" || cType == "datetime" || cType == "timestamp" {
		return "date_recent", "Tanggal & Waktu Saat Ini"
	}

	// 9. Produk / Barang / Menu
	if strings.Contains(name, "produk") || strings.Contains(name, "barang") || strings.Contains(name, "menu") ||
		strings.Contains(name, "item") || strings.Contains(name, "obat") {
		return "nama_produk", "Nama Produk / Barang / Obat"
	}

	// 10. Harga / Biaya / Tarif / Total / Subtotal / Gaji
	if strings.Contains(name, "harga") || strings.Contains(name, "price") || strings.Contains(name, "biaya") ||
		strings.Contains(name, "tarif") || strings.Contains(name, "total") || strings.Contains(name, "subtotal") ||
		strings.Contains(name, "bayar") || strings.Contains(name, "gaji") || strings.Contains(name, "ongkir") {
		return "harga_rupiah", "Harga / Nilai Rupiah (Rp 10.000 - Rp 500.000)"
	}

	// 11. Stok / Jumlah / Qty / Umur / Usia
	if strings.Contains(name, "stok") || strings.Contains(name, "stock") || strings.Contains(name, "qty") ||
		strings.Contains(name, "jumlah") || strings.Contains(name, "kuantitas") {
		return "stok", "Jumlah / Stok Angka (5 - 100)"
	}
	if strings.Contains(name, "umur") || strings.Contains(name, "usia") || strings.Contains(name, "age") {
		return "umur", "Usia / Umur (18 - 70)"
	}

	// 12. Kategori / Golongan
	if strings.Contains(name, "kategori") || strings.Contains(name, "category") || strings.Contains(name, "golongan") ||
		strings.Contains(name, "jenis") {
		return "kategori", "Nama Kategori / Golongan"
	}

	// 13. Status / Active
	if strings.Contains(name, "status") || strings.Contains(name, "state") || strings.Contains(name, "is_active") {
		return "status", "Status (Aktif / Selesai / Proses)"
	}

	// 14. Username & Password
	if strings.Contains(name, "user") || strings.Contains(name, "username") {
		return "username", "Username Akun"
	}
	if strings.Contains(name, "pass") || strings.Contains(name, "password") || strings.Contains(name, "pin") {
		return "password", "Password Terenkripsi"
	}

	// 15. Deskripsi / Keterangan / Catatan
	if strings.Contains(name, "deskripsi") || strings.Contains(name, "description") || strings.Contains(name, "keterangan") ||
		strings.Contains(name, "catatan") || strings.Contains(name, "note") || strings.Contains(name, "isi") ||
		strings.Contains(cType, "text") {
		return "deskripsi", "Teks Deskripsi / Keterangan"
	}

	// Fallback based on SQL Data Type
	if strings.HasPrefix(cType, "enum(") {
		return "enum", "Pilihan Nilai Enum"
	}
	if strings.Contains(cType, "int") || strings.Contains(cType, "decimal") || strings.Contains(cType, "float") || strings.Contains(cType, "double") {
		return "number_generic", "Angka Acak (10 - 1000)"
	}
	if strings.Contains(cType, "varchar") || strings.Contains(cType, "char") {
		return "text_generic", "Teks Acak"
	}

	return "text_generic", "Teks Acak"
}

// Generator datasets for realistic Indonesian test data
var (
	indonesianMaleFirstNames = []string{
		"Ahmad", "Budi", "Dedi", "Eko", "Fajar", "Galih", "Hendra", "Indra", "Joko", "Kurniawan",
		"Lukman", "Maulana", "Nugroho", "Pratama", "Rizky", "Surya", "Taufik", "Wahyu", "Yusuf",
		"Aditya", "Bayu", "Candra", "Dimas", "Farhan", "Gilang", "Hafiz", "Ilham", "Agus", "Bambang",
	}
	indonesianFemaleFirstNames = []string{
		"Annisa", "Cindy", "Dewi", "Fitri", "Gita", "Hani", "Indah", "Kartika", "Lestari", "Maya",
		"Nurul", "Putri", "Rina", "Siti", "Tri", "Utami", "Vina", "Wulan", "Yuliana", "Zahra",
		"Aulia", "Bella", "Citra", "Dinda", "Eka", "Febri", "Intan", "Laras", "Rahma", "Ratna",
	}
	indonesianLastNames = []string{
		"Santoso", "Wijaya", "Saputra", "Hidayat", "Setiawan", "Pratama", "Kusuma", "Utomo", "Nugraha",
		"Wibowo", "Permana", "Siregar", "Nasution", "Lubis", "Simanjuntak", "Panjaitan", "Hartono",
		"Gunawan", "Susanto", "Tanoto", "Hermanto", "Subagyo", "Marlina", "Rahmawati", "Wahyuni",
	}
	indonesianStreetNames = []string{
		"Jl. Merdeka", "Jl. Sudirman", "Jl. Gatot Subroto", "Jl. Diponegoro", "Jl. Ahmad Yani",
		"Jl. Pahlawan", "Jl. Gajah Mada", "Jl. Thamrin", "Jl. Asia Afrika", "Jl. Malioboro",
		"Jl. Pemuda", "Jl. Veteran", "Jl. Cendrawasih", "Jl. Melati", "Jl. Mawar", "Jl. Kenanga",
	}
	indonesianCities = []string{
		"Jakarta", "Bandung", "Surabaya", "Yogyakarta", "Semarang", "Medan", "Makassar",
		"Palembang", "Denpasar", "Malang", "Bekasi", "Tangerang", "Bogor", "Depok", "Solo",
	}
	indonesianMedicalComplaints = []string{
		"Demam tinggi dan flu sejak 3 hari",
		"Sakit kepala berdenyut disertai mual",
		"Pemeriksaan tensi darah dan kolesterol rutin",
		"Nyeri sendi lutut dan pegal linu",
		"Batuk kering dan radang tenggorokan",
		"Keluhan maag perih dan kembung",
		"Kontrol asam urat dan gula darah",
		"Alergi debu dan gatal pada kulit",
		"Sesak nafas ringan saat beraktivitas",
		"Luka lecet pada jari tangan",
		"Pusing berputar (vertigo) kambuh",
		"Pemeriksaan kesehatan umum (Medical Checkup)",
		"Nyeri ulu hati setelah makan pedas",
		"Gigi ngilu dan gusi bengkak",
		"Badan lemas dan kurang nafsu makan",
	}
	indonesianProducts = []string{
		"Kopi Susu Gula Aren", "Teh Tarik Dingin", "Roti Bakar Coklat Keju", "Nasi Goreng Spesial",
		"Mie Ayam Bakso", "Air Mineral 600ml", "Sabun Mandi Cair 450ml", "Shampoo Anti Dandruff 180ml",
		"Minyak Goreng 2L", "Beras Premium 5kg", "Gula Pasir 1kg", "Telur Ayam 1kg",
		"Kripik Singkong Balado", "Biskuit Coklat Renyah", "Paracetamol 500mg (Strip)", "Vitamin C 500mg",
		"Antasida Doen (Strip)", "Madu Murni 250ml", "Kapas Medis 100gr", "Minyak Kayu Putih 60ml",
	}
	indonesianCategories = []string{
		"Makanan", "Minuman", "Sembako", "Perawatan Tubuh", "Snack & Camilan", "Obat & Vitamin", "Umum",
	}
	sampleDescriptions = []string{
		"Data contoh otomatis hasil generate MyLokalWebserver untuk uji coba.",
		"Kondisi baik dan terdaftar dalam sistem verifikasi lokal.",
		"Pemeriksaan berkala tercatat aktif dan sesuai prosedur operasional standar.",
		"Stok barang baru masuk dari distributor resmi terpercaya.",
		"Pasien dalam masa observasi pemulihan dengan resep obat teratur.",
	}
)

// GenerateSmartSeedData creates and inserts realistic seed data into the specified table
func GenerateSmartSeedData(dbName, tableName string, count int) (int, error) {
	if dbName == "" || tableName == "" {
		return 0, fmt.Errorf("nama database dan tabel tidak boleh kosong")
	}
	if count <= 0 {
		count = 10
	}
	if count > 500 {
		count = 500
	}

	columns, err := DescribeTable(dbName, tableName)
	if err != nil {
		return 0, err
	}
	if len(columns) == 0 {
		return 0, fmt.Errorf("tidak ada kolom yang ditemukan pada tabel '%s'", tableName)
	}

	// Filter out auto increment columns for insert query
	var insertCols []string
	var activeCols []TableColumnInfo
	for _, c := range columns {
		if c.DetectedType != "auto_increment" && !c.IsAutoIncr {
			insertCols = append(insertCols, fmt.Sprintf("`%s`", c.Field))
			activeCols = append(activeCols, c)
		}
	}

	if len(insertCols) == 0 {
		return 0, fmt.Errorf("semua kolom bertipe auto-increment, tidak ada yang perlu diisi")
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))

	var valuesList []string
	for i := 0; i < count; i++ {
		isMale := rnd.Intn(2) == 0
		var firstName string
		if isMale {
			firstName = indonesianMaleFirstNames[rnd.Intn(len(indonesianMaleFirstNames))]
		} else {
			firstName = indonesianFemaleFirstNames[rnd.Intn(len(indonesianFemaleFirstNames))]
		}
		lastName := indonesianLastNames[rnd.Intn(len(indonesianLastNames))]
		fullName := fmt.Sprintf("%s %s", firstName, lastName)

		var rowValues []string
		for _, col := range activeCols {
			val := generateValueForColumn(col, rnd, isMale, fullName, firstName, lastName, i)
			rowValues = append(rowValues, val)
		}
		valuesList = append(valuesList, "("+strings.Join(rowValues, ", ")+")")
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES\n%s;",
		tableName,
		strings.Join(insertCols, ", "),
		strings.Join(valuesList, ",\n"),
	)

	settings := GetCurrentSettings()
	clientBin := getMariaDBClientBin()
	if clientBin == "" {
		return 0, fmt.Errorf("mariadb client binary tidak ditemukan")
	}

	cmd := exec.Command(clientBin, "-u", "root", "-P", fmt.Sprintf("%d", settings.MariaDBPort), dbName, "-e", query)
	var errBuf bytes.Buffer
	cmd.Stderr = &errBuf
	SetCmdHideWindow(cmd)

	if err := cmd.Run(); err != nil {
		errStr := strings.TrimSpace(errBuf.String())
		return 0, fmt.Errorf("gagal menyisipkan data ke '%s': %s", tableName, errStr)
	}

	return count, nil
}

func generateValueForColumn(col TableColumnInfo, rnd *rand.Rand, isMale bool, fullName, firstName, lastName string, index int) string {
	escapeSQL := func(s string) string {
		s = strings.ReplaceAll(s, "\\", "\\\\")
		s = strings.ReplaceAll(s, "'", "\\'")
		return "'" + s + "'"
	}

	cType := strings.ToLower(col.Type)

	switch col.DetectedType {
	case "nama_lengkap":
		return escapeSQL(fullName)

	case "nik":
		// Format NIK: 320101 + DDMMYY (6 digits) + 0001 (4 digits)
		provCode := 31 + rnd.Intn(5) // 31-35 (DKI, Jabar, Jateng, Jatim, Banten)
		cityCode := 10 + rnd.Intn(80)
		kecCode := 10 + rnd.Intn(80)
		birthDay := 1 + rnd.Intn(28)
		if !isMale {
			birthDay += 40 // female NIK date offset
		}
		birthMonth := 1 + rnd.Intn(12)
		birthYear := 70 + rnd.Intn(33) // 1970-2003
		serial := 1000 + rnd.Intn(8999)
		nikStr := fmt.Sprintf("%d%02d%02d%02d%02d%02d%d", provCode, cityCode, kecCode, birthDay, birthMonth, birthYear, serial)
		return escapeSQL(nikStr)

	case "phone":
		prefixes := []string{"0812", "0813", "0821", "0852", "0857", "0858", "0877", "0878", "0896"}
		prefix := prefixes[rnd.Intn(len(prefixes))]
		suffix := 10000000 + rnd.Intn(89999999)
		return escapeSQL(fmt.Sprintf("%s%d", prefix, suffix))

	case "email":
		domainList := []string{"gmail.com", "yahoo.com", "outlook.com", "dikodein.com"}
		cleanFirst := strings.ToLower(strings.ReplaceAll(firstName, " ", ""))
		cleanLast := strings.ToLower(strings.ReplaceAll(lastName, " ", ""))
		dom := domainList[rnd.Intn(len(domainList))]
		emailStr := fmt.Sprintf("%s.%s%d@%s", cleanFirst, cleanLast, rnd.Intn(99), dom)
		return escapeSQL(emailStr)

	case "alamat":
		street := indonesianStreetNames[rnd.Intn(len(indonesianStreetNames))]
		no := 1 + rnd.Intn(150)
		city := indonesianCities[rnd.Intn(len(indonesianCities))]
		return escapeSQL(fmt.Sprintf("%s No. %d, %s", street, no, city))

	case "gender":
		if strings.HasPrefix(cType, "enum(") {
			raw := strings.TrimPrefix(cType, "enum(")
			raw = strings.TrimSuffix(raw, ")")
			opts := strings.Split(raw, ",")
			if len(opts) > 0 {
				if isMale {
					return strings.TrimSpace(opts[0])
				}
				if len(opts) > 1 {
					return strings.TrimSpace(opts[1])
				}
				return strings.TrimSpace(opts[0])
			}
		}
		if strings.Contains(cType, "char(1)") {
			if isMale {
				return "'L'"
			}
			return "'P'"
		}
		if isMale {
			return "'Laki-laki'"
		}
		return "'Perempuan'"

	case "keluhan_medis":
		complaint := indonesianMedicalComplaints[rnd.Intn(len(indonesianMedicalComplaints))]
		return escapeSQL(complaint)

	case "birth_date":
		year := 1960 + rnd.Intn(45)
		month := 1 + rnd.Intn(12)
		day := 1 + rnd.Intn(28)
		return escapeSQL(fmt.Sprintf("%04d-%02d-%02d", year, month, day))

	case "date_recent":
		offsetDays := rnd.Intn(30)
		t := time.Now().AddDate(0, 0, -offsetDays)
		if cType == "date" {
			return escapeSQL(t.Format("2006-01-02"))
		}
		return escapeSQL(t.Format("2006-01-02 15:04:05"))

	case "nama_produk":
		prod := indonesianProducts[rnd.Intn(len(indonesianProducts))]
		return escapeSQL(prod)

	case "harga_rupiah":
		prices := []int{5000, 10000, 15000, 20000, 25000, 35000, 50000, 75000, 100000, 150000, 250000, 500000}
		p := prices[rnd.Intn(len(prices))]
		if strings.Contains(cType, "int") || strings.Contains(cType, "decimal") || strings.Contains(cType, "float") || strings.Contains(cType, "double") {
			return strconv.Itoa(p)
		}
		return escapeSQL(strconv.Itoa(p))

	case "stok":
		val := 5 + rnd.Intn(95)
		if strings.Contains(cType, "int") {
			return strconv.Itoa(val)
		}
		return escapeSQL(strconv.Itoa(val))

	case "umur":
		val := 18 + rnd.Intn(52)
		if strings.Contains(cType, "int") {
			return strconv.Itoa(val)
		}
		return escapeSQL(strconv.Itoa(val))

	case "kategori":
		cat := indonesianCategories[rnd.Intn(len(indonesianCategories))]
		return escapeSQL(cat)

	case "status":
		if strings.Contains(cType, "int") || strings.Contains(cType, "tinyint(1)") {
			return "1"
		}
		statuses := []string{"Aktif", "Selesai", "Proses", "Menunggu"}
		return escapeSQL(statuses[rnd.Intn(len(statuses))])

	case "username":
		u := strings.ToLower(firstName) + strconv.Itoa(rnd.Intn(999))
		return escapeSQL(u)

	case "password":
		// Standard default hash for 'password123' (md5 or bcrypt fallback)
		if strings.Contains(cType, "varchar(32)") || strings.Contains(cType, "char(32)") {
			return "'482c811da5d5b4bc6d497ffa98491e38'" // md5 of password123
		}
		return escapeSQL("password123")

	case "deskripsi":
		desc := sampleDescriptions[rnd.Intn(len(sampleDescriptions))]
		return escapeSQL(desc)

	case "enum":
		// Extract enum values from enum('val1','val2',...)
		raw := strings.TrimPrefix(cType, "enum(")
		raw = strings.TrimSuffix(raw, ")")
		opts := strings.Split(raw, ",")
		if len(opts) > 0 {
			chosen := strings.Trim(strings.TrimSpace(opts[rnd.Intn(len(opts))]), "'\"")
			return escapeSQL(chosen)
		}
		return "'default'"

	case "number_generic":
		val := 10 + rnd.Intn(990)
		if strings.Contains(cType, "int") || strings.Contains(cType, "decimal") || strings.Contains(cType, "float") || strings.Contains(cType, "double") {
			return strconv.Itoa(val)
		}
		return escapeSQL(strconv.Itoa(val))

	default:
		return escapeSQL(fmt.Sprintf("Data Contoh %d", index+1))
	}
}

// ApplyPresetDatabaseTemplate creates predefined tables and populates sample data
func ApplyPresetDatabaseTemplate(dbName, templateKey string) error {
	if dbName == "" {
		return fmt.Errorf("nama database tidak boleh kosong")
	}

	var sqlSchema string

	switch templateKey {
	case "klinik_pasien":
		sqlSchema = fmt.Sprintf(`
CREATE DATABASE IF NOT EXISTS `+"`%s`"+`;
USE `+"`%s`"+`;

CREATE TABLE IF NOT EXISTS `+"`pasien`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`nik`"+` VARCHAR(20) NOT NULL,
    `+"`nama_pasien`"+` VARCHAR(100) NOT NULL,
    `+"`jenis_kelamin`"+` ENUM('Laki-laki', 'Perempuan') NOT NULL,
    `+"`tgl_lahir`"+` DATE NOT NULL,
    `+"`no_hp`"+` VARCHAR(20),
    `+"`alamat`"+` TEXT,
    `+"`keluhan_utama`"+` TEXT,
    `+"`created_at`"+` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS `+"`dokter`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`nip`"+` VARCHAR(20) NOT NULL,
    `+"`nama_dokter`"+` VARCHAR(100) NOT NULL,
    `+"`spesialis`"+` VARCHAR(50) NOT NULL,
    `+"`no_hp`"+` VARCHAR(20),
    `+"`tarif_konsultasi`"+` DECIMAL(10,2) DEFAULT 75000
);

CREATE TABLE IF NOT EXISTS `+"`kunjungan_berobat`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`pasien_id`"+` INT,
    `+"`dokter_id`"+` INT,
    `+"`tgl_kunjungan`"+` DATE,
    `+"`diagnosa`"+` TEXT,
    `+"`biaya_total`"+` DECIMAL(10,2),
    `+"`status`"+` VARCHAR(30) DEFAULT 'Selesai'
);
`, dbName, dbName)

	case "kasir_pos":
		sqlSchema = fmt.Sprintf(`
CREATE DATABASE IF NOT EXISTS `+"`%s`"+`;
USE `+"`%s`"+`;

CREATE TABLE IF NOT EXISTS `+"`kategori`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`nama_kategori`"+` VARCHAR(50) NOT NULL,
    `+"`keterangan`"+` TEXT
);

CREATE TABLE IF NOT EXISTS `+"`produk`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`kode_produk`"+` VARCHAR(30) NOT NULL,
    `+"`nama_produk`"+` VARCHAR(100) NOT NULL,
    `+"`kategori`"+` VARCHAR(50),
    `+"`harga_beli`"+` DECIMAL(10,2) NOT NULL,
    `+"`harga_jual`"+` DECIMAL(10,2) NOT NULL,
    `+"`stok`"+` INT DEFAULT 0,
    `+"`created_at`"+` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS `+"`transaksi`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`no_faktur`"+` VARCHAR(30) NOT NULL,
    `+"`tgl_transaksi`"+` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `+"`total_belanja`"+` DECIMAL(10,2) NOT NULL,
    `+"`bayar`"+` DECIMAL(10,2) NOT NULL,
    `+"`kembalian`"+` DECIMAL(10,2) NOT NULL,
    `+"`kasir`"+` VARCHAR(50) DEFAULT 'Admin'
);
`, dbName, dbName)

	case "manajemen_users":
		sqlSchema = fmt.Sprintf(`
CREATE DATABASE IF NOT EXISTS `+"`%s`"+`;
USE `+"`%s`"+`;

CREATE TABLE IF NOT EXISTS `+"`users`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`nama`"+` VARCHAR(100) NOT NULL,
    `+"`username`"+` VARCHAR(50) NOT NULL UNIQUE,
    `+"`email`"+` VARCHAR(100) NOT NULL UNIQUE,
    `+"`password`"+` VARCHAR(255) NOT NULL,
    `+"`role`"+` ENUM('admin', 'staff', 'member') DEFAULT 'staff',
    `+"`is_active`"+` TINYINT(1) DEFAULT 1,
    `+"`created_at`"+` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS `+"`karyawan`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`nip`"+` VARCHAR(20) NOT NULL,
    `+"`nama_lengkap`"+` VARCHAR(100) NOT NULL,
    `+"`jabatan`"+` VARCHAR(50),
    `+"`gaji`"+` DECIMAL(10,2),
    `+"`no_hp`"+` VARCHAR(20),
    `+"`alamat`"+` TEXT
);
`, dbName, dbName)

	case "toko_online":
		sqlSchema = fmt.Sprintf(`
CREATE DATABASE IF NOT EXISTS `+"`%s`"+`;
USE `+"`%s`"+`;

CREATE TABLE IF NOT EXISTS `+"`customers`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`nama_customer`"+` VARCHAR(100) NOT NULL,
    `+"`email`"+` VARCHAR(100) NOT NULL,
    `+"`no_hp`"+` VARCHAR(20),
    `+"`alamat_pengiriman`"+` TEXT,
    `+"`kota`"+` VARCHAR(50)
);

CREATE TABLE IF NOT EXISTS `+"`products`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`nama_produk`"+` VARCHAR(100) NOT NULL,
    `+"`kategori`"+` VARCHAR(50),
    `+"`harga`"+` DECIMAL(10,2) NOT NULL,
    `+"`stok`"+` INT DEFAULT 10,
    `+"`deskripsi`"+` TEXT
);

CREATE TABLE IF NOT EXISTS `+"`orders`"+` (
    `+"`id`"+` INT AUTO_INCREMENT PRIMARY KEY,
    `+"`no_order`"+` VARCHAR(30) NOT NULL,
    `+"`customer_id`"+` INT,
    `+"`total_tagihan`"+` DECIMAL(10,2),
    `+"`status_pembayaran`"+` VARCHAR(30) DEFAULT 'Menunggu Pembayaran',
    `+"`tgl_order`"+` DATETIME DEFAULT CURRENT_TIMESTAMP
);
`, dbName, dbName)

	default:
		return fmt.Errorf("template '%s' tidak dikenali", templateKey)
	}

	// 1. Run schema creation
	if err := RestoreDatabaseFromSQL(dbName, strings.NewReader(sqlSchema)); err != nil {
		return fmt.Errorf("gagal membuat skema tabel: %w", err)
	}

	// 2. Auto-seed 10 realistic rows for each created table
	tables, err := ListTables(dbName)
	if err == nil {
		for _, tbl := range tables {
			_, _ = GenerateSmartSeedData(dbName, tbl, 10)
		}
	}

	return nil
}

// Quick Database Creator Structs & Engine
type QuickColumnDef struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	AllowNull  bool   `json:"allow_null"`
	DefaultVal string `json:"default_val"`
}

type QuickTableDef struct {
	Name    string           `json:"name"`
	Columns []QuickColumnDef `json:"columns"`
}

type QuickDatabaseCreateRequest struct {
	DatabaseName   string          `json:"database_name"`
	Tables         []QuickTableDef `json:"tables"`
	SeedSampleData bool            `json:"seed_sample_data"`
	SeedCount      int             `json:"seed_count"`
}

// CreateQuickDatabase builds a database and user-defined tables, with optional smart sample data
func CreateQuickDatabase(req QuickDatabaseCreateRequest) error {
	req.DatabaseName = strings.TrimSpace(req.DatabaseName)
	if req.DatabaseName == "" {
		return fmt.Errorf("nama database tidak boleh kosong")
	}

	// Sanitize DB Name
	cleanDb := strings.ToLower(req.DatabaseName)
	cleanDb = strings.ReplaceAll(cleanDb, "-", "_")
	cleanDb = strings.ReplaceAll(cleanDb, " ", "_")

	if len(req.Tables) == 0 {
		return fmt.Errorf("minimal sertakan 1 tabel untuk dibuat")
	}

	settings := GetCurrentSettings()
	if !Manager.IsPortOpen(settings.MariaDBPort) {
		if err := Manager.StartMariaDB(); err != nil {
			return fmt.Errorf("MariaDB tidak aktif: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`;\nUSE `%s`;\n\n", cleanDb, cleanDb))

	for _, tbl := range req.Tables {
		tblName := strings.TrimSpace(tbl.Name)
		if tblName == "" {
			continue
		}
		cleanTbl := strings.ToLower(tblName)
		cleanTbl = strings.ReplaceAll(cleanTbl, "-", "_")
		cleanTbl = strings.ReplaceAll(cleanTbl, " ", "_")

		sb.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS `%s` (\n", cleanTbl))

		var colDefs []string
		hasId := false

		for _, col := range tbl.Columns {
			cName := strings.TrimSpace(col.Name)
			if cName == "" {
				continue
			}
			cleanCol := strings.ToLower(cName)
			cleanCol = strings.ReplaceAll(cleanCol, " ", "_")
			cleanCol = strings.ReplaceAll(cleanCol, "-", "_")

			cType := strings.TrimSpace(col.Type)
			if cType == "" {
				cType = "VARCHAR(100)"
			}

			if cleanCol == "id" {
				hasId = true
				colDefs = append(colDefs, "    `id` INT AUTO_INCREMENT PRIMARY KEY")
				continue
			}

			nullStr := "NOT NULL"
			if col.AllowNull {
				nullStr = "NULL"
			}

			defStr := ""
			if col.DefaultVal != "" {
				defStr = fmt.Sprintf(" DEFAULT '%s'", strings.ReplaceAll(col.DefaultVal, "'", "''"))
			} else if strings.EqualFold(cleanCol, "created_at") && strings.Contains(strings.ToLower(cType), "datetime") {
				defStr = " DEFAULT CURRENT_TIMESTAMP"
			}

			colDefs = append(colDefs, fmt.Sprintf("    `%s` %s %s%s", cleanCol, cType, nullStr, defStr))
		}

		// If no primary key id was defined, automatically prepend standard auto-increment id
		if !hasId {
			colDefs = append([]string{"    `id` INT AUTO_INCREMENT PRIMARY KEY"}, colDefs...)
		}

		sb.WriteString(strings.Join(colDefs, ",\n"))
		sb.WriteString("\n);\n\n")
	}

	// 1. Execute SQL Schema Creation
	if err := RestoreDatabaseFromSQL(cleanDb, strings.NewReader(sb.String())); err != nil {
		return fmt.Errorf("gagal membuat skema database: %w", err)
	}

	// 2. Auto-seed sample data if requested
	if req.SeedSampleData {
		count := req.SeedCount
		if count <= 0 {
			count = 10
		}
		for _, tbl := range req.Tables {
			cleanTbl := strings.ToLower(strings.TrimSpace(tbl.Name))
			cleanTbl = strings.ReplaceAll(cleanTbl, "-", "_")
			cleanTbl = strings.ReplaceAll(cleanTbl, " ", "_")
			if cleanTbl != "" {
				_, _ = GenerateSmartSeedData(cleanDb, cleanTbl, count)
			}
		}
	}

	return nil
}

// ==========================================
// Database Editor & Relations Engine
// ==========================================

type TableDataResult struct {
	Columns    []TableColumnInfo `json:"columns"`
	Rows       []map[string]any  `json:"rows"`
	TotalRows  int               `json:"total_rows"`
	Page       int               `json:"page"`
	Limit      int               `json:"limit"`
	TotalPages int               `json:"total_pages"`
	PrimaryKey string            `json:"primary_key"`
}

type TableRelationInfo struct {
	ConstraintName       string `json:"constraint_name"`
	TableName            string `json:"table_name"`
	ColumnName           string `json:"column_name"`
	ReferencedTableName  string `json:"referenced_table_name"`
	ReferencedColumnName string `json:"referenced_column_name"`
	UpdateRule           string `json:"update_rule"`
	DeleteRule           string `json:"delete_rule"`
}

type TableRelationRequest struct {
	DatabaseName         string `json:"database_name"`
	TableName            string `json:"table_name"`
	ColumnName           string `json:"column_name"`
	ReferencedTableName  string `json:"referenced_table_name"`
	ReferencedColumnName string `json:"referenced_column_name"`
	OnDelete             string `json:"on_delete"`
	OnUpdate             string `json:"on_update"`
}

func execMariaDBRawQuery(dbName, query string) (string, error) {
	settings := GetCurrentSettings()
	clientBin := getMariaDBClientBin()
	if clientBin == "" {
		return "", fmt.Errorf("mariadb client binary tidak ditemukan")
	}

	if !Manager.IsPortOpen(settings.MariaDBPort) {
		if err := Manager.StartMariaDB(); err != nil {
			return "", fmt.Errorf("MariaDB tidak aktif: %w", err)
		}
		time.Sleep(1 * time.Second)
	}

	var args []string
	args = append(args, "-u", "root", "-P", fmt.Sprintf("%d", settings.MariaDBPort), "-A", "--default-character-set=utf8mb4")
	if dbName != "" {
		args = append(args, dbName)
	}
	args = append(args, "-e", query)

	cmd := exec.Command(clientBin, args...)
	SetCmdHideWindow(cmd)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("%s", strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// GetTableData retrieves paginated table rows with optional text search and column metadata
func GetTableData(dbName, tableName, search string, page, limit int) (*TableDataResult, error) {
	if dbName == "" || tableName == "" {
		return nil, fmt.Errorf("nama database dan tabel tidak boleh kosong")
	}

	cols, err := DescribeTable(dbName, tableName)
	if err != nil {
		return nil, err
	}

	pkCol := "id"
	foundPk := false
	for _, c := range cols {
		if c.Key == "PRI" {
			pkCol = c.Field
			foundPk = true
			break
		}
	}
	if !foundPk && len(cols) > 0 {
		pkCol = cols[0].Field
	}

	if page < 1 {
		page = 1
	}
	if limit < 1 || limit > 500 {
		limit = 25
	}
	offset := (page - 1) * limit

	// Build WHERE clause if search query provided
	whereClause := ""
	search = strings.TrimSpace(search)
	if search != "" {
		safeSearch := strings.ReplaceAll(search, "\\", "\\\\")
		safeSearch = strings.ReplaceAll(safeSearch, "'", "\\'")
		var likes []string
		for _, c := range cols {
			likes = append(likes, fmt.Sprintf("CAST(`%s` AS CHAR) LIKE '%%%s%%'", c.Field, safeSearch))
		}
		if len(likes) > 0 {
			whereClause = " WHERE " + strings.Join(likes, " OR ")
		}
	}

	// 1. Total Rows Query
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM `%s`%s;", tableName, whereClause)
	countOut, err := execMariaDBRawQuery(dbName, countQuery)
	totalRows := 0
	if err == nil {
		lines := strings.Split(strings.TrimSpace(countOut), "\n")
		if len(lines) >= 2 {
			totalRows, _ = strconv.Atoi(strings.TrimSpace(lines[1]))
		}
	}

	totalPages := 1
	if totalRows > 0 {
		totalPages = (totalRows + limit - 1) / limit
	}

	// 2. Fetch Page Rows
	orderClause := ""
	if pkCol != "" {
		orderClause = fmt.Sprintf(" ORDER BY `%s` DESC", pkCol)
	}
	dataQuery := fmt.Sprintf("SELECT * FROM `%s`%s%s LIMIT %d OFFSET %d;", tableName, whereClause, orderClause, limit, offset)
	dataOut, err := execMariaDBRawQuery(dbName, dataQuery)
	if err != nil {
		return nil, err
	}

	var rows []map[string]any
	lines := strings.Split(dataOut, "\n")
	if len(lines) > 1 {
		headers := strings.Split(strings.TrimRight(lines[0], "\r"), "\t")
		for _, line := range lines[1:] {
			line = strings.TrimRight(line, "\r")
			if line == "" {
				continue
			}
			parts := strings.Split(line, "\t")
			rowMap := make(map[string]any)
			for idx, h := range headers {
				if idx < len(parts) {
					val := parts[idx]
					if val == "NULL" {
						rowMap[h] = nil
					} else {
						rowMap[h] = val
					}
				} else {
					rowMap[h] = nil
				}
			}
			rows = append(rows, rowMap)
		}
	}

	return &TableDataResult{
		Columns:    cols,
		Rows:       rows,
		TotalRows:  totalRows,
		Page:       page,
		Limit:      limit,
		TotalPages: totalPages,
		PrimaryKey: pkCol,
	}, nil
}

// InsertTableRow inserts a new row into the specified table
func InsertTableRow(dbName, tableName string, rowData map[string]any) error {
	if dbName == "" || tableName == "" {
		return fmt.Errorf("database dan tabel tidak boleh kosong")
	}

	var colNames []string
	var valPlaceholders []string

	for k, v := range rowData {
		cleanCol := strings.TrimSpace(k)
		if cleanCol == "" {
			continue
		}
		colNames = append(colNames, fmt.Sprintf("`%s`", cleanCol))

		if v == nil {
			valPlaceholders = append(valPlaceholders, "NULL")
		} else {
			strVal := fmt.Sprintf("%v", v)
			safeVal := strings.ReplaceAll(strVal, "\\", "\\\\")
			safeVal = strings.ReplaceAll(safeVal, "'", "\\'")
			valPlaceholders = append(valPlaceholders, fmt.Sprintf("'%s'", safeVal))
		}
	}

	if len(colNames) == 0 {
		return fmt.Errorf("tidak ada data kolom untuk disimpan")
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s);",
		tableName,
		strings.Join(colNames, ", "),
		strings.Join(valPlaceholders, ", "))

	_, err := execMariaDBRawQuery(dbName, query)
	return err
}

// UpdateTableRow updates an existing row identified by its Primary Key
func UpdateTableRow(dbName, tableName, pkCol string, pkVal any, rowData map[string]any) error {
	if dbName == "" || tableName == "" || pkCol == "" {
		return fmt.Errorf("database, tabel, dan primary key tidak boleh kosong")
	}

	var setClauses []string
	for k, v := range rowData {
		cleanCol := strings.TrimSpace(k)
		if cleanCol == "" || cleanCol == pkCol {
			continue // Do not update primary key
		}

		if v == nil {
			setClauses = append(setClauses, fmt.Sprintf("`%s` = NULL", cleanCol))
		} else {
			strVal := fmt.Sprintf("%v", v)
			safeVal := strings.ReplaceAll(strVal, "\\", "\\\\")
			safeVal = strings.ReplaceAll(safeVal, "'", "\\'")
			setClauses = append(setClauses, fmt.Sprintf("`%s` = '%s'", cleanCol, safeVal))
		}
	}

	if len(setClauses) == 0 {
		return fmt.Errorf("tidak ada perubahan data")
	}

	safePkVal := fmt.Sprintf("%v", pkVal)
	safePkVal = strings.ReplaceAll(safePkVal, "'", "\\'")

	query := fmt.Sprintf("UPDATE `%s` SET %s WHERE `%s` = '%s' LIMIT 1;",
		tableName,
		strings.Join(setClauses, ", "),
		pkCol,
		safePkVal)

	_, err := execMariaDBRawQuery(dbName, query)
	return err
}

// DeleteTableRow removes a row identified by primary key
func DeleteTableRow(dbName, tableName, pkCol string, pkVal any) error {
	if dbName == "" || tableName == "" || pkCol == "" {
		return fmt.Errorf("database, tabel, dan primary key tidak boleh kosong")
	}

	safePkVal := fmt.Sprintf("%v", pkVal)
	safePkVal = strings.ReplaceAll(safePkVal, "'", "\\'")

	query := fmt.Sprintf("DELETE FROM `%s` WHERE `%s` = '%s' LIMIT 1;", tableName, pkCol, safePkVal)
	_, err := execMariaDBRawQuery(dbName, query)
	return err
}

// AddTableColumn adds a new column to an existing table
func AddTableColumn(dbName, tableName string, col QuickColumnDef, afterCol string, isFirst bool) error {
	colName := strings.ToLower(strings.TrimSpace(col.Name))
	colName = strings.ReplaceAll(colName, " ", "_")
	colName = strings.ReplaceAll(colName, "-", "_")
	if colName == "" {
		return fmt.Errorf("nama kolom tidak boleh kosong")
	}

	colType := strings.TrimSpace(col.Type)
	if colType == "" {
		colType = "VARCHAR(100)"
	}

	nullStr := "NOT NULL"
	if col.AllowNull {
		nullStr = "NULL"
	}

	defStr := ""
	if col.DefaultVal != "" {
		defStr = fmt.Sprintf(" DEFAULT '%s'", strings.ReplaceAll(col.DefaultVal, "'", "\\'"))
	}

	posStr := ""
	if isFirst {
		posStr = " FIRST"
	} else if strings.TrimSpace(afterCol) != "" {
		posStr = fmt.Sprintf(" AFTER `%s`", strings.TrimSpace(afterCol))
	}

	query := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s %s%s%s;",
		tableName, colName, colType, nullStr, defStr, posStr)

	_, err := execMariaDBRawQuery(dbName, query)
	return err
}

// ModifyTableColumn alters an existing column definition
func ModifyTableColumn(dbName, tableName, oldColName string, col QuickColumnDef) error {
	oldColName = strings.TrimSpace(oldColName)
	newColName := strings.ToLower(strings.TrimSpace(col.Name))
	newColName = strings.ReplaceAll(newColName, " ", "_")
	newColName = strings.ReplaceAll(newColName, "-", "_")
	if oldColName == "" || newColName == "" {
		return fmt.Errorf("nama kolom tidak boleh kosong")
	}

	colType := strings.TrimSpace(col.Type)
	if colType == "" {
		colType = "VARCHAR(100)"
	}

	nullStr := "NOT NULL"
	if col.AllowNull {
		nullStr = "NULL"
	}

	defStr := ""
	if col.DefaultVal != "" {
		defStr = fmt.Sprintf(" DEFAULT '%s'", strings.ReplaceAll(col.DefaultVal, "'", "\\'"))
	}

	query := fmt.Sprintf("ALTER TABLE `%s` CHANGE COLUMN `%s` `%s` %s %s%s;",
		tableName, oldColName, newColName, colType, nullStr, defStr)

	_, err := execMariaDBRawQuery(dbName, query)
	return err
}

// DropTableColumn drops a column from a table
func DropTableColumn(dbName, tableName, colName string) error {
	colName = strings.TrimSpace(colName)
	if colName == "" {
		return fmt.Errorf("nama kolom tidak boleh kosong")
	}

	query := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`;", tableName, colName)
	_, err := execMariaDBRawQuery(dbName, query)
	return err
}

// DropTable drops a table from the database
func DropTable(dbName, tableName string) error {
	tableName = strings.TrimSpace(tableName)
	if tableName == "" {
		return fmt.Errorf("nama tabel tidak boleh kosong")
	}

	query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`;", tableName)
	_, err := execMariaDBRawQuery(dbName, query)
	return err
}

// ListTableRelations retrieves all foreign key constraints in a database
func ListTableRelations(dbName string) ([]TableRelationInfo, error) {
	if dbName == "" {
		return nil, fmt.Errorf("nama database tidak boleh kosong")
	}

	query := fmt.Sprintf(`
		SELECT 
			k.CONSTRAINT_NAME,
			k.TABLE_NAME,
			k.COLUMN_NAME,
			k.REFERENCED_TABLE_NAME,
			k.REFERENCED_COLUMN_NAME,
			COALESCE(r.UPDATE_RULE, 'RESTRICT'),
			COALESCE(r.DELETE_RULE, 'RESTRICT')
		FROM information_schema.KEY_COLUMN_USAGE k
		LEFT JOIN information_schema.REFERENTIAL_CONSTRAINTS r 
			ON k.CONSTRAINT_NAME = r.CONSTRAINT_NAME 
			AND k.CONSTRAINT_SCHEMA = r.CONSTRAINT_SCHEMA
		WHERE k.CONSTRAINT_SCHEMA = '%s' 
			AND k.REFERENCED_TABLE_NAME IS NOT NULL;
	`, strings.ReplaceAll(dbName, "'", "\\'"))

	out, err := execMariaDBRawQuery(dbName, query)
	if err != nil {
		return nil, err
	}

	var list []TableRelationInfo
	lines := strings.Split(out, "\n")
	for i, line := range lines {
		line = strings.TrimRight(line, "\r")
		if line == "" || i == 0 { // skip header
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) >= 5 {
			rel := TableRelationInfo{
				ConstraintName:       parts[0],
				TableName:            parts[1],
				ColumnName:           parts[2],
				ReferencedTableName:  parts[3],
				ReferencedColumnName: parts[4],
				UpdateRule:           "RESTRICT",
				DeleteRule:           "RESTRICT",
			}
			if len(parts) > 5 && parts[5] != "NULL" {
				rel.UpdateRule = parts[5]
			}
			if len(parts) > 6 && parts[6] != "NULL" {
				rel.DeleteRule = parts[6]
			}
			list = append(list, rel)
		}
	}

	return list, nil
}

// AddTableRelation creates a foreign key relationship between two tables
func AddTableRelation(req TableRelationRequest) error {
	dbName := strings.TrimSpace(req.DatabaseName)
	srcTable := strings.TrimSpace(req.TableName)
	srcCol := strings.TrimSpace(req.ColumnName)
	refTable := strings.TrimSpace(req.ReferencedTableName)
	refCol := strings.TrimSpace(req.ReferencedColumnName)

	if dbName == "" || srcTable == "" || srcCol == "" || refTable == "" || refCol == "" {
		return fmt.Errorf("seluruh field tabel dan kolom relasi wajib diisi")
	}

	onDelete := strings.ToUpper(strings.TrimSpace(req.OnDelete))
	if onDelete == "" {
		onDelete = "CASCADE"
	}
	onUpdate := strings.ToUpper(strings.TrimSpace(req.OnUpdate))
	if onUpdate == "" {
		onUpdate = "CASCADE"
	}

	constraintName := fmt.Sprintf("fk_%s_%s_%s", srcTable, srcCol, refTable)
	if len(constraintName) > 60 {
		constraintName = constraintName[:60]
	}

	query := fmt.Sprintf("ALTER TABLE `%s` ADD CONSTRAINT `%s` FOREIGN KEY (`%s`) REFERENCES `%s`(`%s`) ON DELETE %s ON UPDATE %s;",
		srcTable, constraintName, srcCol, refTable, refCol, onDelete, onUpdate)

	_, err := execMariaDBRawQuery(dbName, query)
	return err
}

// DropTableRelation removes a foreign key relationship
func DropTableRelation(dbName, tableName, constraintName string) error {
	dbName = strings.TrimSpace(dbName)
	tableName = strings.TrimSpace(tableName)
	constraintName = strings.TrimSpace(constraintName)

	if dbName == "" || tableName == "" || constraintName == "" {
		return fmt.Errorf("database, tabel, dan nama relasi (constraint) tidak boleh kosong")
	}

	query := fmt.Sprintf("ALTER TABLE `%s` DROP FOREIGN KEY `%s`;", tableName, constraintName)
	_, err := execMariaDBRawQuery(dbName, query)
	return err
}

