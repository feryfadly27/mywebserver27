<?php
require_once 'koneksi.php';

$id = $_GET['id'] ?? '';
if ($id === '') {
    header("Location: index.php?pesan=ID+tidak+valid&tipe=danger");
    exit;
}

try {
    $stmt = $pdo->prepare("DELETE FROM `obat` WHERE `id` = :id");
    $stmt->execute([':id' => $id]);

    header("Location: index.php?pesan=Data+berhasil+dihapus!&tipe=success");
    exit;
} catch (PDOException $e) {
    header("Location: index.php?pesan=Gagal+menghapus+data:+relasi+data+masih+terikat+di+tabel+lain&tipe=danger");
    exit;
}
