<?php
/**
 * Koneksi Database MariaDB / MySQL via PDO
 * Dibuat secara otomatis oleh MyLokalWebserver
 */

$db_host = '127.0.0.1';
$db_port = '3307';
$db_name = 'db_apotek';
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
