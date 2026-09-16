<?php
require_once 'koneksi.php';
$page_title = 'Daftar Data kategori_ya';

// Pengaturan Pencarian & Paginasi
$search = trim($_GET['q'] ?? '');
$page = max(1, (int)($_GET['page'] ?? 1));
$limit = 10;
$offset = ($page - 1) * $limit;

// Menghitung Total Data
$countSql = "SELECT COUNT(*) FROM `kategori_ya` ";
$params = [];

if ($search !== '') {
    $countSql .= " WHERE (`kategori_ya`.`nama` LIKE :search OR `kategori_ya`.`keterangan` LIKE :search)";
    $params[':search'] = "%{$search}%";
}

$stmtCount = $pdo->prepare($countSql);
$stmtCount->execute($params);
$totalRows = (int)$stmtCount->fetchColumn();
$totalPages = ceil($totalRows / $limit);

// Query Mengambil Data
$sql = "SELECT `kategori_ya`.* FROM `kategori_ya` ";
if ($search !== '') {
    $sql .= " WHERE (`kategori_ya`.`nama` LIKE :search OR `kategori_ya`.`keterangan` LIKE :search)";
}
$sql .= " ORDER BY `kategori_ya`.`id` DESC LIMIT {$limit} OFFSET {$offset}";

$stmt = $pdo->prepare($sql);
$stmt->execute($params);
$rows = $stmt->fetchAll();

// Notifikasi pesan
$pesan = $_GET['pesan'] ?? '';
$tipePesan = $_GET['tipe'] ?? 'success';

require_once 'header.php';
?>

<div class="card card-custom">
    <div class="card-body p-4">
        <!-- Header Aksi & Pencarian -->
        <div class="d-flex flex-column flex-md-row justify-content-between align-items-md-center gap-3 mb-4">
            <div>
                <h4 class="card-title fw-bold mb-1"><i class="bi bi-journal-text me-2 text-primary"></i>Tabel Data kategori_ya</h4>
                <p class="text-muted small mb-0">Total: <strong><?= number_format($totalRows) ?></strong> data ditemukan</p>
            </div>
            <div class="d-flex gap-2">
                <a href="kategori_ya_tambah.php" class="btn btn-primary">
                    <i class="bi bi-plus-lg me-1"></i> Tambah Data Baru
                </a>
            </div>
        </div>

        <?php if ($pesan): ?>
            <div class="alert alert-<?= htmlspecialchars($tipePesan) ?> alert-dismissible fade show" role="alert">
                <i class="bi bi-check-circle-fill me-2"></i><?= htmlspecialchars($pesan) ?>
                <button type="button" class="btn-close" data-bs-dismiss="alert" aria-label="Close"></button>
            </div>
        <?php endif; ?>

        <!-- Form Pencarian -->
        <form method="GET" action="kategori_ya.php" class="row g-2 mb-3">
            <div class="col-md-5 col-lg-4">
                <div class="input-group">
                    <span class="input-group-text bg-light"><i class="bi bi-search"></i></span>
                    <input type="text" name="q" class="form-control" placeholder="Cari data..." value="<?= htmlspecialchars($search) ?>">
                    <?php if ($search !== ''): ?>
                        <a href="kategori_ya.php" class="btn btn-outline-secondary" title="Reset Pencarian"><i class="bi bi-x-lg"></i></a>
                    <?php endif; ?>
                    <button type="submit" class="btn btn-outline-primary">Cari</button>
                </div>
            </div>
        </form>

        <!-- Tabel Data -->
        <div class="table-responsive">
            <table class="table table-hover align-middle">
                <thead>
                    <tr>
                        <th style="width: 50px;" class="text-center">No</th>
                        <th>Id</th>
                        <th>Nama</th>
                        <th>Keterangan</th>
                        <th style="width: 140px;" class="text-center">Aksi</th>
                    </tr>
                </thead>
                <tbody>
                    <?php if (empty($rows)): ?>
                        <tr>
                            <td colspan="100" class="text-center py-4 text-muted">
                                <i class="bi bi-inbox fs-2 d-block mb-2 text-secondary"></i>
                                Tidak ada data yang ditemukan.
                            </td>
                        </tr>
                    <?php else: ?>
                        <?php $no = $offset + 1; foreach ($rows as $row): ?>
                            <tr>
                                <td class="text-center text-muted fw-bold"><?= $no++ ?></td>
                                <td><?= htmlspecialchars($row['id'] ?? '-') ?></td>
                                <td><?= htmlspecialchars($row['nama'] ?? '-') ?></td>
                                <td><?= htmlspecialchars($row['keterangan'] ?? '-') ?></td>
                                <td class="text-center">
                                    <div class="btn-group btn-group-sm">
                                        <a href="kategori_ya_edit.php?id=<?= urlencode($row['id']) ?>" class="btn btn-outline-primary" title="Ubah Data">
                                            <i class="bi bi-pencil-square"></i>
                                        </a>
                                        <a href="kategori_ya_hapus.php?id=<?= urlencode($row['id']) ?>" class="btn btn-outline-danger" onclick="return confirm('Apakah Anda yakin ingin menghapus data ini?');" title="Hapus Data">
                                            <i class="bi bi-trash"></i>
                                        </a>
                                    </div>
                                </td>
                            </tr>
                        <?php endforeach; ?>
                    <?php endif; ?>
                </tbody>
            </table>
        </div>

        <!-- Paginasi -->
        <?php if ($totalPages > 1): ?>
            <div class="d-flex justify-content-between align-items-center mt-3">
                <small class="text-muted">Halaman <?= $page ?> dari <?= $totalPages ?></small>
                <nav>
                    <ul class="pagination pagination-sm mb-0">
                        <li class="page-item <?= ($page <= 1) ? 'disabled' : '' ?>">
                            <a class="page-link" href="kategori_ya.php?page=<?= $page - 1 ?>&q=<?= urlencode($search) ?>">&laquo;</a>
                        </li>
                        <?php for ($i = 1; $i <= $totalPages; $i++): ?>
                            <li class="page-item <?= ($page == $i) ? 'active' : '' ?>">
                                <a class="page-link" href="kategori_ya.php?page=<?= $i ?>&q=<?= urlencode($search) ?>"><?= $i ?></a>
                            </li>
                        <?php endfor; ?>
                        <li class="page-item <?= ($page >= $totalPages) ? 'disabled' : '' ?>">
                            <a class="page-link" href="kategori_ya.php?page=<?= $page + 1 ?>&q=<?= urlencode($search) ?>">&raquo;</a>
                        </li>
                    </ul>
                </nav>
            </div>
        <?php endif; ?>
    </div>
</div>

<?php
require_once 'footer.php';
?>
