package main

import (
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type ServiceStatus struct {
	Name      string    `json:"name"`
	Running   bool      `json:"running"`
	PID       int       `json:"pid"`
	Port      int       `json:"port"`
	StartTime time.Time `json:"start_time,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
	Version   string    `json:"version,omitempty"`
}

type ServicesManager struct {
	lock         sync.Mutex
	apacheCmd    *exec.Cmd
	apacheStart  time.Time
	mariadbCmd   *exec.Cmd
	mariadbStart time.Time
}

var Manager = &ServicesManager{}

func (sm *ServicesManager) IsPortOpen(port int) bool {
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 400*time.Millisecond)
	if err == nil {
		conn.Close()
		return true
	}
	return false
}

func (sm *ServicesManager) StartApache() error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	settings := GetCurrentSettings()
	if sm.apacheCmd != nil && sm.apacheCmd.Process != nil {
		if sm.IsPortOpen(settings.ApachePort) {
			return nil // already running
		}
	} else if sm.IsPortOpen(settings.ApachePort) {
		return fmt.Errorf("Port %d is already in use by another application. Please close the conflicting application or change Apache port in Settings.", settings.ApachePort)
	}

	// Regenerate configs in case port changed
	_ = GenerateApacheConfig(settings)
	_ = GeneratePHPConfig(settings)

	exeExt := GetExecutableExt()
	apacheBin := filepath.Join(AppRootDir, "bin", "apache", "bin", "httpd"+exeExt)
	if !fileExists(apacheBin) {
		apacheBin = filepath.Join(AppRootDir, "bin", "apache", "bin", "httpd")
	}
	if !fileExists(apacheBin) {
		return fmt.Errorf("apache binary not found at %s. Please ensure Apache is downloaded/installed via Component Downloader.", apacheBin)
	}

	confPath := filepath.Join(AppRootDir, "bin", "apache", "conf", "httpd.conf")
	phpDir := filepath.Join(AppRootDir, "bin", "php")
	mariadbBinDir := filepath.Join(AppRootDir, "bin", "mariadb", "bin")
	apacheBinDir := filepath.Join(AppRootDir, "bin", "apache", "bin")
	sep := GetPathListSeparator()
	envPath := fmt.Sprintf("PATH=%s%s%s%s%s%s%s", phpDir, sep, mariadbBinDir, sep, apacheBinDir, sep, os.Getenv("PATH"))

	// Pre-flight Apache syntax check
	testCmd := exec.Command(apacheBin, "-t", "-f", confPath)
	testCmd.Dir = filepath.Join(AppRootDir, "bin", "apache")
	SetCmdHideWindow(testCmd)
	testCmd.Env = append(os.Environ(), envPath)
	if output, err := testCmd.CombinedOutput(); err != nil {
		outStr := strings.TrimSpace(string(output))
		if outStr != "" {
			return fmt.Errorf("Apache configuration error: %s", outStr)
		}
		return fmt.Errorf("Apache pre-flight test failed: %w", err)
	}

	cmd := exec.Command(apacheBin, "-f", confPath)
	cmd.Dir = filepath.Join(AppRootDir, "bin", "apache")
	SetCmdHideWindow(cmd)
	cmd.Env = append(os.Environ(), envPath)

	// Forward logs
	logsDir := filepath.Join(AppRootDir, "logs")
	_ = os.MkdirAll(logsDir, 0755)
	logFile, _ := os.OpenFile(filepath.Join(logsDir, "apache_runner.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Apache: %w", err)
	}

	sm.apacheCmd = cmd
	sm.apacheStart = time.Now()

	// Monitor process
	go func() {
		_ = cmd.Wait()
		sm.lock.Lock()
		if sm.apacheCmd == cmd {
			sm.apacheCmd = nil
		}
		sm.lock.Unlock()
	}()

	// Brief wait to ensure Apache hasn't crashed on boot
	time.Sleep(500 * time.Millisecond)
	sm.lock.Lock()
	isExited := (sm.apacheCmd == nil)
	sm.lock.Unlock()

	if isExited {
		logBytes, _ := os.ReadFile(filepath.Join(logsDir, "apache_runner.log"))
		errBytes, _ := os.ReadFile(filepath.Join(logsDir, "apache_error.log"))
		combinedLogs := strings.TrimSpace(string(logBytes) + "\n" + string(errBytes))
		if len(combinedLogs) > 350 {
			combinedLogs = combinedLogs[len(combinedLogs)-350:]
		}
		if combinedLogs != "" {
			return fmt.Errorf("Apache stopped immediately after start: %s", combinedLogs)
		}
		return fmt.Errorf("Apache failed to start. On Windows, please ensure Microsoft Visual C++ 2015-2022 Redistributable (x64) is installed.")
	}

	return nil
}

func (sm *ServicesManager) StopApache() error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	exeExt := GetExecutableExt()
	apacheBin := filepath.Join(AppRootDir, "bin", "apache", "bin", "httpd"+exeExt)
	if !fileExists(apacheBin) {
		apacheBin = filepath.Join(AppRootDir, "bin", "apache", "bin", "httpd")
	}
	confPath := filepath.Join(AppRootDir, "bin", "apache", "conf", "httpd.conf")

	// Try graceful stop
	if fileExists(apacheBin) {
		stopCmd := exec.Command(apacheBin, "-k", "stop", "-f", confPath)
		SetCmdHideWindow(stopCmd)
		_ = stopCmd.Run()
	}

	if sm.apacheCmd != nil && sm.apacheCmd.Process != nil {
		_ = KillProcessTree(sm.apacheCmd.Process.Pid)
		sm.apacheCmd = nil
	}

	// Also kill any lingering httpd processes
	KillByName("httpd" + exeExt)
	KillByName("httpd")

	return nil
}

func (sm *ServicesManager) RestartApache() error {
	_ = sm.StopApache()
	time.Sleep(1 * time.Second)
	return sm.StartApache()
}

func (sm *ServicesManager) StartMariaDB() error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	settings := GetCurrentSettings()
	if sm.mariadbCmd != nil && sm.mariadbCmd.Process != nil {
		if sm.IsPortOpen(settings.MariaDBPort) {
			return nil // already running
		}
	}

	_ = InitMariaDBData()
	_ = GenerateMariaDBConfig(settings)
	_ = GeneratePhpMyAdminConfig(settings)

	exeExt := GetExecutableExt()
	mariadbBin := filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mariadbd"+exeExt)
	if !fileExists(mariadbBin) {
		mariadbBin = filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mysqld"+exeExt)
	}
	if !fileExists(mariadbBin) {
		mariadbBin = filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mariadbd")
	}
	if !fileExists(mariadbBin) {
		mariadbBin = filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mysqld")
	}
	if !fileExists(mariadbBin) {
		return fmt.Errorf("mariadb binary not found")
	}

	myIni := filepath.Join(AppRootDir, "bin", "mariadb", "my.ini")
	cmd := exec.Command(mariadbBin, fmt.Sprintf("--defaults-file=%s", myIni), "--console")
	cmd.Dir = filepath.Join(AppRootDir, "bin", "mariadb")
	SetCmdHideWindow(cmd)

	logsDir := filepath.Join(AppRootDir, "logs")
	_ = os.MkdirAll(logsDir, 0755)
	logFile, _ := os.OpenFile(filepath.Join(logsDir, "mariadb_runner.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if logFile != nil {
		cmd.Stdout = logFile
		cmd.Stderr = logFile
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start MariaDB: %w", err)
	}

	sm.mariadbCmd = cmd
	sm.mariadbStart = time.Now()

	go func() {
		_ = cmd.Wait()
		sm.lock.Lock()
		if sm.mariadbCmd == cmd {
			sm.mariadbCmd = nil
		}
		sm.lock.Unlock()
	}()

	return nil
}

func (sm *ServicesManager) StopMariaDB() error {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	settings := GetCurrentSettings()

	// Try graceful shutdown via mariadb-admin
	exeExt := GetExecutableExt()
	adminBin := filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mariadb-admin"+exeExt)
	if !fileExists(adminBin) {
		adminBin = filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mysqladmin"+exeExt)
	}
	if !fileExists(adminBin) {
		adminBin = filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mariadb-admin")
	}
	if !fileExists(adminBin) {
		adminBin = filepath.Join(AppRootDir, "bin", "mariadb", "bin", "mysqladmin")
	}

	if fileExists(adminBin) {
		shutdownCmd := exec.Command(adminBin, "-u", "root", fmt.Sprintf("--port=%d", settings.MariaDBPort), "shutdown")
		SetCmdHideWindow(shutdownCmd)
		_ = shutdownCmd.Run()
	}

	if sm.mariadbCmd != nil && sm.mariadbCmd.Process != nil {
		_ = KillProcessTree(sm.mariadbCmd.Process.Pid)
		sm.mariadbCmd = nil
	}

	// Also kill any lingering mysqld / mariadbd
	KillByName("mariadbd" + exeExt)
	KillByName("mysqld" + exeExt)
	KillByName("mariadbd")
	KillByName("mysqld")

	return nil
}

func (sm *ServicesManager) RestartMariaDB() error {
	_ = sm.StopMariaDB()
	time.Sleep(1 * time.Second)
	return sm.StartMariaDB()
}

func (sm *ServicesManager) StopAll() {
	_ = sm.StopApache()
	_ = sm.StopMariaDB()
}

func (sm *ServicesManager) GetStatus() map[string]ServiceStatus {
	sm.lock.Lock()
	defer sm.lock.Unlock()

	settings := GetCurrentSettings()
	result := make(map[string]ServiceStatus)

	// Apache Status
	apacheRunning := sm.IsPortOpen(settings.ApachePort)
	apachePid := 0
	if sm.apacheCmd != nil && sm.apacheCmd.Process != nil {
		apachePid = sm.apacheCmd.Process.Pid
	}
	apacheUptime := ""
	if apacheRunning && !sm.apacheStart.IsZero() {
		apacheUptime = formatDuration(time.Since(sm.apacheStart))
	}

	result["apache"] = ServiceStatus{
		Name:      "Apache HTTP Server",
		Running:   apacheRunning,
		PID:       apachePid,
		Port:      settings.ApachePort,
		StartTime: sm.apacheStart,
		Uptime:    apacheUptime,
	}

	// MariaDB Status
	mariadbRunning := sm.IsPortOpen(settings.MariaDBPort)
	mariadbPid := 0
	if sm.mariadbCmd != nil && sm.mariadbCmd.Process != nil {
		mariadbPid = sm.mariadbCmd.Process.Pid
	}
	mariadbUptime := ""
	if mariadbRunning && !sm.mariadbStart.IsZero() {
		mariadbUptime = formatDuration(time.Since(sm.mariadbStart))
	}

	result["mariadb"] = ServiceStatus{
		Name:      "MariaDB Database",
		Running:   mariadbRunning,
		PID:       mariadbPid,
		Port:      settings.MariaDBPort,
		StartTime: sm.mariadbStart,
		Uptime:    mariadbUptime,
	}

	return result
}

func formatDuration(d time.Duration) string {
	d = d.Round(time.Second)
	h := d / time.Hour
	d -= h * time.Hour
	m := d / time.Minute
	d -= m * time.Minute
	s := d / time.Second

	if h > 0 {
		return fmt.Sprintf("%02dh %02dm %02ds", h, m, s)
	}
	if m > 0 {
		return fmt.Sprintf("%02dm %02ds", m, s)
	}
	return fmt.Sprintf("%02ds", s)
}

// Global convenience functions
func StartApache() error   { return Manager.StartApache() }
func StopApache() error    { return Manager.StopApache() }
func RestartApache() error { return Manager.RestartApache() }
func StartMariaDB() error  { return Manager.StartMariaDB() }
func StopMariaDB() error   { return Manager.StopMariaDB() }
func RestartMariaDB() error { return Manager.RestartMariaDB() }
