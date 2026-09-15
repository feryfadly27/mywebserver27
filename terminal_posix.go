//go:build !windows

package main

import (
	"fmt"
	"io"
	"os/exec"

	"github.com/gorilla/websocket"
)

func startTerminalBackend(conn *websocket.Conn, htdocsDir string, env []string, settings Settings, writeToWS func([]byte) error) {
	shell := "/bin/zsh"
	if !fileExists(shell) {
		shell = "/bin/bash"
	}
	if !fileExists(shell) {
		shell = "/bin/sh"
	}

	cmd := exec.Command(shell, "-l")
	cmd.Dir = htdocsDir
	cmd.Env = append(env, "TERM=xterm-256color")
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
		_ = writeToWS([]byte(fmt.Sprintf("Failed to start shell %s: %v\r\n", shell, err)))
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
			_, err = io.WriteString(stdin, string(p))
			if err != nil {
				break
			}
		}
	}
}
