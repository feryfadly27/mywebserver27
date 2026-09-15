package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type VirtualHost struct {
	Domain       string `json:"domain"`        // e.g. "toko.test"
	Folder       string `json:"folder"`        // relative to www/htdocs, e.g. "toko" or "toko/public"
	DocumentRoot string `json:"document_root"` // full absolute path
	Enabled      bool   `json:"enabled"`
}

type Settings struct {
	ApachePort       int           `json:"apache_port"`
	MariaDBPort      int           `json:"mariadb_port"`
	PanelPort        int           `json:"panel_port"`
	AutoStart        bool          `json:"auto_start"`
	AutoPortFallback bool          `json:"auto_port_fallback"`
	Shell            string        `json:"shell"` // "cmd" or "powershell"
	GitHubRepo       string        `json:"github_repo,omitempty"`
	GitHubToken      string        `json:"github_token,omitempty"`
	AutoCheckUpdate  bool          `json:"auto_check_update"`
	VirtualHosts     []VirtualHost `json:"virtual_hosts,omitempty"`
}

var (
	settingsLock    sync.RWMutex
	currentSettings Settings
)

func DefaultSettings() Settings {
	return Settings{
		ApachePort:       8080,
		MariaDBPort:      3307,
		PanelPort:        3000,
		AutoStart:        true,
		AutoPortFallback: true,
		Shell:            "cmd",
		GitHubRepo:       "feryfadly27/mywebserver27",
		AutoCheckUpdate:  true,
		VirtualHosts:     []VirtualHost{},
	}
}

func getSettingsPath() string {
	return filepath.Join(AppRootDir, "settings.json")
}

func LoadSettings() (Settings, error) {
	settingsLock.Lock()
	defer settingsLock.Unlock()

	s := DefaultSettings()
	path := getSettingsPath()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		currentSettings = s
		// Save default settings
		data, _ := json.MarshalIndent(s, "", "  ")
		_ = os.WriteFile(path, data, 0644)
		return s, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		currentSettings = s
		return s, err
	}

	if err := json.Unmarshal(data, &s); err != nil {
		currentSettings = DefaultSettings()
		return currentSettings, err
	}

	// Validate defaults if zero
	if s.ApachePort == 0 {
		s.ApachePort = 8080
	}
	if s.MariaDBPort == 0 {
		s.MariaDBPort = 3306
	}
	if s.PanelPort == 0 {
		s.PanelPort = 3000
	}
	if s.Shell == "" {
		s.Shell = "cmd"
	}

	currentSettings = s
	return s, nil
}

func SaveSettings(s Settings) error {
	settingsLock.Lock()
	defer settingsLock.Unlock()

	if s.ApachePort == 0 {
		s.ApachePort = 8080
	}
	if s.MariaDBPort == 0 {
		s.MariaDBPort = 3306
	}
	if s.PanelPort == 0 {
		s.PanelPort = 3000
	}
	if s.Shell == "" {
		s.Shell = "cmd"
	}

	currentSettings = s
	path := getSettingsPath()
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0644)
}

func GetCurrentSettings() Settings {
	settingsLock.RLock()
	defer settingsLock.RUnlock()
	return currentSettings
}
