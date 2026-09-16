<?php
require_once 'koneksi.php';

$id = $_GET['id'] ?? '';
if ($id === '') {
    header("Location: dokter.php?pesan=ID+tidak+valid&tipe=danger");
    exit;
}

try {
    $stmt = $pdo->prepare("DELETE FROM `dokter` WHERE `id` = :id");
    $stmt->execute([':id' => $id]);

    header("Location: dokter.php?pesan=Data+berhasil+dihapus!&tipe=success");
    exit;
} catch (PDOException $e) {
    header("Location: dokter.php?pesan=Gagal+menghapus+data:+relasi+data+masih+terikat+di+tabel+lain&tipe=danger");
    exit;
}
