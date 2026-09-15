//go:build windows

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/UserExistsError/conpty"
	"github.com/gorilla/websocket"
)

func startTerminalBackend(conn *websocket.Conn, htdocsDir string, env []string, settings Settings, writeToWS func([]byte) error) {
	shellCmd := "cmd.exe"
	if strings.ToLower(settings.Shell) == "powershell" || strings.ToLower(settings.Shell) == "pwsh" {
		shellCmd = "powershell.exe -NoLogo -ExecutionPolicy Bypass"
	}

	// Try Windows ConPTY first
	if conpty.IsConPtyAvailable() {
		cpty, err := conpty.Start(shellCmd,
			conpty.ConPtyWorkDir(htdocsDir),
			conpty.ConPtyEnv(env),
			conpty.ConPtyDimensions(120, 30),
		)
		if err == nil {
			defer cpty.Close()

			go func() {
				buf := make([]byte, 4096)
				for {
					n, err := cpty.Read(buf)
					if n > 0 {
						if err := writeToWS(buf[:n]); err != nil {
							return
						}
					}
					if err != nil {
						return
					}
				}
			}()

			for {
				messageType, p, err := conn.ReadMessage()
				if err != nil {
					break
				}

				if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
					var ctrlMsg TerminalControlMsg
					if err := json.Unmarshal(p, &ctrlMsg); err == nil && ctrlMsg.Type == "resize" {
						if ctrlMsg.Cols > 0 && ctrlMsg.Rows > 0 {
							_ = cpty.Resize(ctrlMsg.Cols, ctrlMsg.Rows)
						}
					} else {
						_, _ = cpty.Write(p)
					}
				}
			}
			return
		}
	}

	// Fallback to standard pipe
	var cmd *exec.Cmd
	if strings.ToLower(settings.Shell) == "powershell" || strings.ToLower(settings.Shell) == "pwsh" {
		cmd = exec.Command("powershell.exe", "-NoExit", "-NoLogo", "-ExecutionPolicy", "Bypass")
	} else {
		cmd = exec.Command("cmd.exe", "/K")
	}

	cmd.Dir = htdocsDir
	cmd.Env = env
	SetCmdHideWindow(cmd)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		_ = writeToWS([]byte(fmt.Sprintf("Failed to get stdin pipe: %v\r\n", err)))
		return
	}
	defer stdin.Close()

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = writeToWS([]byte(fmt.Sprintf("Failed to get stdout pipe: %v\r\n", err)))
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		_ = writeToWS([]byte(fmt.Sprintf("Failed to get stderr pipe: %v\r\n", err)))
		return
	}

	if err := cmd.Start(); err != nil {
		_ = writeToWS([]byte(fmt.Sprintf("Failed to start shell: %v\r\n", err)))
		return
	}

	defer func() {
		if cmd.Process != nil {
			_ = KillProcessTree(cmd.Process.Pid)
		}
	}()

	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				if err := writeToWS(buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	go func() {
		buf := make([]byte, 2048)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				if err := writeToWS(buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if messageType == websocket.TextMessage || messageType == websocket.BinaryMessage {
			msgStr := string(p)
			if msgStr == "\r" {
				msgStr = "\r\n"
			}
			_, err = io.WriteString(stdin, msgStr)
			if err != nil {
				break
			}
		}
	}
}
