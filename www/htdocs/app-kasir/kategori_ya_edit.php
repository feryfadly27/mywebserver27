<?php
require_once 'koneksi.php';
$page_title = 'Ubah Data kategori_ya';

$id = $_GET['id'] ?? '';
if ($id === '') {
    header("Location: kategori_ya.php?pesan=ID+tidak+valid&tipe=danger");
    exit;
}

// Ambil data lama
$stmtOld = $pdo->prepare("SELECT * FROM `kategori_ya` WHERE `id` = :id");
$stmtOld->execute([':id' => $id]);
$row = $stmtOld->fetch();

if (!$row) {
    header("Location: kategori_ya.php?pesan=Data+tidak+ditemukan&tipe=danger");
    exit;
}

$errors = [];
$values = $row;

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $values['nama'] = trim($_POST['nama'] ?? '');
    if ($values['nama'] === '') {
        $errors['nama'] = 'Nama wajib diisi.';
    }
    $values['keterangan'] = trim($_POST['keterangan'] ?? '');

    if (empty($errors)) {
        try {
            $sql = "UPDATE `kategori_ya` SET `nama` = :nama, `keterangan` = :keterangan WHERE `id` = :primary_id";
            $stmt = $pdo->prepare($sql);
            $stmt->execute([
                ':nama' => $values['nama'],
                ':keterangan' => $values['keterangan'] !== '' ? $values['keterangan'] : null,
                ':primary_id' => $id,
            ]);

            header("Location: kategori_ya.php?pesan=Data+berhasil+diperbarui!&tipe=success");
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
                    <h5 class="fw-bold mb-0"><i class="bi bi-pencil-square text-primary me-2"></i>Ubah Data kategori_ya</h5>
                    <a href="kategori_ya.php" class="btn btn-outline-secondary btn-sm"><i class="bi bi-arrow-left me-1"></i>Kembali</a>
                </div>
            </div>
            <div class="card-body p-4">
                <?php if (!empty($errors['db'])): ?>
                    <div class="alert alert-danger"><i class="bi bi-exclamation-triangle-fill me-2"></i><?= htmlspecialchars($errors['db']) ?></div>
                <?php endif; ?>

                <form method="POST" action="kategori_ya_edit.php?id=<?= urlencode($id) ?>">
                    <div class="mb-3">
                        <label for="nama" class="form-label fw-semibold">Nama <span class="text-danger">*</span></label>
                        <input type="text" class="form-control <?= isset($errors['nama']) ? 'is-invalid' : '' ?>" id="nama" name="nama" value="<?= htmlspecialchars($values['nama'] ?? '') ?>">
                        <?php if (isset($errors['nama'])): ?>
                            <div class="invalid-feedback"><?= $errors['nama'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="keterangan" class="form-label fw-semibold">Keterangan </label>
                        <textarea class="form-control <?= isset($errors['keterangan']) ? 'is-invalid' : '' ?>" id="keterangan" name="keterangan" rows="3"><?= htmlspecialchars($values['keterangan'] ?? '') ?></textarea>
                        <?php if (isset($errors['keterangan'])): ?>
                            <div class="invalid-feedback"><?= $errors['keterangan'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="d-flex justify-content-end gap-2 mt-4">
                        <a href="kategori_ya.php" class="btn btn-light border">Batal</a>
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
