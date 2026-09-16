package main

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// CheckDomainInHosts checks if domain is mapped to 127.0.0.1 or ::1 in OS hosts file
func CheckDomainInHosts(domain string) bool {
	domain = strings.TrimSpace(strings.ToLower(domain))
	if domain == "" {
		return false
	}
	// *.localhost domains automatically resolve to 127.0.0.1 without hosts entry
	if strings.HasSuffix(domain, ".localhost") || domain == "localhost" {
		return true
	}

	hostsPath := GetHostsFilePath()
	f, err := os.Open(hostsPath)
	if err != nil {
		return false
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// Strip inline comment
		if idx := strings.Index(line, "#"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			ip := parts[0]
			if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
				for _, host := range parts[1:] {
					if strings.ToLower(host) == domain {
						return true
					}
				}
			}
		}
	}
	return false
}

// GetMissingHostsDomains filters out domains that are already in hosts file or end with .localhost
func GetMissingHostsDomains(domains []string) []string {
	var missing []string
	for _, d := range domains {
		d = strings.TrimSpace(d)
		if d == "" {
			continue
		}
		if !CheckDomainInHosts(d) {
			missing = append(missing, d)
		}
	}
	return missing
}

// SyncDomainsToHosts adds missing domains to the OS hosts file
func SyncDomainsToHosts(domains []string) error {
	missing := GetMissingHostsDomains(domains)
	if len(missing) == 0 {
		return nil // already synced
	}

	if runtime.GOOS == "windows" {
		hostsPath := GetHostsFilePath()
		
		// 1. Create a reliable batch file for UAC elevation
		var batContent strings.Builder
		batContent.WriteString("@echo off\r\n")
		batContent.WriteString("echo Menyinkronkan domain ke hosts Windows...\r\n")
		batContent.WriteString(fmt.Sprintf("echo. >> \"%s\"\r\n", hostsPath))
		batContent.WriteString(fmt.Sprintf("echo # Added by MyLokalWebserver >> \"%s\"\r\n", hostsPath))
		for _, d := range missing {
			batContent.WriteString(fmt.Sprintf("echo 127.0.0.1  %s >> \"%s\"\r\n", d, hostsPath))
			batContent.WriteString(fmt.Sprintf("echo 127.0.0.1  www.%s >> \"%s\"\r\n", d, hostsPath))
		}
		batContent.WriteString("ipconfig /flushdns >nul 2>&1\r\n")
		batContent.WriteString("echo Selesai.\r\n")

		// Write to temp bat file and project root sync_hosts.bat for convenience
		tmpBat := filepath.Join(os.TempDir(), "sync_hosts_mylokal.bat")
		_ = os.WriteFile(tmpBat, []byte(batContent.String()), 0755)

		if AppRootDir != "" {
			_ = os.WriteFile(filepath.Join(AppRootDir, "sync_hosts.bat"), []byte(batContent.String()), 0755)
		}

		// Run elevated cmd with runAs
		psCmd := fmt.Sprintf(`Start-Process -FilePath "cmd.exe" -ArgumentList "/c ` + "`" + `"%s` + "`" + `" " -Verb runAs -Wait`, tmpBat)
		cmd := exec.Command("powershell", "-NoProfile", "-Command", psCmd)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("izin Administrator (UAC) diperlukan untuk mengubah berkas hosts Windows. Anda juga dapat menjalankan 'sync_hosts.bat' sebagai Administrator.")
		}

		// Clean up temp bat
		_ = os.Remove(tmpBat)

		// Check if domains are now registered
		if len(GetMissingHostsDomains(domains)) == 0 {
			return nil
		}
		return nil
	}

	// Unix / macOS fallback
	hostsPath := GetHostsFilePath()
	f, err := os.OpenFile(hostsPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("permission denied writing to %s. Please run with sudo or add entries manually.", hostsPath)
	}
	defer f.Close()

	_, _ = f.WriteString("\n# Added by MyLokalWebserver\n")
	for _, d := range missing {
		_, _ = f.WriteString(fmt.Sprintf("127.0.0.1 %s www.%s\n", d, d))
	}
	return nil
}
