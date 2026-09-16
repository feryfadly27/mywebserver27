<?php
require_once 'koneksi.php';

$id = $_GET['id'] ?? '';
if ($id === '') {
    header("Location: kategori_ya.php?pesan=ID+tidak+valid&tipe=danger");
    exit;
}

try {
    $stmt = $pdo->prepare("DELETE FROM `kategori_ya` WHERE `id` = :id");
    $stmt->execute([':id' => $id]);

    header("Location: kategori_ya.php?pesan=Data+berhasil+dihapus!&tipe=success");
    exit;
} catch (PDOException $e) {
    header("Location: kategori_ya.php?pesan=Gagal+menghapus+data:+relasi+data+masih+terikat+di+tabel+lain&tipe=danger");
    exit;
}
