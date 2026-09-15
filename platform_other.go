//go:build !windows && !darwin

package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
)

func OpenURL(url string) {
	_ = exec.Command("xdg-open", url).Start()
}

func OpenFolder(targetPath string) {
	_ = os.MkdirAll(targetPath, 0755)
	_ = exec.Command("xdg-open", filepath.Clean(targetPath)).Start()
}

func OpenNativeTerminal(targetPath string, settings Settings) {
	_ = os.MkdirAll(targetPath, 0755)
	_ = exec.Command("x-terminal-emulator", "--working-directory="+targetPath).Start()
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
