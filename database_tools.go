package main

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
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
