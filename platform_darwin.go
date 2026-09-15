//go:build darwin

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func OpenURL(url string) {
	_ = exec.Command("open", url).Start()
}

func OpenFolder(targetPath string) {
	_ = os.MkdirAll(targetPath, 0755)
	_ = exec.Command("open", filepath.Clean(targetPath)).Start()
}

func OpenNativeTerminal(targetPath string, settings Settings) {
	_ = os.MkdirAll(targetPath, 0755)
	phpDir := filepath.Join(AppRootDir, "bin", "php")
	mariadbBinDir := filepath.Join(AppRootDir, "bin", "mariadb", "bin")
	apacheBinDir := filepath.Join(AppRootDir, "bin", "apache", "bin")
	customPath := fmt.Sprintf("%s:%s:%s:$PATH", phpDir, mariadbBinDir, apacheBinDir)

	script := fmt.Sprintf(`tell application "Terminal"
    do script "export PATH=%s; export MYSQL_TCP_PORT=%d; export MYSQL_HOST=127.0.0.1; cd '%s'; clear; echo '⚡ MyLokalWebserver macOS Terminal'; echo '📁 Folder: %s'; echo ''"
    activate
end tell`, customPath, settings.MariaDBPort, targetPath, targetPath)

	_ = exec.Command("osascript", "-e", script).Start()
}

func SetCmdHideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func SetProcessGroupAttributes(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}
}

func KillProcessTree(pid int) error {
	_ = syscall.Kill(-pid, syscall.SIGKILL)
	_ = exec.Command("pkill", "-P", fmt.Sprintf("%d", pid)).Run()
	return nil
}

func KillByName(name string) {
	_ = exec.Command("pkill", "-f", name).Run()
}

func GetExecutableExt() string {
	return ""
}

func GetPathListSeparator() string {
	return ":"
}

func GetHostsFilePath() string {
	return "/etc/hosts"
}
