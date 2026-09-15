package main

import (
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"
)

const AppVersion = "v1.0.0"

type GitHubAsset struct {
	Name               string `json:"name"`
	Size               int64  `json:"size"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

type GitHubRelease struct {
	TagName     string        `json:"tag_name"`
	Name        string        `json:"name"`
	Body        string        `json:"body"`
	PublishedAt time.Time     `json:"published_at"`
	Assets      []GitHubAsset `json:"assets"`
}

type UpdateCheckResult struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	ReleaseName    string `json:"release_name"`
	ReleaseNotes   string `json:"release_notes"`
	PublishedAt    string `json:"published_at"`
	DownloadURL    string `json:"download_url,omitempty"`
	AssetName      string `json:"asset_name,omitempty"`
	AssetSize      int64  `json:"asset_size,omitempty"`
	AssetSizeStr   string `json:"asset_size_str,omitempty"`
}

type UpdateProgress struct {
	Status     string `json:"status"` // "idle", "downloading", "extracting", "applying", "completed", "error"
	Progress   int    `json:"progress"`
	BytesRead  int64  `json:"bytes_read"`
	TotalBytes int64  `json:"total_bytes"`
	Message    string `json:"message"`
	Error      string `json:"error,omitempty"`
}

var (
	updateLock     sync.Mutex
	isUpdating     bool
	updateProgress = UpdateProgress{Status: "idle"}
)

func CheckForUpdates(repo string, token string) (*UpdateCheckResult, error) {
	if repo == "" {
		repo = "fery/mylokalwebserver"
	}
	repo = strings.TrimPrefix(repo, "https://github.com/")
	repo = strings.TrimSuffix(repo, "/")

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo)
	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "MyLokalWebserver-Updater/"+AppVersion)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   15 * time.Second,
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("gagal terhubung ke GitHub: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return &UpdateCheckResult{
			HasUpdate:      false,
			CurrentVersion: AppVersion,
			LatestVersion:  AppVersion,
			ReleaseNotes:   "Belum ada rilis versi publik di repository GitHub.",
		}, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API mengembalikan status %d", resp.StatusCode)
	}

	var rel GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("gagal membaca data rilis: %w", err)
	}

	latestVer := strings.TrimSpace(rel.TagName)
	hasUpdate := isVersionNewer(latestVer, AppVersion)

	matchedAsset := findMatchingAsset(rel.Assets)
	var downloadURL, assetName, assetSizeStr string
	var assetSize int64

	if matchedAsset != nil {
		downloadURL = matchedAsset.BrowserDownloadURL
		assetName = matchedAsset.Name
		assetSize = matchedAsset.Size
		assetSizeStr = formatBytes(assetSize)
	}

	pubDate := ""
	if !rel.PublishedAt.IsZero() {
		pubDate = rel.PublishedAt.Format("02 Jan 2006, 15:04 MST")
	}

	return &UpdateCheckResult{
		HasUpdate:      hasUpdate,
		CurrentVersion: AppVersion,
		LatestVersion:  latestVer,
		ReleaseName:    rel.Name,
		ReleaseNotes:   rel.Body,
		PublishedAt:    pubDate,
		DownloadURL:    downloadURL,
		AssetName:      assetName,
		AssetSize:      assetSize,
		AssetSizeStr:   assetSizeStr,
	}, nil
}

func findMatchingAsset(assets []GitHubAsset) *GitHubAsset {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	for _, a := range assets {
		name := strings.ToLower(a.Name)
		if goos == "windows" {
			if strings.HasSuffix(name, ".exe") || strings.Contains(name, "windows") || strings.Contains(name, "win") {
				return &a
			}
		} else if goos == "darwin" {
			if goarch == "arm64" && (strings.Contains(name, "arm64") || strings.Contains(name, "apple") || strings.Contains(name, "macos-arm64")) {
				return &a
			}
			if goarch == "amd64" && (strings.Contains(name, "amd64") || strings.Contains(name, "intel") || strings.Contains(name, "macos-intel")) {
				return &a
			}
			if strings.Contains(name, "darwin") || strings.Contains(name, "macos") {
				return &a
			}
		} else if goos == "linux" {
			if strings.Contains(name, "linux") {
				return &a
			}
		}
	}

	// Fallback to first executable-like asset
	if len(assets) > 0 {
		return &assets[0]
	}
	return nil
}

func isVersionNewer(latest, current string) bool {
	latest = strings.TrimPrefix(strings.TrimSpace(latest), "v")
	current = strings.TrimPrefix(strings.TrimSpace(current), "v")

	if latest == "" || latest == current {
		return false
	}

	latestParts := strings.Split(latest, ".")
	currentParts := strings.Split(current, ".")

	for i := 0; i < len(latestParts) && i < len(currentParts); i++ {
		lNum, err1 := strconv.Atoi(latestParts[i])
		cNum, err2 := strconv.Atoi(currentParts[i])
		if err1 == nil && err2 == nil {
			if lNum > cNum {
				return true
			}
			if lNum < cNum {
				return false
			}
		} else {
			if latestParts[i] > currentParts[i] {
				return true
			}
			if latestParts[i] < currentParts[i] {
				return false
			}
		}
	}

	return len(latestParts) > len(currentParts)
}

func ApplySelfUpdate(downloadURL string, token string) error {
	updateLock.Lock()
	if isUpdating {
		updateLock.Unlock()
		return fmt.Errorf("pembaruan sedang berjalan")
	}
	isUpdating = true
	updateProgress = UpdateProgress{
		Status:   "downloading",
		Progress: 0,
		Message:  "Menghubungkan ke GitHub untuk mengunduh versi baru...",
	}
	updateLock.Unlock()

	defer func() {
		updateLock.Lock()
		isUpdating = false
		updateLock.Unlock()
	}()

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("gagal mendapatkan path binary saat ini: %w", err)
	}
	currentExe, err = filepath.EvalSymlinks(currentExe)
	if err != nil {
		currentExe, _ = os.Executable()
	}

	newExePath := currentExe + ".download"
	oldExePath := currentExe + ".old"

	// 1. Download file
	req, err := http.NewRequest(http.MethodGet, downloadURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", "MyLokalWebserver-Updater/"+AppVersion)
	req.Header.Set("Accept", "application/octet-stream")
	if token != "" {
		req.Header.Set("Authorization", "token "+token)
	}

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Transport: tr,
		Timeout:   5 * time.Minute,
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("gagal mengunduh file rilis: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unduhan mengembalikan status %d", resp.StatusCode)
	}

	outFile, err := os.OpenFile(newExePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("gagal membuat file sementara: %w", err)
	}

	totalSize := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)

	for {
		n, rErr := resp.Body.Read(buf)
		if n > 0 {
			_, wErr := outFile.Write(buf[:n])
			if wErr != nil {
				outFile.Close()
				_ = os.Remove(newExePath)
				return wErr
			}
			downloaded += int64(n)
			if totalSize > 0 {
				pct := int(float64(downloaded) / float64(totalSize) * 100)
				updateLock.Lock()
				updateProgress = UpdateProgress{
					Status:     "downloading",
					Progress:   pct,
					BytesRead:  downloaded,
					TotalBytes: totalSize,
					Message:    fmt.Sprintf("Mengunduh pembaruan (%s / %s)... %d%%", formatBytes(downloaded), formatBytes(totalSize), pct),
				}
				updateLock.Unlock()
			}
		}
		if rErr != nil {
			if rErr == io.EOF {
				break
			}
			outFile.Close()
			_ = os.Remove(newExePath)
			return rErr
		}
	}
	outFile.Close()

	// Ensure permissions
	_ = os.Chmod(newExePath, 0755)

	// Clean previous .old file
	_ = os.Remove(oldExePath)

	updateLock.Lock()
	updateProgress = UpdateProgress{
		Status:   "applying",
		Progress: 95,
		Message:  "Menerapkan binary baru...",
	}
	updateLock.Unlock()

	// 2. Perform atomic swap
	if runtime.GOOS == "windows" {
		if err := os.Rename(currentExe, oldExePath); err != nil {
			_ = os.Remove(newExePath)
			return fmt.Errorf("gagal memindahkan binary aktif: %w", err)
		}
		if err := os.Rename(newExePath, currentExe); err != nil {
			_ = os.Rename(oldExePath, currentExe)
			_ = os.Remove(newExePath)
			return fmt.Errorf("gagal mengaktifkan binary baru: %w", err)
		}
	} else {
		if err := os.Rename(newExePath, currentExe); err != nil {
			_ = os.Remove(newExePath)
			return fmt.Errorf("gagal mengganti binary: %w", err)
		}
	}

	updateLock.Lock()
	updateProgress = UpdateProgress{
		Status:   "completed",
		Progress: 100,
		Message:  "Pembaruan sukses dipasang! Memulai ulang aplikasi...",
	}
	updateLock.Unlock()

	// 3. Restart application
	go func() {
		time.Sleep(1200 * time.Millisecond)
		Manager.StopAll()
		cmd := exec.Command(currentExe, "-no-browser")
		cmd.Dir = AppRootDir
		SetProcessGroupAttributes(cmd)
		_ = cmd.Start()
		os.Exit(0)
	}()

	return nil
}

func GetUpdateProgress() UpdateProgress {
	updateLock.Lock()
	defer updateLock.Unlock()
	return updateProgress
}

func CleanupOldBinary() {
	exe, err := os.Executable()
	if err == nil {
		oldExe := exe + ".old"
		if fileExists(oldExe) {
			_ = os.Remove(oldExe)
		}
	}
}

func formatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(b)/float64(div), "KMGTPE"[exp])
}
