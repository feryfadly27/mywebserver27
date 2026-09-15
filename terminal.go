package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local control panel
	},
}

type TerminalControlMsg struct {
	Type string `json:"type"`
	Cols int    `json:"cols,omitempty"`
	Rows int    `json:"rows,omitempty"`
}

func HandleTerminalWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	var wsLock sync.Mutex
	writeToWS := func(data []byte) error {
		wsLock.Lock()
		defer wsLock.Unlock()
		return conn.WriteMessage(websocket.TextMessage, data)
	}

	settings := GetCurrentSettings()
	htdocsDir := filepath.Join(AppRootDir, "www", "htdocs")
	_ = os.MkdirAll(htdocsDir, 0755)

	// Configure PATH & MySQL environment
	phpDir := filepath.Join(AppRootDir, "bin", "php")
	mariadbBinDir := filepath.Join(AppRootDir, "bin", "mariadb", "bin")
	apacheBinDir := filepath.Join(AppRootDir, "bin", "apache", "bin")

	var env []string
	for _, e := range os.Environ() {
		if strings.HasPrefix(strings.ToUpper(e), "PATH=") {
			continue
		}
		if strings.HasPrefix(strings.ToUpper(e), "MYSQL_") || strings.HasPrefix(strings.ToUpper(e), "MARIADB_") {
			continue
		}
		env = append(env, e)
	}

	sep := GetPathListSeparator()
	customPath := fmt.Sprintf("PATH=%s%s%s%s%s%s%s", phpDir, sep, mariadbBinDir, sep, apacheBinDir, sep, os.Getenv("PATH"))
	env = append(env,
		customPath,
		fmt.Sprintf("MYSQL_TCP_PORT=%d", settings.MariaDBPort),
		"MYSQL_HOST=127.0.0.1",
		fmt.Sprintf("MARIADB_PORT=%d", settings.MariaDBPort),
		"MARIADB_HOST=127.0.0.1",
	)

	startTerminalBackend(conn, htdocsDir, env, settings, writeToWS)
}
