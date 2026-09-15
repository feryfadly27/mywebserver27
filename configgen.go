package main

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func toApachePath(path string) string {
	return strings.ReplaceAll(path, "\\", "/")
}

func GenerateApacheConfig(settings Settings) error {
	apacheDir := filepath.Join(AppRootDir, "bin", "apache")
	phpDir := filepath.Join(AppRootDir, "bin", "php")
	htdocsDir := filepath.Join(AppRootDir, "www", "htdocs")
	logsDir := filepath.Join(AppRootDir, "logs")

	// Find the PHP apache module dll name
	phpDll := "php8apache2_4.dll"
	files, _ := os.ReadDir(phpDir)
	for _, f := range files {
		if strings.HasPrefix(strings.ToLower(f.Name()), "php") && strings.HasSuffix(strings.ToLower(f.Name()), "apache2_4.dll") {
			phpDll = f.Name()
			break
		}
	}

	confContent := fmt.Sprintf(`# MyLokalWebserver - Auto-generated Apache Configuration
Define SRVROOT "%s"
ServerRoot "${SRVROOT}"

Listen %d
ServerName localhost:%d

# Modules
LoadModule actions_module modules/mod_actions.so
LoadModule alias_module modules/mod_alias.so
LoadModule authz_core_module modules/mod_authz_core.so
LoadModule authz_host_module modules/mod_authz_host.so
LoadModule autoindex_module modules/mod_autoindex.so
LoadModule dir_module modules/mod_dir.so
LoadModule env_module modules/mod_env.so
LoadModule headers_module modules/mod_headers.so
LoadModule mime_module modules/mod_mime.so
LoadModule negotiation_module modules/mod_negotiation.so
LoadModule rewrite_module modules/mod_rewrite.so
LoadModule setenvif_module modules/mod_setenvif.so
LoadModule log_config_module modules/mod_log_config.so

# PHP Module
LoadModule php_module "%s/%s"
PHPIniDir "%s"

<FilesMatch \.php$>
    SetHandler application/x-httpd-php
</FilesMatch>

DocumentRoot "%s"
<Directory "%s">
    Options Indexes FollowSymLinks MultiViews
    AllowOverride All
    Require all granted
</Directory>

<IfModule dir_module>
    DirectoryIndex index.php index.html index.htm
</IfModule>

<IfModule log_config_module>
    LogFormat "%%h %%l %%u %%t \"%%r\" %%>s %%b" common
    CustomLog "%s/apache_access.log" common
</IfModule>

ErrorLog "%s/apache_error.log"
LogLevel warn

<IfModule mime_module>
    TypesConfig conf/mime.types
    AddType application/x-compress .Z
    AddType application/x-gzip .gz .tgz
    AddType application/x-httpd-php .php
    AddType application/x-httpd-php-source .phps
</IfModule>

# Virtual Hosts
IncludeOptional conf/vhosts.conf
`,
		toApachePath(apacheDir),
		settings.ApachePort,
		settings.ApachePort,
		toApachePath(phpDir), phpDll,
		toApachePath(phpDir),
		toApachePath(htdocsDir),
		toApachePath(htdocsDir),
		toApachePath(logsDir),
		toApachePath(logsDir),
	)

	confDir := filepath.Join(apacheDir, "conf")
	_ = os.MkdirAll(confDir, 0755)
	if err := os.WriteFile(filepath.Join(confDir, "httpd.conf"), []byte(confContent), 0644); err != nil {
		return err
	}
	return GenerateVhostsConfig(settings)
}

func sanitizeFilename(s string) string {
	s = strings.ReplaceAll(s, ":", "_")
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, "\\", "_")
	s = strings.ReplaceAll(s, ".", "_")
	return s
}

func GenerateVhostsConfig(settings Settings) error {
	apacheDir := filepath.Join(AppRootDir, "bin", "apache")
	htdocsDir := filepath.Join(AppRootDir, "www", "htdocs")
	logsDir := filepath.Join(AppRootDir, "logs")

	var sb strings.Builder
	sb.WriteString("# MyLokalWebserver - Auto-generated Virtual Hosts\n\n")

	// 1. Default Host (localhost & 127.0.0.1) -> points to www/htdocs (Ensures standard PHP native & phpMyAdmin work 100%)
	sb.WriteString(fmt.Sprintf(`<VirtualHost *:%d>
    ServerName localhost
    ServerAlias 127.0.0.1
    DocumentRoot "%s"
    <Directory "%s">
        Options Indexes FollowSymLinks MultiViews
        AllowOverride All
        Require all granted
    </Directory>
    CustomLog "%s/access_localhost.log" common
    ErrorLog "%s/error_localhost.log"
</VirtualHost>

`, settings.ApachePort, toApachePath(htdocsDir), toApachePath(htdocsDir), toApachePath(logsDir), toApachePath(logsDir)))

	// 2. Custom Virtual Hosts
	for _, vh := range settings.VirtualHosts {
		if !vh.Enabled || strings.TrimSpace(vh.Domain) == "" {
			continue
		}

		docRoot := vh.DocumentRoot
		if docRoot == "" {
			docRoot = filepath.Join(htdocsDir, filepath.FromSlash(vh.Folder))
		}

		_ = os.MkdirAll(docRoot, 0755)

		sb.WriteString(fmt.Sprintf(`<VirtualHost *:%d>
    ServerName %s
    ServerAlias www.%s
    DocumentRoot "%s"
    <Directory "%s">
        Options Indexes FollowSymLinks MultiViews
        AllowOverride All
        Require all granted
    </Directory>
    CustomLog "%s/access_%s.log" common
    ErrorLog "%s/error_%s.log"
</VirtualHost>

`, settings.ApachePort, vh.Domain, vh.Domain, toApachePath(docRoot), toApachePath(docRoot), toApachePath(logsDir), sanitizeFilename(vh.Domain), toApachePath(logsDir), sanitizeFilename(vh.Domain)))
	}

	confDir := filepath.Join(apacheDir, "conf")
	_ = os.MkdirAll(confDir, 0755)
	return os.WriteFile(filepath.Join(confDir, "vhosts.conf"), []byte(sb.String()), 0644)
}

func GeneratePHPConfig(settings Settings) error {
	phpDir := filepath.Join(AppRootDir, "bin", "php")
	extDir := filepath.Join(phpDir, "ext")
	tmpDir := filepath.Join(AppRootDir, "tmp")
	logsDir := filepath.Join(AppRootDir, "logs")

	iniContent := fmt.Sprintf(`[PHP]
engine = On
short_open_tag = Off
precision = 14
output_buffering = 4096
zlib.output_compression = Off
implicit_flush = Off
serialize_precision = -1
zend.enable_gc = On

max_execution_time = 120
max_input_time = 60
memory_limit = 256M

error_reporting = E_ALL & ~E_DEPRECATED & ~E_STRICT
display_errors = On
display_startup_errors = On
log_errors = On
error_log = "%s/php_error.log"

post_max_size = 64M
upload_max_filesize = 64M
max_file_uploads = 20
upload_tmp_dir = "%s"

extension_dir = "%s"

; Extensions
extension=curl
extension=fileinfo
extension=gd
extension=mbstring
extension=exif
extension=mysqli
extension=openssl
extension=pdo_mysql

[Date]
date.timezone = "Asia/Jakarta"

[Session]
session.save_handler = files
session.save_path = "%s"
session.use_strict_mode = 0
session.use_cookies = 1
session.use_only_cookies = 1
session.name = PHPSESSID
session.auto_start = 0
session.cookie_lifetime = 0
session.cookie_path = /
session.gc_probability = 1
session.gc_divisor = 1000
session.gc_maxlifetime = 1440

[MySQLi]
mysqli.max_persistent = -1
mysqli.allow_persistent = On
mysqli.max_links = -1
mysqli.default_port = %d
mysqli.default_socket =
mysqli.default_host = 127.0.0.1
mysqli.default_user = root
mysqli.default_pw =
mysqli.reconnect = Off

[Pdo_mysql]
pdo_mysql.default_socket=
`,
		toApachePath(logsDir),
		toApachePath(tmpDir),
		toApachePath(extDir),
		toApachePath(tmpDir),
		settings.MariaDBPort,
	)

	return os.WriteFile(filepath.Join(phpDir, "php.ini"), []byte(iniContent), 0644)
}

func GenerateMariaDBConfig(settings Settings) error {
	mariadbDir := filepath.Join(AppRootDir, "bin", "mariadb")
	dataDir := filepath.Join(AppRootDir, "data", "mysql")
	logsDir := filepath.Join(AppRootDir, "logs")

	iniContent := fmt.Sprintf(`[mysqld]
basedir = "%s"
datadir = "%s"
port = %d
bind-address = 127.0.0.1
character-set-server = utf8mb4
collation-server = utf8mb4_unicode_ci
default-storage-engine = InnoDB
innodb_buffer_pool_size = 128M
log-error = "%s/mariadb_error.log"
max_allowed_packet = 64M

[client]
host = 127.0.0.1
port = %d
user = root
protocol = tcp
ssl = 0
default-character-set = utf8mb4

[mysql]
host = 127.0.0.1
port = %d
user = root
protocol = tcp
ssl = 0
default-character-set = utf8mb4

[mariadb-client]
host = 127.0.0.1
port = %d
user = root
protocol = tcp
ssl = 0
default-character-set = utf8mb4
`,
		toApachePath(mariadbDir),
		toApachePath(dataDir),
		settings.MariaDBPort,
		toApachePath(logsDir),
		settings.MariaDBPort,
		settings.MariaDBPort,
		settings.MariaDBPort,
	)

	_ = os.WriteFile(filepath.Join(mariadbDir, "my.ini"), []byte(iniContent), 0644)
	_ = os.WriteFile(filepath.Join(mariadbDir, "bin", "my.ini"), []byte(iniContent), 0644)
	_ = os.WriteFile(filepath.Join(AppRootDir, "www", "htdocs", "my.ini"), []byte(iniContent), 0644)
	return nil
}

func GeneratePhpMyAdminConfig(settings Settings) error {
	pmaDir := filepath.Join(AppRootDir, "www", "htdocs", "phpmyadmin")
	if _, err := os.Stat(pmaDir); os.IsNotExist(err) {
		return nil
	}

	secretBytes := make([]byte, 16)
	_, _ = rand.Read(secretBytes)
	secret := hex.EncodeToString(secretBytes)

	confContent := fmt.Sprintf(`<?php
/**
 * MyLokalWebserver - Auto-generated phpMyAdmin Configuration
 */
declare(strict_types=1);

$cfg['blowfish_secret'] = '%s';

$i = 0;
$i++;
$cfg['Servers'][$i]['auth_type'] = 'config';
$cfg['Servers'][$i]['user'] = 'root';
$cfg['Servers'][$i]['password'] = '';
$cfg['Servers'][$i]['host'] = '127.0.0.1';
$cfg['Servers'][$i]['port'] = '%d';
$cfg['Servers'][$i]['Compress'] = false;
$cfg['Servers'][$i]['AllowNoPassword'] = true;

$cfg['UploadDir'] = '';
$cfg['SaveDir'] = '';
$cfg['TempDir'] = '%s';
$cfg['SendErrorReports'] = 'never';
`,
		secret,
		settings.MariaDBPort,
		toApachePath(filepath.Join(AppRootDir, "tmp")),
	)

	return os.WriteFile(filepath.Join(pmaDir, "config.inc.php"), []byte(confContent), 0644)
}

func GenerateAllConfigs(settings Settings) error {
	_ = os.MkdirAll(filepath.Join(AppRootDir, "logs"), 0755)
	_ = os.MkdirAll(filepath.Join(AppRootDir, "tmp"), 0755)
	_ = os.MkdirAll(filepath.Join(AppRootDir, "data", "mysql"), 0755)
	_ = os.MkdirAll(filepath.Join(AppRootDir, "www", "htdocs"), 0755)

	if err := GenerateApacheConfig(settings); err != nil {
		return err
	}
	if err := GeneratePHPConfig(settings); err != nil {
		return err
	}
	if err := GenerateMariaDBConfig(settings); err != nil {
		return err
	}
	if err := GeneratePhpMyAdminConfig(settings); err != nil {
		return err
	}
	return nil
}
