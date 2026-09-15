package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

//go:embed web/*
var webFS embed.FS

var AppRootDir string

func main() {
	noBrowser := flag.Bool("no-browser", false, "Do not auto-open browser")
	customPort := flag.Int("port", 0, "Override control panel port")
	flag.Parse()

	// Determine Application Root Directory
	exePath, err := os.Executable()
	if err != nil {
		AppRootDir, _ = os.Getwd()
	} else {
		exeDir := filepath.Dir(exePath)
		if strings.Contains(exeDir, ".app/Contents/MacOS") || strings.Contains(exeDir, ".app\\Contents\\MacOS") {
			cwd, _ := os.Getwd()
			if cwd != "" && !strings.Contains(cwd, ".app/Contents") {
				AppRootDir = cwd
			} else {
				AppRootDir = filepath.Clean(filepath.Join(exeDir, "..", "..", ".."))
			}
		} else {
			AppRootDir = exeDir
		}
	}

	log.Printf("==================================================")
	log.Printf("  ⚡ MyLokalWebserver (Portable Web Server)")
	log.Printf("  📁 Location: %s", AppRootDir)
	log.Printf("==================================================")

	// Create necessary folders
	_ = os.MkdirAll(filepath.Join(AppRootDir, "bin"), 0755)
	_ = os.MkdirAll(filepath.Join(AppRootDir, "data", "mysql"), 0755)
	_ = os.MkdirAll(filepath.Join(AppRootDir, "www", "htdocs"), 0755)
	_ = os.MkdirAll(filepath.Join(AppRootDir, "logs"), 0755)
	_ = os.MkdirAll(filepath.Join(AppRootDir, "tmp"), 0755)

	// Load Settings
	settings, err := LoadSettings()
	if err != nil {
		log.Printf("Warning: Failed to load settings: %v", err)
	}

	if *customPort > 0 {
		settings.PanelPort = *customPort
	}

	// Create starter page if empty
	CreateStarterPage()

	// Clean any leftover .old binary from previous update
	CleanupOldBinary()

	// Check if binaries are already installed
	allInstalled, _ := CheckBinariesExist()
	if allInstalled {
		_ = GenerateAllConfigs(settings)
		if settings.AutoStart {
			log.Println("Auto-starting Apache & MariaDB...")
			go func() {
				time.Sleep(500 * time.Millisecond)
				_ = StartMariaDB()
				_ = StartApache()
			}()
		}
	} else {
		log.Println("Binaries not detected. Please complete setup in the web control panel.")
	}

	// Setup HTTP Routes
	mux := http.NewServeMux()

	// API Routes
	mux.HandleFunc("/api/status", HandleStatus)
	mux.HandleFunc("/api/service", HandleServiceAction)
	mux.HandleFunc("/api/settings", HandleSettings)
	mux.HandleFunc("/api/download/start", HandleDownloadStart)
	mux.HandleFunc("/api/download/progress", HandleDownloadProgressSSE)
	mux.HandleFunc("/api/open-folder", HandleOpenFolder)
	mux.HandleFunc("/api/open-terminal", HandleOpenNativeTerminal)
	mux.HandleFunc("/api/projects", HandleProjects)
	mux.HandleFunc("/api/projects/create", HandleProjectCreate)
	mux.HandleFunc("/api/vhosts/save", HandleVHostSave)
	mux.HandleFunc("/api/vhosts/delete", HandleVHostDelete)
	mux.HandleFunc("/api/vhosts/toggle", HandleVHostToggle)
	mux.HandleFunc("/api/shutdown", HandleShutdown)
	mux.HandleFunc("/api/restart", HandleRestartApp)
	mux.HandleFunc("/api/update/check", HandleCheckUpdate)
	mux.HandleFunc("/api/update/progress", HandleUpdateProgress)
	mux.HandleFunc("/ws/terminal", HandleTerminalWebSocket)

	// Network LAN IP & QR Sharing
	mux.HandleFunc("/api/network/ips", HandleNetworkIPs)

	// Database Backup & Restore Tools
	mux.HandleFunc("/api/db/list", HandleDatabaseList)
	mux.HandleFunc("/api/db/backup", HandleDatabaseBackup)
	mux.HandleFunc("/api/db/restore", HandleDatabaseRestore)
	mux.HandleFunc("/api/db/download", HandleDatabaseBackupDownload)
	mux.HandleFunc("/api/db/delete-backup", HandleDatabaseDeleteBackup)

	// Static Web Frontend (Prefer disk if folder exists next to binary, fallback to embedded)
	var staticFS http.FileSystem
	diskWeb := filepath.Join(AppRootDir, "web")
	if info, err := os.Stat(diskWeb); err == nil && info.IsDir() {
		staticFS = http.Dir(diskWeb)
	} else {
		webSubFS, err := fs.Sub(webFS, "web")
		if err != nil {
			log.Fatalf("Failed to load embedded web assets: %v", err)
		}
		staticFS = http.FS(webSubFS)
	}
	mux.Handle("/", http.FileServer(staticFS))

	// Setup Graceful Shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\rShutting down MyLokalWebserver and child services...")
		Manager.StopAll()
		log.Println("Goodbye!")
		os.Exit(0)
	}()

	panelURL := fmt.Sprintf("http://localhost:%d", settings.PanelPort)
	log.Printf("🌐 Control Panel running at: %s", panelURL)

	// Open browser
	if !*noBrowser {
		go func() {
			time.Sleep(800 * time.Millisecond)
			OpenURL(panelURL)
		}()
	}

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", settings.PanelPort),
		Handler: mux,
	}

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP Server error: %v", err)
	}
}
