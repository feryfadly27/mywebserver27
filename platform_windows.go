//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func OpenURL(url string) {
	cmd := exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
}

func OpenFolder(targetPath string) {
	_ = os.MkdirAll(targetPath, 0755)
	cmd := exec.Command("cmd.exe", "/c", "start", "", filepath.Clean(targetPath))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
}

func OpenNativeTerminal(targetPath string, settings Settings) {
	_ = os.MkdirAll(targetPath, 0755)
	phpDir := filepath.Join(AppRootDir, "bin", "php")
	mariadbBinDir := filepath.Join(AppRootDir, "bin", "mariadb", "bin")
	apacheBinDir := filepath.Join(AppRootDir, "bin", "apache", "bin")
	customPath := fmt.Sprintf("PATH=%s;%s;%s;%s", phpDir, mariadbBinDir, apacheBinDir, os.Getenv("PATH"))

	var cmd *exec.Cmd
	if strings.ToLower(settings.Shell) == "powershell" || strings.ToLower(settings.Shell) == "pwsh" {
		psInit := fmt.Sprintf("$env:PATH='%s;'+$env:PATH; $env:MYSQL_TCP_PORT='%d'; $env:MYSQL_HOST='127.0.0.1'; $env:MARIADB_PORT='%d'; Set-Location '%s'; Write-Host '⚡ MyLokalWebserver Terminal' -ForegroundColor Cyan; Write-Host '📁 Folder: %s' -ForegroundColor Gray; Write-Host '💡 Tip: Ketik \"mysql -u root\" untuk langsung masuk ke database' -ForegroundColor DarkGray; Write-Host ''", strings.ReplaceAll(fmt.Sprintf("%s;%s;%s", phpDir, mariadbBinDir, apacheBinDir), "'", "''"), settings.MariaDBPort, settings.MariaDBPort, targetPath, targetPath)
		cmd = exec.Command("cmd.exe", "/c", "start", "powershell.exe", "-NoExit", "-ExecutionPolicy", "Bypass", "-Command", psInit)
	} else {
		cmdInit := fmt.Sprintf("title MyLokalWebserver Terminal & set MYSQL_TCP_PORT=%d & set MYSQL_HOST=127.0.0.1 & set MARIADB_PORT=%d & cd /d \"%s\" & echo ⚡ MyLokalWebserver Native Terminal & echo 📁 Folder: %s & echo 💡 Tip: Ketik \"mysql -u root\" untuk langsung masuk ke database & echo.", settings.MariaDBPort, settings.MariaDBPort, targetPath, targetPath)
		cmd = exec.Command("cmd.exe", "/c", "start", "cmd.exe", "/K", cmdInit)
	}

	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") || strings.HasPrefix(strings.ToUpper(e), "MYSQL_") || strings.HasPrefix(strings.ToUpper(e), "MARIADB_") {
			continue
		}
		env = append(env, e)
	}

	env = append(env,
		customPath,
		fmt.Sprintf("MYSQL_TCP_PORT=%d", settings.MariaDBPort),
		"MYSQL_HOST=127.0.0.1",
		fmt.Sprintf("MARIADB_PORT=%d", settings.MariaDBPort),
		"MARIADB_HOST=127.0.0.1",
	)
	cmd.Env = env
	cmd.Dir = targetPath
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Start()
}

func SetCmdHideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
}

func SetProcessGroupAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
		HideWindow:    true,
	}
}

func KillProcessTree(pid int) error {
	cmd := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprintf("%d", pid))
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	return cmd.Run()
}

func KillByName(name string) {
	cmd := exec.Command("taskkill", "/F", "/IM", name)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	_ = cmd.Run()
}

func GetExecutableExt() string {
	return ".exe"
}

func GetPathListSeparator() string {
	return ";"
}

func GetHostsFilePath() string {
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot == "" {
		systemRoot = `C:\Windows`
	}
	return filepath.Join(systemRoot, "System32", "drivers", "etc", "hosts")
}
