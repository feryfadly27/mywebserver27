<?php
require_once 'koneksi.php';
$page_title = 'Tambah Data kategori_ya';

$errors = [];
$values = [];

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $values['nama'] = trim($_POST['nama'] ?? '');
    if ($values['nama'] === '') {
        $errors['nama'] = 'Nama wajib diisi.';
    }
    $values['keterangan'] = trim($_POST['keterangan'] ?? '');

    if (empty($errors)) {
        try {
            $sql = "INSERT INTO `kategori_ya` (`nama`, `keterangan`) VALUES (:nama, :keterangan)";
            $stmt = $pdo->prepare($sql);
            $stmt->execute([
                ':nama' => $values['nama'],
                ':keterangan' => $values['keterangan'] !== '' ? $values['keterangan'] : null,
            ]);

            header("Location: kategori_ya.php?pesan=Data+berhasil+ditambahkan!&tipe=success");
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
                    <h5 class="fw-bold mb-0"><i class="bi bi-plus-circle text-primary me-2"></i>Tambah Data kategori_ya</h5>
                    <a href="kategori_ya.php" class="btn btn-outline-secondary btn-sm"><i class="bi bi-arrow-left me-1"></i>Kembali</a>
                </div>
            </div>
            <div class="card-body p-4">
                <?php if (!empty($errors['db'])): ?>
                    <div class="alert alert-danger"><i class="bi bi-exclamation-triangle-fill me-2"></i><?= htmlspecialchars($errors['db']) ?></div>
                <?php endif; ?>

                <form method="POST" action="kategori_ya_tambah.php">
                    <div class="mb-3">
                        <label for="nama" class="form-label fw-semibold">Nama <span class="text-danger">*</span></label>
                        <input type="text" class="form-control <?= isset($errors['nama']) ? 'is-invalid' : '' ?>" id="nama" name="nama" value="<?= htmlspecialchars($values['Nama'] ?? '') ?>" placeholder="Masukkan %!s(MISSING)">
                        <?php if (isset($errors['nama'])): ?>
                            <div class="invalid-feedback"><?= $errors['nama'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="keterangan" class="form-label fw-semibold">Keterangan </label>
                        <textarea class="form-control <?= isset($errors['keterangan']) ? 'is-invalid' : '' ?>" id="keterangan" name="keterangan" rows="3" placeholder="Masukkan Keterangan"><?= htmlspecialchars($values['keterangan'] ?? '') ?></textarea>
                        <?php if (isset($errors['keterangan'])): ?>
                            <div class="invalid-feedback"><?= $errors['keterangan'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="d-flex justify-content-end gap-2 mt-4">
                        <a href="kategori_ya.php" class="btn btn-light border">Batal</a>
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
