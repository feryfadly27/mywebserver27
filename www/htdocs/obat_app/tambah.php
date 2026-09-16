<?php
require_once 'koneksi.php';
$page_title = 'Tambah Data obat';

$errors = [];
$values = [];

// Ambil data referensi untuk dropdown relasi Foreign Key
$options_kategori_id = $pdo->query("SELECT `id`, `keterangan` FROM `kategori_ya` ORDER BY `keterangan` ASC")->fetchAll();

if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $values['kode_obat'] = trim($_POST['kode_obat'] ?? '');
    if ($values['kode_obat'] === '') {
        $errors['kode_obat'] = 'Kode Obat wajib diisi.';
    }
    $values['nama_obat'] = trim($_POST['nama_obat'] ?? '');
    if ($values['nama_obat'] === '') {
        $errors['nama_obat'] = 'Nama Obat wajib diisi.';
    }
    $values['stok'] = trim($_POST['stok'] ?? '');
    if ($values['stok'] === '') {
        $errors['stok'] = 'Stok wajib diisi.';
    }
    $values['harga_beli'] = trim($_POST['harga_beli'] ?? '');
    if ($values['harga_beli'] === '') {
        $errors['harga_beli'] = 'Harga Beli wajib diisi.';
    }
    $values['harga_jual'] = trim($_POST['harga_jual'] ?? '');
    if ($values['harga_jual'] === '') {
        $errors['harga_jual'] = 'Harga Jual wajib diisi.';
    }
    $values['tgl_kadaluarsa'] = trim($_POST['tgl_kadaluarsa'] ?? '');
    if ($values['tgl_kadaluarsa'] === '') {
        $errors['tgl_kadaluarsa'] = 'Tgl Kadaluarsa wajib diisi.';
    }
    $values['status_obat'] = trim($_POST['status_obat'] ?? '');
    if ($values['status_obat'] === '') {
        $errors['status_obat'] = 'Status Obat wajib diisi.';
    }
    $values['kategori'] = trim($_POST['kategori'] ?? '');
    $values['kategori_id'] = trim($_POST['kategori_id'] ?? '');

    if (empty($errors)) {
        try {
            $sql = "INSERT INTO `obat` (`kode_obat`, `nama_obat`, `stok`, `harga_beli`, `harga_jual`, `tgl_kadaluarsa`, `status_obat`, `kategori`, `kategori_id`) VALUES (:kode_obat, :nama_obat, :stok, :harga_beli, :harga_jual, :tgl_kadaluarsa, :status_obat, :kategori, :kategori_id)";
            $stmt = $pdo->prepare($sql);
            $stmt->execute([
                ':kode_obat' => $values['kode_obat'],
                ':nama_obat' => $values['nama_obat'],
                ':stok' => $values['stok'],
                ':harga_beli' => $values['harga_beli'],
                ':harga_jual' => $values['harga_jual'],
                ':tgl_kadaluarsa' => $values['tgl_kadaluarsa'],
                ':status_obat' => $values['status_obat'],
                ':kategori' => $values['kategori'] !== '' ? $values['kategori'] : null,
                ':kategori_id' => $values['kategori_id'] !== '' ? $values['kategori_id'] : null,
            ]);

            header("Location: index.php?pesan=Data+berhasil+ditambahkan!&tipe=success");
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
                    <h5 class="fw-bold mb-0"><i class="bi bi-plus-circle text-primary me-2"></i>Tambah Data obat</h5>
                    <a href="index.php" class="btn btn-outline-secondary btn-sm"><i class="bi bi-arrow-left me-1"></i>Kembali</a>
                </div>
            </div>
            <div class="card-body p-4">
                <?php if (!empty($errors['db'])): ?>
                    <div class="alert alert-danger"><i class="bi bi-exclamation-triangle-fill me-2"></i><?= htmlspecialchars($errors['db']) ?></div>
                <?php endif; ?>

                <form method="POST" action="tambah.php">
                    <div class="mb-3">
                        <label for="kode_obat" class="form-label fw-semibold">Kode Obat <span class="text-danger">*</span></label>
                        <input type="text" class="form-control <?= isset($errors['kode_obat']) ? 'is-invalid' : '' ?>" id="kode_obat" name="kode_obat" value="<?= htmlspecialchars($values['Kode Obat'] ?? '') ?>" placeholder="Masukkan %!s(MISSING)">
                        <?php if (isset($errors['kode_obat'])): ?>
                            <div class="invalid-feedback"><?= $errors['kode_obat'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="nama_obat" class="form-label fw-semibold">Nama Obat <span class="text-danger">*</span></label>
                        <input type="text" class="form-control <?= isset($errors['nama_obat']) ? 'is-invalid' : '' ?>" id="nama_obat" name="nama_obat" value="<?= htmlspecialchars($values['Nama Obat'] ?? '') ?>" placeholder="Masukkan %!s(MISSING)">
                        <?php if (isset($errors['nama_obat'])): ?>
                            <div class="invalid-feedback"><?= $errors['nama_obat'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="stok" class="form-label fw-semibold">Stok <span class="text-danger">*</span></label>
                        <input type="number" step="1" class="form-control <?= isset($errors['stok']) ? 'is-invalid' : '' ?>" id="stok" name="stok" value="<?= htmlspecialchars($values['stok'] ?? '') ?>" placeholder="0">
                        <?php if (isset($errors['stok'])): ?>
                            <div class="invalid-feedback"><?= $errors['stok'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="harga_beli" class="form-label fw-semibold">Harga Beli <span class="text-danger">*</span></label>
                        <input type="number" step="0.01" class="form-control <?= isset($errors['harga_beli']) ? 'is-invalid' : '' ?>" id="harga_beli" name="harga_beli" value="<?= htmlspecialchars($values['harga_beli'] ?? '') ?>" placeholder="0">
                        <?php if (isset($errors['harga_beli'])): ?>
                            <div class="invalid-feedback"><?= $errors['harga_beli'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="harga_jual" class="form-label fw-semibold">Harga Jual <span class="text-danger">*</span></label>
                        <input type="number" step="0.01" class="form-control <?= isset($errors['harga_jual']) ? 'is-invalid' : '' ?>" id="harga_jual" name="harga_jual" value="<?= htmlspecialchars($values['harga_jual'] ?? '') ?>" placeholder="0">
                        <?php if (isset($errors['harga_jual'])): ?>
                            <div class="invalid-feedback"><?= $errors['harga_jual'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="tgl_kadaluarsa" class="form-label fw-semibold">Tgl Kadaluarsa <span class="text-danger">*</span></label>
                        <input type="date" class="form-control <?= isset($errors['tgl_kadaluarsa']) ? 'is-invalid' : '' ?>" id="tgl_kadaluarsa" name="tgl_kadaluarsa" value="<?= htmlspecialchars($values['tgl_kadaluarsa'] ?? '') ?>">
                        <?php if (isset($errors['tgl_kadaluarsa'])): ?>
                            <div class="invalid-feedback"><?= $errors['tgl_kadaluarsa'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="status_obat" class="form-label fw-semibold">Status Obat <span class="text-danger">*</span></label>
                        <select class="form-select <?= isset($errors['status_obat']) ? 'is-invalid' : '' ?>" id="status_obat" name="status_obat">
                            <option value="">-- Pilih Status Obat --</option>
                            <option value="Tersedia" <?= (($values['status_obat'] ?? '') === 'Tersedia') ? 'selected' : '' ?>>Tersedia</option>
                            <option value="Menipis" <?= (($values['status_obat'] ?? '') === 'Menipis') ? 'selected' : '' ?>>Menipis</option>
                            <option value="Habis" <?= (($values['status_obat'] ?? '') === 'Habis') ? 'selected' : '' ?>>Habis</option>
                            <option value="Indent" <?= (($values['status_obat'] ?? '') === 'Indent') ? 'selected' : '' ?>>Indent</option>
                        </select>
                        <?php if (isset($errors['status_obat'])): ?>
                            <div class="invalid-feedback"><?= $errors['status_obat'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="kategori" class="form-label fw-semibold">Kategori </label>
                        <select class="form-select <?= isset($errors['kategori']) ? 'is-invalid' : '' ?>" id="kategori" name="kategori">
                            <option value="">-- Pilih Kategori --</option>
                            <option value="Makanan" <?= (($values['kategori'] ?? '') === 'Makanan') ? 'selected' : '' ?>>Makanan</option>
                            <option value="Minuman" <?= (($values['kategori'] ?? '') === 'Minuman') ? 'selected' : '' ?>>Minuman</option>
                            <option value="Obat" <?= (($values['kategori'] ?? '') === 'Obat') ? 'selected' : '' ?>>Obat</option>
                        </select>
                        <?php if (isset($errors['kategori'])): ?>
                            <div class="invalid-feedback"><?= $errors['kategori'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="mb-3">
                        <label for="kategori_id" class="form-label fw-semibold">Kategori Id </label>
                        <select class="form-select <?= isset($errors['kategori_id']) ? 'is-invalid' : '' ?>" id="kategori_id" name="kategori_id">
                            <option value="">-- Pilih Kategori Id --</option>
                            <?php foreach ($options_kategori_id as $opt): ?>
                                <option value="<?= htmlspecialchars($opt['id']) ?>" <?= (($values['kategori_id'] ?? '') == $opt['id']) ? 'selected' : '' ?>>
                                    <?= htmlspecialchars($opt['keterangan']) ?>
                                </option>
                            <?php endforeach; ?>
                        </select>
                        <?php if (isset($errors['kategori_id'])): ?>
                            <div class="invalid-feedback"><?= $errors['kategori_id'] ?></div>
                        <?php endif; ?>
                    </div>
                    <div class="d-flex justify-content-end gap-2 mt-4">
                        <a href="index.php" class="btn btn-light border">Batal</a>
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
