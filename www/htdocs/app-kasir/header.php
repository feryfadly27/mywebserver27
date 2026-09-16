<?php
// Header & Sidebar template
?>
<!DOCTYPE html>
<html lang="id">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title><?= isset($page_title) ? htmlspecialchars($page_title) . ' - ' : '' ?>Sistem Data Kategori Ya</title>
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
            width: 100%%;
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
            width: calc(100%% - var(--sidebar-width));
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
                transform: translateX(-100%%);
            }
            .app-sidebar.show {
                transform: translateX(0);
            }
            .app-main {
                margin-left: 0;
                width: 100%%;
            }
        }
    </style>
</head>
<body>
    <div class="app-wrapper">
        <!-- Sidebar Navigation -->
        <aside class="app-sidebar" id="appSidebar">
            <a href="index.php" class="sidebar-brand text-decoration-none">
                <i class="bi bi-grid-1x2-fill text-info me-2 fs-5"></i>
                <span class="text-truncate">Sistem Data Kategori Ya</span>
            </a>
            <div class="sidebar-nav">

            <div class="sidebar-heading">MENU UTAMA</div>
            <a class="sidebar-link <?= basename($_SERVER['PHP_SELF']) == 'index.php' ? 'active' : '' ?>" href="index.php">
                <i class="bi bi-speedometer2"></i>
                <span>Dashboard</span>
            </a>
            <div class="sidebar-heading">KELOLA DATA</div>
            <a class="sidebar-link <?= in_array(basename($_SERVER['PHP_SELF']), ['kategori_ya.php', 'kategori_ya_edit.php']) ? 'active' : '' ?>" href="kategori_ya.php">
                <i class="bi bi-table"></i>
                <span>Data Kategori Ya</span>
            </a>
            <a class="sidebar-link <?= basename($_SERVER['PHP_SELF']) == 'kategori_ya_tambah.php' ? 'active' : '' ?>" href="kategori_ya_tambah.php">
                <i class="bi bi-plus-circle"></i>
                <span>Tambah Data</span>
            </a>
            <div class="sidebar-heading">MODUL LAINNYA</div>
            <a class="sidebar-link <?= basename($_SERVER['PHP_SELF']) == 'obat.php' ? 'active' : '' ?>" href="obat.php">
                <i class="bi bi-folder2-open"></i>
                <span>Total Obat</span>
            </a>
            <div class="sidebar-heading">TOOLS</div>
            <a class="sidebar-link" href="http://localhost:8080/phpmyadmin" target="_blank">
                <i class="bi bi-database-fill-gear"></i>
                <span>phpMyAdmin</span>
            </a>
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
                    <span class="fw-bold text-dark fs-6"><?= isset($page_title) ? htmlspecialchars($page_title) : 'Sistem Data Kategori Ya' ?></span>
                </div>
                <div class="d-flex align-items-center gap-2">
                    <span class="badge bg-light text-dark border px-3 py-2">
                        <i class="bi bi-person-fill text-primary me-1"></i> Administrator
                    </span>
                </div>
            </header>

            <!-- Page Body Content -->
            <main class="app-content">
