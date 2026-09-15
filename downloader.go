package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ComponentInfo struct {
	Name             string `json:"name"`
	TargetDir        string `json:"target_dir"`
	URLs             []string `json:"urls"` // primary and fallback mirrors
	SizeEstimate     string `json:"size_estimate"`
	StripLeadingDir  bool   `json:"strip_leading_dir"`
	Status           string `json:"status"` // "pending", "downloading", "extracting", "completed", "error"
	Progress         int    `json:"progress"` // 0 - 100
	ErrorMsg         string `json:"error_msg,omitempty"`
}

var (
	downloadLock      sync.Mutex
	isDownloading     bool
	downloadStatusList []ComponentInfo
	downloadListeners = make(map[chan ComponentProgress]bool)
	listenersLock     sync.Mutex
)

type ComponentProgress struct {
	ComponentName string `json:"component_name"`
	Status        string `json:"status"`
	Progress      int    `json:"progress"`
	BytesRead     int64  `json:"bytes_read"`
	TotalBytes    int64  `json:"total_bytes"`
	Message       string `json:"message"`
	AllDone       bool   `json:"all_done"`
	Error         string `json:"error,omitempty"`
}

func GetDefaultComponents() []ComponentInfo {
	return []ComponentInfo{
		{
			Name: "PHP 8.4 TS",
			TargetDir: filepath.Join("bin", "php"),
			URLs: []string{
				"https://windows.php.net/downloads/releases/archives/php-8.4.4-Win32-vs17-x64.zip",
				"https://downloads.php.net/~windows/releases/archives/php-8.5.10-Win32-vs17-x64.zip",
				"https://windows.php.net/downloads/releases/archives/php-8.3.17-Win32-vs16-x64.zip",
			},
			SizeEstimate:    "~35 MB",
			StripLeadingDir: false,
			Status:          "pending",
		},
		{
			Name: "Apache 2.4",
			TargetDir: filepath.Join("bin", "apache"),
			URLs: []string{
				"https://www.apachelounge.com/download/VS18/binaries/httpd-2.4.68-260827-Win64-VS18.zip",
				"https://www.apachelounge.com/download/VS18/binaries/httpd-2.4.68-260827-win64-VS18.zip",
				"https://www.apachelounge.com/download/VS18/binaries/httpd-2.4.66-260307-win64-VS18.zip",
			},
			SizeEstimate:    "~16 MB",
			StripLeadingDir: true, // "Apache24/..." -> bin/apache/...
			Status:          "pending",
		},
		{
			Name: "MariaDB 11.4",
			TargetDir: filepath.Join("bin", "mariadb"),
			URLs: []string{
				"https://archive.mariadb.org/mariadb-11.4.5/winx64-packages/mariadb-11.4.5-winx64.zip",
				"https://mirror.mariadb.org/mariadb-11.4.5/winx64-packages/mariadb-11.4.5-winx64.zip",
				"https://downloads.mariadb.org/f/mariadb-11.4.5/winx64-packages/mariadb-11.4.5-winx64.zip",
			},
			SizeEstimate:    "~85 MB",
			StripLeadingDir: true, // "mariadb-11.4.5-winx64/..." -> bin/mariadb/...
			Status:          "pending",
		},
		{
			Name: "phpMyAdmin",
			TargetDir: filepath.Join("www", "htdocs", "phpmyadmin"),
			URLs: []string{
				"https://files.phpmyadmin.net/phpMyAdmin/5.2.2/phpMyAdmin-5.2.2-all-languages.zip",
				"https://files.phpmyadmin.net/phpMyAdmin/5.2.1/phpMyAdmin-5.2.1-all-languages.zip",
			},
			SizeEstimate:    "~15 MB",
			StripLeadingDir: true, // "phpMyAdmin-5.2.x-all-languages/..." -> www/htdocs/phpmyadmin/...
			Status:          "pending",
		},
	}
}

func CheckBinariesExist() (bool, map[string]bool) {
	status := make(map[string]bool)

	apacheExe := filepath.Join(AppRootDir, "bin", "apache", "bin", "httpd.exe")
	phpExe := filepath.Join(AppRootDir, "bin", "php", "php.exe")
	mariadbExe := filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mariadbd.exe")
	if _, err := os.Stat(mariadbExe); os.IsNotExist(err) {
		mariadbExe = filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mysqld.exe")
	}
	pmaDir := filepath.Join(AppRootDir, "www", "htdocs", "phpmyadmin", "index.php")

	status["apache"] = fileExists(apacheExe)
	status["php"] = fileExists(phpExe)
	status["mariadb"] = fileExists(mariadbExe)
	status["phpmyadmin"] = fileExists(pmaDir)

	allExist := status["apache"] && status["php"] && status["mariadb"]
	return allExist, status
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

func RegisterProgressListener(ch chan ComponentProgress) {
	listenersLock.Lock()
	defer listenersLock.Unlock()
	downloadListeners[ch] = true
}

func UnregisterProgressListener(ch chan ComponentProgress) {
	listenersLock.Lock()
	defer listenersLock.Unlock()
	delete(downloadListeners, ch)
	close(ch)
}

func broadcastProgress(p ComponentProgress) {
	listenersLock.Lock()
	defer listenersLock.Unlock()
	for ch := range downloadListeners {
		select {
		case ch <- p:
		default:
		}
	}
}

func StartAutoDownload() error {
	downloadLock.Lock()
	if isDownloading {
		downloadLock.Unlock()
		return fmt.Errorf("download already in progress")
	}
	isDownloading = true
	downloadStatusList = GetDefaultComponents()
	downloadLock.Unlock()

	go func() {
		defer func() {
			downloadLock.Lock()
			isDownloading = false
			downloadLock.Unlock()
		}()

		components := GetDefaultComponents()
		for i := range components {
			comp := &components[i]
			fullTargetDir := filepath.Join(AppRootDir, comp.TargetDir)

			// Check if already installed
			if comp.Name == "Apache 2.4" && fileExists(filepath.Join(fullTargetDir, "bin", "httpd.exe")) {
				comp.Status = "completed"
				comp.Progress = 100
				broadcastProgress(ComponentProgress{
					ComponentName: comp.Name,
					Status:        "completed",
					Progress:      100,
					Message:       fmt.Sprintf("%s already installed", comp.Name),
				})
				continue
			}
			if strings.HasPrefix(comp.Name, "PHP") && fileExists(filepath.Join(fullTargetDir, "php.exe")) {
				comp.Status = "completed"
				comp.Progress = 100
				broadcastProgress(ComponentProgress{
					ComponentName: comp.Name,
					Status:        "completed",
					Progress:      100,
					Message:       fmt.Sprintf("%s already installed", comp.Name),
				})
				continue
			}
			if strings.HasPrefix(comp.Name, "MariaDB") && (fileExists(filepath.Join(fullTargetDir, "bin", "mariadbd.exe")) || fileExists(filepath.Join(fullTargetDir, "bin", "mysqld.exe"))) {
				comp.Status = "completed"
				comp.Progress = 100
				broadcastProgress(ComponentProgress{
					ComponentName: comp.Name,
					Status:        "completed",
					Progress:      100,
					Message:       fmt.Sprintf("%s already installed", comp.Name),
				})
				continue
			}
			if comp.Name == "phpMyAdmin" && fileExists(filepath.Join(fullTargetDir, "index.php")) {
				comp.Status = "completed"
				comp.Progress = 100
				broadcastProgress(ComponentProgress{
					ComponentName: comp.Name,
					Status:        "completed",
					Progress:      100,
					Message:       fmt.Sprintf("%s already installed", comp.Name),
				})
				continue
			}

			// Download and extract
			err := downloadAndExtractWithFallbacks(comp, fullTargetDir)
			if err != nil {
				comp.Status = "error"
				comp.ErrorMsg = err.Error()
				broadcastProgress(ComponentProgress{
					ComponentName: comp.Name,
					Status:        "error",
					Error:         err.Error(),
					Message:       fmt.Sprintf("Failed to install %s: %v", comp.Name, err),
				})
				return
			}

			comp.Status = "completed"
			comp.Progress = 100
			broadcastProgress(ComponentProgress{
				ComponentName: comp.Name,
				Status:        "completed",
				Progress:      100,
				Message:       fmt.Sprintf("%s installed successfully!", comp.Name),
			})
		}

		// Initialize Database & Generate Configs
		settings := GetCurrentSettings()
		_ = InitMariaDBData()
		_ = GenerateAllConfigs(settings)

		// Create starter index.php
		CreateStarterPage()

		broadcastProgress(ComponentProgress{
			ComponentName: "Setup",
			Status:        "completed",
			Progress:      100,
			AllDone:       true,
			Message:       "All components downloaded and configured successfully!",
		})

		// Auto start services if enabled
		if settings.AutoStart {
			_ = StartApache()
			_ = StartMariaDB()
		}
	}()

	return nil
}

func downloadAndExtractWithFallbacks(comp *ComponentInfo, targetDir string) error {
	var lastErr error
	tmpDir := filepath.Join(AppRootDir, "tmp")
	_ = os.MkdirAll(tmpDir, 0755)
	zipPath := filepath.Join(tmpDir, fmt.Sprintf("%s_download.zip", strings.ReplaceAll(comp.Name, " ", "_")))
	defer os.Remove(zipPath)

	for _, url := range comp.URLs {
		broadcastProgress(ComponentProgress{
			ComponentName: comp.Name,
			Status:        "downloading",
			Progress:      0,
			Message:       fmt.Sprintf("Downloading %s from %s...", comp.Name, url),
		})

		err := downloadFileWithProgress(url, zipPath, comp.Name)
		if err == nil {
			// Extract
			broadcastProgress(ComponentProgress{
				ComponentName: comp.Name,
				Status:        "extracting",
				Progress:      90,
				Message:       fmt.Sprintf("Extracting %s...", comp.Name),
			})

			err = extractZip(zipPath, targetDir, comp.StripLeadingDir)
			if err == nil {
				return nil
			}
			lastErr = fmt.Errorf("extraction error: %w", err)
		} else {
			lastErr = err
		}
	}

	return fmt.Errorf("all download sources failed for %s. Last error: %v", comp.Name, lastErr)
}

type progressWriter struct {
	total      int64
	current    int64
	compName   string
	lastUpdate time.Time
}

func (pw *progressWriter) Write(p []byte) (int, error) {
	n := len(p)
	pw.current += int64(n)

	if time.Since(pw.lastUpdate) > 200*time.Millisecond {
		pw.lastUpdate = time.Now()
		var pct int
		if pw.total > 0 {
			pct = int((float64(pw.current) / float64(pw.total)) * 85) // 0-85% for download
		}
		broadcastProgress(ComponentProgress{
			ComponentName: pw.compName,
			Status:        "downloading",
			Progress:      pct,
			BytesRead:     pw.current,
			TotalBytes:    pw.total,
			Message:       fmt.Sprintf("Downloading %s (%d MB / %d MB)...", pw.compName, pw.current/(1024*1024), pw.total/(1024*1024)),
		})
	}

	return n, nil
}

func downloadFileWithProgress(url string, destPath string, compName string) error {
	client := &http.Client{
		Timeout: 30 * time.Minute,
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "*/*")
	if strings.Contains(url, "apachelounge.com") {
		req.Header.Set("Referer", "https://www.apachelounge.com/download/")
	} else if strings.Contains(url, "php.net") {
		req.Header.Set("Referer", "https://windows.php.net/download/")
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	out, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer out.Close()

	pw := &progressWriter{
		total:      resp.ContentLength,
		compName:   compName,
		lastUpdate: time.Now(),
	}

	_, err = io.Copy(out, io.TeeReader(resp.Body, pw))
	return err
}

func extractZip(zipPath string, targetDir string, stripLeadingDir bool) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	_ = os.MkdirAll(targetDir, 0755)

	// Determine if there is a common top level dir
	var leadingDir string
	if stripLeadingDir {
		for _, f := range r.File {
			parts := strings.Split(strings.TrimPrefix(filepath.ToSlash(f.Name), "/"), "/")
			if len(parts) > 0 && parts[0] != "" {
				if leadingDir == "" {
					leadingDir = parts[0]
				} else if leadingDir != parts[0] {
					leadingDir = ""
					break
				}
			}
		}
	}

	for _, f := range r.File {
		cleanName := filepath.ToSlash(f.Name)
		if stripLeadingDir && leadingDir != "" {
			if cleanName == leadingDir || cleanName == leadingDir+"/" {
				continue
			}
			cleanName = strings.TrimPrefix(cleanName, leadingDir+"/")
		}

		destPath := filepath.Join(targetDir, filepath.FromSlash(cleanName))

		// Prevent zip slip vulnerability
		if !strings.HasPrefix(filepath.Clean(destPath), filepath.Clean(targetDir)) {
			continue
		}

		if f.FileInfo().IsDir() {
			_ = os.MkdirAll(destPath, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return err
		}

		outFile, err := os.OpenFile(destPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			outFile.Close()
			return err
		}

		_, err = io.Copy(outFile, rc)
		rc.Close()
		outFile.Close()
		if err != nil {
			return err
		}
	}

	// Post extraction: If Apache24 subfolder exists inside targetDir, move contents to targetDir
	nestedApache := filepath.Join(targetDir, "Apache24")
	if info, err := os.Stat(nestedApache); err == nil && info.IsDir() {
		_ = copyDir(nestedApache, targetDir)
		_ = os.RemoveAll(nestedApache)
	}

	return nil
}

func CreateStarterPage() {
	htdocsDir := filepath.Join(AppRootDir, "www", "htdocs")
	_ = os.MkdirAll(htdocsDir, 0755)
	indexPath := filepath.Join(htdocsDir, "index.php")

	if _, err := os.Stat(indexPath); os.IsNotExist(err) {
		starterContent := `<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>MyLokalWebserver - Server Lokal Aktif</title>
    <style>
        :root {
            --bg: #f8fafc;
            --card: #ffffff;
            --border: #e2e8f0;
            --text: #0f172a;
            --text-sub: #475569;
            --text-muted: #64748b;
            --primary: #0f172a;
            --primary-hover: #1e293b;
            --accent-green-bg: #ecfdf5;
            --accent-green-text: #065f46;
            --accent-green-border: #a7f3d0;
        }

        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, Helvetica, Arial, sans-serif;
            background-color: var(--bg);
            color: var(--text);
            display: flex;
            align-items: center;
            justify-content: center;
            min-height: 100vh;
            padding: 24px;
            font-size: 14px;
            line-height: 1.5;
            -webkit-font-smoothing: antialiased;
        }

        .card {
            background: var(--card);
            border: 1px solid var(--border);
            border-radius: 12px;
            padding: 36px 32px;
            max-width: 540px;
            width: 100%;
            box-shadow: 0 1px 3px 0 rgba(0, 0, 0, 0.05), 0 1px 2px -1px rgba(0, 0, 0, 0.05);
        }

        .header {
            margin-bottom: 20px;
        }

        .badge {
            display: inline-flex;
            align-items: center;
            gap: 6px;
            background: var(--accent-green-bg);
            color: var(--accent-green-text);
            border: 1px solid var(--accent-green-border);
            padding: 3px 10px;
            border-radius: 999px;
            font-size: 12px;
            font-weight: 600;
            margin-bottom: 12px;
        }

        .badge-dot {
            width: 6px;
            height: 6px;
            border-radius: 50%;
            background-color: #10b981;
        }

        h1 {
            font-size: 20px;
            font-weight: 700;
            color: var(--text);
            letter-spacing: -0.3px;
            margin-bottom: 6px;
        }

        p {
            color: var(--text-sub);
            font-size: 13px;
            line-height: 1.6;
        }

        .links {
            display: flex;
            gap: 10px;
            margin: 24px 0;
            flex-wrap: wrap;
        }

        .btn {
            display: inline-flex;
            align-items: center;
            justify-content: center;
            padding: 8px 16px;
            border-radius: 6px;
            text-decoration: none;
            font-weight: 500;
            font-size: 13px;
            transition: all 0.15s ease;
            border: 1px solid transparent;
        }

        .btn-primary {
            background: var(--primary);
            color: #ffffff;
            border-color: var(--primary);
        }

        .btn-primary:hover {
            background: var(--primary-hover);
        }

        .btn-default {
            background: #ffffff;
            border-color: var(--border);
            color: var(--text);
        }

        .btn-default:hover {
            background: #f1f5f9;
        }

        .info-table {
            background: #f8fafc;
            border: 1px solid #f1f5f9;
            border-radius: 8px;
            padding: 12px 16px;
            font-size: 12px;
            display: flex;
            flex-direction: column;
            gap: 8px;
        }

        .info-row {
            display: flex;
            justify-content: space-between;
            align-items: center;
        }

        .info-label {
            color: var(--text-muted);
        }

        .info-val {
            font-family: "Cascadia Code", Consolas, monospace;
            font-weight: 600;
            color: var(--text);
        }
    </style>
</head>
<body>
    <div class="card">
        <div class="header">
            <div class="badge">
                <span class="badge-dot"></span>
                <span>Server Aktif &bull; PHP <?php echo phpversion(); ?></span>
            </div>
            <h1>MyLokalWebserver Siap Digunakan</h1>
            <p>File ini berada di direktori <code>www/htdocs/index.php</code>. Ganti atau tambahkan file proyek web Anda di folder tersebut.</p>
        </div>

        <div class="links">
            <a href="/phpmyadmin" class="btn btn-primary">Buka phpMyAdmin</a>
            <a href="http://localhost:3000" class="btn btn-default" target="_blank">Control Panel</a>
        </div>

        <div class="info-table">
            <div class="info-row">
                <span class="info-label">Document Root</span>
                <span class="info-val"><?php echo $_SERVER['DOCUMENT_ROOT']; ?></span>
            </div>
            <div class="info-row">
                <span class="info-label">Server Software</span>
                <span class="info-val"><?php echo $_SERVER['SERVER_SOFTWARE']; ?></span>
            </div>
            <div class="info-row">
                <span class="info-label">Ekstensi Aktif</span>
                <span class="info-val"><?php echo count(get_loaded_extensions()); ?> modul</span>
            </div>
        </div>

        <div style="margin-top: 24px; padding-top: 16px; border-top: 1px solid var(--border); text-align: center; font-size: 12px; color: var(--text-muted);">
            Created by <strong style="color: var(--text); font-weight: 600;">Fery Fadly</strong> and <strong style="color: var(--text); font-weight: 600;">Team Dikodein</strong>
        </div>
    </div>
</body>
</html>`
		_ = os.WriteFile(indexPath, []byte(starterContent), 0644)
	}
}
