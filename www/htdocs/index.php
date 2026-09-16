<!DOCTYPE html>
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
            <p>File ini berada di direktori <code>www/htdocs/index.php</code>. Ganti atau tambahkan file proyek web Anda
                di folder tersebut.</p>
        </div>

        <div class="links">
            <a href="/phpmyadmin" class="btn btn-primary">Buka phpMyAdmin</a>
            <a href="http://localhost:3001" class="btn btn-default" target="_blank">Control Panel</a>
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

        <div
            style="margin-top: 24px; padding-top: 16px; border-top: 1px solid var(--border); text-align: center; font-size: 12px; color: var(--text-muted);">
            Created by <strong style="color: var(--text); font-weight: 600;">Fery Fadly</strong> and <strong
                style="color: var(--text); font-weight: 600;">Team Dikodein</strong>
        </div>
    </div>
</body>

</html>