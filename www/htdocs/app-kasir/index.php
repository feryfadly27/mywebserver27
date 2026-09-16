<?php
require_once 'koneksi.php';
$page_title = 'Dashboard Admin Kategori Ya';

// Query Statistik

$count_kategori_ya_0 = 0;
try {
    $stmt = $pdo->query("SELECT COUNT(*) FROM `kategori_ya`");
    $count_kategori_ya_0 = (int)$stmt->fetchColumn();
} catch (Exception $e) {}

$count_obat_1 = 0;
try {
    $stmt = $pdo->query("SELECT COUNT(*) FROM `obat`");
    $count_obat_1 = (int)$stmt->fetchColumn();
} catch (Exception $e) {}


require_once 'header.php';
?>

<!-- Header Halaman -->
<div class="d-flex flex-column flex-md-row justify-content-between align-items-md-center mb-4 pb-2 border-bottom">
    <div>
        <h4 class="fw-bold text-dark mb-1">Dashboard Admin Kategori Ya</h4>
        <p class="text-muted mb-0 small">Ringkasan statistik data dan status sistem</p>
    </div>
    <div class="mt-2 mt-md-0">
        <span class="badge bg-primary bg-opacity-10 text-primary px-3 py-2 border border-primary border-opacity-25">
            <i class="bi bi-database me-1"></i> Database: db_apotek
        </span>
    </div>
</div>

<!-- Stat Cards Row -->
<div class="row mb-2">

        <div class="col-md-6 col-lg-4 mb-4">
            <div class="card card-custom h-100 shadow-sm border-0" style="border-left: 4px solid var(--bs-primary, #0284c7) !important;">
                <div class="card-body p-3">
                    <div class="d-flex align-items-center justify-content-between mb-2">
                        <span class="text-muted text-uppercase fw-semibold small">Total Kategori Ya</span>
                        <div class="rounded p-2 bg-primary bg-opacity-10 text-primary">
                            <i class="bi bi-table fs-5"></i>
                        </div>
                    </div>
                    <div class="d-flex align-items-baseline mb-2">
                        <h3 class="fw-bold mb-0 text-dark"><?= number_format($count_kategori_ya_0) ?></h3>
                        <span class="text-muted ms-2 small">data</span>
                    </div>
                    <a href="kategori_ya.php" class="text-decoration-none small fw-medium text-primary">
                        Kelola data <i class="bi bi-arrow-right ms-1"></i>
                    </a>
                </div>
            </div>
        </div>
        <div class="col-md-6 col-lg-4 mb-4">
            <div class="card card-custom h-100 shadow-sm border-0" style="border-left: 4px solid var(--bs-success, #0284c7) !important;">
                <div class="card-body p-3">
                    <div class="d-flex align-items-center justify-content-between mb-2">
                        <span class="text-muted text-uppercase fw-semibold small">Total Obat</span>
                        <div class="rounded p-2 bg-success bg-opacity-10 text-success">
                            <i class="bi bi-folder2-open fs-5"></i>
                        </div>
                    </div>
                    <div class="d-flex align-items-baseline mb-2">
                        <h3 class="fw-bold mb-0 text-dark"><?= number_format($count_obat_1) ?></h3>
                        <span class="text-muted ms-2 small">data</span>
                    </div>
                    <a href="obat.php" class="text-decoration-none small fw-medium text-success">
                        Kelola data <i class="bi bi-arrow-right ms-1"></i>
                    </a>
                </div>
            </div>
        </div>
</div>


    <!-- Recent Data Table Card -->
    <div class="card card-custom shadow-sm mb-4 border-0">
        <div class="card-header bg-white py-3 border-bottom d-flex justify-content-between align-items-center">
            <div class="d-flex align-items-center gap-2">
                <i class="bi bi-clock-history text-primary fs-5"></i>
                <h5 class="mb-0 fw-bold text-dark">5 Data Terbaru: Kategori Ya</h5>
            </div>
            <a href="kategori_ya.php" class="btn btn-sm btn-outline-primary">
                Lihat Semua Data <i class="bi bi-arrow-right ms-1"></i>
            </a>
        </div>
        <div class="card-body p-0">
            <?php
            $recentRows = [];
            try {
                $stmtRecent = $pdo->query("SELECT * FROM `kategori_ya` ORDER BY 1 DESC LIMIT 5");
                $recentRows = $stmtRecent->fetchAll();
            } catch (Exception $e) {}
            ?>
            <?php if (empty($recentRows)): ?>
                <div class="text-center py-5 text-muted">
                    <i class="bi bi-inbox fs-1 d-block mb-2 text-secondary"></i>
                    <p class="mb-0">Belum ada data di tabel <code>kategori_ya</code>.</p>
                </div>
            <?php else: ?>
                <div class="table-responsive">
                    <table class="table table-hover align-middle mb-0">
                        <thead class="table-light">
                            <tr>
                                <?php 
                                $firstRow = reset($recentRows);
                                $displayCols = array_slice(array_keys($firstRow), 0, 5);
                                foreach ($displayCols as $col): ?>
                                    <th class="px-3 py-2 text-uppercase small"><?= htmlspecialchars(str_replace('_', ' ', $col)) ?></th>
                                <?php endforeach; ?>
                            </tr>
                        </thead>
                        <tbody>
                            <?php foreach ($recentRows as $r): ?>
                                <tr>
                                    <?php foreach ($displayCols as $idx => $col): ?>
                                        <td class="px-3 py-2">
                                            <?php if ($idx === 0): ?>
                                                <span class="badge bg-secondary bg-opacity-10 text-dark font-monospace">#<?= htmlspecialchars((string)($r[$col] ?? '')) ?></span>
                                            <?php else: ?>
                                                <?= htmlspecialchars((string)($r[$col] ?? '-')) ?>
                                            <?php endif; ?>
                                        </td>
                                    <?php endforeach; ?>
                                </tr>
                            <?php endforeach; ?>
                        </tbody>
                    </table>
                </div>
            <?php endif; ?>
        </div>
    </div>

<?php require_once 'footer.php'; ?>
