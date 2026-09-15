package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func InitMariaDBData() error {
	dataDir := filepath.Join(AppRootDir, "data", "mysql")
	mariadbDir := filepath.Join(AppRootDir, "bin", "mariadb")

	// Check if already initialized
	mysqlSystemDb := filepath.Join(dataDir, "mysql")
	if _, err := os.Stat(mysqlSystemDb); err == nil {
		return nil // Already initialized
	}

	_ = os.MkdirAll(dataDir, 0755)

	exeExt := GetExecutableExt()
	installExe := filepath.Join(mariadbDir, "bin", "mariadb-install-db"+exeExt)
	if _, err := os.Stat(installExe); os.IsNotExist(err) {
		installExe = filepath.Join(mariadbDir, "bin", "mysql_install_db"+exeExt)
	}
	if _, err := os.Stat(installExe); os.IsNotExist(err) {
		installExe = filepath.Join(mariadbDir, "bin", "mariadb-install-db")
	}
	if _, err := os.Stat(installExe); os.IsNotExist(err) {
		installExe = filepath.Join(mariadbDir, "bin", "mysql_install_db")
	}
	if _, err := os.Stat(installExe); os.IsNotExist(err) {
		// If installer binary doesn't exist, check if data template exists in mariadb/data
		templateData := filepath.Join(mariadbDir, "data")
		if _, err := os.Stat(templateData); err == nil {
			// Copy files from mariadb/data to data/mysql
			return copyDir(templateData, dataDir)
		}
		return fmt.Errorf("mariadb initialization tool not found")
	}

	// Run mariadb-install-db --datadir=...
	cmd := exec.Command(installExe, fmt.Sprintf("--datadir=%s", dataDir), "--default-user=root")
	cmd.Dir = filepath.Join(mariadbDir, "bin")
	SetCmdHideWindow(cmd)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("database init failed: %s (err: %w)", string(out), err)
	}

	return nil
}

func copyDir(src string, dst string) error {
	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if entry.IsDir() {
			_ = os.MkdirAll(dstPath, 0755)
			if err := copyDir(srcPath, dstPath); err != nil {
				return err
			}
		} else {
			data, err := os.ReadFile(srcPath)
			if err != nil {
				return err
			}
			if err := os.WriteFile(dstPath, data, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}
