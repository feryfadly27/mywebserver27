package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type APIResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
	Error   string `json:"error,omitempty"`
}

func jsonResponse(w http.ResponseWriter, statusCode int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	_ = json.NewEncoder(w).Encode(resp)
}

func HandleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	allInstalled, binaryStatus := CheckBinariesExist()
	servicesStatus := Manager.GetStatus()
	settings := GetCurrentSettings()

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"app_version":        AppVersion,
			"binaries_installed": allInstalled,
			"binary_status":      binaryStatus,
			"services":           servicesStatus,
			"settings":           settings,
			"document_root":      filepath.Join(AppRootDir, "www", "htdocs"),
			"app_dir":            AppRootDir,
		},
	})
}

func HandleServiceAction(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	// URL format: /api/service?name=apache&action=start
	name := strings.ToLower(r.URL.Query().Get("name"))
	action := strings.ToLower(r.URL.Query().Get("action"))

	var err error
	switch name {
	case "apache":
		switch action {
		case "start":
			err = StartApache()
		case "stop":
			err = StopApache()
		case "restart":
			err = RestartApache()
		default:
			err = fmt.Errorf("invalid action: %s", action)
		}
	case "mariadb", "mysql":
		switch action {
		case "start":
			err = StartMariaDB()
		case "stop":
			err = StopMariaDB()
		case "restart":
			err = RestartMariaDB()
		default:
			err = fmt.Errorf("invalid action: %s", action)
		}
	case "all":
		switch action {
		case "start":
			_ = StartMariaDB()
			err = StartApache()
		case "stop":
			Manager.StopAll()
		case "restart":
			Manager.StopAll()
			_ = StartMariaDB()
			err = StartApache()
		default:
			err = fmt.Errorf("invalid action: %s", action)
		}
	default:
		err = fmt.Errorf("unknown service: %s", name)
	}

	if err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Service %s %s executed successfully", name, action),
		Data:    Manager.GetStatus(),
	})
}

func HandleSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		settings := GetCurrentSettings()
		jsonResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Data:    settings,
		})
	case http.MethodPost:
		var newSettings Settings
		if err := json.NewDecoder(r.Body).Decode(&newSettings); err != nil {
			jsonResponse(w, http.StatusBadRequest, APIResponse{
				Success: false,
				Error:   "Invalid JSON payload",
			})
			return
		}

		// Validation
		if newSettings.ApachePort < 1 || newSettings.ApachePort > 65535 {
			jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid Apache port (1-65535)"})
			return
		}
		if newSettings.MariaDBPort < 1 || newSettings.MariaDBPort > 65535 {
			jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid MariaDB port (1-65535)"})
			return
		}
		if newSettings.ApachePort == newSettings.MariaDBPort || newSettings.ApachePort == newSettings.PanelPort {
			jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Port conflict detected between services"})
			return
		}

		oldSettings := GetCurrentSettings()
		if newSettings.PanelPort == 0 {
			newSettings.PanelPort = oldSettings.PanelPort
		}
		if newSettings.VirtualHosts == nil {
			newSettings.VirtualHosts = oldSettings.VirtualHosts
		}

		if err := SaveSettings(newSettings); err != nil {
			jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Failed to save settings: " + err.Error()})
			return
		}

		// Re-generate configs with new ports
		_ = GenerateAllConfigs(newSettings)

		// Restart services if port changed and was running
		status := Manager.GetStatus()
		if oldSettings.ApachePort != newSettings.ApachePort && status["apache"].Running {
			_ = RestartApache()
		}
		if oldSettings.MariaDBPort != newSettings.MariaDBPort && status["mariadb"].Running {
			_ = RestartMariaDB()
		}

		jsonResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Settings saved successfully",
			Data:    newSettings,
		})
	default:
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
	}
}

func HandleDownloadStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	err := StartAutoDownload()
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Download started",
	})
}

func HandleDownloadProgressSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan ComponentProgress, 10)
	RegisterProgressListener(ch)
	defer UnregisterProgressListener(ch)

	for {
		select {
		case <-r.Context().Done():
			return
		case prog, ok := <-ch:
			if !ok {
				return
			}
			data, _ := json.Marshal(prog)
			_, _ = fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
			if prog.AllDone || prog.Error != "" {
				return
			}
		}
	}
}

func HandleOpenFolder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	folder := r.URL.Query().Get("folder")
	customSubPath := r.URL.Query().Get("path")
	targetPath := filepath.Join(AppRootDir, "www", "htdocs")
	if customSubPath != "" {
		targetPath = filepath.Join(AppRootDir, "www", "htdocs", filepath.FromSlash(filepath.Clean(customSubPath)))
	} else if folder == "logs" {
		targetPath = filepath.Join(AppRootDir, "logs")
	} else if folder == "root" {
		targetPath = AppRootDir
	}

	OpenFolder(targetPath)

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Folder opened in file manager",
	})
}

func HandleOpenNativeTerminal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	settings := GetCurrentSettings()
	folder := r.URL.Query().Get("folder")
	customSubPath := r.URL.Query().Get("path")
	targetPath := filepath.Join(AppRootDir, "www", "htdocs")
	if customSubPath != "" {
		targetPath = filepath.Join(AppRootDir, "www", "htdocs", filepath.FromSlash(filepath.Clean(customSubPath)))
	} else if folder == "root" {
		targetPath = AppRootDir
	}

	OpenNativeTerminal(targetPath, settings)

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Native terminal opened",
	})
}

type ProjectInfo struct {
	Name          string `json:"name"`
	RelativePath  string `json:"relative_path"`
	FullPath      string `json:"full_path"`
	Framework     string `json:"framework"`
	HasPublicDir  bool   `json:"has_public_dir"`
	DefaultURL    string `json:"default_url"`
	VHostDomain   string `json:"vhost_domain,omitempty"`
	VHostURL      string `json:"vhost_url,omitempty"`
	LocalhostURL  string `json:"localhost_url,omitempty"`
	VHostEnabled  bool   `json:"vhost_enabled"`
	IsInHostsFile bool   `json:"is_in_hosts_file"`
}

func detectFramework(dir string) (string, bool) {
	hasPublic := false
	if fi, err := os.Stat(filepath.Join(dir, "public")); err == nil && fi.IsDir() {
		hasPublic = true
	}

	// Laravel: artisan file + public/
	if fileExists(filepath.Join(dir, "artisan")) {
		return "Laravel", hasPublic
	}
	// CodeIgniter 4: spark file
	if fileExists(filepath.Join(dir, "spark")) {
		return "CodeIgniter 4", hasPublic
	}
	// WordPress: wp-config.php or wp-login.php
	if fileExists(filepath.Join(dir, "wp-config.php")) || fileExists(filepath.Join(dir, "wp-login.php")) {
		return "WordPress", hasPublic
	}
	// Symfony
	if fileExists(filepath.Join(dir, "bin", "console")) {
		return "Symfony", hasPublic
	}

	return "PHP Native", hasPublic
}

func HandleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	htdocsDir := filepath.Join(AppRootDir, "www", "htdocs")
	settings := GetCurrentSettings()

	var projects []ProjectInfo
	entries, err := os.ReadDir(htdocsDir)
	if err == nil {
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			folderName := entry.Name()
			if strings.EqualFold(folderName, "phpmyadmin") {
				continue // handled by dedicated phpMyAdmin button
			}

			fullPath := filepath.Join(htdocsDir, folderName)
			framework, hasPublic := detectFramework(fullPath)

			defaultURL := fmt.Sprintf("http://localhost:%d/%s/", settings.ApachePort, folderName)
			if hasPublic && (framework == "Laravel" || framework == "CodeIgniter 4" || framework == "Symfony") {
				defaultURL = fmt.Sprintf("http://localhost:%d/%s/public/", settings.ApachePort, folderName)
			}

			pInfo := ProjectInfo{
				Name:         folderName,
				RelativePath: folderName,
				FullPath:     fullPath,
				Framework:    framework,
				HasPublicDir: hasPublic,
				DefaultURL:   defaultURL,
			}

			// Check if mapped to any virtual host
			for _, vh := range settings.VirtualHosts {
				cleanFolder := filepath.ToSlash(filepath.Clean(vh.Folder))
				if strings.HasPrefix(cleanFolder, folderName) || cleanFolder == folderName {
					pInfo.VHostDomain = vh.Domain
					pInfo.VHostEnabled = vh.Enabled
					pInfo.VHostURL = fmt.Sprintf("http://%s:%d/", vh.Domain, settings.ApachePort)
					pInfo.IsInHostsFile = CheckDomainInHosts(vh.Domain)

					parts := strings.Split(vh.Domain, ".")
					prefix := parts[0]
					if prefix == "" {
						prefix = folderName
					}
					pInfo.LocalhostURL = fmt.Sprintf("http://%s.localtest.me:%d/", prefix, settings.ApachePort)
					break
				}
			}

			projects = append(projects, pInfo)
		}
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"projects":      projects,
			"virtual_hosts": settings.VirtualHosts,
			"apache_port":   settings.ApachePort,
		},
	})
}

type CreateProjectRequest struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "native" or "public_structure"
	SetVHost bool   `json:"set_vhost"`
	Domain   string `json:"domain"`
}

func HandleProjectCreate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req CreateProjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid JSON format"})
		return
	}

	folderName := strings.TrimSpace(req.Name)
	if folderName == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Nama folder proyek tidak boleh kosong"})
		return
	}
	folderName = sanitizeFilename(folderName)

	htdocsDir := filepath.Join(AppRootDir, "www", "htdocs")
	projectDir := filepath.Join(htdocsDir, folderName)

	if _, err := os.Stat(projectDir); err == nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: fmt.Sprintf("Folder '%s' sudah ada di www/htdocs", folderName)})
		return
	}

	_ = os.MkdirAll(projectDir, 0755)

	starterPHP := fmt.Sprintf(`<?php
/**
 * Proyek: %s
 * Dibuat via MyLokalWebserver
 */
?>
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s — MyLokalWebserver</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; background: #f8fafc; color: #1e293b; display: flex; align-items: center; justify-content: center; min-height: 100vh; margin: 0; }
        .card { background: #ffffff; border: 1px solid #e2e8f0; border-radius: 12px; padding: 32px; max-width: 500px; box-shadow: 0 4px 6px -1px rgba(0,0,0,0.05); text-align: center; }
        h1 { color: #0284c7; margin-top: 0; }
        code { background: #f1f5f9; padding: 2px 6px; border-radius: 4px; font-family: monospace; }
    </style>
</head>
<body>
    <div class="card">
        <h1>⚡ %s</h1>
        <p>Proyek PHP lokal Anda siap dikembangkan!</p>
        <p>Lokasi berkas: <code>www/htdocs/%s/index.php</code></p>
        <p><small>PHP Versi: <?= phpversion(); ?></small></p>
    </div>
</body>
</html>
`, folderName, folderName, folderName, folderName)

	if req.Type == "public_structure" {
		publicDir := filepath.Join(projectDir, "public")
		_ = os.MkdirAll(publicDir, 0755)
		_ = os.WriteFile(filepath.Join(publicDir, "index.php"), []byte(starterPHP), 0644)
	} else {
		_ = os.WriteFile(filepath.Join(projectDir, "index.php"), []byte(starterPHP), 0644)
	}

	if req.SetVHost && req.Domain != "" {
		settings := GetCurrentSettings()
		targetFolder := folderName
		if req.Type == "public_structure" {
			targetFolder = folderName + "/public"
		}
		docRoot := filepath.Join(htdocsDir, filepath.FromSlash(targetFolder))

		settings.VirtualHosts = append(settings.VirtualHosts, VirtualHost{
			Domain:       strings.ToLower(strings.TrimSpace(req.Domain)),
			Folder:       targetFolder,
			DocumentRoot: docRoot,
			Enabled:      true,
		})
		_ = SaveSettings(settings)
		_ = GenerateVhostsConfig(settings)
		if Manager.GetStatus()["apache"].Running {
			_ = RestartApache()
		}
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Proyek '%s' berhasil dibuat", folderName),
	})
}

type VHostRequest struct {
	Domain       string `json:"domain"`
	Folder       string `json:"folder"`
	DocumentRoot string `json:"document_root"`
	Enabled      bool   `json:"enabled"`
}

func HandleVHostSave(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req VHostRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid JSON format"})
		return
	}

	req.Domain = strings.ToLower(strings.TrimSpace(req.Domain))
	req.Folder = strings.TrimSpace(req.Folder)
	if req.Domain == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Nama domain tidak boleh kosong"})
		return
	}
	if req.Domain == "localhost" || req.Domain == "127.0.0.1" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Domain 'localhost' sudah menjadi host utama"})
		return
	}

	htdocsDir := filepath.Join(AppRootDir, "www", "htdocs")
	docRoot := req.DocumentRoot
	if docRoot == "" {
		docRoot = filepath.Join(htdocsDir, filepath.FromSlash(req.Folder))
	}
	_ = os.MkdirAll(docRoot, 0755)

	settings := GetCurrentSettings()
	found := false
	for i, vh := range settings.VirtualHosts {
		if strings.EqualFold(vh.Domain, req.Domain) {
			settings.VirtualHosts[i] = VirtualHost{
				Domain:       req.Domain,
				Folder:       req.Folder,
				DocumentRoot: docRoot,
				Enabled:      req.Enabled,
			}
			found = true
			break
		}
	}

	if !found {
		settings.VirtualHosts = append(settings.VirtualHosts, VirtualHost{
			Domain:       req.Domain,
			Folder:       req.Folder,
			DocumentRoot: docRoot,
			Enabled:      req.Enabled,
		})
	}

	if err := SaveSettings(settings); err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Gagal menyimpan settings: " + err.Error()})
		return
	}

	_ = GenerateVhostsConfig(settings)

	// If Apache is running, reload configs
	status := Manager.GetStatus()
	if status["apache"].Running {
		_ = RestartApache()
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Virtual Host '%s' berhasil disimpan", req.Domain),
		Data:    settings.VirtualHosts,
	})
}

func HandleVHostDelete(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	domain := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("domain")))
	if domain == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Domain parameter required"})
		return
	}

	settings := GetCurrentSettings()
	var newVHosts []VirtualHost
	for _, vh := range settings.VirtualHosts {
		if !strings.EqualFold(vh.Domain, domain) {
			newVHosts = append(newVHosts, vh)
		}
	}

	settings.VirtualHosts = newVHosts
	if err := SaveSettings(settings); err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Gagal menyimpan settings: " + err.Error()})
		return
	}

	_ = GenerateVhostsConfig(settings)

	status := Manager.GetStatus()
	if status["apache"].Running {
		_ = RestartApache()
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Virtual Host '%s' berhasil dihapus", domain),
		Data:    settings.VirtualHosts,
	})
}

func HandleVHostToggle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	domain := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("domain")))
	settings := GetCurrentSettings()
	for i, vh := range settings.VirtualHosts {
		if strings.EqualFold(vh.Domain, domain) {
			settings.VirtualHosts[i].Enabled = !settings.VirtualHosts[i].Enabled
			break
		}
	}

	if err := SaveSettings(settings); err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Gagal menyimpan settings: " + err.Error()})
		return
	}

	_ = GenerateVhostsConfig(settings)

	status := Manager.GetStatus()
	if status["apache"].Running {
		_ = RestartApache()
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Status Virtual Host diperbarui",
		Data:    settings.VirtualHosts,
	})
}

func HandleShutdown(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "MyLokalWebserver beserta seluruh layanannya telah berhasil dimatikan.",
	})

	go func() {
		// Wait brief moment for HTTP response to be flushed to client
		time.Sleep(500 * time.Millisecond)
		Manager.StopAll()
		os.Exit(0)
	}()
}

func HandleRestartApp(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	mode := r.URL.Query().Get("mode")
	if mode == "app" {
		jsonResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Aplikasi sedang dimulai ulang...",
		})
		go func() {
			time.Sleep(500 * time.Millisecond)
			Manager.StopAll()
			exe, err := os.Executable()
			if err == nil {
				cmd := exec.Command(exe, "-no-browser")
				cmd.Dir = AppRootDir
				SetProcessGroupAttributes(cmd)
				_ = cmd.Start()
			}
			os.Exit(0)
		}()
		return
	}

	// Default: restart all services (Apache & MariaDB)
	Manager.StopAll()
	time.Sleep(500 * time.Millisecond)
	settings := GetCurrentSettings()
	_ = GenerateApacheConfig(settings)
	_ = GeneratePHPConfig(settings)
	_ = GenerateMariaDBConfig(settings)
	_ = GenerateVhostsConfig(settings)
	_ = StartMariaDB()
	time.Sleep(400 * time.Millisecond)
	_ = StartApache()

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Seluruh layanan Apache & MariaDB berhasil dimulai ulang.",
		Data:    Manager.GetStatus(),
	})
}

func HandleCheckUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	settings := GetCurrentSettings()
	repo := r.URL.Query().Get("repo")
	if repo == "" {
		repo = settings.GitHubRepo
	}
	token := r.URL.Query().Get("token")
	if token == "" {
		token = settings.GitHubToken
	}

	res, err := CheckForUpdates(repo, token)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   err.Error(),
		})
		return
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    res,
	})
}

func HandleApplyUpdate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req struct {
		DownloadURL string `json:"download_url"`
		Token       string `json:"token,omitempty"`
	}

	_ = json.NewDecoder(r.Body).Decode(&req)

	if req.DownloadURL == "" {
		req.DownloadURL = r.URL.Query().Get("url")
	}

	if req.DownloadURL == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Tautan unduhan tidak boleh kosong"})
		return
	}

	settings := GetCurrentSettings()
	if req.Token == "" {
		req.Token = settings.GitHubToken
	}

	go func() {
		_ = ApplySelfUpdate(req.DownloadURL, req.Token)
	}()

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Proses pembaruan dimulai...",
	})
}

func HandleUpdateProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	prog := GetUpdateProgress()
	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    prog,
	})
}

// Network LAN IP Info
type NetworkIPInfo struct {
	IP        string `json:"ip"`
	Interface string `json:"interface"`
	IsWiFi    bool   `json:"is_wifi"`
}

func GetLocalIPAddresses() []NetworkIPInfo {
	var ips []NetworkIPInfo
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}

	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}

		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}

		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}

			if ip != nil && ip.To4() != nil && !ip.IsLoopback() {
				ipStr := ip.String()
				name := iface.Name
				isWiFi := strings.Contains(strings.ToLower(name), "wi-fi") || strings.Contains(strings.ToLower(name), "wireless") || strings.Contains(strings.ToLower(name), "wlan")
				ips = append(ips, NetworkIPInfo{
					IP:        ipStr,
					Interface: name,
					IsWiFi:    isWiFi,
				})
			}
		}
	}

	// Prioritize WiFi and Standard LAN IPs (192.168.x.x) over virtual adapters
	sort.SliceStable(ips, func(i, j int) bool {
		score := func(item NetworkIPInfo) int {
			s := 0
			if item.IsWiFi {
				s += 100
			}
			if strings.HasPrefix(item.IP, "192.168.") {
				s += 50
			} else if strings.HasPrefix(item.IP, "10.") {
				s += 40
			}
			lowerName := strings.ToLower(item.Interface)
			if strings.Contains(lowerName, "wsl") || strings.Contains(lowerName, "vethernet") || strings.Contains(lowerName, "tailscale") || strings.Contains(lowerName, "zerotier") || strings.Contains(lowerName, "virtual") {
				s -= 30
			}
			return s
		}
		return score(ips[i]) > score(ips[j])
	})

	return ips
}

func HandleNetworkIPs(w http.ResponseWriter, r *http.Request) {
	settings := GetCurrentSettings()
	ips := GetLocalIPAddresses()
	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"ips":         ips,
			"apache_port": settings.ApachePort,
			"panel_port":  settings.PanelPort,
		},
	})
}

// Database Tools Handlers
func HandleDatabaseList(w http.ResponseWriter, r *http.Request) {
	dbs, err := ListDatabases()
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	backups, _ := ListBackups()
	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"databases": dbs,
			"backups":   backups,
		},
	})
}

func HandleDatabaseBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	dbName := strings.TrimSpace(r.URL.Query().Get("database"))
	if dbName == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Nama database harus dipilih"})
		return
	}
	backupInfo, err := BackupDatabase(dbName)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Database '%s' berhasil di-backup!", dbName),
		Data:    backupInfo,
	})
}

func HandleDatabaseRestore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	filename := r.URL.Query().Get("filename")
	targetDB := r.URL.Query().Get("database")

	if filename != "" {
		filePath := filepath.Join(AppRootDir, "data", "backups", filepath.Base(filename))
		f, err := os.Open(filePath)
		if err != nil {
			jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "File backup tidak ditemukan: " + err.Error()})
			return
		}
		defer f.Close()

		if err := RestoreDatabaseFromSQL(targetDB, f); err != nil {
			jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
			return
		}

		jsonResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Database berhasil dipulihkan dari " + filepath.Base(filename),
		})
		return
	}

	// Multipart upload
	file, _, err := r.FormFile("file")
	if err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Pilih file .sql untuk di-upload"})
		return
	}
	defer file.Close()

	if err := RestoreDatabaseFromSQL(targetDB, file); err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "File .sql berhasil diimpor ke database",
	})
}

func HandleDatabaseBackupDownload(w http.ResponseWriter, r *http.Request) {
	filename := filepath.Base(r.URL.Query().Get("filename"))
	if filename == "" || !strings.HasSuffix(strings.ToLower(filename), ".sql") {
		http.Error(w, "Invalid filename", http.StatusBadRequest)
		return
	}
	filePath := filepath.Join(AppRootDir, "data", "backups", filename)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", filename))
	w.Header().Set("Content-Type", "application/sql")
	http.ServeFile(w, r, filePath)
}

func HandleDatabaseDeleteBackup(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}
	filename := r.URL.Query().Get("filename")
	if err := DeleteBackupFile(filename); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "File backup berhasil dihapus",
	})
}

func HandleHostsStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	settings := GetCurrentSettings()
	var allDomains []string
	for _, vh := range settings.VirtualHosts {
		if vh.Enabled && strings.TrimSpace(vh.Domain) != "" {
			allDomains = append(allDomains, vh.Domain)
		}
	}

	missing := GetMissingHostsDomains(allDomains)
	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"hosts_file":    GetHostsFilePath(),
			"all_domains":   allDomains,
			"missing":       missing,
			"is_all_synced": len(missing) == 0,
		},
	})
}

func HandleHostsSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	settings := GetCurrentSettings()
	var allDomains []string
	for _, vh := range settings.VirtualHosts {
		if vh.Enabled && strings.TrimSpace(vh.Domain) != "" {
			allDomains = append(allDomains, vh.Domain)
		}
	}

	if len(allDomains) == 0 {
		jsonResponse(w, http.StatusOK, APIResponse{
			Success: true,
			Message: "Tidak ada domain Virtual Host yang perlu disinkronkan.",
		})
		return
	}

	err := SyncDomainsToHosts(allDomains)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{
			Success: false,
			Error:   fmt.Sprintf("Gagal menyinkronkan hosts: %v", err),
		})
		return
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: "Domain berhasil disinkronkan ke file hosts Windows!",
	})
}

// Database Seeder API Handlers
func HandleDatabaseTables(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	if dbName == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Nama database harus ditentukan"})
		return
	}
	tables, err := ListTables(dbName)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"database": dbName,
			"tables":   tables,
		},
	})
}

func HandleDatabaseColumns(w http.ResponseWriter, r *http.Request) {
	dbName := strings.TrimSpace(r.URL.Query().Get("db"))
	tableName := strings.TrimSpace(r.URL.Query().Get("table"))
	if dbName == "" || tableName == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Nama database dan tabel harus ditentukan"})
		return
	}
	cols, err := DescribeTable(dbName, tableName)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}
	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data: map[string]any{
			"database": dbName,
			"table":    tableName,
			"columns":  cols,
		},
	})
}

type DatabaseSeedRequest struct {
	Database string `json:"database"`
	Table    string `json:"table"`
	Count    int    `json:"count"`
}

func HandleDatabaseSeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req DatabaseSeedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid JSON body"})
		return
	}

	req.Database = strings.TrimSpace(req.Database)
	req.Table = strings.TrimSpace(req.Table)
	if req.Database == "" || req.Table == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Pilih database dan tabel target"})
		return
	}
	if req.Count <= 0 {
		req.Count = 10
	}
	if req.Count > 500 {
		req.Count = 500
	}

	inserted, err := GenerateSmartSeedData(req.Database, req.Table, req.Count)
	if err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Berhasil menambahkan %d baris data contoh realistis ke tabel '%s'!", inserted, req.Table),
		Data: map[string]any{
			"database": req.Database,
			"table":    req.Table,
			"count":    inserted,
		},
	})
}

type DatabaseSeedTemplateRequest struct {
	Database string `json:"database"`
	Template string `json:"template"`
}

func HandleDatabaseSeedTemplate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method not allowed"})
		return
	}

	var req DatabaseSeedTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Invalid JSON body"})
		return
	}

	req.Database = strings.TrimSpace(req.Database)
	req.Template = strings.TrimSpace(req.Template)
	if req.Database == "" || req.Template == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Nama database dan template harus dipilih"})
		return
	}

	if err := ApplyPresetDatabaseTemplate(req.Database, req.Template); err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: err.Error()})
		return
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Template database '%s' berhasil dibuat dan diisi data contoh pada database '%s'!", req.Template, req.Database),
	})
}
