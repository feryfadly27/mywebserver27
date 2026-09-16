            </main>
            <footer class="footer-custom d-flex justify-content-between align-items-center">
                <span>&copy; <?= date('Y') ?> &bull; Panel Administrasi Data</span>
                <span class="text-muted">MyLokalWebserver</span>
            </footer>
        </div> <!-- /.app-main -->
    </div> <!-- /.app-wrapper -->

    <!-- Bootstrap 5 JS Bundle -->
    <script src="https://cdn.jsdelivr.net/npm/bootstrap@5.3.2/dist/js/bootstrap.bundle.min.js"></script>
    <script>
        const btnToggle = document.getElementById('btnToggleSidebar');
        const sidebar = document.getElementById('appSidebar');
        if (btnToggle && sidebar) {
            btnToggle.addEventListener('click', () => {
                sidebar.classList.toggle('show');
            });
        }
    </script>
</body>
</html>
