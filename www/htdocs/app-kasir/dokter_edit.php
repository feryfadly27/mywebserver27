<?php
require_once 'koneksi.php';
$page_title = 'Ubah Data dokter';

$id = $_GET['id'] ?? '';
if ($id === '') {
    header("Location: dokter.php?pesan=ID+tidak+valid&tipe=danger");
    exit;
}

// Ambil data lama
$stmtOld = $pdo->prepare("SELECT * FROM `dokter` WHERE `id` = :id");
$stmtOld->execute([':id' => $id]);
$row = $stmtOld->fetch();

if (!$row) {
    header("Location: dokter.php?pesan=Data+tidak+ditemukan&tipe=danger");
    exit;
}

$errors = [];
$values = $row;

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $values['nip'] = trim($_POST['nip'] ?? '');
    if ($values['nip'] === '') {
        $errors['nip'] = 'Nip wajib diisi.';
    }
    $values['nama_dokter'] = trim($_POST['nama_dokter'] ?? '');
    if ($values['nama_dokter'] === '') {
        $errors['nama_dokter'] = 'Nama Dokter wajib diisi.';
    }
    $values['spesialis'] = trim($_POST['spesialis'] ?? '');
    if ($values['spesialis'] === '') {
        $errors['spesialis'] = 'Spesialis wajib diisi.';
    }
    $values['no_hp'] = trim($_POST['no_hp'] ?? '');
    $values['tarif_konsultasi'] = trim($_POST['tarif_konsultasi'] ?? '');

    if (empty($errors)) {
        try {
            $sql = "UPDATE `dokter` SET `nip` = :nip, `nama_dokter` = :nama_dokter, `spesialis` = :spesialis, `no_hp` = :no_hp, `tarif_konsultasi` = :tarif_konsultasi WHERE `id` = :primary_id";
            $stmt = $pdo->prepare($sql);
            $stmt->execute([
                ':nip' => $values['nip'],
                ':nama_dokter' => $values['nama_dokter'],
                ':spesialis' => $values['spesialis'],
                ':no_hp' => $values['no_hp'] !== '' ? $values['no_hp'] : null,
                ':tarif_konsultasi' => $values['tarif_konsultasi'] !== '' ? $values['tarif_konsultasi'] : null,
                ':primary_id' => $id,
            ]);

            header("Location: dokter.php?pesan=Data+berhasil+diperbarui!&tipe=success");
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
                    <h5 class="fw-bold mb-0"><i class="bi bi-pencil-square text-primary me-2"></i>Ubah Data dokter</h5>
                    <a href="dokter.php" class="btn btn-outline-secondary btn-sm"><i class="bi bi-arrow-left me-1"></i>Kembali</a>
                </div>
            </div>
            <div class="card-body p-4">
                <?php if (!empty($errors['db'])): ?>
                    <div class="alert alert-danger"><i class="bi bi-exclamation-triangle-fill me-2"></i><?= htmlspecialchars($errors['db']) ?></div>
                <?php endif; ?>

                <form method="POST" action="dokter_edit.php?id=<?= urlencode($id) ?>">
                    <div class="mb-3">
                        <label for="nip" class="form-label fw-semibold">Nip <span class="text-danger">*</span></label>
                        <input type="text" class="form-control <?= isset($errors['nip']) ? 'is-invalid' : '' ?>" id="nip" name="nip" value="<?= htmlspecialchars($values['nip'] ?? '') ?>">
                        <?php if (isset($errors['nip'])): ?>
                            <div class="invalid-feedback"><?= $errors['nip'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="nama_dokter" class="form-label fw-semibold">Nama Dokter <span class="text-danger">*</span></label>
                        <input type="text" class="form-control <?= isset($errors['nama_dokter']) ? 'is-invalid' : '' ?>" id="nama_dokter" name="nama_dokter" value="<?= htmlspecialchars($values['nama_dokter'] ?? '') ?>">
                        <?php if (isset($errors['nama_dokter'])): ?>
                            <div class="invalid-feedback"><?= $errors['nama_dokter'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="spesialis" class="form-label fw-semibold">Spesialis <span class="text-danger">*</span></label>
                        <input type="text" class="form-control <?= isset($errors['spesialis']) ? 'is-invalid' : '' ?>" id="spesialis" name="spesialis" value="<?= htmlspecialchars($values['spesialis'] ?? '') ?>">
                        <?php if (isset($errors['spesialis'])): ?>
                            <div class="invalid-feedback"><?= $errors['spesialis'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="no_hp" class="form-label fw-semibold">No Hp </label>
                        <input type="text" class="form-control <?= isset($errors['no_hp']) ? 'is-invalid' : '' ?>" id="no_hp" name="no_hp" value="<?= htmlspecialchars($values['no_hp'] ?? '') ?>">
                        <?php if (isset($errors['no_hp'])): ?>
                            <div class="invalid-feedback"><?= $errors['no_hp'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="tarif_konsultasi" class="form-label fw-semibold">Tarif Konsultasi </label>
                        <input type="number" step="0.01" class="form-control <?= isset($errors['tarif_konsultasi']) ? 'is-invalid' : '' ?>" id="tarif_konsultasi" name="tarif_konsultasi" value="<?= htmlspecialchars($values['tarif_konsultasi'] ?? '') ?>">
                        <?php if (isset($errors['tarif_konsultasi'])): ?>
                            <div class="invalid-feedback"><?= $errors['tarif_konsultasi'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="d-flex justify-content-end gap-2 mt-4">
                        <a href="dokter.php" class="btn btn-light border">Batal</a>
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
