package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// PHPColumnSetting holds configuration for each column in PHP CRUD generation
type PHPColumnSetting struct {
	Field       string   `json:"field"`
	Label       string   `json:"label"`
	Type        string   `json:"type"`
	ShowInList  bool     `json:"show_in_list"`
	ShowInForm  bool     `json:"show_in_form"`
	InputType   string   `json:"input_type"` // text, number, date, datetime, textarea, select_enum, select_fk
	Required    bool     `json:"required"`
	EnumOptions []string `json:"enum_options,omitempty"`
}

// PHPRelationSetting holds foreign key relational configuration
type PHPRelationSetting struct {
	ColumnName      string `json:"column_name"`       // e.g. kategori_id
	ReferencedTable string `json:"referenced_table"`  // e.g. kategori
	ReferencedPK    string `json:"referenced_pk"`     // e.g. id
	DisplayColumn   string `json:"display_column"`    // e.g. nama
	DisplayAlias    string `json:"display_alias"`     // e.g. kategori_nama
}

// PHPDashboardStatTable defines a table stat widget for the Admin Dashboard
type PHPDashboardStatTable struct {
	TableName string `json:"table_name"`
	Label     string `json:"label"`
	Icon      string `json:"icon"`  // e.g. bi-capsule, bi-tags, bi-box, bi-people
	Color     string `json:"color"` // primary, success, warning, info, danger, dark
}

// PHPDashboardSetting defines options for generating an Admin Dashboard homepage
type PHPDashboardSetting struct {
	Enabled        bool                    `json:"enabled"`
	DashboardTitle string                  `json:"dashboard_title"`
	WelcomeMsg     string                  `json:"welcome_msg"`
	StatTables     []PHPDashboardStatTable `json:"stat_tables"`
	ShowRecent     bool                    `json:"show_recent"`
	RecentTable    string                  `json:"recent_table"`
	ShowQuickLinks bool                    `json:"show_quick_links"`
}

// PHPGenerateRequest is the payload sent from the wizard
type PHPGenerateRequest struct {
	DatabaseName  string               `json:"database_name"`
	TableName     string               `json:"table_name"`
	ProjectFolder string               `json:"project_folder"`
	AppTitle      string               `json:"app_title"`
	PrimaryCol    string               `json:"primary_col"`
	NamingMode    string               `json:"naming_mode"` // "table" (obat.php, obat_tambah.php) or "index" (index.php, tambah.php)
	Overwrite     bool                 `json:"overwrite"`
	Columns       []PHPColumnSetting   `json:"columns"`
	Relations     []PHPRelationSetting `json:"relations"`
	Dashboard     PHPDashboardSetting  `json:"dashboard"`
}

// PHPGeneratedFile represents a single PHP source file
type PHPGeneratedFile struct {
	Filename    string `json:"filename"`
	Content     string `json:"content"`
	Description string `json:"description"`
}

// PHPScaffoldResult is the bundle of all generated files
type PHPScaffoldResult struct {
	DatabaseName  string             `json:"database_name"`
	TableName     string             `json:"table_name"`
	ProjectFolder string             `json:"project_folder"`
	WebURL        string             `json:"web_url"`
	Files         []PHPGeneratedFile `json:"files"`
	ExistingFiles []string           `json:"existing_files"`
	HasExisting   bool               `json:"has_existing"`
}

// PHPPageFilenames defines filenames for CRUD pages
type PHPPageFilenames struct {
	Index  string
	Tambah string
	Edit   string
	Hapus  string
}

func getPageFilenames(req PHPGenerateRequest) PHPPageFilenames {
	cleanTable := sanitizeFilename(req.TableName)
	if cleanTable == "" {
		cleanTable = "data"
	}
	if req.Dashboard.Enabled || req.NamingMode == "table" {
		return PHPPageFilenames{
			Index:  fmt.Sprintf("%s.php", cleanTable),
			Tambah: fmt.Sprintf("%s_tambah.php", cleanTable),
			Edit:   fmt.Sprintf("%s_edit.php", cleanTable),
			Hapus:  fmt.Sprintf("%s_hapus.php", cleanTable),
		}
	}
	return PHPPageFilenames{
		Index:  "index.php",
		Tambah: "tambah.php",
		Edit:   "edit.php",
		Hapus:  "hapus.php",
	}
}

// GenerateKoneksiPHP creates a PDO database connection script
func GenerateKoneksiPHP(dbName string, port int) string {
	return fmt.Sprintf(`<?php
/**
 * Koneksi Database MariaDB / MySQL via PDO
 * Dibuat secara otomatis oleh MyLokalWebserver
 */

$db_host = '127.0.0.1';
$db_port = '%d';
$db_name = '%s';
$db_user = 'root';
$db_pass = '';

try {
    $dsn = "mysql:host={$db_host};port={$db_port};dbname={$db_name};charset=utf8mb4";
    $pdo = new PDO($dsn, $db_user, $db_pass, [
        PDO::ATTR_ERRMODE            => PDO::ERRMODE_EXCEPTION,
        PDO::ATTR_DEFAULT_FETCH_MODE => PDO::FETCH_ASSOC,
        PDO::ATTR_EMULATE_PREPARES   => false,
    ]);
} catch (PDOException $e) {
    die("<div style='font-family: sans-serif; padding: 20px; background: #fee2e2; border: 1px solid #ef4444; border-radius: 8px; color: #991b1b;'>
        <h3 style='margin-top:0;'>❌ Koneksi Database Gagal</h3>
        <p><strong>Pesan Kesalahan:</strong> " . htmlspecialchars($e->getMessage()) . "</p>
        <p><small>Pastikan MariaDB aktif di port {$db_port} dan database '{$db_name}' sudah dibuat.</small></p>
    </div>");
}
`, port, dbName)
}

// GenerateHeaderPHP creates a responsive AdminLTE-style sidebar and header layout
func GenerateHeaderPHP(appTitle, tableName string, fn PHPPageFilenames, dash PHPDashboardSetting) string {
	brandHref := fn.Index
	if dash.Enabled {
		brandHref = "index.php"
	}

	var menuItems strings.Builder
	if dash.Enabled {
		menuItems.WriteString(`
            <div class="sidebar-heading">MENU UTAMA</div>
            <a class="sidebar-link <?= basename($_SERVER['PHP_SELF']) == 'index.php' ? 'active' : '' ?>" href="index.php">
                <i class="bi bi-speedometer2"></i>
                <span>Dashboard</span>
            </a>`)
	}

	cleanTableTitle := strings.Title(strings.ReplaceAll(tableName, "_", " "))
	menuItems.WriteString(fmt.Sprintf(`
            <div class="sidebar-heading">KELOLA DATA</div>
            <a class="sidebar-link <?= in_array(basename($_SERVER['PHP_SELF']), ['%s', '%s']) ? 'active' : '' ?>" href="%s">
                <i class="bi bi-table"></i>
                <span>Data %s</span>
            </a>
            <a class="sidebar-link <?= basename($_SERVER['PHP_SELF']) == '%s' ? 'active' : '' ?>" href="%s">
                <i class="bi bi-plus-circle"></i>
                <span>Tambah Data</span>
            </a>`, fn.Index, fn.Edit, fn.Index, cleanTableTitle, fn.Tambah, fn.Tambah))

	if dash.Enabled && len(dash.StatTables) > 1 {
		var otherLinks strings.Builder
		for _, st := range dash.StatTables {
			tbl := sanitizeFilename(st.TableName)
			if tbl != sanitizeFilename(tableName) {
				lbl := st.Label
				if lbl == "" {
					lbl = strings.Title(strings.ReplaceAll(tbl, "_", " "))
				}
				icon := st.Icon
				if icon == "" {
					icon = "bi-folder2"
				}
				otherLinks.WriteString(fmt.Sprintf(`
            <a class="sidebar-link <?= basename($_SERVER['PHP_SELF']) == '%s.php' ? 'active' : '' ?>" href="%s.php">
                <i class="bi %s"></i>
                <span>%s</span>
            </a>`, tbl, tbl, icon, lbl))
			}
		}
		if otherLinks.Len() > 0 {
			menuItems.WriteString(`
            <div class="sidebar-heading">MODUL LAINNYA</div>` + otherLinks.String())
		}
	}

	menuItems.WriteString(`
            <div class="sidebar-heading">TOOLS</div>
            <a class="sidebar-link" href="http://localhost:8080/phpmyadmin" target="_blank">
                <i class="bi bi-database-fill-gear"></i>
                <span>phpMyAdmin</span>
            </a>`)

	return fmt.Sprintf(`<?php
// Header & Sidebar template
?>
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title><?= isset($page_title) ? htmlspecialchars($page_title) . ' - ' : '' ?>%s</title>
    <!-- Bootstrap 5 CSS -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/css/bootstrap.min.css" rel="stylesheet">
    <!-- Bootstrap Icons -->
    <link href="https://cdn.jsdelivr.net/npm/bootstrap-icons@1.11.3/font/bootstrap-icons.min.css" rel="stylesheet">
    <style>
        :root {
            --sidebar-width: 250px;
            --sidebar-bg: #1e293b;
            --sidebar-hover: #334155;
            --sidebar-active: #0284c7;
            --sidebar-text: #cbd5e1;
            --sidebar-muted: #64748b;
            --topbar-height: 56px;
            --bg-body: #f8fafc;
        }
        * { box-sizing: border-box; }
        body {
            background-color: var(--bg-body);
            font-family: system-ui, -apple-system, "Segoe UI", Roboto, sans-serif;
            color: #334155;
            margin: 0;
            padding: 0;
            min-height: 100vh;
        }
        .app-wrapper {
            display: flex;
            min-height: 100vh;
            width: 100%%%%;
        }
        /* Sidebar */
        .app-sidebar {
            width: var(--sidebar-width);
            background-color: var(--sidebar-bg);
            color: var(--sidebar-text);
            flex-shrink: 0;
            display: flex;
            flex-direction: column;
            position: fixed;
            top: 0;
            bottom: 0;
            left: 0;
            z-index: 1030;
            transition: transform 0.25s ease-in-out;
            box-shadow: 2px 0 8px rgba(0, 0, 0, 0.08);
        }
        .sidebar-brand {
            height: var(--topbar-height);
            display: flex;
            align-items: center;
            padding: 0 18px;
            font-size: 1.05rem;
            font-weight: 700;
            color: #ffffff !important;
            text-decoration: none;
            border-bottom: 1px solid rgba(255, 255, 255, 0.08);
            background-color: #0f172a;
        }
        .sidebar-brand:hover {
            color: #38bdf8 !important;
        }
        .sidebar-nav {
            padding: 12px 10px;
            flex-grow: 1;
            overflow-y: auto;
        }
        .sidebar-heading {
            font-size: 0.68rem;
            text-transform: uppercase;
            letter-spacing: 0.08em;
            color: var(--sidebar-muted);
            padding: 12px 12px 4px;
            font-weight: 700;
        }
        .sidebar-link {
            display: flex;
            align-items: center;
            gap: 10px;
            padding: 8px 12px;
            color: var(--sidebar-text);
            text-decoration: none;
            border-radius: 6px;
            font-size: 0.88rem;
            font-weight: 500;
            margin-bottom: 3px;
            transition: all 0.15s ease;
        }
        .sidebar-link i {
            font-size: 1.05rem;
            width: 20px;
            text-align: center;
            color: #94a3b8;
        }
        .sidebar-link:hover {
            background-color: var(--sidebar-hover);
            color: #ffffff;
        }
        .sidebar-link:hover i {
            color: #ffffff;
        }
        .sidebar-link.active {
            background-color: var(--sidebar-active);
            color: #ffffff;
            font-weight: 600;
        }
        .sidebar-link.active i {
            color: #ffffff;
        }
        .sidebar-footer {
            padding: 12px 16px;
            border-top: 1px solid rgba(255, 255, 255, 0.08);
            font-size: 0.78rem;
            color: var(--sidebar-muted);
            background-color: #0f172a;
        }
        /* Main Layout */
        .app-main {
            margin-left: var(--sidebar-width);
            flex-grow: 1;
            display: flex;
            flex-direction: column;
            min-height: 100vh;
            width: calc(100%%%% - var(--sidebar-width));
        }
        .app-topbar {
            height: var(--topbar-height);
            background: #ffffff;
            border-bottom: 1px solid #e2e8f0;
            display: flex;
            align-items: center;
            justify-content: space-between;
            padding: 0 24px;
            position: sticky;
            top: 0;
            z-index: 1020;
        }
        .app-content {
            padding: 24px;
            flex-grow: 1;
        }
        .card-custom {
            border: 1px solid #e2e8f0;
            border-radius: 8px;
            box-shadow: 0 1px 3px rgba(0, 0, 0, 0.05);
            background: #ffffff;
        }
        .table th {
            background-color: #f8fafc;
            color: #475569;
            font-weight: 600;
            font-size: 0.78rem;
            letter-spacing: 0.04em;
        }
        .btn-primary {
            background-color: #0284c7;
            border-color: #0284c7;
        }
        .btn-primary:hover {
            background-color: #0369a1;
            border-color: #0369a1;
        }
        .footer-custom {
            background: #ffffff;
            border-top: 1px solid #e2e8f0;
            padding: 14px 24px;
            font-size: 0.8rem;
            color: #64748b;
        }
        @media (max-width: 991.98px) {
            .app-sidebar {
                transform: translateX(-100%%%%);
            }
            .app-sidebar.show {
                transform: translateX(0);
            }
            .app-main {
                margin-left: 0;
                width: 100%%%%;
            }
        }
    </style>
</head>
<body>
    <div class="app-wrapper">
        <!-- Sidebar Navigation -->
        <aside class="app-sidebar" id="appSidebar">
            <a href="%s" class="sidebar-brand text-decoration-none">
                <i class="bi bi-grid-1x2-fill text-info me-2 fs-5"></i>
                <span class="text-truncate">%s</span>
            </a>
            <div class="sidebar-nav">
%s
            </div>
            <div class="sidebar-footer d-flex align-items-center justify-content-between">
                <span class="badge bg-success bg-opacity-25 text-success border border-success border-opacity-50"><i class="bi bi-circle-fill me-1" style="font-size:6px;"></i> MariaDB</span>
                <span><?= date('d M Y') ?></span>
            </div>
        </aside>

        <!-- Main Content Wrapper -->
        <div class="app-main">
            <!-- Top Navigation Bar -->
            <header class="app-topbar">
                <div class="d-flex align-items-center gap-3">
                    <button class="btn btn-sm btn-outline-secondary d-lg-none" id="btnToggleSidebar" type="button">
                        <i class="bi bi-list fs-5"></i>
                    </button>
                    <span class="fw-bold text-dark fs-6"><?= isset($page_title) ? htmlspecialchars($page_title) : '%s' ?></span>
                </div>
                <div class="d-flex align-items-center gap-2">
                    <span class="badge bg-light text-dark border px-3 py-2">
                        <i class="bi bi-person-fill text-primary me-1"></i> Administrator
                    </span>
                </div>
            </header>

            <!-- Page Body Content -->
            <main class="app-content">
`, appTitle, brandHref, appTitle, menuItems.String(), appTitle)
}

// GenerateFooterPHP creates the footer and layout closure
func GenerateFooterPHP() string {
	return `            </main>
            <footer class="footer-custom d-flex justify-content-between align-items-center">
                <span>&copy; <?= date('Y') ?> &bull; Panel Administrasi Data</span>
                <span class="text-muted">MyLokalWebserver</span>
            </footer>
        </div> <!-- /.app-main -->
    </div> <!-- /.app-wrapper -->

    <!-- Bootstrap 5 JS Bundle -->
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
    <script>
        const btnToggle = document.getElementById('btnToggleSidebar');
        const sidebar = document.getElementById('appSidebar');
        if (btnToggle && sidebar) {
            btnToggle.addEventListener('click', () => {
                sidebar.classList.toggle('show');
            });
        }
    </script>
</body>
</html>
`
}

// GenerateIndexPHP creates index/table.php with search, pagination, FK JOINs, and data table
func GenerateIndexPHP(req PHPGenerateRequest, fn PHPPageFilenames) string {
	var sb strings.Builder

	primaryCol := req.PrimaryCol
	if primaryCol == "" {
		primaryCol = "id"
	}

	// Build SELECT query and JOIN clauses
	var selectCols []string
	var joinClauses []string
	selectCols = append(selectCols, fmt.Sprintf("`%s`.*", req.TableName))

	for _, rel := range req.Relations {
		alias := rel.DisplayAlias
		if alias == "" {
			alias = fmt.Sprintf("%s_%s", rel.ReferencedTable, rel.DisplayColumn)
		}
		selectCols = append(selectCols, fmt.Sprintf("`%s`.`%s` AS `%s`", rel.ReferencedTable, rel.DisplayColumn, alias))
		joinClauses = append(joinClauses, fmt.Sprintf("LEFT JOIN `%s` ON `%s`.`%s` = `%s`.`%s`",
			rel.ReferencedTable, req.TableName, rel.ColumnName, rel.ReferencedTable, rel.ReferencedPK))
	}

	selectSql := strings.Join(selectCols, ", ")
	joinSql := strings.Join(joinClauses, " ")

	// Searchable columns
	var searchCols []string
	for _, c := range req.Columns {
		if c.ShowInList && !strings.Contains(strings.ToLower(c.Type), "int") && !strings.Contains(strings.ToLower(c.Type), "decimal") {
			searchCols = append(searchCols, fmt.Sprintf("`%s`.`%s` LIKE :search", req.TableName, c.Field))
		}
	}
	if len(searchCols) == 0 {
		searchCols = append(searchCols, fmt.Sprintf("`%s`.`%s` LIKE :search", req.TableName, primaryCol))
	}
	searchWhere := strings.Join(searchCols, " OR ")

	sb.WriteString(fmt.Sprintf(`<?php
require_once 'koneksi.php';
$page_title = 'Daftar Data %s';

// Pengaturan Pencarian & Paginasi
$search = trim($_GET['q'] ?? '');
$page = max(1, (int)($_GET['page'] ?? 1));
$limit = 10;
$offset = ($page - 1) * $limit;

// Menghitung Total Data
$countSql = "SELECT COUNT(*) FROM ` + "`%s`" + ` %s";
$params = [];

if ($search !== '') {
    $countSql .= " WHERE (%s)";
    $params[':search'] = "%%{$search}%%";
}

$stmtCount = $pdo->prepare($countSql);
$stmtCount->execute($params);
$totalRows = (int)$stmtCount->fetchColumn();
$totalPages = ceil($totalRows / $limit);

// Query Mengambil Data
$sql = "SELECT %s FROM ` + "`%s`" + ` %s";
if ($search !== '') {
    $sql .= " WHERE (%s)";
}
$sql .= " ORDER BY ` + "`%s`.`%s`" + ` DESC LIMIT {$limit} OFFSET {$offset}";

$stmt = $pdo->prepare($sql);
$stmt->execute($params);
$rows = $stmt->fetchAll();

// Notifikasi pesan
$pesan = $_GET['pesan'] ?? '';
$tipePesan = $_GET['tipe'] ?? 'success';

require_once 'header.php';
?>

<div class="card card-custom">
    <div class="card-body p-4">
        <!-- Header Aksi & Pencarian -->
        <div class="d-flex flex-column flex-md-row justify-content-between align-items-md-center gap-3 mb-4">
            <div>
                <h4 class="card-title fw-bold mb-1"><i class="bi bi-journal-text me-2 text-primary"></i>Tabel Data %s</h4>
                <p class="text-muted small mb-0">Total: <strong><?= number_format($totalRows) ?></strong> data ditemukan</p>
            </div>
            <div class="d-flex gap-2">
                <a href="%s" class="btn btn-primary">
                    <i class="bi bi-plus-lg me-1"></i> Tambah Data Baru
                </a>
            </div>
        </div>

        <?php if ($pesan): ?>
            <div class="alert alert-<?= htmlspecialchars($tipePesan) ?> alert-dismissible fade show" role="alert">
                <i class="bi bi-check-circle-fill me-2"></i><?= htmlspecialchars($pesan) ?>
                <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
            </div>
        <?php endif; ?>

        <!-- Form Pencarian -->
        <form method="GET" action="%s" class="row g-2 mb-3">
            <div class="col-md-5 col-lg-4">
                <div class="input-group">
                    <span class="input-group-text bg-light"><i class="bi bi-search"></i></span>
                    <input type="text" name="q" class="form-control" placeholder="Cari data..." value="<?= htmlspecialchars($search) ?>">
                    <?php if ($search !== ''): ?>
                        <a href="%s" class="btn btn-outline-secondary" title="Reset Pencarian"><i class="bi bi-x-lg"></i></a>
                    <?php endif; ?>
                    <button type="submit" class="btn btn-outline-primary">Cari</button>
                </div>
            </div>
        </form>

        <!-- Tabel Data -->
        <div class="table-responsive">
            <table class="table table-hover align-middle">
                <thead>
                    <tr>
                        <th style="width: 50px;" class="text-center">No</th>
`, req.TableName, req.TableName, joinSql, searchWhere, selectSql, req.TableName, joinSql, searchWhere, req.TableName, primaryCol, req.TableName, fn.Tambah, fn.Index, fn.Index))

	// Table headers
	for _, col := range req.Columns {
		if !col.ShowInList {
			continue
		}
		label := col.Label
		if label == "" {
			label = col.Field
		}

		sb.WriteString(fmt.Sprintf("                        <th>%s</th>\n", htmlspecialcharsGo(label)))
	}
	sb.WriteString(`                        <th style="width: 140px;" class="text-center">Aksi</th>
                    </tr>
                </thead>
                <tbody>
                    <?php if (empty($rows)): ?>
                        <tr>
                            <td colspan="100" class="text-center py-4 text-muted">
                                <i class="bi bi-inbox fs-2 d-block mb-2 text-secondary"></i>
                                Tidak ada data yang ditemukan.
                            </td>
                        </tr>
                    <?php else: ?>
                        <?php $no = $offset + 1; foreach ($rows as $row): ?>
                            <tr>
                                <td class="text-center text-muted fw-bold"><?= $no++ ?></td>
`)

	// Table columns output
	for _, col := range req.Columns {
		if !col.ShowInList {
			continue
		}

		// Check if this column is a foreign key
		var rel *PHPRelationSetting
		for i := range req.Relations {
			if req.Relations[i].ColumnName == col.Field {
				rel = &req.Relations[i]
				break
			}
		}

		colTypeLower := strings.ToLower(col.Type)

		if rel != nil {
			alias := rel.DisplayAlias
			if alias == "" {
				alias = fmt.Sprintf("%s_%s", rel.ReferencedTable, rel.DisplayColumn)
			}
			sb.WriteString(fmt.Sprintf(`                                <td>
                                    <span class="badge bg-light text-dark border">
                                        <i class="bi bi-link-45deg text-primary me-1"></i><?= htmlspecialchars($row['%s'] ?? $row['%s'] ?? '-') ?>
                                    </span>
                                </td>
`, alias, col.Field))
		} else if strings.HasPrefix(colTypeLower, "enum") {
			sb.WriteString(fmt.Sprintf(`                                <td>
                                    <span class="badge bg-info-subtle text-info-emphasis px-2 py-1">
                                        <?= htmlspecialchars($row['%s'] ?? '-') ?>
                                    </span>
                                </td>
`, col.Field))
		} else if strings.Contains(colTypeLower, "decimal") || strings.Contains(colTypeLower, "float") || strings.Contains(colTypeLower, "double") {
			sb.WriteString(fmt.Sprintf(`                                <td>Rp <?= number_format((float)($row['%s'] ?? 0), 2, ',', '.') ?></td>
`, col.Field))
		} else if strings.HasPrefix(colTypeLower, "date") || strings.HasPrefix(colTypeLower, "datetime") {
			sb.WriteString(fmt.Sprintf(`                                <td><?= !empty($row['%s']) ? htmlspecialchars($row['%s']) : '-' ?></td>
`, col.Field, col.Field))
		} else {
			sb.WriteString(fmt.Sprintf(`                                <td><?= htmlspecialchars($row['%s'] ?? '-') ?></td>
`, col.Field))
		}
	}

	sb.WriteString(fmt.Sprintf(`                                <td class="text-center">
                                    <div class="btn-group btn-group-sm">
                                        <a href="%s?%s=<?= urlencode($row['%s']) ?>" class="btn btn-outline-primary" title="Ubah Data">
                                            <i class="bi bi-pencil-square"></i>
                                        </a>
                                        <a href="%s?%s=<?= urlencode($row['%s']) ?>" class="btn btn-outline-danger" onclick="return confirm('Apakah Anda yakin ingin menghapus data ini?');" title="Hapus Data">
                                            <i class="bi bi-trash"></i>
                                        </a>
                                    </div>
                                </td>
                            </tr>
                        <?php endforeach; ?>
                    <?php endif; ?>
                </tbody>
            </table>
        </div>

        <!-- Paginasi -->
        <?php if ($totalPages > 1): ?>
            <div class="d-flex justify-content-between align-items-center mt-3">
                <small class="text-muted">Halaman <?= $page ?> dari <?= $totalPages ?></small>
                <nav>
                    <ul class="pagination pagination-sm mb-0">
                        <li class="page-item <?= ($page <= 1) ? 'disabled' : '' ?>">
                            <a class="page-link" href="%s?page=<?= $page - 1 ?>&q=<?= urlencode($search) ?>">&laquo;</a>
                        </li>
                        <?php for ($i = 1; $i <= $totalPages; $i++): ?>
                            <li class="page-item <?= ($page == $i) ? 'active' : '' ?>">
                                <a class="page-link" href="%s?page=<?= $i ?>&q=<?= urlencode($search) ?>"><?= $i ?></a>
                            </li>
                        <?php endfor; ?>
                        <li class="page-item <?= ($page >= $totalPages) ? 'disabled' : '' ?>">
                            <a class="page-link" href="%s?page=<?= $page + 1 ?>&q=<?= urlencode($search) ?>">&raquo;</a>
                        </li>
                    </ul>
                </nav>
            </div>
        <?php endif; ?>
    </div>
</div>

<?php
require_once 'footer.php';
?>
`, fn.Edit, primaryCol, primaryCol, fn.Hapus, primaryCol, primaryCol, fn.Index, fn.Index, fn.Index))

	return sb.String()
}

// GenerateTambahPHP creates tambah.php for creating new records
func GenerateTambahPHP(req PHPGenerateRequest, fn PHPPageFilenames) string {
	var sb strings.Builder

	primaryCol := req.PrimaryCol
	if primaryCol == "" {
		primaryCol = "id"
	}

	sb.WriteString(fmt.Sprintf(`<?php
require_once 'koneksi.php';
$page_title = 'Tambah Data %s';

$errors = [];
$values = [];

`, req.TableName))

	// Fetch relation choices for foreign keys
	if len(req.Relations) > 0 {
		sb.WriteString("// Ambil data referensi untuk dropdown relasi Foreign Key\n")
		for _, rel := range req.Relations {
			sb.WriteString(fmt.Sprintf(`$options_%s = $pdo->query("SELECT `+"`%s`"+`, `+"`%s`"+` FROM `+"`%s`"+` ORDER BY `+"`%s`"+` ASC")->fetchAll();
`, rel.ColumnName, rel.ReferencedPK, rel.DisplayColumn, rel.ReferencedTable, rel.DisplayColumn))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`if ($_SERVER['REQUEST_METHOD'] === 'POST') {
`)

	var formCols []PHPColumnSetting
	for _, col := range req.Columns {
		if col.Field == primaryCol && (strings.Contains(strings.ToLower(col.Type), "int") || strings.Contains(strings.ToLower(col.Type), "auto")) {
			continue // skip auto increment PK
		}
		if col.ShowInForm {
			formCols = append(formCols, col)
		}
	}

	for _, col := range formCols {
		sb.WriteString(fmt.Sprintf("    $values['%s'] = trim($_POST['%s'] ?? '');\n", col.Field, col.Field))
		if col.Required {
			sb.WriteString(fmt.Sprintf(`    if ($values['%s'] === '') {
        $errors['%s'] = '%s wajib diisi.';
    }
`, col.Field, col.Field, col.Label))
		}
	}

	sb.WriteString(`
    if (empty($errors)) {
        try {
`)

	var insertFields []string
	var insertPlaceholders []string
	for _, col := range formCols {
		insertFields = append(insertFields, fmt.Sprintf("`%s`", col.Field))
		insertPlaceholders = append(insertPlaceholders, fmt.Sprintf(":%s", col.Field))
	}

	insertSql := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES (%s)",
		req.TableName, strings.Join(insertFields, ", "), strings.Join(insertPlaceholders, ", "))

	sb.WriteString(fmt.Sprintf(`            $sql = "%s";
            $stmt = $pdo->prepare($sql);
            $stmt->execute([
`, insertSql))

	for _, col := range formCols {
		if !col.Required {
			sb.WriteString(fmt.Sprintf("                ':%s' => $values['%s'] !== '' ? $values['%s'] : null,\n", col.Field, col.Field, col.Field))
		} else {
			sb.WriteString(fmt.Sprintf("                ':%s' => $values['%s'],\n", col.Field, col.Field))
		}
	}

	sb.WriteString(fmt.Sprintf(`            ]);

            header("Location: %s?pesan=Data+berhasil+ditambahkan!&tipe=success");
            exit;
        } catch (PDOException $e) {
            $errors['db'] = "Gagal menyimpan data: " . $e->getMessage();
        }
    }
}

require_once 'header.php';
?>

<div class="row justify-content-center">
    <div class="col-lg-8">
        <div class="card card-custom">
            <div class="card-header bg-white border-bottom p-4">
                <div class="d-flex justify-content-between align-items-center">
                    <h5 class="fw-bold mb-0"><i class="bi bi-plus-circle text-primary me-2"></i>Tambah Data %s</h5>
                    <a href="%s" class="btn btn-outline-secondary btn-sm"><i class="bi bi-arrow-left me-1"></i>Kembali</a>
                </div>
            </div>
            <div class="card-body p-4">
                <?php if (!empty($errors['db'])): ?>
                    <div class="alert alert-danger"><i class="bi bi-exclamation-triangle-fill me-2"></i><?= htmlspecialchars($errors['db']) ?></div>
                <?php endif; ?>

                <form method="POST" action="%s">
`, fn.Index, req.TableName, fn.Index, fn.Tambah))

	// Generate form fields
	for _, col := range formCols {
		label := col.Label
		if label == "" {
			label = col.Field
		}

		// Check if relation
		var rel *PHPRelationSetting
		for i := range req.Relations {
			if req.Relations[i].ColumnName == col.Field {
				rel = &req.Relations[i]
				break
			}
		}

		colTypeLower := strings.ToLower(col.Type)

		sb.WriteString(fmt.Sprintf(`                    <div class="mb-3">
                        <label for="%s" class="form-label fw-semibold">%s %s</label>
`, col.Field, htmlspecialcharsGo(label), requiredStar(col.Required)))

		if rel != nil {
			sb.WriteString(fmt.Sprintf(`                        <select class="form-select <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s">
                            <option value="">-- Pilih %s --</option>
                            <?php foreach ($options_%s as $opt): ?>
                                <option value="<?= htmlspecialchars($opt['%s']) ?>" <?= (($values['%s'] ?? '') == $opt['%s']) ? 'selected' : '' ?>>
                                    <?= htmlspecialchars($opt['%s']) ?>
                                </option>
                            <?php endforeach; ?>
                        </select>
`, col.Field, col.Field, col.Field, htmlspecialcharsGo(label), rel.ColumnName, rel.ReferencedPK, col.Field, rel.ReferencedPK, rel.DisplayColumn))
		} else if len(col.EnumOptions) > 0 || strings.HasPrefix(colTypeLower, "enum") {
			opts := col.EnumOptions
			if len(opts) == 0 {
				opts = extractEnumValues(col.Type)
			}
			sb.WriteString(fmt.Sprintf(`                        <select class="form-select <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s">
                            <option value="">-- Pilih %s --</option>
`, col.Field, col.Field, col.Field, htmlspecialcharsGo(label)))
			for _, o := range opts {
				sb.WriteString(fmt.Sprintf(`                            <option value="%s" <?= (($values['%s'] ?? '') === '%s') ? 'selected' : '' ?>>%s</option>
`, htmlspecialcharsGo(o), col.Field, htmlspecialcharsGo(o), htmlspecialcharsGo(o)))
			}
			sb.WriteString(`                        </select>
`)
		} else if strings.Contains(colTypeLower, "text") {
			sb.WriteString(fmt.Sprintf(`                        <textarea class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" rows="3" placeholder="Masukkan %s"><?= htmlspecialchars($values['%s'] ?? '') ?></textarea>
`, col.Field, col.Field, col.Field, htmlspecialcharsGo(label), col.Field))
		} else if strings.Contains(colTypeLower, "int") || strings.Contains(colTypeLower, "decimal") || strings.Contains(colTypeLower, "float") {
			step := "1"
			if strings.Contains(colTypeLower, "decimal") || strings.Contains(colTypeLower, "float") {
				step = "0.01"
			}
			sb.WriteString(fmt.Sprintf(`                        <input type="number" step="%s" class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" value="<?= htmlspecialchars($values['%s'] ?? '') ?>" placeholder="0">
`, step, col.Field, col.Field, col.Field, col.Field))
		} else if strings.HasPrefix(colTypeLower, "date") && !strings.HasPrefix(colTypeLower, "datetime") {
			sb.WriteString(fmt.Sprintf(`                        <input type="date" class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" value="<?= htmlspecialchars($values['%s'] ?? '') ?>">
`, col.Field, col.Field, col.Field, col.Field))
		} else if strings.HasPrefix(colTypeLower, "datetime") {
			sb.WriteString(fmt.Sprintf(`                        <input type="datetime-local" class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" value="<?= htmlspecialchars($values['%s'] ?? '') ?>">
`, col.Field, col.Field, col.Field, col.Field))
		} else {
			sb.WriteString(fmt.Sprintf(`                        <input type="text" class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" value="<?= htmlspecialchars($values['%s'] ?? '') ?>" placeholder="Masukkan %s">
`, col.Field, col.Field, col.Field, htmlspecialcharsGo(label)))
		}

		sb.WriteString(fmt.Sprintf(`                        <?php if (isset($errors['%s'])): ?>
                            <div class="invalid-feedback"><?= $errors['%s'] ?></div>
                        <?php endif; ?>
                    </div>
`, col.Field, col.Field))
	}

	sb.WriteString(fmt.Sprintf(`                    <div class="d-flex justify-content-end gap-2 mt-4">
                        <a href="%s" class="btn btn-light border">Batal</a>
                        <button type="submit" class="btn btn-primary"><i class="bi bi-save me-1"></i>Simpan Data</button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>

<?php
require_once 'footer.php';
?>
`, fn.Index))

	return sb.String()
}

// GenerateEditPHP creates edit.php for updating existing records
func GenerateEditPHP(req PHPGenerateRequest, fn PHPPageFilenames) string {
	var sb strings.Builder

	primaryCol := req.PrimaryCol
	if primaryCol == "" {
		primaryCol = "id"
	}

	sb.WriteString(fmt.Sprintf(`<?php
require_once 'koneksi.php';
$page_title = 'Ubah Data %s';

$id = $_GET['%s'] ?? '';
if ($id === '') {
    header("Location: %s?pesan=ID+tidak+valid&tipe=danger");
    exit;
}

// Ambil data lama
$stmtOld = $pdo->prepare("SELECT * FROM `+"`%s`"+` WHERE `+"`%s`"+` = :id");
$stmtOld->execute([':id' => $id]);
$row = $stmtOld->fetch();

if (!$row) {
    header("Location: %s?pesan=Data+tidak+ditemukan&tipe=danger");
    exit;
}

$errors = [];
$values = $row;

`, req.TableName, primaryCol, fn.Index, req.TableName, primaryCol, fn.Index))

	// Fetch relation choices for foreign keys
	if len(req.Relations) > 0 {
		sb.WriteString("// Ambil data referensi untuk dropdown relasi Foreign Key\n")
		for _, rel := range req.Relations {
			sb.WriteString(fmt.Sprintf(`$options_%s = $pdo->query("SELECT `+"`%s`"+`, `+"`%s`"+` FROM `+"`%s`"+` ORDER BY `+"`%s`"+` ASC")->fetchAll();
`, rel.ColumnName, rel.ReferencedPK, rel.DisplayColumn, rel.ReferencedTable, rel.DisplayColumn))
		}
		sb.WriteString("\n")
	}

	sb.WriteString(`if ($_SERVER['REQUEST_METHOD'] === 'POST') {
`)

	var formCols []PHPColumnSetting
	for _, col := range req.Columns {
		if col.Field == primaryCol {
			continue // primary key is in WHERE clause
		}
		if col.ShowInForm {
			formCols = append(formCols, col)
		}
	}

	for _, col := range formCols {
		sb.WriteString(fmt.Sprintf("    $values['%s'] = trim($_POST['%s'] ?? '');\n", col.Field, col.Field))
		if col.Required {
			sb.WriteString(fmt.Sprintf(`    if ($values['%s'] === '') {
        $errors['%s'] = '%s wajib diisi.';
    }
`, col.Field, col.Field, col.Label))
		}
	}

	sb.WriteString(`
    if (empty($errors)) {
        try {
`)

	var updateSets []string
	for _, col := range formCols {
		updateSets = append(updateSets, fmt.Sprintf("`%s` = :%s", col.Field, col.Field))
	}

	updateSql := fmt.Sprintf("UPDATE `%s` SET %s WHERE `%s` = :primary_id",
		req.TableName, strings.Join(updateSets, ", "), primaryCol)

	sb.WriteString(fmt.Sprintf(`            $sql = "%s";
            $stmt = $pdo->prepare($sql);
            $stmt->execute([
`, updateSql))

	for _, col := range formCols {
		if !col.Required {
			sb.WriteString(fmt.Sprintf("                ':%s' => $values['%s'] !== '' ? $values['%s'] : null,\n", col.Field, col.Field, col.Field))
		} else {
			sb.WriteString(fmt.Sprintf("                ':%s' => $values['%s'],\n", col.Field, col.Field))
		}
	}
	sb.WriteString(fmt.Sprintf("                ':primary_id' => $id,\n"))

	sb.WriteString(fmt.Sprintf(`            ]);

            header("Location: %s?pesan=Data+berhasil+diperbarui!&tipe=success");
            exit;
        } catch (PDOException $e) {
            $errors['db'] = "Gagal memperbarui data: " . $e->getMessage();
        }
    }
}

require_once 'header.php';
?>

<div class="row justify-content-center">
    <div class="col-lg-8">
        <div class="card card-custom">
            <div class="card-header bg-white border-bottom p-4">
                <div class="d-flex justify-content-between align-items-center">
                    <h5 class="fw-bold mb-0"><i class="bi bi-pencil-square text-primary me-2"></i>Ubah Data %s</h5>
                    <a href="%s" class="btn btn-outline-secondary btn-sm"><i class="bi bi-arrow-left me-1"></i>Kembali</a>
                </div>
            </div>
            <div class="card-body p-4">
                <?php if (!empty($errors['db'])): ?>
                    <div class="alert alert-danger"><i class="bi bi-exclamation-triangle-fill me-2"></i><?= htmlspecialchars($errors['db']) ?></div>
                <?php endif; ?>

                <form method="POST" action="%s?%s=<?= urlencode($id) ?>">
`, fn.Index, req.TableName, fn.Index, fn.Edit, primaryCol))

	// Generate form fields
	for _, col := range formCols {
		label := col.Label
		if label == "" {
			label = col.Field
		}

		// Check if relation
		var rel *PHPRelationSetting
		for i := range req.Relations {
			if req.Relations[i].ColumnName == col.Field {
				rel = &req.Relations[i]
				break
			}
		}

		colTypeLower := strings.ToLower(col.Type)

		sb.WriteString(fmt.Sprintf(`                    <div class="mb-3">
                        <label for="%s" class="form-label fw-semibold">%s %s</label>
`, col.Field, htmlspecialcharsGo(label), requiredStar(col.Required)))

		if rel != nil {
			sb.WriteString(fmt.Sprintf(`                        <select class="form-select <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s">
                            <option value="">-- Pilih %s --</option>
                            <?php foreach ($options_%s as $opt): ?>
                                <option value="<?= htmlspecialchars($opt['%s']) ?>" <?= (($values['%s'] ?? '') == $opt['%s']) ? 'selected' : '' ?>>
                                    <?= htmlspecialchars($opt['%s']) ?>
                                </option>
                            <?php endforeach; ?>
                        </select>
`, col.Field, col.Field, col.Field, htmlspecialcharsGo(label), rel.ColumnName, rel.ReferencedPK, col.Field, rel.ReferencedPK, rel.DisplayColumn))
		} else if len(col.EnumOptions) > 0 || strings.HasPrefix(colTypeLower, "enum") {
			opts := col.EnumOptions
			if len(opts) == 0 {
				opts = extractEnumValues(col.Type)
			}
			sb.WriteString(fmt.Sprintf(`                        <select class="form-select <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s">
                            <option value="">-- Pilih %s --</option>
`, col.Field, col.Field, col.Field, htmlspecialcharsGo(label)))
			for _, o := range opts {
				sb.WriteString(fmt.Sprintf(`                            <option value="%s" <?= (($values['%s'] ?? '') === '%s') ? 'selected' : '' ?>>%s</option>
`, htmlspecialcharsGo(o), col.Field, htmlspecialcharsGo(o), htmlspecialcharsGo(o)))
			}
			sb.WriteString(`                        </select>
`)
		} else if strings.Contains(colTypeLower, "text") {
			sb.WriteString(fmt.Sprintf(`                        <textarea class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" rows="3"><?= htmlspecialchars($values['%s'] ?? '') ?></textarea>
`, col.Field, col.Field, col.Field, col.Field))
		} else if strings.Contains(colTypeLower, "int") || strings.Contains(colTypeLower, "decimal") || strings.Contains(colTypeLower, "float") {
			step := "1"
			if strings.Contains(colTypeLower, "decimal") || strings.Contains(colTypeLower, "float") {
				step = "0.01"
			}
			sb.WriteString(fmt.Sprintf(`                        <input type="number" step="%s" class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" value="<?= htmlspecialchars($values['%s'] ?? '') ?>">
`, step, col.Field, col.Field, col.Field, col.Field))
		} else if strings.HasPrefix(colTypeLower, "date") && !strings.HasPrefix(colTypeLower, "datetime") {
			sb.WriteString(fmt.Sprintf(`                        <input type="date" class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" value="<?= htmlspecialchars($values['%s'] ?? '') ?>">
`, col.Field, col.Field, col.Field, col.Field))
		} else if strings.HasPrefix(colTypeLower, "datetime") {
			sb.WriteString(fmt.Sprintf(`                        <input type="datetime-local" class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" value="<?= htmlspecialchars($values['%s'] ?? '') ?>">
`, col.Field, col.Field, col.Field, col.Field))
		} else {
			sb.WriteString(fmt.Sprintf(`                        <input type="text" class="form-control <?= isset($errors['%s']) ? 'is-invalid' : '' ?>" id="%s" name="%s" value="<?= htmlspecialchars($values['%s'] ?? '') ?>">
`, col.Field, col.Field, col.Field, col.Field))
		}

		sb.WriteString(fmt.Sprintf(`                        <?php if (isset($errors['%s'])): ?>
                            <div class="invalid-feedback"><?= $errors['%s'] ?></div>
                        <?php endif; ?>
                    </div>
`, col.Field, col.Field))
	}

	sb.WriteString(fmt.Sprintf(`                    <div class="d-flex justify-content-end gap-2 mt-4">
                        <a href="%s" class="btn btn-light border">Batal</a>
                        <button type="submit" class="btn btn-primary"><i class="bi bi-save me-1"></i>Simpan Perubahan</button>
                    </div>
                </form>
            </div>
        </div>
    </div>
</div>

<?php
require_once 'footer.php';
?>
`, fn.Index))

	return sb.String()
}

// GenerateHapusPHP creates hapus.php for deleting records safely
func GenerateHapusPHP(req PHPGenerateRequest, fn PHPPageFilenames) string {
	primaryCol := req.PrimaryCol
	if primaryCol == "" {
		primaryCol = "id"
	}

	return fmt.Sprintf(`<?php
require_once 'koneksi.php';

$id = $_GET['%s'] ?? '';
if ($id === '') {
    header("Location: %s?pesan=ID+tidak+valid&tipe=danger");
    exit;
}

try {
    $stmt = $pdo->prepare("DELETE FROM `+"`%s`"+` WHERE `+"`%s`"+` = :id");
    $stmt->execute([':id' => $id]);

    header("Location: %s?pesan=Data+berhasil+dihapus!&tipe=success");
    exit;
} catch (PDOException $e) {
    header("Location: %s?pesan=Gagal+menghapus+data:+relasi+data+masih+terikat+di+tabel+lain&tipe=danger");
    exit;
}
`, primaryCol, fn.Index, req.TableName, primaryCol, fn.Index, fn.Index)
}

// GenerateDashboardPHP creates a modern, clean Admin Dashboard (index.php) homepage
func GenerateDashboardPHP(req PHPGenerateRequest, mariaPort int) string {
	dash := req.Dashboard
	dashTitle := dash.DashboardTitle
	if dashTitle == "" {
		dashTitle = fmt.Sprintf("Dashboard %s", req.AppTitle)
	}
	welcomeMsg := dash.WelcomeMsg
	if welcomeMsg == "" {
		welcomeMsg = "Ringkasan statistik data dan pemantauan sistem secara langsung."
	}

	recentTable := dash.RecentTable
	if recentTable == "" {
		recentTable = req.TableName
	}

	var statQueries strings.Builder
	var statCards strings.Builder

	// Helper for contextual icons & badge colors
	getIconForTable := func(t string) (string, string) {
		lt := strings.ToLower(t)
		if strings.Contains(lt, "obat") || strings.Contains(lt, "resep") || strings.Contains(lt, "medis") {
			return "bi-capsule", "success"
		}
		if strings.Contains(lt, "pasien") || strings.Contains(lt, "dokter") || strings.Contains(lt, "user") || strings.Contains(lt, "pengguna") || strings.Contains(lt, "karyawan") {
			return "bi-people-fill", "primary"
		}
		if strings.Contains(lt, "kategori") || strings.Contains(lt, "jenis") || strings.Contains(lt, "tag") {
			return "bi-tags-fill", "info"
		}
		if strings.Contains(lt, "transaksi") || strings.Contains(lt, "order") || strings.Contains(lt, "jual") || strings.Contains(lt, "beli") {
			return "bi-cart-check-fill", "warning"
		}
		if strings.Contains(lt, "produk") || strings.Contains(lt, "barang") || strings.Contains(lt, "item") || strings.Contains(lt, "stok") {
			return "bi-box-seam-fill", "primary"
		}
		return "bi-database-fill", "secondary"
	}

	statTables := dash.StatTables
	if len(statTables) == 0 {
		statTables = []PHPDashboardStatTable{
			{
				TableName: req.TableName,
				Label:     fmt.Sprintf("Total %s", strings.Title(strings.ReplaceAll(req.TableName, "_", " "))),
				Icon:      "bi-table",
				Color:     "primary",
			},
		}
	}

	for i, st := range statTables {
		tbl := sanitizeFilename(st.TableName)
		varName := fmt.Sprintf("count_%s_%d", tbl, i)
		statQueries.WriteString(fmt.Sprintf(`
$%s = 0;
try {
    $stmt = $pdo->query("SELECT COUNT(*) FROM `+"`%s`"+`");
    $%s = (int)$stmt->fetchColumn();
} catch (Exception $e) {}
`, varName, tbl, varName))

		icon := st.Icon
		color := st.Color
		if icon == "" || color == "" {
			autoIcon, autoColor := getIconForTable(tbl)
			if icon == "" {
				icon = autoIcon
			}
			if color == "" {
				color = autoColor
			}
		}

		label := st.Label
		if label == "" {
			label = fmt.Sprintf("Total %s", strings.Title(strings.ReplaceAll(tbl, "_", " ")))
		}

		targetLink := fmt.Sprintf("%s.php", tbl)

		statCards.WriteString(fmt.Sprintf(`
        <div class="col-md-6 col-lg-4 mb-4">
            <div class="card card-custom h-100 shadow-sm border-0" style="border-left: 4px solid var(--bs-%s, #0284c7) !important;">
                <div class="card-body p-3">
                    <div class="d-flex align-items-center justify-content-between mb-2">
                        <span class="text-muted text-uppercase fw-semibold small">%s</span>
                        <div class="rounded p-2 bg-%s bg-opacity-10 text-%s">
                            <i class="bi %s fs-5"></i>
                        </div>
                    </div>
                    <div class="d-flex align-items-baseline mb-2">
                        <h3 class="fw-bold mb-0 text-dark"><?= number_format($%s) ?></h3>
                        <span class="text-muted ms-2 small">data</span>
                    </div>
                    <a href="%s" class="text-decoration-none small fw-medium text-%s">
                        Kelola data <i class="bi bi-arrow-right ms-1"></i>
                    </a>
                </div>
            </div>
        </div>`, color, label, color, color, icon, varName, targetLink, color))
	}

	// Recent data table section
	var recentDataBlock string
	if dash.ShowRecent {
		recentTbl := sanitizeFilename(recentTable)
		recentDataBlock = fmt.Sprintf(`
    <!-- Recent Data Table Card -->
    <div class="card card-custom shadow-sm mb-4 border-0">
        <div class="card-header bg-white py-3 border-bottom d-flex justify-content-between align-items-center">
            <div class="d-flex align-items-center gap-2">
                <i class="bi bi-clock-history text-primary fs-5"></i>
                <h5 class="mb-0 fw-bold text-dark">5 Data Terbaru: %s</h5>
            </div>
            <a href="%s.php" class="btn btn-sm btn-outline-primary">
                Lihat Semua Data <i class="bi bi-arrow-right ms-1"></i>
            </a>
        </div>
        <div class="card-body p-0">
            <?php
            $recentRows = [];
            try {
                $stmtRecent = $pdo->query("SELECT * FROM `+"`%s`"+` ORDER BY 1 DESC LIMIT 5");
                $recentRows = $stmtRecent->fetchAll();
            } catch (Exception $e) {}
            ?>
            <?php if (empty($recentRows)): ?>
                <div class="text-center py-5 text-muted">
                    <i class="bi bi-inbox fs-1 d-block mb-2 text-secondary"></i>
                    <p class="mb-0">Belum ada data di tabel <code>%s</code>.</p>
                </div>
            <?php else: ?>
                <div class="table-responsive">
                    <table class="table table-hover align-middle mb-0">
                        <thead class="table-light">
                            <tr>
                                <?php 
                                $firstRow = reset($recentRows);
                                $displayCols = array_slice(array_keys($firstRow), 0, 5);
                                foreach ($displayCols as $col): ?>
                                    <th class="px-3 py-2 text-uppercase small"><?= htmlspecialchars(str_replace('_', ' ', $col)) ?></th>
                                <?php endforeach; ?>
                            </tr>
                        </thead>
                        <tbody>
                            <?php foreach ($recentRows as $r): ?>
                                <tr>
                                    <?php foreach ($displayCols as $idx => $col): ?>
                                        <td class="px-3 py-2">
                                            <?php if ($idx === 0): ?>
                                                <span class="badge bg-secondary bg-opacity-10 text-dark font-monospace">#<?= htmlspecialchars((string)($r[$col] ?? '')) ?></span>
                                            <?php else: ?>
                                                <?= htmlspecialchars((string)($r[$col] ?? '-')) ?>
                                            <?php endif; ?>
                                        </td>
                                    <?php endforeach; ?>
                                </tr>
                            <?php endforeach; ?>
                        </tbody>
                    </table>
                </div>
            <?php endif; ?>
        </div>
    </div>`, strings.Title(strings.ReplaceAll(recentTbl, "_", " ")), recentTbl, recentTbl, recentTbl)
	}

	return fmt.Sprintf(`<?php
require_once 'koneksi.php';
$page_title = '%s';

// Query Statistik
%s

require_once 'header.php';
?>

<!-- Header Halaman -->
<div class="d-flex flex-column flex-md-row justify-content-between align-items-md-center mb-4 pb-2 border-bottom">
    <div>
        <h4 class="fw-bold text-dark mb-1">%s</h4>
        <p class="text-muted mb-0 small">%s</p>
    </div>
    <div class="mt-2 mt-md-0">
        <span class="badge bg-primary bg-opacity-10 text-primary px-3 py-2 border border-primary border-opacity-25">
            <i class="bi bi-database me-1"></i> Database: %s
        </span>
    </div>
</div>

<!-- Stat Cards Row -->
<div class="row mb-2">
%s
</div>

%s

<?php require_once 'footer.php'; ?>
`, dashTitle, statQueries.String(), dashTitle, welcomeMsg, req.DatabaseName, statCards.String(), recentDataBlock)
}

// GenerateAllScaffold assembles all scaffold files and inspects disk for existing files
func GenerateAllScaffold(req PHPGenerateRequest) PHPScaffoldResult {
	settings := GetCurrentSettings()
	mariaPort := settings.MariaDBPort
	if mariaPort == 0 {
		mariaPort = 3307
	}

	appTitle := req.AppTitle
	if appTitle == "" {
		appTitle = fmt.Sprintf("Aplikasi %s", strings.Title(strings.ReplaceAll(req.TableName, "_", " ")))
	}

	folder := req.ProjectFolder
	if folder == "" {
		folder = fmt.Sprintf("%s_app", sanitizeFilename(req.TableName))
	} else {
		folder = sanitizeFilename(folder)
	}

	fn := getPageFilenames(req)

	var files []PHPGeneratedFile

	if req.Dashboard.Enabled {
		files = append(files, PHPGeneratedFile{
			Filename:    "index.php",
			Content:     GenerateDashboardPHP(req, mariaPort),
			Description: "Halaman Beranda Admin Dashboard (Ringkasan Statistik & Navigasi)",
		})
	}

	files = append(files, []PHPGeneratedFile{
		{
			Filename:    "koneksi.php",
			Content:     GenerateKoneksiPHP(req.DatabaseName, mariaPort),
			Description: "Konfigurasi koneksi database MariaDB via PDO",
		},
		{
			Filename:    "header.php",
			Content:     GenerateHeaderPHP(appTitle, req.TableName, fn, req.Dashboard),
			Description: "Layout template navigasi atas & stylesheet Bootstrap 5",
		},
		{
			Filename:    "footer.php",
			Content:     GenerateFooterPHP(),
			Description: "Layout template footer & script Bootstrap 5",
		},
		{
			Filename:    fn.Index,
			Content:     GenerateIndexPHP(req, fn),
			Description: fmt.Sprintf("Halaman utama tabel data %s (pencarian, paginasi, JOIN relasi)", req.TableName),
		},
		{
			Filename:    fn.Tambah,
			Content:     GenerateTambahPHP(req, fn),
			Description: fmt.Sprintf("Formulir input data baru %s dengan dropdown relasi otomatis", req.TableName),
		},
		{
			Filename:    fn.Edit,
			Content:     GenerateEditPHP(req, fn),
			Description: fmt.Sprintf("Formulir ubah data %s dengan data awal terisi otomatis", req.TableName),
		},
		{
			Filename:    fn.Hapus,
			Content:     GenerateHapusPHP(req, fn),
			Description: fmt.Sprintf("Proses penghapusan data %s dengan perlindungan foreign key", req.TableName),
		},
	}...)

	// Check if files already exist on disk
	targetDir := filepath.Join(AppRootDir, "www", "htdocs", folder)
	var existingFiles []string
	for _, file := range files {
		filePath := filepath.Join(targetDir, file.Filename)
		if _, err := os.Stat(filePath); err == nil {
			existingFiles = append(existingFiles, file.Filename)
		}
	}

	webTarget := fn.Index
	if req.Dashboard.Enabled {
		webTarget = "index.php"
	}
	webUrl := fmt.Sprintf("http://localhost:%d/%s/%s", settings.ApachePort, folder, webTarget)

	return PHPScaffoldResult{
		DatabaseName:  req.DatabaseName,
		TableName:     req.TableName,
		ProjectFolder: folder,
		WebURL:        webUrl,
		Files:         files,
		ExistingFiles: existingFiles,
		HasExisting:   len(existingFiles) > 0,
	}
}

// HandlePHPGeneratorPreview returns generated code for review without writing to disk
func HandlePHPGeneratorPreview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method tidak diizinkan"})
		return
	}

	var req PHPGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Format JSON tidak valid: " + err.Error()})
		return
	}

	if req.DatabaseName == "" || req.TableName == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Database dan tabel wajib dipilih"})
		return
	}

	result := GenerateAllScaffold(req)
	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Data:    result,
	})
}

// HandlePHPGeneratorGenerate writes confirmed PHP files to www/htdocs/<folder>
func HandlePHPGeneratorGenerate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		jsonResponse(w, http.StatusMethodNotAllowed, APIResponse{Success: false, Error: "Method tidak diizinkan"})
		return
	}

	var req PHPGenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Format JSON tidak valid: " + err.Error()})
		return
	}

	if req.DatabaseName == "" || req.TableName == "" {
		jsonResponse(w, http.StatusBadRequest, APIResponse{Success: false, Error: "Database dan tabel wajib dipilih"})
		return
	}

	result := GenerateAllScaffold(req)
	targetDir := filepath.Join(AppRootDir, "www", "htdocs", result.ProjectFolder)

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: "Gagal membuat folder htdocs: " + err.Error()})
		return
	}

	for _, file := range result.Files {
		filePath := filepath.Join(targetDir, file.Filename)
		if err := os.WriteFile(filePath, []byte(file.Content), 0644); err != nil {
			jsonResponse(w, http.StatusInternalServerError, APIResponse{Success: false, Error: fmt.Sprintf("Gagal menulis file '%s': %s", file.Filename, err.Error())})
			return
		}
	}

	jsonResponse(w, http.StatusOK, APIResponse{
		Success: true,
		Message: fmt.Sprintf("Berhasil membuat seluruh berkas PHP di folder 'www/htdocs/%s'!", result.ProjectFolder),
		Data:    result,
	})
}

// Helper utilities
func requiredStar(req bool) string {
	if req {
		return `<span class="text-danger">*</span>`
	}
	return ""
}

func htmlspecialcharsGo(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	return s
}

func extractEnumValues(enumDef string) []string {
	clean := strings.TrimSpace(enumDef)
	clean = strings.TrimPrefix(clean, "enum(")
	clean = strings.TrimPrefix(clean, "ENUM(")
	clean = strings.TrimSuffix(clean, ")")
	parts := strings.Split(clean, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, "'\"")
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}
