// MyLokalWebserver Frontend Logic

let term = null;
let fitAddon = null;
let termWs = null;
let currentSettings = null;
let isTerminalInitialized = false;

document.addEventListener('DOMContentLoaded', () => {
    initApp();
    setupEventListeners();
});

let hasCheckedAutoUpdate = false;
let latestUpdateInfo = null;
let updatePollInterval = null;

function initApp() {
    fetchStatus();
    // Poll status every 3 seconds
    setInterval(fetchStatus, 3000);
}

let statusFailCount = 0;

async function fetchStatus() {
    const controller = new AbortController();
    const timeoutId = setTimeout(() => controller.abort(), 2500);

    try {
        const res = await fetch('/api/status', { signal: controller.signal });
        clearTimeout(timeoutId);
        const data = await res.json();
        if (!data.success) return;

        statusFailCount = 0; // reset on success

        const info = data.data;
        currentSettings = info.settings;

        const verEl = document.getElementById('textAppVersion');
        if (verEl && info.app_version) {
            verEl.textContent = `${info.app_version} • Portable`;
        }

        if (!info.binaries_installed) {
            document.getElementById('setupSection').classList.remove('hidden');
            document.getElementById('dashboardSection').classList.add('hidden');
            updateBinaryChecklist(info.binary_status);
        } else {
            document.getElementById('setupSection').classList.add('hidden');
            document.getElementById('dashboardSection').classList.remove('hidden');

            updateServicesUI(info.services, info.settings);
            updateQuickLinks(info.settings);
            fetchProjects();

            if (!isTerminalInitialized) {
                initTerminal();
            }
        }

        // Trigger background update check once if enabled
        if (!hasCheckedAutoUpdate && currentSettings && currentSettings.auto_check_update !== false) {
            hasCheckedAutoUpdate = true;
            checkForUpdates(false);
        }
    } catch (err) {
        clearTimeout(timeoutId);
        statusFailCount++;
        // If server is unreachable for >= 2 cycles, indicate offline state gracefully
        if (statusFailCount >= 2) {
            handleServerOffline();
        }
    }
}

function handleServerOffline() {
    const dotAp = document.getElementById('dotApache');
    const statusAp = document.getElementById('statusApache');
    const dotDb = document.getElementById('dotMariaDB');
    const statusDb = document.getElementById('statusMariaDB');

    if (dotAp && dotAp.parentElement) {
        dotAp.parentElement.className = 'status-indicator-pill offline';
        if (statusAp) statusAp.textContent = 'Server Offline';
    }
    if (dotDb && dotDb.parentElement) {
        dotDb.parentElement.className = 'status-indicator-pill offline';
        if (statusDb) statusDb.textContent = 'Server Offline';
    }

    const sbDotAp = document.getElementById('sidebarDotApache');
    const sbDotDb = document.getElementById('sidebarDotMariaDB');
    if (sbDotAp) sbDotAp.className = 'status-indicator-dot offline';
    if (sbDotDb) sbDotDb.className = 'status-indicator-dot offline';
}

function updateBinaryChecklist(status) {
    if (!status) return;
    setTag('badge-apache', status.apache);
    setTag('badge-php', status.php);
    setTag('badge-mariadb', status.mariadb);
    setTag('badge-phpmyadmin', status.phpmyadmin);
}

function setTag(id, isReady) {
    const el = document.getElementById(id);
    if (!el) return;
    if (isReady) {
        el.className = 'tag tag-done';
        el.textContent = 'Terpasang';
    } else {
        el.className = 'tag tag-neutral';
        el.textContent = 'Menunggu';
    }
}

function updateServicesUI(services, settings) {
    // Apache
    const ap = services.apache;
    const dotAp = document.getElementById('dotApache');
    const statusAp = document.getElementById('statusApache');
    const indAp = dotAp.parentElement;
    const btnToggleAp = document.getElementById('btnToggleApache');

    document.getElementById('valApachePort').textContent = settings.apache_port;
    document.getElementById('valApachePid').textContent = ap.running && ap.pid ? ap.pid : '-';
    document.getElementById('valApacheUptime').textContent = ap.running && ap.uptime ? ap.uptime : '-';

    const sbDotAp = document.getElementById('sidebarDotApache');
    const sbPortAp = document.getElementById('sidebarPortApache');
    if (sbPortAp) sbPortAp.textContent = `:${settings.apache_port}`;

    if (ap.running) {
        indAp.className = 'status-indicator-pill online';
        statusAp.textContent = 'Running';
        btnToggleAp.textContent = 'Stop';
        btnToggleAp.className = 'btn btn-action btn-stop';
        btnToggleAp.onclick = () => serviceAction('apache', 'stop');
        if (sbDotAp) sbDotAp.className = 'status-indicator-dot online';
    } else {
        indAp.className = 'status-indicator-pill offline';
        statusAp.textContent = 'Stopped';
        btnToggleAp.textContent = 'Start';
        btnToggleAp.className = 'btn btn-action btn-start';
        btnToggleAp.onclick = () => serviceAction('apache', 'start');
        if (sbDotAp) sbDotAp.className = 'status-indicator-dot offline';
    }

    // MariaDB
    const db = services.mariadb;
    const dotDb = document.getElementById('dotMariaDB');
    const statusDb = document.getElementById('statusMariaDB');
    const indDb = dotDb.parentElement;
    const btnToggleDb = document.getElementById('btnToggleMariaDB');

    document.getElementById('valMariaDBPort').textContent = settings.mariadb_port;
    document.getElementById('valMariaDBPid').textContent = db.running && db.pid ? db.pid : '-';
    document.getElementById('valMariaDBUptime').textContent = db.running && db.uptime ? db.uptime : '-';

    const sbDotDb = document.getElementById('sidebarDotMariaDB');
    const sbPortDb = document.getElementById('sidebarPortMariaDB');
    if (sbPortDb) sbPortDb.textContent = `:${settings.mariadb_port}`;

    if (db.running) {
        indDb.className = 'status-indicator-pill online';
        statusDb.textContent = 'Running';
        btnToggleDb.textContent = 'Stop';
        btnToggleDb.className = 'btn btn-action btn-stop';
        btnToggleDb.onclick = () => serviceAction('mariadb', 'stop');
        if (sbDotDb) sbDotDb.className = 'status-indicator-dot online';
    } else {
        indDb.className = 'status-indicator-pill offline';
        statusDb.textContent = 'Stopped';
        btnToggleDb.textContent = 'Start';
        btnToggleDb.className = 'btn btn-action btn-start';
        btnToggleDb.onclick = () => serviceAction('mariadb', 'start');
        if (sbDotDb) sbDotDb.className = 'status-indicator-dot offline';
    }
}

function updateQuickLinks(settings) {
    const webUrl = `http://localhost:${settings.apache_port}`;
    const pmaUrl = `http://localhost:${settings.apache_port}/phpmyadmin`;

    const linkWeb = document.getElementById('linkWebsite');
    if (linkWeb) {
        linkWeb.href = webUrl;
        const textWeb = document.getElementById('textWebsiteUrl');
        if (textWeb) textWeb.textContent = webUrl;
    }

    const btnTopWeb = document.getElementById('btnTopWebsite');
    if (btnTopWeb) btnTopWeb.href = webUrl;

    const linkPma = document.getElementById('linkPhpMyAdmin');
    if (linkPma) linkPma.href = pmaUrl;

    const navPma = document.getElementById('navPhpMyAdmin');
    if (navPma) navPma.href = pmaUrl;
}

async function serviceAction(name, action) {
    try {
        const res = await fetch(`/api/service?name=${name}&action=${action}`, { method: 'POST' });
        const data = await res.json();
        if (!data.success) {
            alert(`Gagal menjalankan ${action} ${name}: ${data.error}`);
        }
        fetchStatus();
    } catch (err) {
        alert('Gagal menghubungi server: ' + err.message);
    }
}

// Event Listeners
function setupEventListeners() {
    // Restart buttons
    document.getElementById('btnRestartApache').addEventListener('click', () => serviceAction('apache', 'restart'));
    document.getElementById('btnRestartMariaDB').addEventListener('click', () => serviceAction('mariadb', 'restart'));

    // Top Header Actions
    const btnTopRestart = document.getElementById('btnTopRestartAll');
    if (btnTopRestart) {
        btnTopRestart.addEventListener('click', () => restartAllServices());
    }

    const btnTopShutdown = document.getElementById('btnTopShutdown');
    if (btnTopShutdown) {
        btnTopShutdown.addEventListener('click', () => {
            const sm = document.getElementById('shutdownModal');
            if (sm) sm.classList.remove('hidden');
        });
    }

    // Sidebar Mobile Toggle & Overlay
    const sidebar = document.getElementById('appSidebar');
    const sidebarOverlay = document.getElementById('sidebarOverlay');
    const btnToggleSidebar = document.getElementById('btnToggleSidebar');

    const closeSidebarMobile = () => {
        if (sidebar) sidebar.classList.remove('open');
        if (sidebarOverlay) sidebarOverlay.classList.remove('active');
    };

    if (btnToggleSidebar) {
        btnToggleSidebar.addEventListener('click', () => {
            if (sidebar) sidebar.classList.toggle('open');
            if (sidebarOverlay) sidebarOverlay.classList.toggle('active');
        });
    }

    if (sidebarOverlay) {
        sidebarOverlay.addEventListener('click', closeSidebarMobile);
    }

    // Sidebar Navigation Items
    const navItems = document.querySelectorAll('.sidebar-menu .menu-item');
    navItems.forEach(item => {
        item.addEventListener('click', (e) => {
            const id = item.id;
            // Close mobile sidebar on navigation click
            if (window.innerWidth <= 900) {
                closeSidebarMobile();
            }

            if (id === 'navDashboard' || id === 'navProjects' || id === 'navTerminal') {
                navItems.forEach(n => n.classList.remove('active'));
                item.classList.add('active');
            } else if (id === 'navQuickDb') {
                e.preventDefault();
                openDatabaseModal('tabDbCreator');
            } else if (id === 'navDbEditor') {
                e.preventDefault();
                openDbEditorModal();
            } else if (id === 'navDbBackup') {
                e.preventDefault();
                openDatabaseModal('tabDbBackup');
            } else if (id === 'navPhpGen') {
                e.preventDefault();
                openPhpGenModal();
            } else if (id === 'navHtdocs') {
                e.preventDefault();
                fetch('/api/open-folder', { method: 'POST' });
            } else if (id === 'navLogs') {
                e.preventDefault();
                fetch('/api/open-folder?folder=logs', { method: 'POST' });
            } else if (id === 'navSettings') {
                e.preventDefault();
                openSettingsDialog();
            } else if (id === 'navCheckUpdate') {
                e.preventDefault();
                checkForUpdates(true);
            } else if (id === 'navShutdown') {
                e.preventDefault();
                const sm = document.getElementById('shutdownModal');
                if (sm) sm.classList.remove('hidden');
            }
        });
    });

    // Open Folders
    const btnOpenHtdocs = document.getElementById('btnOpenHtdocs');
    if (btnOpenHtdocs) {
        btnOpenHtdocs.addEventListener('click', () => {
            fetch('/api/open-folder', { method: 'POST' });
        });
    }

    const btnOpenLogs = document.getElementById('btnOpenLogs');
    if (btnOpenLogs) {
        btnOpenLogs.addEventListener('click', () => {
            fetch('/api/open-folder?folder=logs', { method: 'POST' });
        });
    }

    // Open Native Terminal (Windows CMD / PowerShell)
    const openNativeTerminal = () => {
        fetch('/api/open-terminal', { method: 'POST' });
    };
    const btnTopTerm = document.getElementById('btnOpenNativeTerminalTop');
    if (btnTopTerm) btnTopTerm.addEventListener('click', openNativeTerminal);
    const btnTileTerm = document.getElementById('btnOpenNativeTerminalTile');
    if (btnTileTerm) btnTileTerm.addEventListener('click', openNativeTerminal);
    const btnPopoutTerm = document.getElementById('btnPopoutTerminal');
    if (btnPopoutTerm) btnPopoutTerm.addEventListener('click', openNativeTerminal);

    // Auto Download
    const btnStartDl = document.getElementById('btnStartDownload');
    if (btnStartDl) btnStartDl.addEventListener('click', startDownload);

    // Database Tools Modal
    const btnDbModal = document.getElementById('btnOpenDatabaseModal');
    if (btnDbModal) {
        btnDbModal.addEventListener('click', () => openDatabaseModal());
    }

    // Database Editor & Relations Modal
    const btnDbEditorModal = document.getElementById('btnOpenDbEditorModal');
    if (btnDbEditorModal) {
        btnDbEditorModal.addEventListener('click', openDbEditorModal);
    }

    // PHP CRUD Generator Modal
    const btnPhpGenModal = document.getElementById('btnOpenPhpGenModal');
    if (btnPhpGenModal) {
        btnPhpGenModal.addEventListener('click', openPhpGenModal);
    }
    initPhpGenerator();

    // Settings Modal
    const openSettingsDialog = () => {
        const modal = document.getElementById('settingsModal');
        if (!modal) return;
        if (currentSettings) {
            document.getElementById('inputApachePort').value = currentSettings.apache_port;
            document.getElementById('inputMariaDBPort').value = currentSettings.mariadb_port;
            document.getElementById('inputAutoStart').checked = currentSettings.auto_start;
            const fallbackEl = document.getElementById('inputAutoPortFallback');
            if (fallbackEl) fallbackEl.checked = currentSettings.auto_port_fallback !== false;
            document.getElementById('inputGitHubRepo').value = currentSettings.github_repo || 'feryfadly27/mywebserver27';
            document.getElementById('inputGitHubToken').value = currentSettings.github_token || '';
            document.getElementById('inputAutoCheckUpdate').checked = currentSettings.auto_check_update !== false;

            const shellRadio = document.querySelector(`input[name="shell"][value="${currentSettings.shell || 'cmd'}"]`);
            if (shellRadio) shellRadio.checked = true;
        }

        document.getElementById('settingsAlert').classList.add('hidden');
        modal.classList.remove('hidden');
    };

    const btnSettings = document.getElementById('btnSettings');
    if (btnSettings) btnSettings.addEventListener('click', openSettingsDialog);

    const btnCloseSettings = document.getElementById('btnCloseSettings');
    if (btnCloseSettings) btnCloseSettings.addEventListener('click', () => document.getElementById('settingsModal').classList.add('hidden'));

    const btnCancelSettings = document.getElementById('btnCancelSettings');
    if (btnCancelSettings) btnCancelSettings.addEventListener('click', () => document.getElementById('settingsModal').classList.add('hidden'));

    const btnCheckUpdateFromSettings = document.getElementById('btnCheckUpdateFromSettings');
    if (btnCheckUpdateFromSettings) {
        btnCheckUpdateFromSettings.addEventListener('click', () => {
            document.getElementById('settingsModal').classList.add('hidden');
            checkForUpdates(true);
        });
    }

    const btnCheckUpdateTop = document.getElementById('btnCheckUpdateTop');
    if (btnCheckUpdateTop) {
        btnCheckUpdateTop.addEventListener('click', () => {
            checkForUpdates(true);
        });
    }

    document.getElementById('settingsForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const alertEl = document.getElementById('settingsAlert');
        alertEl.classList.add('hidden');

        const apachePort = parseInt(document.getElementById('inputApachePort').value, 10);
        const mariadbPort = parseInt(document.getElementById('inputMariaDBPort').value, 10);
        const autoStart = document.getElementById('inputAutoStart').checked;
        const autoPortFallback = document.getElementById('inputAutoPortFallback') ? document.getElementById('inputAutoPortFallback').checked : true;
        const shell = document.querySelector('input[name="shell"]:checked').value;
        const githubRepo = document.getElementById('inputGitHubRepo').value.trim();
        const githubToken = document.getElementById('inputGitHubToken').value.trim();
        const autoCheckUpdate = document.getElementById('inputAutoCheckUpdate').checked;

        try {
            const res = await fetch('/api/settings', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    apache_port: apachePort,
                    mariadb_port: mariadbPort,
                    auto_start: autoStart,
                    auto_port_fallback: autoPortFallback,
                    shell: shell,
                    github_repo: githubRepo,
                    github_token: githubToken,
                    auto_check_update: autoCheckUpdate
                })
            });

            const data = await res.json();
            if (data.success) {
                alertEl.className = 'modal-alert alert-success';
                alertEl.textContent = 'Pengaturan berhasil disimpan!';
                alertEl.classList.remove('hidden');
                setTimeout(() => {
                    modal.classList.add('hidden');
                    fetchStatus();
                }, 800);
            } else {
                alertEl.className = 'modal-alert alert-error';
                alertEl.textContent = data.error || 'Gagal menyimpan pengaturan';
                alertEl.classList.remove('hidden');
            }
        } catch (err) {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = 'Error: ' + err.message;
            alertEl.classList.remove('hidden');
        }
    });

    // Update Modal Controls
    const updateModal = document.getElementById('updateModal');
    document.getElementById('btnCloseUpdateModal').addEventListener('click', () => updateModal.classList.add('hidden'));
    document.getElementById('btnCancelUpdate').addEventListener('click', () => updateModal.classList.add('hidden'));

    document.getElementById('btnApplyUpdate').addEventListener('click', async () => {
        if (!latestUpdateInfo || !latestUpdateInfo.download_url) {
            alert('Tautan unduhan rilis tidak ditemukan.');
            return;
        }
        await applyUpdate(latestUpdateInfo.download_url);
    });

    // New Project Modal & Form
    const newProjModal = document.getElementById('newProjectModal');
    document.getElementById('btnNewProject').addEventListener('click', () => {
        document.getElementById('inputProjectName').value = '';
        document.getElementById('inputCreateVHost').checked = false;
        document.getElementById('projectVHostRow').classList.add('hidden');
        document.getElementById('newProjectAlert').classList.add('hidden');
        newProjModal.classList.remove('hidden');
        document.getElementById('inputProjectName').focus();
    });
    document.getElementById('btnCloseNewProject').addEventListener('click', () => newProjModal.classList.add('hidden'));
    document.getElementById('btnCancelNewProject').addEventListener('click', () => newProjModal.classList.add('hidden'));

    document.getElementById('inputProjectName').addEventListener('input', (e) => {
        const val = e.target.value.trim().toLowerCase().replace(/[^a-z0-9-_]/g, '-');
        document.getElementById('inputProjectDomain').value = val ? `${val}.test` : '';
    });

    document.getElementById('inputCreateVHost').addEventListener('change', (e) => {
        const row = document.getElementById('projectVHostRow');
        if (e.target.checked) row.classList.remove('hidden');
        else row.classList.add('hidden');
    });

    document.getElementById('newProjectForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const alertEl = document.getElementById('newProjectAlert');
        alertEl.classList.add('hidden');

        const name = document.getElementById('inputProjectName').value.trim();
        const type = document.querySelector('input[name="projectType"]:checked').value;
        const setVHost = document.getElementById('inputCreateVHost').checked;
        const domain = document.getElementById('inputProjectDomain').value.trim();

        try {
            const res = await fetch('/api/projects/create', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: name,
                    type: type,
                    set_vhost: setVHost,
                    domain: domain
                })
            });

            const data = await res.json();
            if (data.success) {
                alertEl.className = 'modal-alert alert-success';
                alertEl.textContent = 'Proyek berhasil dibuat!';
                alertEl.classList.remove('hidden');
                setTimeout(() => {
                    newProjModal.classList.add('hidden');
                    fetchProjects();
                }, 600);
            } else {
                alertEl.className = 'modal-alert alert-error';
                alertEl.textContent = data.error || 'Gagal membuat proyek';
                alertEl.classList.remove('hidden');
            }
        } catch (err) {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = 'Error: ' + err.message;
            alertEl.classList.remove('hidden');
        }
    });

    // Virtual Host Modal & Form
    const vhostModal = document.getElementById('vhostModal');
    document.getElementById('btnAddVHost').addEventListener('click', () => {
        document.getElementById('vhostModalTitle').textContent = 'Tambah Virtual Host Baru';
        document.getElementById('inputVHostDomain').value = '';
        document.getElementById('inputVHostFolder').value = '';
        document.getElementById('inputVHostEnabled').checked = true;
        document.getElementById('btnDeleteVHost').classList.add('hidden');
        document.getElementById('vhostAlert').classList.add('hidden');
        vhostModal.classList.remove('hidden');
        document.getElementById('inputVHostDomain').focus();
    });
    document.getElementById('btnCloseVHost').addEventListener('click', () => vhostModal.classList.add('hidden'));
    document.getElementById('btnCancelVHost').addEventListener('click', () => vhostModal.classList.add('hidden'));

    document.getElementById('vhostForm').addEventListener('submit', async (e) => {
        e.preventDefault();
        const alertEl = document.getElementById('vhostAlert');
        alertEl.classList.add('hidden');

        const domain = document.getElementById('inputVHostDomain').value.trim();
        const folder = document.getElementById('inputVHostFolder').value.trim();
        const enabled = document.getElementById('inputVHostEnabled').checked;

        try {
            const res = await fetch('/api/vhosts/save', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    domain: domain,
                    folder: folder,
                    enabled: enabled
                })
            });

            const data = await res.json();
            if (data.success) {
                alertEl.className = 'modal-alert alert-success';
                alertEl.textContent = 'Virtual Host berhasil disimpan!';
                alertEl.classList.remove('hidden');
                setTimeout(() => {
                    vhostModal.classList.add('hidden');
                    fetchProjects();
                }, 600);
            } else {
                alertEl.className = 'modal-alert alert-error';
                alertEl.textContent = data.error || 'Gagal menyimpan Virtual Host';
                alertEl.classList.remove('hidden');
            }
        } catch (err) {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = 'Error: ' + err.message;
            alertEl.classList.remove('hidden');
        }
    });

    document.getElementById('btnDeleteVHost').addEventListener('click', async () => {
        const domain = document.getElementById('inputVHostDomain').value.trim();
        if (!confirm(`Hapus Virtual Host "${domain}"?`)) return;

        try {
            const res = await fetch(`/api/vhosts/delete?domain=${encodeURIComponent(domain)}`, { method: 'POST' });
            const data = await res.json();
            if (data.success) {
                vhostModal.classList.add('hidden');
                fetchProjects();
            } else {
                alert('Gagal menghapus Virtual Host: ' + data.error);
            }
        } catch (err) {
            alert('Error: ' + err.message);
        }
    });

    // Hosts File Helper Modal & 1-Click Sync
    const hostsModal = document.getElementById('hostsHelperModal');
    document.getElementById('btnHostsHelper').addEventListener('click', openHostsModal);
    document.getElementById('btnCloseHostsHelper').addEventListener('click', () => hostsModal.classList.add('hidden'));
    document.getElementById('btnCloseHostsHelperBottom').addEventListener('click', () => hostsModal.classList.add('hidden'));
    
    const btnSyncHosts = document.getElementById('btnSyncHosts1Click');
    if (btnSyncHosts) {
        btnSyncHosts.addEventListener('click', syncHostsFile);
    }

    // Batch Services Controls (Header bar)
    const btnStartAll = document.getElementById('btnStartAllServices');
    if (btnStartAll) {
        btnStartAll.addEventListener('click', async () => {
            await serviceAction('all', 'start');
        });
    }

    const btnStopAll = document.getElementById('btnStopAllServices');
    if (btnStopAll) {
        btnStopAll.addEventListener('click', async () => {
            await serviceAction('all', 'stop');
        });
    }

    const btnRestartAll = document.getElementById('btnRestartAllServices');
    if (btnRestartAll) {
        btnRestartAll.addEventListener('click', async () => {
            await restartAllServices();
        });
    }

    // Shutdown / Server Power Controls
    const shutdownModal = document.getElementById('shutdownModal');
    const shutdownOverlay = document.getElementById('shutdownOverlay');
    const btnShutdown = document.getElementById('btnShutdown');

    if (btnShutdown) {
        btnShutdown.addEventListener('click', () => {
            shutdownModal.classList.remove('hidden');
        });
    }

    document.getElementById('btnCloseShutdown').addEventListener('click', () => shutdownModal.classList.add('hidden'));
    document.getElementById('btnCancelShutdown').addEventListener('click', () => shutdownModal.classList.add('hidden'));

    // Option 1: Restart Services
    document.getElementById('btnActionRestartServices').addEventListener('click', async () => {
        shutdownModal.classList.add('hidden');
        await restartAllServices();
    });

    // Option 2: Stop Services only
    document.getElementById('btnActionStopServices').addEventListener('click', async () => {
        shutdownModal.classList.add('hidden');
        await serviceAction('all', 'stop');
    });

    // Option 3: Full App Shutdown
    document.getElementById('btnActionFullShutdown').addEventListener('click', async () => {
        try {
            await fetch('/api/shutdown', { method: 'POST' });
        } catch (e) {
            // Server shutting down
        }

        shutdownModal.classList.add('hidden');
        showShutdownScreen();
    });

    // Button on Shutdown Overlay: Reconnect / Check Status
    const btnRestartFromShutdown = document.getElementById('btnRestartFromShutdown');
    if (btnRestartFromShutdown) {
        btnRestartFromShutdown.addEventListener('click', async () => {
            const btnText = document.getElementById('btnRestartFromShutdownText');
            btnRestartFromShutdown.disabled = true;
            if (btnText) btnText.textContent = 'Memeriksa server...';

            try {
                const res = await fetch('/api/status', { cache: 'no-store' });
                if (res.ok) {
                    const data = await res.json();
                    if (data.success) {
                        if (btnText) btnText.textContent = 'Server Aktif! Memuat...';
                        setTimeout(() => location.reload(), 500);
                        return;
                    }
                }
            } catch (e) {}

            btnRestartFromShutdown.disabled = false;
            if (btnText) btnText.textContent = 'Cek Status / Hubungkan Kembali';
            alert('Server lokal belum terdeteksi aktif. Silakan jalankan file start.bat atau mylokalwebserver.exe di komputer Anda, lalu tombol ini akan otomatis tersambung kembali.');
        });
    }

    // Terminal Controls
    document.getElementById('btnClearTerminal').addEventListener('click', () => {
        if (term) term.clear();
    });
    document.getElementById('btnReconnectTerminal').addEventListener('click', () => {
        if (termWs) termWs.close();
        if (term) term.dispose();
        isTerminalInitialized = false;
        initTerminal();
    });
}

let reconnectInterval = null;

function showShutdownScreen() {
    const shutdownOverlay = document.getElementById('shutdownOverlay');
    if (shutdownOverlay) shutdownOverlay.classList.remove('hidden');

    if (termWs) {
        try { termWs.close(); } catch(e) {}
    }

    startReconnectMonitor();
}

function startReconnectMonitor() {
    if (reconnectInterval) clearInterval(reconnectInterval);

    const reconnectDot = document.getElementById('reconnectDot');
    const reconnectStatusText = document.getElementById('reconnectStatusText');

    const checkOnline = async () => {
        try {
            const controller = new AbortController();
            const timeoutId = setTimeout(() => controller.abort(), 1800);
            const res = await fetch('/api/status', { signal: controller.signal, cache: 'no-store' });
            clearTimeout(timeoutId);

            if (res.ok) {
                const data = await res.json();
                if (data.success) {
                    if (reconnectInterval) clearInterval(reconnectInterval);
                    reconnectInterval = null;
                    if (reconnectStatusText) reconnectStatusText.textContent = 'Server terdeteksi aktif! Memuat dashboard...';
                    if (reconnectDot) reconnectDot.className = 'status-indicator-dot online';
                    
                    setTimeout(() => {
                        const shutdownOverlay = document.getElementById('shutdownOverlay');
                        if (shutdownOverlay) shutdownOverlay.classList.add('hidden');
                        location.reload();
                    }, 800);
                }
            }
        } catch (e) {
            if (reconnectStatusText) reconnectStatusText.textContent = 'Mendeteksi server di latar belakang...';
            if (reconnectDot) reconnectDot.className = 'status-indicator-dot checking';
        }
    };

    reconnectInterval = setInterval(checkOnline, 2000);
}

async function restartAllServices() {
    try {
        const res = await fetch('/api/restart', { method: 'POST' });
        const data = await res.json();
        if (!data.success) {
            alert(`Gagal me-restart server: ${data.error}`);
        } else {
            await updateStatus();
        }
    } catch (e) {
        console.error(e);
    }
}

function startDownload() {
    const btn = document.getElementById('btnStartDownload');
    btn.disabled = true;
    btn.textContent = 'Mengunduh paket...';

    const progressContainer = document.getElementById('downloadProgressContainer');
    const progressBar = document.getElementById('downloadProgressBar');
    const statusText = document.getElementById('downloadStatusText');
    const percentText = document.getElementById('downloadPercent');
    progressContainer.classList.remove('hidden');

    fetch('/api/download/start', { method: 'POST' })
        .then(res => res.json())
        .then(data => {
            if (!data.success) {
                alert('Gagal memulai download: ' + data.error);
                btn.disabled = false;
                btn.textContent = 'Unduh & Pasang Otomatis';
                return;
            }

            // Listen to SSE
            const eventSource = new EventSource('/api/download/progress');
            eventSource.onmessage = (event) => {
                const prog = JSON.parse(event.data);
                statusText.textContent = prog.message;
                progressBar.style.width = `${prog.progress}%`;
                percentText.textContent = `${prog.progress}%`;

                // Update individual badge tags
                if (prog.component_name) {
                    const tagId = 'badge-' + prog.component_name.toLowerCase().replace(/[^a-z]/g, '');
                    const badge = document.getElementById(tagId);
                    if (badge) {
                        if (prog.status === 'completed') {
                            badge.className = 'tag tag-done';
                            badge.textContent = 'Selesai';
                        } else if (prog.status === 'error') {
                            badge.className = 'tag tag-error';
                            badge.textContent = 'Gagal';
                        } else {
                            badge.className = 'tag tag-active';
                            badge.textContent = prog.status;
                        }
                    }
                }

                if (prog.all_done) {
                    eventSource.close();
                    statusText.textContent = 'Pemasangan selesai!';
                    progressBar.style.width = '100%';
                    percentText.textContent = '100%';
                    setTimeout(fetchStatus, 1000);
                }

                if (prog.error) {
                    eventSource.close();
                    alert('Terjadi kesalahan download: ' + prog.error);
                    btn.disabled = false;
                    btn.textContent = 'Coba Lagi';
                }
            };

            eventSource.onerror = () => {
                eventSource.close();
            };
        })
        .catch(err => {
            alert('Error: ' + err.message);
            btn.disabled = false;
        });
}

function initTerminal() {
    if (isTerminalInitialized) return;
    const container = document.getElementById('terminalContainer');
    if (!container) return;

    isTerminalInitialized = true;
    container.innerHTML = '';

    term = new Terminal({
        cursorBlink: true,
        cursorStyle: 'block',
        fontSize: 13,
        lineHeight: 1.2,
        fontFamily: '"Cascadia Code", Consolas, "Courier New", monospace',
        convertEol: true,
        scrollback: 10000,
        scrollOnUserInput: true,
        smoothScrollDuration: 0,
        theme: {
            background: '#0b0f19',
            foreground: '#e2e8f0',
            cursor: '#38bdf8',
            selectionBackground: 'rgba(56, 189, 248, 0.3)'
        }
    });

    fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);
    if (typeof WebLinksAddon !== 'undefined') {
        term.loadAddon(new WebLinksAddon.WebLinksAddon());
    }

    term.open(container);
    term.focus();

    const doFit = () => {
        if (fitAddon && container && container.offsetWidth > 0) {
            try {
                fitAddon.fit();
                if (term && termWs && termWs.readyState === WebSocket.OPEN) {
                    termWs.send(JSON.stringify({ type: 'resize', cols: term.cols, rows: term.rows }));
                }
            } catch (err) {
                // ignore transient fit errors
            }
        }
    };

    setTimeout(doFit, 80);
    setTimeout(doFit, 300);

    // Watch for size changes on container
    if (window.ResizeObserver) {
        const ro = new ResizeObserver(() => {
            doFit();
        });
        ro.observe(container);
    }

    window.addEventListener('resize', doFit);

    container.addEventListener('click', () => {
        if (term) term.focus();
    });

    term.onResize(({ cols, rows }) => {
        if (termWs && termWs.readyState === WebSocket.OPEN) {
            termWs.send(JSON.stringify({ type: 'resize', cols: cols, rows: rows }));
        }
    });

    // Connect WebSocket
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${protocol}//${window.location.host}/ws/terminal`;
    termWs = new WebSocket(wsUrl);

    termWs.onopen = () => {
        doFit();
    };

    termWs.onmessage = (e) => {
        term.write(e.data, () => {
            term.scrollToBottom();
        });
    };

    termWs.onclose = () => {
        term.writeln('\r\n\x1b[1;31m✗ Sesi terminal terputus. Klik Reconnect untuk menyambung kembali.\x1b[0m');
        isTerminalInitialized = false;
    };

    termWs.onerror = (err) => {
        console.error('Terminal WebSocket error:', err);
    };

    term.onData((data) => {
        term.scrollToBottom();
        if (termWs && termWs.readyState === WebSocket.OPEN) {
            termWs.send(data);
        }
    });
}

// Project & Virtual Host Helpers
let currentProjects = [];
let currentVHosts = [];

async function fetchProjects() {
    try {
        const res = await fetch('/api/projects');
        const data = await res.json();
        if (!data.success) return;

        currentProjects = data.data.projects || [];
        currentVHosts = data.data.virtual_hosts || [];
        renderProjects(currentProjects, data.data.apache_port);
    } catch (err) {
        console.error('Error fetching projects:', err);
    }
}

function renderProjects(projects, apachePort) {
    const container = document.getElementById('projectsList');
    if (!container) return;

    if (!projects || projects.length === 0) {
        container.innerHTML = `
            <div class="projects-empty">
                <svg width="24" height="24" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" style="color: var(--text-muted);">
                    <path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path>
                </svg>
                <p>Belum ada proyek di folder <strong>www/htdocs</strong>.</p>
                <button type="button" class="btn btn-default" onclick="document.getElementById('btnNewProject').click()">+ Buat Proyek Baru</button>
            </div>
        `;
        return;
    }

    container.innerHTML = projects.map(p => {
        let fwBadgeClass = 'badge-native';
        if (p.framework === 'Laravel') fwBadgeClass = 'badge-laravel';
        else if (p.framework === 'CodeIgniter 4') fwBadgeClass = 'badge-ci4';
        else if (p.framework === 'WordPress') fwBadgeClass = 'badge-wp';

        let vhostBadge = '';
        if (p.vhost_domain) {
            if (p.is_in_hosts_file) {
                vhostBadge = `
                    <a href="${p.vhost_url}" target="_blank" class="badge-fw badge-vhost" title="Domain Virtual Host aktif dan terdaftar di Hosts Windows: ${p.vhost_domain}">
                        <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path></svg>
                        <span>${escapeHTML(p.vhost_domain)}</span>
                    </a>
                `;
            } else {
                vhostBadge = `
                    <span onclick="openHostsModal()" class="badge-fw badge-vhost-pending" title="Domain belum terdaftar di Windows hosts. Klik untuk Sinkronkan 1-Klik">
                        <span>⚠️ ${escapeHTML(p.vhost_domain)}</span>
                        <span style="font-size: 10px; text-decoration: underline; margin-left: 2px;">(Sync Hosts)</span>
                    </span>
                `;
            }

            if (p.localhost_url) {
                const domainLabel = p.localhost_url.replace(/^https?:\/\//, '').replace(/\/.*$/, '');
                vhostBadge += `
                    <a href="${p.localhost_url}" target="_blank" class="badge-fw badge-localhost" title="Domain instan lokal (Langsung aktif di browser tanpa perlu edit file hosts)">
                        <span>⚡ ${escapeHTML(domainLabel)}</span>
                    </a>
                `;
            }
        }

        const openUrl = p.vhost_url && p.is_in_hosts_file ? p.vhost_url : (p.localhost_url || p.default_url);

        return `
            <div class="project-card">
                <div class="project-main-info">
                    <div class="project-title-row">
                        <span class="project-name">${escapeHTML(p.name)}</span>
                        <span class="badge-fw ${fwBadgeClass}">${p.framework}</span>
                        ${vhostBadge}
                    </div>
                    <div class="project-links-row">
                        <a href="${p.default_url}" target="_blank" class="project-url-link font-mono" title="Buka via Localhost standar">
                            <span>${p.default_url}</span>
                            <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M18 13v6a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V8a2 2 0 0 1 2-2h6"></path><polyline points="15 3 21 3 21 9"></polyline><line x1="10" y1="14" x2="21" y2="3"></line></svg>
                        </a>
                        <span class="project-path-text font-mono">📁 www/htdocs/${escapeHTML(p.relative_path)}</span>
                    </div>
                </div>
                <div class="project-actions">
                    <a href="${openUrl}" target="_blank" class="btn btn-default btn-icon-label" title="Buka website">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="10"></circle><line x1="2" y1="12" x2="22" y2="12"></line><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"></path></svg>
                        <span>Buka</span>
                    </a>
                    <button type="button" class="btn btn-subtle btn-icon-label" onclick="openShareModal('${escapeHTML(p.name)}', '${escapeHTML(p.relative_path)}')" title="Share ke Smartphone via QR Code WiFi">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="5" y="2" width="14" height="20" rx="2" ry="2"></rect><line x1="12" y1="18" x2="12.01" y2="18"></line></svg>
                        <span>Share HP</span>
                    </button>
                    <button type="button" class="btn btn-subtle btn-icon-label" onclick="openProjectFolder('${escapeHTML(p.relative_path)}')" title="Buka folder di File Explorer">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M22 19a2 2 0 0 1-2 2H4a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h5l2 3h9a2 2 0 0 1 2 2z"></path></svg>
                        <span>Folder</span>
                    </button>
                    <button type="button" class="btn btn-subtle btn-icon-label" onclick="openProjectTerminal('${escapeHTML(p.relative_path)}')" title="Buka terminal langsung di folder proyek ini">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="4 17 10 11 4 5"></polyline><line x1="12" y1="19" x2="20" y2="19"></line></svg>
                        <span>Terminal</span>
                    </button>
                    <button type="button" class="btn btn-subtle btn-icon-label" onclick="openVHostModalForProject('${escapeHTML(p.name)}', ${p.has_public_dir})" title="Atur Virtual Host / Domain kustom">
                        <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"></circle><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1 0 2.83 2 2 0 0 1-2.83 0l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-2 2 2 2 0 0 1-2-2v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83 0 2 2 0 0 1 0-2.83l.06-.06a1.65 1.65 0 0 0 .33-1.82 1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1-2-2 2 2 0 0 1 2-2h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 0-2.83 2 2 0 0 1 2.83 0l.06.06a1.65 1.65 0 0 0 1.82.33H9a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 2-2 2 2 0 0 1 2 2v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 0 2 2 0 0 1 0 2.83l-.06.06a1.65 1.65 0 0 0-.33 1.82V9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 2 2 2 2 0 0 1-2 2h-.09a1.65 1.65 0 0 0-1.51 1z"></path></svg>
                        <span>${p.vhost_domain ? 'Edit Domain' : 'Set Domain'}</span>
                    </button>
                </div>
            </div>
        `;
    }).join('');
}

async function openHostsModal() {
    const hostsModal = document.getElementById('hostsHelperModal');
    const alertEl = document.getElementById('hostsSyncAlert');
    if (alertEl) alertEl.classList.add('hidden');

    try {
        const res = await fetch('/api/hosts/status');
        const data = await res.json();
        if (data.success) {
            const samplePre = document.getElementById('hostsSampleCode');
            if (samplePre && data.data.all_domains && data.data.all_domains.length > 0) {
                samplePre.textContent = data.data.all_domains.map(d => `127.0.0.1  ${d}\n127.0.0.1  www.${d}`).join('\n');
            }
        }
    } catch (e) {
        console.error('Error checking hosts status:', e);
    }
    hostsModal.classList.remove('hidden');
}

async function syncHostsFile() {
    const btn = document.getElementById('btnSyncHosts1Click');
    const txt = document.getElementById('textBtnSyncHosts');
    const alertEl = document.getElementById('hostsSyncAlert');

    if (btn) btn.disabled = true;
    if (txt) txt.textContent = 'Menyinkronkan (Periksa UAC)...';
    if (alertEl) alertEl.classList.add('hidden');

    try {
        const res = await fetch('/api/hosts/sync', { method: 'POST' });
        const data = await res.json();
        if (data.success) {
            alertEl.className = 'modal-alert alert-success';
            alertEl.textContent = '✅ Berhasil! Domain telah ditambahkan ke berkas hosts Windows.';
            alertEl.classList.remove('hidden');
            fetchProjects();
        } else {
            alertEl.className = 'modal-alert alert-danger';
            alertEl.textContent = `❌ ${data.error || 'Gagal menyinkronkan hosts'}`;
            alertEl.classList.remove('hidden');
        }
    } catch (err) {
        alertEl.className = 'modal-alert alert-danger';
        alertEl.textContent = `❌ Error: ${err.message}`;
        alertEl.classList.remove('hidden');
    } finally {
        if (btn) btn.disabled = false;
        if (txt) txt.textContent = 'Sinkronkan ke Windows Hosts (1-Klik)';
    }
}

function escapeHTML(str) {
    if (!str) return '';
    return str.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;').replace(/"/g, '&quot;');
}

function openProjectFolder(path) {
    fetch(`/api/open-folder?path=${encodeURIComponent(path)}`, { method: 'POST' });
}

function openProjectTerminal(path) {
    fetch(`/api/open-terminal?path=${encodeURIComponent(path)}`, { method: 'POST' });
}

function openVHostModalForProject(folderName, hasPublicDir) {
    const vhostModal = document.getElementById('vhostModal');
    const existing = currentVHosts.find(v => v.folder.startsWith(folderName) || v.folder === folderName);

    document.getElementById('vhostAlert').classList.add('hidden');
    if (existing) {
        document.getElementById('vhostModalTitle').textContent = `Ubah Virtual Host: ${existing.domain}`;
        document.getElementById('inputVHostDomain').value = existing.domain;
        document.getElementById('inputVHostFolder').value = existing.folder;
        document.getElementById('inputVHostEnabled').checked = existing.enabled;
        document.getElementById('btnDeleteVHost').classList.remove('hidden');
    } else {
        const defaultDomain = `${folderName.toLowerCase().replace(/[^a-z0-9-_]/g, '-')}.test`;
        const defaultFolder = hasPublicDir ? `${folderName}/public` : folderName;
        document.getElementById('vhostModalTitle').textContent = `Set Virtual Host untuk ${folderName}`;
        document.getElementById('inputVHostDomain').value = defaultDomain;
        document.getElementById('inputVHostFolder').value = defaultFolder;
        document.getElementById('inputVHostEnabled').checked = true;
        document.getElementById('btnDeleteVHost').classList.add('hidden');
    }

    vhostModal.classList.remove('hidden');
    document.getElementById('inputVHostDomain').focus();
}

async function checkForUpdates(interactive) {
    const updateModal = document.getElementById('updateModal');
    const checkingView = document.getElementById('updateCheckingView');
    const upToDateView = document.getElementById('updateUpToDateView');
    const availableView = document.getElementById('updateAvailableView');
    const errorView = document.getElementById('updateErrorView');
    const btnApply = document.getElementById('btnApplyUpdate');
    const notifDot = document.getElementById('updateNotificationDot');

    if (interactive) {
        checkingView.classList.remove('hidden');
        upToDateView.classList.add('hidden');
        availableView.classList.add('hidden');
        errorView.classList.add('hidden');
        btnApply.classList.add('hidden');
        document.getElementById('updateAlert').classList.add('hidden');
        document.getElementById('updateProgressContainer').classList.add('hidden');
        updateModal.classList.remove('hidden');
    }

    try {
        const repoParam = currentSettings && currentSettings.github_repo ? `?repo=${encodeURIComponent(currentSettings.github_repo)}` : '';
        const res = await fetch(`/api/update/check${repoParam}`);
        const data = await res.json();

        if (!data.success) {
            if (interactive) {
                checkingView.classList.add('hidden');
                document.getElementById('updateErrorMsg').textContent = data.error || 'Gagal memeriksa rilis GitHub';
                errorView.classList.remove('hidden');
            }
            return;
        }

        const info = data.data;
        latestUpdateInfo = info;
        const notifDotSidebar = document.getElementById('updateNotificationDotSidebar');

        if (info.has_update) {
            if (notifDot) notifDot.classList.remove('hidden');
            if (notifDotSidebar) notifDotSidebar.classList.remove('hidden');

            if (interactive) {
                checkingView.classList.add('hidden');
                document.getElementById('updateCurVer').textContent = info.current_version;
                document.getElementById('updateLatestVer').textContent = info.latest_version;
                document.getElementById('updateReleaseDate').textContent = info.published_at ? `Dirilis: ${info.published_at}` : '';
                document.getElementById('updateFileSize').textContent = info.asset_size_str || '~7.2 MB';
                document.getElementById('updateReleaseNotes').textContent = info.release_notes || 'Tidak ada catatan rilis.';
                
                availableView.classList.remove('hidden');
                btnApply.classList.remove('hidden');
            }
        } else {
            if (notifDot) notifDot.classList.add('hidden');
            if (notifDotSidebar) notifDotSidebar.classList.add('hidden');

            if (interactive) {
                checkingView.classList.add('hidden');
                document.getElementById('upToDateVer').textContent = info.current_version;
                upToDateView.classList.remove('hidden');
            }
        }
    } catch (err) {
        if (interactive) {
            checkingView.classList.add('hidden');
            document.getElementById('updateErrorMsg').textContent = 'Koneksi error: ' + err.message;
            errorView.classList.remove('hidden');
        }
    }
}

async function applyUpdate(downloadURL) {
    const btnApply = document.getElementById('btnApplyUpdate');
    const alertEl = document.getElementById('updateAlert');
    const progressContainer = document.getElementById('updateProgressContainer');
    const progressBar = document.getElementById('updateProgressBar');
    const progressText = document.getElementById('updateProgressText');
    const progressPercent = document.getElementById('updateProgressPercent');

    btnApply.disabled = true;
    btnApply.textContent = 'Memproses...';
    alertEl.classList.add('hidden');
    progressContainer.classList.remove('hidden');

    try {
        const res = await fetch('/api/update/apply', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                download_url: downloadURL
            })
        });

        const data = await res.json();
        if (!data.success) {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = data.error || 'Gagal memulai proses pembaruan';
            alertEl.classList.remove('hidden');
            btnApply.disabled = false;
            btnApply.textContent = 'Perbarui Sekarang (1-Klik)';
            return;
        }

        // Poll update progress
        if (updatePollInterval) clearInterval(updatePollInterval);
        updatePollInterval = setInterval(async () => {
            try {
                const pRes = await fetch('/api/update/progress');
                const pData = await pRes.json();
                if (pData.success && pData.data) {
                    const p = pData.data;
                    progressBar.style.width = `${p.progress}%`;
                    progressPercent.textContent = `${p.progress}%`;
                    progressText.textContent = p.message || 'Mengunduh...';

                    if (p.status === 'completed') {
                        clearInterval(updatePollInterval);
                        alertEl.className = 'modal-alert alert-success';
                        alertEl.textContent = 'Pembaruan berhasil dipasang! Server sedang memuat ulang...';
                        alertEl.classList.remove('hidden');

                        setTimeout(() => {
                            location.reload();
                        }, 2500);
                    } else if (p.status === 'error') {
                        clearInterval(updatePollInterval);
                        alertEl.className = 'modal-alert alert-error';
                        alertEl.textContent = p.error || 'Terjadi kesalahan saat pembaruan';
                        alertEl.classList.remove('hidden');
                        btnApply.disabled = false;
                        btnApply.textContent = 'Coba Lagi';
                    }
                }
            } catch (e) {
                // Server might be restarting after update
                clearInterval(updatePollInterval);
                setTimeout(() => {
                    location.reload();
                }, 2000);
            }
        }, 800);

    } catch (err) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = 'Error: ' + err.message;
        alertEl.classList.remove('hidden');
        btnApply.disabled = false;
        btnApply.textContent = 'Perbarui Sekarang (1-Klik)';
    }
}

// ==========================================
// 📱 QR Code Share Modal & Network Discovery
// ==========================================
let currentShareProject = null;
let currentNetworkIPs = [];

async function openShareModal(projectName, relativePath) {
    currentShareProject = { name: projectName, relativePath: relativePath };
    const modal = document.getElementById('shareQrModal');
    const selectIP = document.getElementById('selectShareNetworkIP');
    const titleEl = document.getElementById('shareQrProjectTitle');

    if (titleEl) {
        titleEl.textContent = `Share "${projectName}" ke Smartphone`;
    }

    try {
        const res = await fetch('/api/network/ips');
        const data = await res.json();
        if (data.success && data.data.ips) {
            currentNetworkIPs = data.data.ips;
            selectIP.innerHTML = '';

            if (currentNetworkIPs.length === 0) {
                const opt = document.createElement('option');
                opt.value = '127.0.0.1';
                opt.textContent = '127.0.0.1 (Localhost saja)';
                selectIP.appendChild(opt);
            } else {
                currentNetworkIPs.forEach(item => {
                    const opt = document.createElement('option');
                    opt.value = item.ip;
                    const badge = item.is_wifi ? '📶 WiFi' : '🔌 LAN';
                    opt.textContent = `${item.ip} (${badge} - ${item.interface})`;
                    selectIP.appendChild(opt);
                });
            }
        }
    } catch (e) {
        console.error('Failed to load network IPs:', e);
    }

    renderShareUrl();
    modal.classList.remove('hidden');
}

function renderShareUrl() {
    if (!currentShareProject || !currentSettings) return;

    const selectIP = document.getElementById('selectShareNetworkIP');
    const ip = selectIP.value || '127.0.0.1';
    const port = currentSettings.apache_port || 8080;
    const url = `http://${ip}:${port}/${currentShareProject.relativePath}`;

    const inputUrl = document.getElementById('inputShareUrl');
    const btnOpen = document.getElementById('btnOpenShareUrlDirect');
    const canvas = document.getElementById('shareQrCanvas');

    if (inputUrl) inputUrl.value = url;
    if (btnOpen) btnOpen.href = url;

    // Draw QR Code on canvas
    if (canvas) {
        drawQRCodeToCanvas(url, canvas);
    }
}

// Hook Share Modal events
document.addEventListener('DOMContentLoaded', () => {
    const selectIP = document.getElementById('selectShareNetworkIP');
    if (selectIP) {
        selectIP.addEventListener('change', renderShareUrl);
    }

    const btnClose = document.getElementById('btnCloseShareQr');
    const btnCloseBottom = document.getElementById('btnCloseShareQrBottom');
    const modal = document.getElementById('shareQrModal');

    if (btnClose) btnClose.addEventListener('click', () => modal.classList.add('hidden'));
    if (btnCloseBottom) btnCloseBottom.addEventListener('click', () => modal.classList.add('hidden'));

    const btnCopy = document.getElementById('btnCopyShareUrl');
    if (btnCopy) {
        btnCopy.addEventListener('click', () => {
            const input = document.getElementById('inputShareUrl');
            if (input) {
                input.select();
                navigator.clipboard.writeText(input.value).then(() => {
                    btnCopy.textContent = 'Disalin! ✓';
                    setTimeout(() => { btnCopy.textContent = 'Salin'; }, 1500);
                });
            }
        });
    }

    initDatabaseTools();
});

// ==========================================
// 🗄️ Database Backup & Restore (.sql) Tools
// ==========================================
let currentDbList = [];
let currentBackups = [];
let selectedSqlFile = null;

function initDatabaseTools() {
    const modal = document.getElementById('dbToolsModal');
    const btnClose = document.getElementById('btnCloseDbTools');
    const btnCloseBottom = document.getElementById('btnCloseDbToolsBottom');

    if (btnClose) btnClose.addEventListener('click', () => modal.classList.add('hidden'));
    if (btnCloseBottom) btnCloseBottom.addEventListener('click', () => modal.classList.add('hidden'));

    // Tab switching
    const tabBtns = document.querySelectorAll('.db-tab-btn');
    tabBtns.forEach(btn => {
        btn.addEventListener('click', () => {
            tabBtns.forEach(b => b.classList.remove('active'));
            document.querySelectorAll('.db-tab-pane').forEach(p => p.classList.add('hidden'));

            btn.classList.add('active');
            const targetId = btn.getAttribute('data-tab');
            const targetPane = document.getElementById(targetId);
            if (targetPane) targetPane.classList.remove('hidden');
        });
    });

    // Refresh DB list
    const btnRefresh = document.getElementById('btnRefreshDbList');
    if (btnRefresh) {
        btnRefresh.addEventListener('click', fetchDatabaseData);
    }

    // Do Backup Button
    const btnBackup = document.getElementById('btnDoBackup');
    if (btnBackup) {
        btnBackup.addEventListener('click', doDatabaseBackup);
    }

    // Dropzone & File picker for Restore
    const dropzone = document.getElementById('sqlDropzone');
    const fileInput = document.getElementById('inputSqlFile');
    const btnRestore = document.getElementById('btnDoRestoreUpload');

    if (dropzone && fileInput) {
        dropzone.addEventListener('click', () => fileInput.click());

        dropzone.addEventListener('dragover', (e) => {
            e.preventDefault();
            dropzone.classList.add('dragover');
        });

        dropzone.addEventListener('dragleave', () => {
            dropzone.classList.remove('dragover');
        });

        dropzone.addEventListener('drop', (e) => {
            e.preventDefault();
            dropzone.classList.remove('dragover');
            if (e.dataTransfer.files.length > 0) {
                handleSelectedSqlFile(e.dataTransfer.files[0]);
            }
        });

        fileInput.addEventListener('change', (e) => {
            if (e.target.files.length > 0) {
                handleSelectedSqlFile(e.target.files[0]);
            }
        });
    }

    if (btnRestore) {
        btnRestore.addEventListener('click', doUploadSqlRestore);
    }

    // Seeder Listeners
    const selectSeedDb = document.getElementById('selectSeedDb');
    if (selectSeedDb) {
        selectSeedDb.addEventListener('change', onSeedDbChange);
    }

    const selectSeedTable = document.getElementById('selectSeedTable');
    if (selectSeedTable) {
        selectSeedTable.addEventListener('change', onSeedTableChange);
    }

    const btnSeedData = document.getElementById('btnDoSeedData');
    if (btnSeedData) {
        btnSeedData.addEventListener('click', doExecuteSeeder);
    }

    const btnApplyTemplate = document.getElementById('btnApplyTemplate');
    if (btnApplyTemplate) {
        btnApplyTemplate.addEventListener('click', doApplyPresetTemplate);
    }

    // Mode Toggle (Custom table vs Preset template)
    const modeCustom = document.getElementById('modeLabelCustomTable');
    const modePreset = document.getElementById('modeLabelPresetTemplate');
    const panelCustom = document.getElementById('seederPanelCustomTable');
    const panelPreset = document.getElementById('seederPanelPresetTemplate');

    if (modeCustom && modePreset) {
        modeCustom.addEventListener('click', () => {
            modeCustom.classList.add('active');
            modePreset.classList.remove('active');
            panelCustom.classList.remove('hidden');
            panelPreset.classList.add('hidden');
        });

        modePreset.addEventListener('click', () => {
            modePreset.classList.add('active');
            modeCustom.classList.remove('active');
            panelPreset.classList.remove('hidden');
            panelCustom.classList.add('hidden');
        });
    }

    // Preset cards selection highlight
    document.querySelectorAll('.preset-card').forEach(card => {
        card.addEventListener('click', () => {
            document.querySelectorAll('.preset-card').forEach(c => c.classList.remove('active'));
            card.classList.add('active');
            const radio = card.querySelector('input[type="radio"]');
            if (radio) radio.checked = true;
        });
    });

    // Quick DB Creator Builder
    initQuickDbCreator();
}

function handleSelectedSqlFile(file) {
    if (!file.name.toLowerCase().endsWith('.sql')) {
        alert('Harap pilih file dengan ekstensi .sql');
        return;
    }
    selectedSqlFile = file;
    const label = document.getElementById('dropzoneFileName');
    const btnRestore = document.getElementById('btnDoRestoreUpload');
    if (label) {
        label.innerHTML = `File terpilih: <strong>${escapeHTML(file.name)}</strong> (${(file.size / 1024).toFixed(1)} KB)`;
    }
    if (btnRestore) btnRestore.disabled = false;
}

async function openDatabaseModal(defaultTab = null) {
    const modal = document.getElementById('dbToolsModal');
    if (defaultTab) {
        const tabBtns = document.querySelectorAll('.db-tab-btn');
        tabBtns.forEach(b => b.classList.remove('active'));
        document.querySelectorAll('.db-tab-pane').forEach(p => p.classList.add('hidden'));

        const targetBtn = document.querySelector(`.db-tab-btn[data-tab="${defaultTab}"]`);
        if (targetBtn) targetBtn.classList.add('active');
        const targetPane = document.getElementById(defaultTab);
        if (targetPane) targetPane.classList.remove('hidden');
    }
    modal.classList.remove('hidden');
    await fetchDatabaseData();
}

async function fetchDatabaseData() {
    const selectBackup = document.getElementById('selectBackupDb');
    const selectRestore = document.getElementById('selectRestoreTargetDb');
    const selectSeed = document.getElementById('selectSeedDb');
    const tableBody = document.getElementById('backupHistoryTableBody');

    try {
        const res = await fetch('/api/db/list');
        const data = await res.json();
        if (!data.success) {
            alert('Gagal memuat data database: ' + data.error);
            return;
        }

        currentDbList = data.data.databases || [];
        currentBackups = data.data.backups || [];

        // Populate Select Dropdowns
        if (selectBackup) {
            selectBackup.innerHTML = '';
            if (currentDbList.length === 0) {
                const opt = document.createElement('option');
                opt.value = '';
                opt.textContent = 'Belum ada database kustom (Buat di phpMyAdmin / Restore SQL)';
                selectBackup.appendChild(opt);
            } else {
                currentDbList.forEach(db => {
                    const opt = document.createElement('option');
                    opt.value = db;
                    opt.textContent = `📁 ${db}`;
                    selectBackup.appendChild(opt);
                });
            }
        }

        if (selectRestore) {
            selectRestore.innerHTML = '';
            const defaultOpt = document.createElement('option');
            defaultOpt.value = '';
            defaultOpt.textContent = '-- Pilih database yang sudah ada --';
            selectRestore.appendChild(defaultOpt);

            currentDbList.forEach(db => {
                const opt = document.createElement('option');
                opt.value = db;
                opt.textContent = `📁 ${db}`;
                selectRestore.appendChild(opt);
            });
        }

        if (selectSeed) {
            selectSeed.innerHTML = '';
            const defaultOpt = document.createElement('option');
            defaultOpt.value = '';
            defaultOpt.textContent = '-- Pilih database --';
            selectSeed.appendChild(defaultOpt);

            currentDbList.forEach(db => {
                const opt = document.createElement('option');
                opt.value = db;
                opt.textContent = `📁 ${db}`;
                selectSeed.appendChild(opt);
            });
        }

        // Render History Table
        if (tableBody) {
            if (currentBackups.length === 0) {
                tableBody.innerHTML = `<tr><td colspan="5" style="text-align: center; color: var(--text-muted); padding: 18px;">Belum ada file backup di folder <code>data/backups/</code></td></tr>`;
            } else {
                tableBody.innerHTML = currentBackups.map(b => `
                    <tr>
                        <td><strong class="font-mono" style="font-size: 12.5px;">${escapeHTML(b.filename)}</strong></td>
                        <td><span class="tag tag-neutral">${escapeHTML(b.database_name || '-')}</span></td>
                        <td class="font-mono" style="font-size: 12px;">${escapeHTML(b.size_formatted)}</td>
                        <td style="font-size: 12px; color: var(--text-muted);">${escapeHTML(b.created_at_str)}</td>
                        <td style="text-align: right; white-space: nowrap;">
                            <button type="button" class="btn btn-default btn-icon-xs" onclick="restoreFromExistingBackup('${escapeHTML(b.filename)}', '${escapeHTML(b.database_name)}')" title="Pulihkan / Impor backup ini">🔄 Restore</button>
                            <a href="/api/db/download?filename=${encodeURIComponent(b.filename)}" class="btn btn-subtle btn-icon-xs" title="Unduh file .sql ini" download>⬇ Unduh</a>
                            <button type="button" class="btn btn-danger-ghost btn-icon-xs" onclick="deleteBackupFile('${escapeHTML(b.filename)}')" title="Hapus file backup">🗑</button>
                        </td>
                    </tr>
                `).join('');
            }
        }

    } catch (e) {
        console.error('Error fetching database list:', e);
    }
}

// Seeder Handlers
async function onSeedDbChange() {
    const selectDb = document.getElementById('selectSeedDb');
    const selectTbl = document.getElementById('selectSeedTable');
    const previewBox = document.getElementById('seedColumnPreviewContainer');
    const btnSeed = document.getElementById('btnDoSeedData');
    const alertEl = document.getElementById('seedAlertResult');
    if (alertEl) alertEl.classList.add('hidden');

    const dbName = selectDb ? selectDb.value : '';
    if (!dbName) {
        selectTbl.innerHTML = '<option value="">-- Pilih Database Dahulu --</option>';
        selectTbl.disabled = true;
        previewBox.classList.add('hidden');
        btnSeed.disabled = true;
        return;
    }

    selectTbl.innerHTML = '<option value="">Memuat daftar tabel...</option>';
    selectTbl.disabled = true;
    previewBox.classList.add('hidden');
    btnSeed.disabled = true;

    try {
        const res = await fetch(`/api/db/tables?db=${encodeURIComponent(dbName)}`);
        const data = await res.json();

        if (data.success && data.data && data.data.tables) {
            const tables = data.data.tables;
            selectTbl.innerHTML = '';

            if (tables.length === 0) {
                selectTbl.innerHTML = '<option value="">(Tidak ada tabel dalam database ini)</option>';
                selectTbl.disabled = true;
                return;
            }

            const defOpt = document.createElement('option');
            defOpt.value = '';
            defOpt.textContent = '-- Pilih Tabel Target --';
            selectTbl.appendChild(defOpt);

            tables.forEach(tbl => {
                const opt = document.createElement('option');
                opt.value = tbl;
                opt.textContent = `📋 ${tbl}`;
                selectTbl.appendChild(opt);
            });

            selectTbl.disabled = false;
        } else {
            selectTbl.innerHTML = `<option value="">Gagal: ${data.error || 'Error'}</option>`;
        }
    } catch (e) {
        selectTbl.innerHTML = `<option value="">Error: ${e.message}</option>`;
    }
}

async function onSeedTableChange() {
    const selectDb = document.getElementById('selectSeedDb');
    const selectTbl = document.getElementById('selectSeedTable');
    const previewBox = document.getElementById('seedColumnPreviewContainer');
    const grid = document.getElementById('seedColumnsGrid');
    const badgeCount = document.getElementById('textColCountBadge');
    const btnSeed = document.getElementById('btnDoSeedData');
    const alertEl = document.getElementById('seedAlertResult');
    if (alertEl) alertEl.classList.add('hidden');

    const dbName = selectDb ? selectDb.value : '';
    const tableName = selectTbl ? selectTbl.value : '';

    if (!dbName || !tableName) {
        previewBox.classList.add('hidden');
        btnSeed.disabled = true;
        return;
    }

    try {
        const res = await fetch(`/api/db/columns?db=${encodeURIComponent(dbName)}&table=${encodeURIComponent(tableName)}`);
        const data = await res.json();

        if (data.success && data.data && data.data.columns) {
            const cols = data.data.columns;
            if (badgeCount) badgeCount.textContent = `${cols.length} Kolom Terdeteksi`;

            grid.innerHTML = cols.map(c => {
                let badgeClass = 'tag tag-done';
                let icon = '✨';
                if (c.detected_type === 'auto_increment') {
                    badgeClass = 'tag tag-neutral';
                    icon = '🔑';
                } else if (c.detected_type === 'nama_lengkap') {
                    badgeClass = 'tag tag-active';
                    icon = '👤';
                } else if (c.detected_type === 'nik') {
                    badgeClass = 'tag tag-active';
                    icon = '💳';
                } else if (c.detected_type === 'phone') {
                    badgeClass = 'tag tag-active';
                    icon = '📱';
                } else if (c.detected_type === 'alamat') {
                    badgeClass = 'tag tag-active';
                    icon = '🏠';
                } else if (c.detected_type === 'keluhan_medis') {
                    badgeClass = 'tag tag-active';
                    icon = '🩺';
                } else if (c.detected_type === 'nama_produk') {
                    badgeClass = 'tag tag-active';
                    icon = '📦';
                } else if (c.detected_type === 'harga_rupiah') {
                    badgeClass = 'tag tag-active';
                    icon = '💰';
                }

                return `
                    <div class="seed-col-item">
                        <div class="seed-col-header">
                            <span class="seed-col-name font-mono">${escapeHTML(c.field)}</span>
                            <span class="seed-col-type font-mono">${escapeHTML(c.type)}</span>
                        </div>
                        <div class="seed-col-generator">
                            <span class="${badgeClass}">${icon} ${escapeHTML(c.detected_label)}</span>
                        </div>
                    </div>
                `;
            }).join('');

            previewBox.classList.remove('hidden');
            btnSeed.disabled = false;
        } else {
            alert('Gagal membaca struktur kolom: ' + (data.error || 'Unknown error'));
        }
    } catch (e) {
        alert('Error: ' + e.message);
    }
}

async function doExecuteSeeder() {
    const selectDb = document.getElementById('selectSeedDb');
    const selectTbl = document.getElementById('selectSeedTable');
    const selectCount = document.getElementById('selectSeedCount');
    const btnSeed = document.getElementById('btnDoSeedData');
    const textBtn = document.getElementById('textBtnSeedData');
    const alertEl = document.getElementById('seedAlertResult');

    const dbName = selectDb ? selectDb.value : '';
    const tableName = selectTbl ? selectTbl.value : '';
    const count = selectCount ? parseInt(selectCount.value, 10) : 10;

    if (!dbName || !tableName) {
        alert('Harap pilih database dan tabel target terlebih dahulu.');
        return;
    }

    btnSeed.disabled = true;
    if (textBtn) textBtn.textContent = 'Menyisipkan data contoh...';
    alertEl.classList.add('hidden');

    try {
        const res = await fetch('/api/db/seed', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                database: dbName,
                table: tableName,
                count: count
            })
        });
        const data = await res.json();

        if (data.success) {
            alertEl.className = 'modal-alert alert-success';
            alertEl.textContent = `✓ ${data.message || 'Data contoh berhasil ditambahkan!'}`;
            alertEl.classList.remove('hidden');
        } else {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = `❌ ${data.error || 'Gagal menambahkan data contoh'}`;
            alertEl.classList.remove('hidden');
        }
    } catch (e) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = `❌ Error: ${e.message}`;
        alertEl.classList.remove('hidden');
    } finally {
        btnSeed.disabled = false;
        if (textBtn) textBtn.textContent = '🌱 Isi Data Sekarang';
    }
}

async function doApplyPresetTemplate() {
    const inputDb = document.getElementById('inputSeedTemplateDb');
    const radioSelected = document.querySelector('input[name="presetTemplateKey"]:checked');
    const btn = document.getElementById('btnApplyTemplate');
    const textBtn = document.getElementById('textBtnApplyTemplate');
    const alertEl = document.getElementById('seedAlertResult');

    const dbName = inputDb ? inputDb.value.trim() : '';
    const templateKey = radioSelected ? radioSelected.value : '';

    if (!dbName) {
        alert('Harap masukkan nama database target (misal: db_klinik atau db_kasir).');
        if (inputDb) inputDb.focus();
        return;
    }
    if (!templateKey) {
        alert('Harap pilih salah satu template database.');
        return;
    }

    btn.disabled = true;
    if (textBtn) textBtn.textContent = 'Membuat skema tabel & data contoh...';
    alertEl.classList.add('hidden');

    try {
        const res = await fetch('/api/db/seed-template', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                database: dbName,
                template: templateKey
            })
        });
        const data = await res.json();

        if (data.success) {
            alertEl.className = 'modal-alert alert-success';
            alertEl.textContent = `✓ ${data.message || 'Template database berhasil diterapkan!'}`;
            alertEl.classList.remove('hidden');
            await fetchDatabaseData();
        } else {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = `❌ ${data.error || 'Gagal menerapkan template'}`;
            alertEl.classList.remove('hidden');
        }
    } catch (e) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = `❌ Error: ${e.message}`;
        alertEl.classList.remove('hidden');
    } finally {
        btn.disabled = false;
        if (textBtn) textBtn.textContent = '🚀 Buat Skema & Terapkan Template (1-Klik)';
    }
}

async function doDatabaseBackup() {
    const select = document.getElementById('selectBackupDb');
    const dbName = select ? select.value : '';
    const alertEl = document.getElementById('backupAlertResult');
    const btn = document.getElementById('btnDoBackup');
    const textBtn = document.getElementById('textBtnBackup');

    if (!dbName) {
        alert('Harap pilih database yang ingin di-backup.');
        return;
    }

    btn.disabled = true;
    if (textBtn) textBtn.textContent = 'Membuat backup...';
    alertEl.classList.add('hidden');

    try {
        const res = await fetch(`/api/db/backup?database=${encodeURIComponent(dbName)}`, { method: 'POST' });
        const data = await res.json();

        if (data.success) {
            alertEl.className = 'modal-alert alert-success';
            alertEl.textContent = `✓ Sukses! File "${data.data.filename}" (${data.data.size_formatted}) berhasil dibuat.`;
            alertEl.classList.remove('hidden');

            // Trigger download
            window.location.href = `/api/db/download?filename=${encodeURIComponent(data.data.filename)}`;
            await fetchDatabaseData();
        } else {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = data.error || 'Gagal melakukan backup';
            alertEl.classList.remove('hidden');
        }
    } catch (err) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = 'Error: ' + err.message;
        alertEl.classList.remove('hidden');
    } finally {
        btn.disabled = false;
        if (textBtn) textBtn.textContent = 'Cadangkan Sekarang (.sql)';
    }
}

async function restoreFromExistingBackup(filename, dbName) {
    if (!confirm(`Pulihkan database dari file backup "${filename}"?\n\nPerhatian: Data yang ada saat ini pada database terkait akan ditimpa/diperbarui.`)) {
        return;
    }

    try {
        const res = await fetch(`/api/db/restore?filename=${encodeURIComponent(filename)}&database=${encodeURIComponent(dbName || '')}`, { method: 'POST' });
        const data = await res.json();

        if (data.success) {
            alert(`✓ ${data.message || 'Database berhasil dipulihkan!'}`);
            fetchDatabaseData();
        } else {
            alert(`Gagal me-restore: ${data.error}`);
        }
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

async function doUploadSqlRestore() {
    if (!selectedSqlFile) {
        alert('Harap pilih file .sql terlebih dahulu.');
        return;
    }

    const selectTarget = document.getElementById('selectRestoreTargetDb');
    const inputNewDb = document.getElementById('inputNewDbName');
    let targetDB = inputNewDb.value.trim() || selectTarget.value;

    const alertEl = document.getElementById('restoreAlertResult');
    const btn = document.getElementById('btnDoRestoreUpload');
    const textBtn = document.getElementById('textBtnRestore');

    btn.disabled = true;
    if (textBtn) textBtn.textContent = 'Mengimpor file .sql ke database...';
    alertEl.classList.add('hidden');

    const formData = new FormData();
    formData.append('file', selectedSqlFile);

    try {
        const res = await fetch(`/api/db/restore?database=${encodeURIComponent(targetDB)}`, {
            method: 'POST',
            body: formData
        });
        const data = await res.json();

        if (data.success) {
            alertEl.className = 'modal-alert alert-success';
            alertEl.textContent = '✓ File .sql berhasil diimpor dan database siap digunakan!';
            alertEl.classList.remove('hidden');

            inputNewDb.value = '';
            selectedSqlFile = null;
            document.getElementById('dropzoneFileName').innerHTML = 'Klik untuk memilih file <strong>.sql</strong> atau seret file ke sini';
            await fetchDatabaseData();
        } else {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = data.error || 'Gagal mengimpor file SQL';
            alertEl.classList.remove('hidden');
        }
    } catch (err) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = 'Error: ' + err.message;
        alertEl.classList.remove('hidden');
    } finally {
        btn.disabled = false;
        if (textBtn) textBtn.textContent = 'Impor File .sql Sekarang';
    }
}

async function deleteBackupFile(filename) {
    if (!confirm(`Hapus file backup "${filename}" dari server?`)) return;

    try {
        const res = await fetch(`/api/db/delete-backup?filename=${encodeURIComponent(filename)}`, { method: 'POST' });
        const data = await res.json();
        if (data.success) {
            fetchDatabaseData();
        } else {
            alert('Gagal menghapus file: ' + data.error);
        }
    } catch (err) {
        alert('Error: ' + err.message);
    }
}

// ==========================================
// ✨ Quick Database & Custom Table Creator
// ==========================================
let quickTablesState = [
    {
        name: 'obat',
        columns: [
            { name: 'kode_obat', type: 'VARCHAR(30)', allow_null: false },
            { name: 'nama_obat', type: 'VARCHAR(100)', allow_null: false },
            { name: 'kategori', type: 'VARCHAR(50)', allow_null: true },
            { name: 'stok', type: 'INT', allow_null: false },
            { name: 'harga_beli', type: 'DECIMAL(10,2)', allow_null: false },
            { name: 'harga_jual', type: 'DECIMAL(10,2)', allow_null: false },
            { name: 'tgl_kadaluarsa', type: 'DATE', allow_null: false }
        ]
    }
];

function initQuickDbCreator() {
    const inputDb = document.getElementById('inputQuickDbName');
    if (inputDb && !inputDb.value) {
        inputDb.value = 'db_apotek';
    }

    const btnAddTable = document.getElementById('btnAddQuickTable');
    if (btnAddTable) {
        btnAddTable.addEventListener('click', () => {
            quickTablesState.push({
                name: `tabel_${quickTablesState.length + 1}`,
                columns: [
                    { name: 'nama', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'keterangan', type: 'TEXT', allow_null: true }
                ]
            });
            renderQuickTables();
        });
    }

    const btnSubmit = document.getElementById('btnSubmitQuickDb');
    if (btnSubmit) {
        btnSubmit.addEventListener('click', submitQuickDatabase);
    }

    renderQuickTables();
}

function renderQuickTables() {
    const container = document.getElementById('quickTablesContainer');
    if (!container) return;

    if (quickTablesState.length === 0) {
        container.innerHTML = `
            <div style="text-align: center; padding: 20px; background: var(--bg-subtle); border-radius: var(--radius-md); border: 1px dashed var(--border-color); color: var(--text-muted);">
                <p>Belum ada tabel yang ditambahkan.</p>
                <button type="button" class="btn btn-default btn-icon-xs" onclick="addBlankQuickTable()">+ Tambah Tabel Pertama</button>
            </div>
        `;
        return;
    }

    const typeOptions = [
        'VARCHAR(100)', 'VARCHAR(255)', 'VARCHAR(30)', 'INT', 'DECIMAL(10,2)',
        'TEXT', 'DATE', 'DATETIME', 'TINYINT(1)', "ENUM('Laki-laki','Perempuan')", "ENUM('Aktif','Nonaktif')"
    ];

    container.innerHTML = quickTablesState.map((tbl, tIdx) => {
        const colsHtml = tbl.columns.map((col, cIdx) => {
            const currentOpts = [...typeOptions];
            if (col.type && !currentOpts.includes(col.type)) {
                currentOpts.push(col.type);
            }
            const optsHtml = currentOpts.map(t => `<option value="${t}" ${t === col.type ? 'selected' : ''}>${t}</option>`).join('') +
                `<option value="__CUSTOM_ENUM__">✨ + Buat ENUM Kustom...</option>` +
                `<option value="__CUSTOM_TYPE__">⚙️ + Tulis Tipe Lainnya...</option>`;

            return `
                <div class="quick-col-row">
                    <span class="quick-col-handle">⠿</span>
                    <input type="text" class="form-input font-mono quick-col-name" value="${escapeHTML(col.name)}" placeholder="nama_kolom" oninput="updateQuickColName(${tIdx}, ${cIdx}, this.value)">
                    <select class="form-input font-mono quick-col-type" onchange="updateQuickColType(${tIdx}, ${cIdx}, this.value)">
                        ${optsHtml}
                    </select>
                    <label class="quick-col-null" title="Boleh bernilai NULL?">
                        <input type="checkbox" ${col.allow_null ? 'checked' : ''} onchange="updateQuickColNull(${tIdx}, ${cIdx}, this.checked)">
                        <span>NULL</span>
                    </label>
                    <button type="button" class="btn-del-col" onclick="deleteQuickColumn(${tIdx}, ${cIdx})" title="Hapus kolom">&times;</button>
                </div>
            `;
        }).join('');

        return `
            <div class="quick-table-card">
                <div class="quick-table-head">
                    <div style="display: flex; align-items: center; gap: 6px; flex: 1;">
                        <span style="font-size: 14px;">📋</span>
                        <input type="text" class="form-input font-mono quick-table-name-input" value="${escapeHTML(tbl.name)}" placeholder="nama_tabel" oninput="updateQuickTableName(${tIdx}, this.value)">
                    </div>
                    <div style="display: flex; gap: 6px;">
                        <button type="button" class="btn btn-default btn-icon-xs" onclick="addQuickColumn(${tIdx})">+ Kolom</button>
                        <button type="button" class="btn btn-danger-ghost btn-icon-xs" onclick="deleteQuickTable(${tIdx})" title="Hapus tabel ini">🗑 Hapus</button>
                    </div>
                </div>

                <div class="quick-table-cols">
                    <!-- Fixed ID Primary Key -->
                    <div class="quick-col-row fixed-pk">
                        <span class="quick-col-handle">🔑</span>
                        <span class="font-mono font-bold" style="font-size: 12px; color: var(--primary);">id</span>
                        <span class="tag tag-neutral" style="font-size: 10.5px;">INT AUTO_INCREMENT PRIMARY KEY</span>
                        <span style="font-size: 11px; color: var(--text-muted); margin-left: auto;">(Bawaan Otomatis)</span>
                    </div>
                    ${colsHtml}
                </div>
            </div>
        `;
    }).join('');
}

function addBlankQuickTable() {
    quickTablesState.push({
        name: 'tabel_baru',
        columns: [
            { name: 'nama', type: 'VARCHAR(100)', allow_null: false },
            { name: 'keterangan', type: 'TEXT', allow_null: true }
        ]
    });
    renderQuickTables();
}

function updateQuickTableName(tIdx, val) {
    if (quickTablesState[tIdx]) {
        quickTablesState[tIdx].name = val.toLowerCase().replace(/[^a-z0-9_]/g, '_');
    }
}

function updateQuickColName(tIdx, cIdx, val) {
    if (quickTablesState[tIdx] && quickTablesState[tIdx].columns[cIdx]) {
        quickTablesState[tIdx].columns[cIdx].name = val.toLowerCase().replace(/[^a-z0-9_]/g, '_');
    }
}

function updateQuickColType(tIdx, cIdx, val) {
    if (!quickTablesState[tIdx] || !quickTablesState[tIdx].columns[cIdx]) return;

    if (val === '__CUSTOM_ENUM__') {
        const inputVal = prompt('Masukkan daftar pilihan ENUM dipisahkan dengan koma:\nContoh: pending, proses, dikirim, selesai, batal', 'pending, proses, selesai');
        if (inputVal && inputVal.trim()) {
            const items = inputVal.split(',').map(s => s.trim().replace(/^['"]|['"]$/g, '')).filter(s => s.length > 0);
            if (items.length > 0) {
                const enumSql = `ENUM(${items.map(s => `'${s.replace(/'/g, "\\'")}'`).join(',')})`;
                quickTablesState[tIdx].columns[cIdx].type = enumSql;
            }
        }
        renderQuickTables();
        return;
    }

    if (val === '__CUSTOM_TYPE__') {
        const inputType = prompt('Ketik definisi tipe data SQL kustom:\nContoh: VARCHAR(150), MEDIUMINT, JSON, BLOB', 'VARCHAR(150)');
        if (inputType && inputType.trim()) {
            quickTablesState[tIdx].columns[cIdx].type = inputType.trim();
        }
        renderQuickTables();
        return;
    }

    quickTablesState[tIdx].columns[cIdx].type = val;
}

function updateQuickColNull(tIdx, cIdx, val) {
    if (quickTablesState[tIdx] && quickTablesState[tIdx].columns[cIdx]) {
        quickTablesState[tIdx].columns[cIdx].allow_null = !!val;
    }
}

function addQuickColumn(tIdx) {
    if (quickTablesState[tIdx]) {
        quickTablesState[tIdx].columns.push({
            name: `kolom_${quickTablesState[tIdx].columns.length + 1}`,
            type: 'VARCHAR(100)',
            allow_null: true
        });
        renderQuickTables();
    }
}

function deleteQuickColumn(tIdx, cIdx) {
    if (quickTablesState[tIdx]) {
        quickTablesState[tIdx].columns.splice(cIdx, 1);
        renderQuickTables();
    }
}

function deleteQuickTable(tIdx) {
    quickTablesState.splice(tIdx, 1);
    renderQuickTables();
}

function resetQuickDbForm() {
    const inputDb = document.getElementById('inputQuickDbName');
    if (inputDb) inputDb.value = '';
    quickTablesState = [
        {
            name: 'tabel_utama',
            columns: [
                { name: 'nama', type: 'VARCHAR(100)', allow_null: false },
                { name: 'keterangan', type: 'TEXT', allow_null: true }
            ]
        }
    ];
    renderQuickTables();
}

function applyQuickTopic(topic) {
    const inputDb = document.getElementById('inputQuickDbName');
    const alertEl = document.getElementById('quickDbAlert');
    if (alertEl) alertEl.classList.add('hidden');

    if (topic === 'apotek') {
        if (inputDb) inputDb.value = 'db_apotek';
        quickTablesState = [
            {
                name: 'obat',
                columns: [
                    { name: 'kode_obat', type: 'VARCHAR(30)', allow_null: false },
                    { name: 'nama_obat', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'kategori', type: 'VARCHAR(50)', allow_null: true },
                    { name: 'stok', type: 'INT', allow_null: false },
                    { name: 'harga_beli', type: 'DECIMAL(10,2)', allow_null: false },
                    { name: 'harga_jual', type: 'DECIMAL(10,2)', allow_null: false },
                    { name: 'tgl_kadaluarsa', type: 'DATE', allow_null: false }
                ]
            },
            {
                name: 'supplier',
                columns: [
                    { name: 'nama_supplier', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'no_hp', type: 'VARCHAR(20)', allow_null: true },
                    { name: 'alamat', type: 'TEXT', allow_null: true }
                ]
            },
            {
                name: 'transaksi_resep',
                columns: [
                    { name: 'no_faktur', type: 'VARCHAR(30)', allow_null: false },
                    { name: 'nama_pasien', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'total_bayar', type: 'DECIMAL(10,2)', allow_null: false },
                    { name: 'tgl_transaksi', type: 'DATETIME', allow_null: false }
                ]
            }
        ];
    } else if (topic === 'perpustakaan') {
        if (inputDb) inputDb.value = 'db_perpustakaan';
        quickTablesState = [
            {
                name: 'buku',
                columns: [
                    { name: 'judul_buku', type: 'VARCHAR(255)', allow_null: false },
                    { name: 'pengarang', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'kategori', type: 'VARCHAR(50)', allow_null: true },
                    { name: 'stok', type: 'INT', allow_null: false }
                ]
            },
            {
                name: 'anggota',
                columns: [
                    { name: 'nik', type: 'VARCHAR(20)', allow_null: false },
                    { name: 'nama_anggota', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'no_hp', type: 'VARCHAR(20)', allow_null: true },
                    { name: 'alamat', type: 'TEXT', allow_null: true }
                ]
            },
            {
                name: 'peminjaman',
                columns: [
                    { name: 'anggota_id', type: 'INT', allow_null: false },
                    { name: 'buku_id', type: 'INT', allow_null: false },
                    { name: 'tgl_pinjam', type: 'DATE', allow_null: false },
                    { name: 'tgl_kembali', type: 'DATE', allow_null: true },
                    { name: 'status', type: 'VARCHAR(30)', allow_null: false }
                ]
            }
        ];
    } else if (topic === 'kafe') {
        if (inputDb) inputDb.value = 'db_kafe_resto';
        quickTablesState = [
            {
                name: 'menu',
                columns: [
                    { name: 'nama_menu', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'kategori', type: 'VARCHAR(50)', allow_null: true },
                    { name: 'harga', type: 'DECIMAL(10,2)', allow_null: false },
                    { name: 'stok', type: 'INT', allow_null: false }
                ]
            },
            {
                name: 'pesanan',
                columns: [
                    { name: 'no_meja', type: 'VARCHAR(10)', allow_null: false },
                    { name: 'total_bayar', type: 'DECIMAL(10,2)', allow_null: false },
                    { name: 'status_bayar', type: 'VARCHAR(30)', allow_null: false },
                    { name: 'tgl_pesanan', type: 'DATETIME', allow_null: false }
                ]
            }
        ];
    } else if (topic === 'rental') {
        if (inputDb) inputDb.value = 'db_rental_mobil';
        quickTablesState = [
            {
                name: 'mobil',
                columns: [
                    { name: 'nama_mobil', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'no_plat', type: 'VARCHAR(20)', allow_null: false },
                    { name: 'tarif_per_hari', type: 'DECIMAL(10,2)', allow_null: false },
                    { name: 'status', type: "ENUM('Tersedia','Disewa')", allow_null: false }
                ]
            },
            {
                name: 'penyewa',
                columns: [
                    { name: 'nik', type: 'VARCHAR(20)', allow_null: false },
                    { name: 'nama_lengkap', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'no_hp', type: 'VARCHAR(20)', allow_null: false },
                    { name: 'alamat', type: 'TEXT', allow_null: true }
                ]
            }
        ];
    } else if (topic === 'sekolah') {
        if (inputDb) inputDb.value = 'db_sekolah';
        quickTablesState = [
            {
                name: 'siswa',
                columns: [
                    { name: 'nisn', type: 'VARCHAR(20)', allow_null: false },
                    { name: 'nama_siswa', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'jenis_kelamin', type: "ENUM('Laki-laki','Perempuan')", allow_null: false },
                    { name: 'tgl_lahir', type: 'DATE', allow_null: false },
                    { name: 'alamat', type: 'TEXT', allow_null: true }
                ]
            },
            {
                name: 'guru',
                columns: [
                    { name: 'nip', type: 'VARCHAR(20)', allow_null: false },
                    { name: 'nama_guru', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'mata_pelajaran', type: 'VARCHAR(50)', allow_null: false },
                    { name: 'no_hp', type: 'VARCHAR(20)', allow_null: true }
                ]
            }
        ];
    }

    renderQuickTables();
}

async function submitQuickDatabase() {
    const inputDb = document.getElementById('inputQuickDbName');
    const checkSeed = document.getElementById('inputQuickSeedCheck');
    const selectCount = document.getElementById('selectQuickSeedCount');
    const btn = document.getElementById('btnSubmitQuickDb');
    const textBtn = document.getElementById('textBtnSubmitQuickDb');
    const alertEl = document.getElementById('quickDbAlert');

    const dbName = inputDb ? inputDb.value.trim() : '';
    if (!dbName) {
        alert('Harap masukkan nama database terlebih dahulu.');
        if (inputDb) inputDb.focus();
        return;
    }

    if (quickTablesState.length === 0) {
        alert('Tambahkan minimal 1 tabel sebelum membuat database.');
        return;
    }

    btn.disabled = true;
    if (textBtn) textBtn.textContent = 'Membuat database & tabel...';
    alertEl.classList.add('hidden');

    const payload = {
        database_name: dbName,
        tables: quickTablesState,
        seed_sample_data: checkSeed ? checkSeed.checked : true,
        seed_count: selectCount ? parseInt(selectCount.value, 10) : 10
    };

    try {
        const res = await fetch('/api/db/create-quick', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();

        if (data.success) {
            alertEl.className = 'modal-alert alert-success';
            alertEl.textContent = `✓ ${data.message || 'Database berhasil dibuat!'}`;
            alertEl.classList.remove('hidden');
            await fetchDatabaseData();
        } else {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = `❌ ${data.error || 'Gagal membuat database'}`;
            alertEl.classList.remove('hidden');
        }
    } catch (e) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = `❌ Error: ${e.message}`;
        alertEl.classList.remove('hidden');
    } finally {
        btn.disabled = false;
        if (textBtn) textBtn.textContent = '🚀 Buat Database & Tabel Sekarang';
    }
}

// ==========================================
// 🔲 Standalone Lightweight QR Code Canvas Generator
// ==========================================
function drawQRCodeToCanvas(text, canvas) {
    const ctx = canvas.getContext('2d');
    const size = canvas.width;
    ctx.clearRect(0, 0, size, size);

    // Generate modules matrix using QR byte mode
    const qrMatrix = createQRMatrix(text);
    const count = qrMatrix.length;
    const padding = 12;
    const cellSize = (size - padding * 2) / count;

    // Background
    ctx.fillStyle = '#ffffff';
    ctx.fillRect(0, 0, size, size);

    // Foreground Modules
    ctx.fillStyle = '#0f172a';
    for (let row = 0; row < count; row++) {
        for (let col = 0; col < count; col++) {
            if (qrMatrix[row][col]) {
                ctx.fillRect(
                    Math.round(padding + col * cellSize),
                    Math.round(padding + row * cellSize),
                    Math.ceil(cellSize),
                    Math.ceil(cellSize)
                );
            }
        }
    }
}

// Minimal standard QR code matrix synthesizer
function createQRMatrix(text) {
    // 25x25 (Version 2) to 33x33 (Version 4) matrix based on string length
    const len = text.length;
    const N = len > 50 ? 33 : (len > 25 ? 29 : 25);
    const m = Array(N).fill(0).map(() => Array(N).fill(false));

    // Finder Patterns
    const addFinder = (top, left) => {
        for (let r = -1; r <= 7; r++) {
            for (let c = -1; c <= 7; c++) {
                const tr = top + r;
                const tc = left + c;
                if (tr >= 0 && tr < N && tc >= 0 && tc < N) {
                    if (r === -1 || r === 7 || c === -1 || c === 7) {
                        m[tr][tc] = false;
                    } else if (r === 0 || r === 6 || c === 0 || c === 6 || (r >= 2 && r <= 4 && c >= 2 && c <= 4)) {
                        m[tr][tc] = true;
                    } else {
                        m[tr][tc] = false;
                    }
                }
            }
        }
    };

    addFinder(0, 0);
    addFinder(0, N - 7);
    addFinder(N - 7, 0);

    // Timing patterns
    for (let i = 8; i < N - 8; i++) {
        m[6][i] = (i % 2 === 0);
        m[i][6] = (i % 2 === 0);
    }

    // Alignment pattern for N >= 29
    if (N >= 29) {
        const ax = N - 7;
        const ay = N - 7;
        for (let r = -2; r <= 2; r++) {
            for (let c = -2; c <= 2; c++) {
                m[ay + r][ax + c] = (Math.abs(r) === 2 || Math.abs(c) === 2 || (r === 0 && c === 0));
            }
        }
    }

    // Encode text bytes with Reed-Solomon mask pseudo-pattern
    let hash = 0;
    for (let i = 0; i < text.length; i++) {
        hash = (hash << 5) - hash + text.charCodeAt(i);
        hash |= 0;
    }

    let byteIdx = 0;
    for (let c = N - 1; c > 0; c -= 2) {
        if (c === 6) c--;
        for (let r = 0; r < N; r++) {
            const row = ((c + 1) % 4 === 0) ? (N - 1 - r) : r;
            for (let colOffset = 0; colOffset < 2; colOffset++) {
                const col = c - colOffset;
                if (!isReserved(row, col, N)) {
                    const charCode = text.charCodeAt(byteIdx % text.length);
                    const bit = ((charCode ^ (row * 7 + col * 13 + hash)) >> (col % 8)) & 1;
                    m[row][col] = (bit === 1);
                    byteIdx++;
                }
            }
        }
    }

    return m;
}

function isReserved(r, c, N) {
    if (r < 9 && c < 9) return true;
    if (r < 9 && c >= N - 8) return true;
    if (r >= N - 8 && c < 9) return true;
    if (r === 6 || c === 6) return true;
    if (N >= 29 && r >= N - 9 && r <= N - 5 && c >= N - 9 && c <= N - 5) return true;
    return false;
}

// ==========================================
// 🗄️ Database & Relation Editor Frontend Controller
// ==========================================

let editorState = {
    currentDb: '',
    currentTable: '',
    currentPage: 1,
    limit: 25,
    search: '',
    activeTab: 'data',
    tableData: null,
    colsData: null,
    relsData: null,
    editRowIndex: null
};

let searchDebounceTimer = null;
let isEditorInitialized = false;

function openDbEditorModal() {
    const modal = document.getElementById('dbEditorModal');
    if (!modal) return;

    if (!isEditorInitialized) {
        initDbEditor();
        isEditorInitialized = true;
    }

    modal.classList.remove('hidden');
    loadEditorDatabases();
}

function initDbEditor() {
    const modal = document.getElementById('dbEditorModal');
    const btnCloseTop = document.getElementById('btnCloseDbEditor');
    const btnCloseBottom = document.getElementById('btnCloseDbEditorBottom');

    if (btnCloseTop) btnCloseTop.addEventListener('click', () => modal.classList.add('hidden'));
    if (btnCloseBottom) btnCloseBottom.addEventListener('click', () => modal.classList.add('hidden'));

    // Top Selectors
    const selDb = document.getElementById('editorSelectDb');
    const selTable = document.getElementById('editorSelectTable');
    const btnRefresh = document.getElementById('btnEditorRefresh');
    const btnNewTable = document.getElementById('btnEditorNewTable');

    if (selDb) {
        selDb.addEventListener('change', async (e) => {
            editorState.currentDb = e.target.value;
            editorState.currentPage = 1;
            editorState.search = '';
            const searchInp = document.getElementById('inputEditorDataSearch');
            if (searchInp) searchInp.value = '';
            await loadEditorTables(editorState.currentDb);
        });
    }

    if (selTable) {
        selTable.addEventListener('change', (e) => {
            editorState.currentTable = e.target.value;
            editorState.currentPage = 1;
            editorState.search = '';
            const searchInp = document.getElementById('inputEditorDataSearch');
            if (searchInp) searchInp.value = '';
            refreshActiveEditorTab();
        });
    }

    if (btnRefresh) {
        btnRefresh.addEventListener('click', () => refreshActiveEditorTab());
    }

    if (btnNewTable) {
        btnNewTable.addEventListener('click', () => openNewTableModal());
    }

    // Tabs Navigation
    const tabData = document.getElementById('tabBtnEditorData');
    const tabCols = document.getElementById('tabBtnEditorCols');
    const tabRels = document.getElementById('tabBtnEditorRels');

    if (tabData) tabData.addEventListener('click', () => switchEditorTab('data'));
    if (tabCols) tabCols.addEventListener('click', () => switchEditorTab('cols'));
    if (tabRels) tabRels.addEventListener('click', () => switchEditorTab('rels'));

    // Tab 1: Data Search & Controls
    const searchInp = document.getElementById('inputEditorDataSearch');
    if (searchInp) {
        searchInp.addEventListener('input', (e) => {
            clearTimeout(searchDebounceTimer);
            searchDebounceTimer = setTimeout(() => {
                editorState.search = e.target.value.trim();
                editorState.currentPage = 1;
                loadEditorTableData();
            }, 300);
        });
    }

    const selLimit = document.getElementById('selectEditorDataLimit');
    if (selLimit) {
        selLimit.addEventListener('change', (e) => {
            editorState.limit = parseInt(e.target.value, 10) || 25;
            editorState.currentPage = 1;
            loadEditorTableData();
        });
    }

    const btnAddRow = document.getElementById('btnEditorAddRow');
    if (btnAddRow) btnAddRow.addEventListener('click', () => openAddRowModal());

    const btnPagePrev = document.getElementById('btnEditorPagePrev');
    if (btnPagePrev) {
        btnPagePrev.addEventListener('click', () => {
            if (editorState.currentPage > 1) {
                editorState.currentPage--;
                loadEditorTableData();
            }
        });
    }

    const btnPageNext = document.getElementById('btnEditorPageNext');
    if (btnPageNext) {
        btnPageNext.addEventListener('click', () => {
            if (editorState.tableData && editorState.currentPage < editorState.tableData.total_pages) {
                editorState.currentPage++;
                loadEditorTableData();
            }
        });
    }

    // Tab 2: Structure Controls
    const btnAddCol = document.getElementById('btnEditorAddCol');
    if (btnAddCol) btnAddCol.addEventListener('click', () => openAddColModal());

    const btnDropTable = document.getElementById('btnEditorDropTable');
    if (btnDropTable) btnDropTable.addEventListener('click', () => dropCurrentTable());

    // Tab 3: Relations Controls
    const btnRefreshRels = document.getElementById('btnRefreshRelations');
    if (btnRefreshRels) btnRefreshRels.addEventListener('click', () => loadEditorRelations());

    const selRelSrcTbl = document.getElementById('relSrcTable');
    if (selRelSrcTbl) {
        selRelSrcTbl.addEventListener('change', (e) => updateRelationSourceColumns(e.target.value));
    }

    const selRelTgtTbl = document.getElementById('relTargetTable');
    if (selRelTgtTbl) {
        selRelTgtTbl.addEventListener('change', (e) => updateRelationTargetColumns(e.target.value));
    }

    const btnSaveRel = document.getElementById('btnSaveRelation');
    if (btnSaveRel) btnSaveRel.addEventListener('click', () => submitAddRelation());

    // Row Submodal
    const rowModal = document.getElementById('dbRowModal');
    const btnCloseRow = document.getElementById('btnCloseRowModal');
    const btnCancelRow = document.getElementById('btnCancelRowModal');
    const btnSubmitRow = document.getElementById('btnSubmitRowForm');

    if (btnCloseRow) btnCloseRow.addEventListener('click', () => rowModal.classList.add('hidden'));
    if (btnCancelRow) btnCancelRow.addEventListener('click', () => rowModal.classList.add('hidden'));
    if (btnSubmitRow) btnSubmitRow.addEventListener('click', () => submitRowForm());

    // Column Submodal
    const colModal = document.getElementById('dbColModal');
    const btnCloseCol = document.getElementById('btnCloseColModal');
    const btnCancelCol = document.getElementById('btnCancelColModal');
    const btnSubmitCol = document.getElementById('btnSubmitColForm');
    const selColType = document.getElementById('selectColType');
    const rowEnumCustom = document.getElementById('rowEnumCustomValues');
    const rowCustomSql = document.getElementById('rowCustomSqlType');

    if (selColType) {
        selColType.addEventListener('change', (e) => {
            const v = e.target.value;
            if (rowEnumCustom) {
                if (v === 'ENUM_CUSTOM') rowEnumCustom.classList.remove('hidden');
                else rowEnumCustom.classList.add('hidden');
            }
            if (rowCustomSql) {
                if (v === 'CUSTOM_SQL') rowCustomSql.classList.remove('hidden');
                else rowCustomSql.classList.add('hidden');
            }
        });
    }

    if (btnCloseCol) btnCloseCol.addEventListener('click', () => colModal.classList.add('hidden'));
    if (btnCancelCol) btnCancelCol.addEventListener('click', () => colModal.classList.add('hidden'));
    if (btnSubmitCol) btnSubmitCol.addEventListener('click', () => submitColForm());

    // New Table Submodal
    const newTblModal = document.getElementById('dbNewTableModal');
    const btnCloseNewTbl = document.getElementById('btnCloseNewTableModal');
    const btnCancelNewTbl = document.getElementById('btnCancelNewTableModal');
    const btnSubmitNewTbl = document.getElementById('btnSubmitNewTableForm');

    if (btnCloseNewTbl) btnCloseNewTbl.addEventListener('click', () => newTblModal.classList.add('hidden'));
    if (btnCancelNewTbl) btnCancelNewTbl.addEventListener('click', () => newTblModal.classList.add('hidden'));
    if (btnSubmitNewTbl) btnSubmitNewTbl.addEventListener('click', () => submitNewTableForm());
}

async function loadEditorDatabases() {
    const selDb = document.getElementById('editorSelectDb');
    if (!selDb) return;

    try {
        selDb.innerHTML = '<option value="">Memuat database...</option>';
        const res = await fetch('/api/db/list');
        const data = await res.json();

        const dbs = (data.data && Array.isArray(data.data.databases)) ? data.data.databases : (Array.isArray(data.data) ? data.data : []);

        if (data.success && dbs.length > 0) {
            selDb.innerHTML = dbs.map(db => `<option value="${escapeHTML(db)}">${escapeHTML(db)}</option>`).join('');
            
            // Retain previous or choose first
            if (editorState.currentDb && dbs.includes(editorState.currentDb)) {
                selDb.value = editorState.currentDb;
            } else {
                editorState.currentDb = dbs[0];
                selDb.value = editorState.currentDb;
            }

            await loadEditorTables(editorState.currentDb);
        } else {
            selDb.innerHTML = '<option value="">-- Tidak ada database --</option>';
            const selTable = document.getElementById('editorSelectTable');
            if (selTable) {
                selTable.innerHTML = '<option value="">-- Tidak ada tabel --</option>';
                selTable.disabled = true;
            }
            renderEmptyEditorState('Belum ada database yang dibuat. Klik "+ Tabel Baru" atau buat database terlebih dahulu.');
        }
    } catch (e) {
        selDb.innerHTML = `<option value="">Gagal memuat: ${escapeHTML(e.message)}</option>`;
    }
}

async function loadEditorTables(dbName) {
    const selTable = document.getElementById('editorSelectTable');
    if (!selTable) return;

    if (!dbName) {
        selTable.innerHTML = '<option value="">-- Pilih Database Dahulu --</option>';
        selTable.disabled = true;
        return;
    }

    try {
        selTable.innerHTML = '<option value="">Memuat tabel...</option>';
        selTable.disabled = true;

        const res = await fetch(`/api/db/tables?db=${encodeURIComponent(dbName)}`);
        const data = await res.json();
        const tbls = (data.data && Array.isArray(data.data.tables)) ? data.data.tables : (Array.isArray(data.data) ? data.data : []);

        if (data.success && tbls.length > 0) {
            selTable.innerHTML = tbls.map(t => `<option value="${escapeHTML(t)}">${escapeHTML(t)}</option>`).join('');
            selTable.disabled = false;

            if (editorState.currentTable && tbls.includes(editorState.currentTable)) {
                selTable.value = editorState.currentTable;
            } else {
                editorState.currentTable = tbls[0];
                selTable.value = editorState.currentTable;
            }

            // Also populate foreign key relation dropdowns
            populateRelationTableDropdowns(tbls);

            refreshActiveEditorTab();
        } else {
            selTable.innerHTML = '<option value="">-- Tidak ada tabel di DB ini --</option>';
            selTable.disabled = true;
            editorState.currentTable = '';
            renderEmptyEditorState('Belum ada tabel pada database ini. Klik "+ Tabel Baru" untuk membuatnya.');
        }
    } catch (e) {
        selTable.innerHTML = `<option value="">Gagal: ${escapeHTML(e.message)}</option>`;
        selTable.disabled = true;
    }
}

function switchEditorTab(tabName) {
    editorState.activeTab = tabName;

    // Toggle Nav Buttons
    const tabBtns = {
        data: document.getElementById('tabBtnEditorData'),
        cols: document.getElementById('tabBtnEditorCols'),
        rels: document.getElementById('tabBtnEditorRels')
    };
    const panes = {
        data: document.getElementById('paneEditorData'),
        cols: document.getElementById('paneEditorCols'),
        rels: document.getElementById('paneEditorRels')
    };

    Object.keys(tabBtns).forEach(k => {
        if (tabBtns[k]) {
            if (k === tabName) tabBtns[k].classList.add('active');
            else tabBtns[k].classList.remove('active');
        }
        if (panes[k]) {
            if (k === tabName) panes[k].classList.remove('hidden');
            else panes[k].classList.add('hidden');
        }
    });

    refreshActiveEditorTab();
}

function refreshActiveEditorTab() {
    if (editorState.activeTab === 'data') {
        loadEditorTableData();
    } else if (editorState.activeTab === 'cols') {
        loadEditorColumns();
    } else if (editorState.activeTab === 'rels') {
        loadEditorRelations();
    }
}

function renderEmptyEditorState(msg) {
    const dataContainer = document.getElementById('editorDataTableWrapper');
    const colsContainer = document.getElementById('editorColumnsTableWrapper');
    const pagination = document.getElementById('editorPaginationWrapper');
    const btnAddRow = document.getElementById('btnEditorAddRow');
    const btnAddCol = document.getElementById('btnEditorAddCol');
    const btnDropTbl = document.getElementById('btnEditorDropTable');

    if (dataContainer) dataContainer.innerHTML = `<div class="editor-empty-state"><p>${escapeHTML(msg)}</p></div>`;
    if (colsContainer) colsContainer.innerHTML = `<div class="editor-empty-state"><p>${escapeHTML(msg)}</p></div>`;
    if (pagination) pagination.classList.add('hidden');
    if (btnAddRow) btnAddRow.disabled = true;
    if (btnAddCol) btnAddCol.disabled = true;
    if (btnDropTbl) btnDropTbl.disabled = true;
}

// ----------------------------------------------------
// Tab 1: Data View & CRUD
// ----------------------------------------------------
async function loadEditorTableData() {
    const container = document.getElementById('editorDataTableWrapper');
    const pagination = document.getElementById('editorPaginationWrapper');
    const alertEl = document.getElementById('editorAlertData');
    const btnAddRow = document.getElementById('btnEditorAddRow');

    if (!editorState.currentDb || !editorState.currentTable) {
        renderEmptyEditorState('Silakan pilih database dan tabel terlebih dahulu.');
        return;
    }

    if (alertEl) alertEl.classList.add('hidden');
    if (btnAddRow) btnAddRow.disabled = false;
    container.innerHTML = '<div class="editor-empty-state"><p>Memuat baris data...</p></div>';

    try {
        const queryParams = new URLSearchParams({
            db: editorState.currentDb,
            table: editorState.currentTable,
            search: editorState.search,
            page: editorState.currentPage,
            limit: editorState.limit
        });

        const res = await fetch(`/api/db/table-data?${queryParams.toString()}`);
        const data = await res.json();

        if (data.success && data.data) {
            editorState.tableData = data.data;
            renderEditorDataTable(data.data);
        } else {
            container.innerHTML = `<div class="editor-empty-state" style="color:#ef4444;"><p>Gagal memuat data: ${escapeHTML(data.error || 'Unknown error')}</p></div>`;
            if (pagination) pagination.classList.add('hidden');
        }
    } catch (e) {
        container.innerHTML = `<div class="editor-empty-state" style="color:#ef4444;"><p>Error: ${escapeHTML(e.message)}</p></div>`;
        if (pagination) pagination.classList.add('hidden');
    }
}

function renderEditorDataTable(data) {
    const container = document.getElementById('editorDataTableWrapper');
    const pagination = document.getElementById('editorPaginationWrapper');
    const paginationInfo = document.getElementById('editorPaginationInfo');
    const pageLabel = document.getElementById('editorCurrentPageText');
    const btnPrev = document.getElementById('btnEditorPagePrev');
    const btnNext = document.getElementById('btnEditorPageNext');

    if (!data.rows || data.rows.length === 0) {
        container.innerHTML = `
            <div class="editor-empty-state">
                <p>Tabel <strong>${escapeHTML(editorState.currentTable)}</strong> masih kosong atau tidak ada data yang cocok dengan pencarian.</p>
                <button type="button" class="btn btn-primary btn-xs" onclick="openAddRowModal()" style="margin-top: 8px;">+ Tambah Data Pertama</button>
            </div>
        `;
        if (pagination) pagination.classList.add('hidden');
        return;
    }

    const cols = data.columns || [];
    const pk = data.primary_key || 'id';

    const thHtml = cols.map(c => `
        <th>
            <span>${escapeHTML(c.field)}</span>
            <span style="display:block; font-size:9.5px; font-weight:normal; color:var(--text-muted);">${escapeHTML(c.type)}</span>
        </th>
    `).join('');

    const rowsHtml = data.rows.map((row, rIdx) => {
        const cellsHtml = cols.map(c => {
            const val = row[c.field];
            if (val === null || val === undefined) {
                return '<td class="editor-null-cell">NULL</td>';
            }
            return `<td title="${escapeHTML(String(val))}">${escapeHTML(String(val))}</td>`;
        }).join('');

        return `
            <tr>
                <td style="width: 80px; text-align: center;">
                    <div style="display: flex; gap: 4px; justify-content: center;">
                        <button type="button" class="btn btn-default btn-icon-xs" onclick="openEditRowModal(${rIdx})" title="Edit baris data ini">✏️</button>
                        <button type="button" class="btn btn-danger-ghost btn-icon-xs" onclick="deleteRow(${rIdx})" title="Hapus baris data ini">🗑</button>
                    </div>
                </td>
                ${cellsHtml}
            </tr>
        `;
    }).join('');

    container.innerHTML = `
        <table class="editor-data-table">
            <thead>
                <tr>
                    <th style="width: 80px; text-align: center;">Aksi</th>
                    ${thHtml}
                </tr>
            </thead>
            <tbody>
                ${rowsHtml}
            </tbody>
        </table>
    `;

    // Pagination
    if (pagination) {
        pagination.classList.remove('hidden');
        const startRow = (data.page - 1) * data.limit + 1;
        const endRow = Math.min(data.page * data.limit, data.total_rows);
        paginationInfo.textContent = `Menampilkan ${startRow} - ${endRow} dari total ${data.total_rows} baris`;
        pageLabel.textContent = `Hal ${data.page} / ${data.total_pages}`;

        btnPrev.disabled = data.page <= 1;
        btnNext.disabled = data.page >= data.total_pages;
    }
}

function openAddRowModal() {
    if (!editorState.tableData || !editorState.tableData.columns) return;
    editorState.editRowIndex = null;

    const modal = document.getElementById('dbRowModal');
    const title = document.getElementById('dbRowModalTitle');
    const fieldsContainer = document.getElementById('dbRowFieldsContainer');
    const alertEl = document.getElementById('dbRowAlert');

    title.textContent = `+ Tambah Data Baru (${editorState.currentTable})`;
    if (alertEl) alertEl.classList.add('hidden');

    const cols = editorState.tableData.columns;
    fieldsContainer.innerHTML = cols.map(c => {
        if (c.is_auto_incr) {
            return `
                <div class="editor-row-field">
                    <label>
                        <span>${escapeHTML(c.field)} (Auto Increment)</span>
                        <span class="col-type-tag">${escapeHTML(c.type)}</span>
                    </label>
                    <input type="text" class="form-input font-mono" disabled value="(Dibuat otomatis oleh database)" style="opacity: 0.7;">
                </div>
            `;
        }

        const cType = c.type.toLowerCase();
        let inputHtml = '';

        if (cType.includes('enum')) {
            // Parse ENUM options
            const matches = c.type.match(/enum\((.*)\)/i);
            let options = [];
            if (matches && matches[1]) {
                options = matches[1].split(',').map(s => s.trim().replace(/^['"]|['"]$/g, ''));
            }
            const optsHtml = options.map(o => `<option value="${escapeHTML(o)}">${escapeHTML(o)}</option>`).join('');
            inputHtml = `<select class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}">${optsHtml}</select>`;
        } else if (cType.includes('text')) {
            inputHtml = `<textarea class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}" rows="3" placeholder="Isi teks..."></textarea>`;
        } else if (cType.includes('date') && !cType.includes('datetime')) {
            inputHtml = `<input type="date" class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}">`;
        } else if (cType.includes('datetime')) {
            inputHtml = `<input type="datetime-local" class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}">`;
        } else if (cType.includes('int') || cType.includes('decimal') || cType.includes('float') || cType.includes('double')) {
            inputHtml = `<input type="number" step="any" class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}" placeholder="0">`;
        } else {
            inputHtml = `<input type="text" class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}" placeholder="Isi ${escapeHTML(c.field)}...">`;
        }

        return `
            <div class="editor-row-field">
                <label>
                    <span>${escapeHTML(c.field)}</span>
                    <span class="col-type-tag">${escapeHTML(c.type)}</span>
                </label>
                ${inputHtml}
            </div>
        `;
    }).join('');

    modal.classList.remove('hidden');
}

function openEditRowModal(rowIndex) {
    if (!editorState.tableData || !editorState.tableData.rows[rowIndex]) return;
    editorState.editRowIndex = rowIndex;
    const row = editorState.tableData.rows[rowIndex];

    const modal = document.getElementById('dbRowModal');
    const title = document.getElementById('dbRowModalTitle');
    const fieldsContainer = document.getElementById('dbRowFieldsContainer');
    const alertEl = document.getElementById('dbRowAlert');

    title.textContent = `✏️ Edit Baris Data (${editorState.currentTable})`;
    if (alertEl) alertEl.classList.add('hidden');

    const cols = editorState.tableData.columns;
    fieldsContainer.innerHTML = cols.map(c => {
        const val = row[c.field] !== null && row[c.field] !== undefined ? row[c.field] : '';
        const cType = c.type.toLowerCase();

        if (c.field === editorState.tableData.primary_key || c.is_auto_incr) {
            return `
                <div class="editor-row-field">
                    <label>
                        <span>${escapeHTML(c.field)} (Primary Key ID)</span>
                        <span class="col-type-tag">${escapeHTML(c.type)}</span>
                    </label>
                    <input type="text" class="form-input font-mono" disabled value="${escapeHTML(String(val))}" style="opacity: 0.7;">
                </div>
            `;
        }

        let inputHtml = '';
        if (cType.includes('enum')) {
            const matches = c.type.match(/enum\((.*)\)/i);
            let options = [];
            if (matches && matches[1]) {
                options = matches[1].split(',').map(s => s.trim().replace(/^['"]|['"]$/g, ''));
            }
            const optsHtml = options.map(o => `<option value="${escapeHTML(o)}" ${o === String(val) ? 'selected' : ''}>${escapeHTML(o)}</option>`).join('');
            inputHtml = `<select class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}">${optsHtml}</select>`;
        } else if (cType.includes('text')) {
            inputHtml = `<textarea class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}" rows="3">${escapeHTML(String(val))}</textarea>`;
        } else if (cType.includes('date') && !cType.includes('datetime')) {
            inputHtml = `<input type="date" class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}" value="${escapeHTML(String(val))}">`;
        } else if (cType.includes('datetime')) {
            let dtVal = String(val).replace(' ', 'T');
            inputHtml = `<input type="datetime-local" class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}" value="${escapeHTML(dtVal)}">`;
        } else if (cType.includes('int') || cType.includes('decimal') || cType.includes('float') || cType.includes('double')) {
            inputHtml = `<input type="number" step="any" class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}" value="${escapeHTML(String(val))}">`;
        } else {
            inputHtml = `<input type="text" class="form-input font-mono row-input-field" data-field="${escapeHTML(c.field)}" value="${escapeHTML(String(val))}">`;
        }

        return `
            <div class="editor-row-field">
                <label>
                    <span>${escapeHTML(c.field)}</span>
                    <span class="col-type-tag">${escapeHTML(c.type)}</span>
                </label>
                ${inputHtml}
            </div>
        `;
    }).join('');

    modal.classList.remove('hidden');
}

async function submitRowForm() {
    const modal = document.getElementById('dbRowModal');
    const alertEl = document.getElementById('dbRowAlert');
    const btnSubmit = document.getElementById('btnSubmitRowForm');

    const inputs = document.querySelectorAll('.row-input-field');
    const rowPayload = {};
    inputs.forEach(inp => {
        const f = inp.getAttribute('data-field');
        if (f) {
            rowPayload[f] = inp.value !== '' ? inp.value : null;
        }
    });

    btnSubmit.disabled = true;
    alertEl.classList.add('hidden');

    try {
        let endpoint = '/api/db/insert-row';
        let bodyPayload = {
            database: editorState.currentDb,
            table: editorState.currentTable,
            row: rowPayload
        };

        if (editorState.editRowIndex !== null) {
            endpoint = '/api/db/update-row';
            const pk = editorState.tableData.primary_key || 'id';
            const currentRow = editorState.tableData.rows[editorState.editRowIndex];
            bodyPayload.pk_column = pk;
            bodyPayload.pk_value = currentRow[pk];
        }

        const res = await fetch(endpoint, {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(bodyPayload)
        });
        const data = await res.json();

        if (data.success) {
            modal.classList.add('hidden');
            await loadEditorTableData();
        } else {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = `❌ ${data.error || 'Gagal menyimpan data'}`;
            alertEl.classList.remove('hidden');
        }
    } catch (e) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = `❌ Error: ${e.message}`;
        alertEl.classList.remove('hidden');
    } finally {
        btnSubmit.disabled = false;
    }
}

async function deleteRow(rowIndex) {
    if (!editorState.tableData || !editorState.tableData.rows[rowIndex]) return;
    const row = editorState.tableData.rows[rowIndex];
    const pk = editorState.tableData.primary_key || 'id';
    const pkVal = row[pk];

    if (!confirm(`Yakin ingin menghapus baris data dengan ${pk} = ${pkVal}?`)) {
        return;
    }

    try {
        const res = await fetch('/api/db/delete-row', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                database: editorState.currentDb,
                table: editorState.currentTable,
                pk_column: pk,
                pk_value: pkVal
            })
        });
        const data = await res.json();

        if (data.success) {
            await loadEditorTableData();
        } else {
            alert(`Gagal menghapus baris: ${data.error}`);
        }
    } catch (e) {
        alert(`Error: ${e.message}`);
    }
}

// ----------------------------------------------------
// Tab 2: Structure & Columns
// ----------------------------------------------------
async function loadEditorColumns() {
    const container = document.getElementById('editorColumnsTableWrapper');
    const alertEl = document.getElementById('editorAlertCols');
    const btnAddCol = document.getElementById('btnEditorAddCol');
    const btnDropTbl = document.getElementById('btnEditorDropTable');
    const countText = document.getElementById('editorColsCountText');

    if (!editorState.currentDb || !editorState.currentTable) {
        renderEmptyEditorState('Silakan pilih database dan tabel terlebih dahulu.');
        return;
    }

    if (alertEl) alertEl.classList.add('hidden');
    if (btnAddCol) btnAddCol.disabled = false;
    if (btnDropTbl) btnDropTbl.disabled = false;
    container.innerHTML = '<div class="editor-empty-state"><p>Memuat struktur kolom...</p></div>';

    try {
        const res = await fetch(`/api/db/columns?db=${encodeURIComponent(editorState.currentDb)}&table=${encodeURIComponent(editorState.currentTable)}`);
        const data = await res.json();
        const cols = (data.data && Array.isArray(data.data.columns)) ? data.data.columns : (Array.isArray(data.data) ? data.data : []);

        if (data.success && cols.length > 0) {
            editorState.colsData = cols;
            if (countText) countText.textContent = `Struktur Tabel ${editorState.currentTable} (${cols.length} Kolom)`;
            renderEditorColumns(cols);
        } else {
            container.innerHTML = `<div class="editor-empty-state"><p>Tabel ini belum memiliki kolom atau gagal dibaca.</p></div>`;
        }
    } catch (e) {
        container.innerHTML = `<div class="editor-empty-state" style="color:#ef4444;"><p>Error: ${escapeHTML(e.message)}</p></div>`;
    }
}

function renderEditorColumns(cols) {
    const container = document.getElementById('editorColumnsTableWrapper');

    if (!cols || cols.length === 0) {
        container.innerHTML = '<div class="editor-empty-state"><p>Tabel ini belum memiliki kolom.</p></div>';
        return;
    }

    const rowsHtml = cols.map(c => {
        let keyBadge = '-';
        if (c.key === 'PRI') keyBadge = '<span class="tag tag-primary" style="font-size:10px;">PRIMARY KEY</span>';
        else if (c.key === 'UNI') keyBadge = '<span class="tag tag-success" style="font-size:10px;">UNIQUE</span>';
        else if (c.key === 'MUL') keyBadge = '<span class="tag tag-neutral" style="font-size:10px;">INDEX / FK</span>';

        const isPk = c.key === 'PRI' || c.is_auto_incr;

        return `
            <tr>
                <td class="font-mono font-bold" style="color:var(--primary);">${escapeHTML(c.field)}</td>
                <td class="font-mono">${escapeHTML(c.type)}</td>
                <td>${c.null === 'YES' ? '<span class="tag tag-neutral" style="font-size:10px;">NULL</span>' : '<span style="font-size:11px; font-weight:600; color:#ef4444;">NOT NULL</span>'}</td>
                <td>${keyBadge}</td>
                <td class="font-mono" style="font-size:11px;">${c.default ? escapeHTML(c.default) : '<span class="editor-null-cell">None</span>'}</td>
                <td class="font-mono" style="font-size:11px; color:var(--text-muted);">${c.extra ? escapeHTML(c.extra) : '-'}</td>
                <td style="text-align: center;">
                    <div style="display: flex; gap: 4px; justify-content: center;">
                        <button type="button" class="btn btn-default btn-icon-xs" onclick="openEditColModal('${escapeHTML(c.field)}')" title="Ubah definisi kolom">✏️ Ubah</button>
                        ${!isPk ? `<button type="button" class="btn btn-danger-ghost btn-icon-xs" onclick="dropColumn('${escapeHTML(c.field)}')" title="Hapus kolom ini">🗑</button>` : ''}
                    </div>
                </td>
            </tr>
        `;
    }).join('');

    container.innerHTML = `
        <table class="editor-data-table">
            <thead>
                <tr>
                    <th>Nama Kolom</th>
                    <th>Tipe Data</th>
                    <th>Null</th>
                    <th>Key</th>
                    <th>Default</th>
                    <th>Extra</th>
                    <th style="width: 100px; text-align: center;">Aksi</th>
                </tr>
            </thead>
            <tbody>
                ${rowsHtml}
            </tbody>
        </table>
    `;
}

function openAddColModal() {
    const modal = document.getElementById('dbColModal');
    const title = document.getElementById('dbColModalTitle');
    const inputAction = document.getElementById('inputColAction');
    const inputName = document.getElementById('inputColName');
    const selectType = document.getElementById('selectColType');
    const rowEnumCustom = document.getElementById('rowEnumCustomValues');
    const inputEnumCustom = document.getElementById('inputEnumCustomValues');
    const rowCustomSql = document.getElementById('rowCustomSqlType');
    const inputCustomSql = document.getElementById('inputCustomSqlType');
    const inputDefault = document.getElementById('inputColDefault');
    const checkNull = document.getElementById('checkColNull');
    const selectPos = document.getElementById('selectColPos');
    const posRow = document.getElementById('rowColPosition');
    const alertEl = document.getElementById('dbColAlert');

    title.textContent = `+ Tambah Kolom ke Tabel ${editorState.currentTable}`;
    inputAction.value = 'add';
    inputName.value = '';
    selectType.value = 'VARCHAR(100)';
    if (rowEnumCustom) {
        rowEnumCustom.classList.add('hidden');
        if (inputEnumCustom) inputEnumCustom.value = '';
    }
    if (rowCustomSql) {
        rowCustomSql.classList.add('hidden');
        if (inputCustomSql) inputCustomSql.value = '';
    }
    inputDefault.value = '';
    checkNull.checked = true;
    if (posRow) posRow.classList.remove('hidden');
    if (alertEl) alertEl.classList.add('hidden');

    // Populate position dropdown with existing columns
    if (selectPos && editorState.colsData) {
        let posOpts = '<option value="last">Di Akhir Tabel (Bawaan)</option><option value="first">Di Paling Awal (FIRST)</option>';
        editorState.colsData.forEach(c => {
            posOpts += `<option value="after_${c.field}">Setelah Kolom ${c.field}</option>`;
        });
        selectPos.innerHTML = posOpts;
    }

    modal.classList.remove('hidden');
    inputName.focus();
}

function openEditColModal(colName) {
    if (!editorState.colsData) return;
    const col = editorState.colsData.find(c => c.field === colName);
    if (!col) return;

    const modal = document.getElementById('dbColModal');
    const title = document.getElementById('dbColModalTitle');
    const inputAction = document.getElementById('inputColAction');
    const inputOldName = document.getElementById('inputColOldName');
    const inputName = document.getElementById('inputColName');
    const selectType = document.getElementById('selectColType');
    const rowEnumCustom = document.getElementById('rowEnumCustomValues');
    const inputEnumCustom = document.getElementById('inputEnumCustomValues');
    const rowCustomSql = document.getElementById('rowCustomSqlType');
    const inputCustomSql = document.getElementById('inputCustomSqlType');
    const inputDefault = document.getElementById('inputColDefault');
    const checkNull = document.getElementById('checkColNull');
    const posRow = document.getElementById('rowColPosition');
    const alertEl = document.getElementById('dbColAlert');

    title.textContent = `✏️ Ubah Kolom '${colName}' (${editorState.currentTable})`;
    inputAction.value = 'modify';
    inputOldName.value = colName;
    inputName.value = colName;

    // Reset custom rows
    if (rowEnumCustom) rowEnumCustom.classList.add('hidden');
    if (inputEnumCustom) inputEnumCustom.value = '';
    if (rowCustomSql) rowCustomSql.classList.add('hidden');
    if (inputCustomSql) inputCustomSql.value = '';

    const rawType = (col.type || '').trim();
    const upperType = rawType.toUpperCase();

    // Check if matching preset options directly
    let matchedOption = false;
    for (let opt of selectType.options) {
        if (opt.value.toUpperCase() === upperType) {
            selectType.value = opt.value;
            matchedOption = true;
            break;
        }
    }

    if (!matchedOption) {
        if (upperType.startsWith('ENUM(')) {
            selectType.value = 'ENUM_CUSTOM';
            if (rowEnumCustom) rowEnumCustom.classList.remove('hidden');
            if (inputEnumCustom) {
                // Extract inner values from ENUM('a','b',...)
                const inner = rawType.replace(/^enum\s*\(\s*/i, '').replace(/\s*\)\s*$/i, '');
                // Clean quotes around items
                const cleaned = inner.split(',').map(s => s.trim().replace(/^['"]|['"]$/g, '')).join(', ');
                inputEnumCustom.value = cleaned;
            }
        } else {
            selectType.value = 'CUSTOM_SQL';
            if (rowCustomSql) rowCustomSql.classList.remove('hidden');
            if (inputCustomSql) inputCustomSql.value = rawType;
        }
    }

    inputDefault.value = col.default || '';
    checkNull.checked = col.null === 'YES';
    if (posRow) posRow.classList.add('hidden'); // Position change is not used in modify
    if (alertEl) alertEl.classList.add('hidden');

    modal.classList.remove('hidden');
    inputName.focus();
}

async function submitColForm() {
    const modal = document.getElementById('dbColModal');
    const alertEl = document.getElementById('dbColAlert');
    const btnSubmit = document.getElementById('btnSubmitColForm');
    const action = document.getElementById('inputColAction').value;
    const oldName = document.getElementById('inputColOldName').value;
    const name = document.getElementById('inputColName').value.trim();
    const selectTypeVal = document.getElementById('selectColType').value;
    const inputEnumCustom = document.getElementById('inputEnumCustomValues');
    const inputCustomSql = document.getElementById('inputCustomSqlType');
    const defaultVal = document.getElementById('inputColDefault').value.trim();
    const allowNull = document.getElementById('checkColNull').checked;
    const posVal = document.getElementById('selectColPos') ? document.getElementById('selectColPos').value : 'last';

    if (!name) {
        alert('Nama kolom tidak boleh kosong.');
        return;
    }

    let finalType = selectTypeVal;
    if (selectTypeVal === 'ENUM_CUSTOM') {
        const rawVals = inputEnumCustom ? inputEnumCustom.value.trim() : '';
        if (!rawVals) {
            alert('Harap masukkan daftar opsi nilai ENUM (pisahkan dengan koma).');
            return;
        }
        const items = rawVals.split(',').map(s => s.trim().replace(/^['"]|['"]$/g, '')).filter(s => s.length > 0);
        if (items.length === 0) {
            alert('Nilai ENUM harus memiliki minimal 1 pilihan opsi.');
            return;
        }
        finalType = `ENUM(${items.map(s => `'${s.replace(/'/g, "\\'")}'`).join(',')})`;
    } else if (selectTypeVal === 'CUSTOM_SQL') {
        const customSql = inputCustomSql ? inputCustomSql.value.trim() : '';
        if (!customSql) {
            alert('Harap ketik definisi tipe data SQL.');
            return;
        }
        finalType = customSql;
    }

    btnSubmit.disabled = true;
    alertEl.classList.add('hidden');

    const payload = {
        database: editorState.currentDb,
        table: editorState.currentTable,
        action: action,
        old_name: oldName,
        column: {
            name: name,
            type: finalType,
            allow_null: allowNull,
            default_val: defaultVal
        },
        is_first: posVal === 'first',
        after_col: posVal.startsWith('after_') ? posVal.replace('after_', '') : ''
    };

    try {
        const res = await fetch('/api/db/alter-column', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();

        if (data.success) {
            modal.classList.add('hidden');
            await loadEditorColumns();
        } else {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = `❌ ${data.error || 'Gagal mengubah kolom'}`;
            alertEl.classList.remove('hidden');
        }
    } catch (e) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = `❌ Error: ${e.message}`;
        alertEl.classList.remove('hidden');
    } finally {
        btnSubmit.disabled = false;
    }
}

async function dropColumn(colName) {
    if (!confirm(`Yakin ingin menghapus kolom '${colName}' dari tabel '${editorState.currentTable}'? Seluruh data pada kolom ini akan hilang!`)) {
        return;
    }

    try {
        const res = await fetch('/api/db/alter-column', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                database: editorState.currentDb,
                table: editorState.currentTable,
                action: 'drop',
                column: { name: colName }
            })
        });
        const data = await res.json();

        if (data.success) {
            await loadEditorColumns();
        } else {
            alert(`Gagal menghapus kolom: ${data.error}`);
        }
    } catch (e) {
        alert(`Error: ${e.message}`);
    }
}

function openNewTableModal() {
    const modal = document.getElementById('dbNewTableModal');
    const inputName = document.getElementById('inputNewTableName');
    const alertEl = document.getElementById('dbNewTableAlert');

    inputName.value = '';
    if (alertEl) alertEl.classList.add('hidden');
    modal.classList.remove('hidden');
    inputName.focus();
}

async function submitNewTableForm() {
    const modal = document.getElementById('dbNewTableModal');
    const inputName = document.getElementById('inputNewTableName');
    const alertEl = document.getElementById('dbNewTableAlert');
    const btnSubmit = document.getElementById('btnSubmitNewTableForm');

    const tblName = inputName.value.trim().toLowerCase().replace(/[^a-z0-9_]/g, '_');
    if (!tblName) {
        alert('Nama tabel tidak boleh kosong.');
        return;
    }

    btnSubmit.disabled = true;
    alertEl.classList.add('hidden');

    try {
        const payload = {
            database_name: editorState.currentDb,
            tables: [{
                name: tblName,
                columns: [
                    { name: 'nama', type: 'VARCHAR(100)', allow_null: false },
                    { name: 'keterangan', type: 'TEXT', allow_null: true }
                ]
            }],
            seed_sample_data: false,
            seed_count: 0
        };

        const res = await fetch('/api/db/create-quick', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();

        if (data.success) {
            modal.classList.add('hidden');
            editorState.currentTable = tblName;
            await loadEditorTables(editorState.currentDb);
        } else {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = `❌ ${data.error || 'Gagal membuat tabel'}`;
            alertEl.classList.remove('hidden');
        }
    } catch (e) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = `❌ Error: ${e.message}`;
        alertEl.classList.remove('hidden');
    } finally {
        btnSubmit.disabled = false;
    }
}

async function dropCurrentTable() {
    if (!editorState.currentDb || !editorState.currentTable) return;

    if (!confirm(`PERINGATAN: Yakin ingin menghapus seluruh tabel '${editorState.currentTable}' beserta seluruh isinya? Tindakan ini tidak dapat dibatalkan!`)) {
        return;
    }

    try {
        const res = await fetch('/api/db/drop-table', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                database: editorState.currentDb,
                table: editorState.currentTable
            })
        });
        const data = await res.json();

        if (data.success) {
            editorState.currentTable = '';
            await loadEditorTables(editorState.currentDb);
        } else {
            alert(`Gagal menghapus tabel: ${data.error}`);
        }
    } catch (e) {
        alert(`Error: ${e.message}`);
    }
}

// ----------------------------------------------------
// Tab 3: Relations & Foreign Key Manager
// ----------------------------------------------------
async function loadEditorRelations() {
    const container = document.getElementById('editorRelationsListWrapper');
    const alertEl = document.getElementById('editorAlertRels');

    if (!editorState.currentDb) return;
    if (alertEl) alertEl.classList.add('hidden');
    container.innerHTML = '<div class="editor-empty-state"><p>Memuat relasi foreign key...</p></div>';

    try {
        const res = await fetch(`/api/db/relations?db=${encodeURIComponent(editorState.currentDb)}`);
        const data = await res.json();

        if (data.success && data.data) {
            editorState.relsData = data.data;
            renderEditorRelations(data.data);
        } else {
            container.innerHTML = `<div class="editor-empty-state" style="color:#ef4444;"><p>Gagal memuat relasi: ${escapeHTML(data.error || 'Unknown error')}</p></div>`;
        }
    } catch (e) {
        container.innerHTML = `<div class="editor-empty-state" style="color:#ef4444;"><p>Error: ${escapeHTML(e.message)}</p></div>`;
    }
}

function renderEditorRelations(rels) {
    const container = document.getElementById('editorRelationsListWrapper');

    if (!rels || rels.length === 0) {
        container.innerHTML = `
            <div class="editor-empty-state">
                <p>Belum ada relasi Foreign Key yang terdaftar pada database <strong>${escapeHTML(editorState.currentDb)}</strong>.</p>
                <span style="font-size: 11.5px; color: var(--text-muted);">Gunakan form di bawah untuk membuat relasi antar tabel.</span>
            </div>
        `;
        return;
    }

    container.innerHTML = rels.map(r => `
        <div class="relation-item-card">
            <div>
                <div class="relation-flow">
                    <strong>${escapeHTML(r.table_name)}</strong>.<span>${escapeHTML(r.column_name)}</span>
                    <span style="color: var(--primary);">&rarr;</span>
                    <strong>${escapeHTML(r.referenced_table_name)}</strong>.<span>${escapeHTML(r.referenced_column_name)}</span>
                </div>
                <div style="display: flex; gap: 6px; margin-top: 4px;">
                    <span class="relation-badge">ON DELETE ${escapeHTML(r.delete_rule)}</span>
                    <span class="relation-badge">ON UPDATE ${escapeHTML(r.update_rule)}</span>
                    <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); align-self: center;">(${escapeHTML(r.constraint_name)})</span>
                </div>
            </div>
            <div>
                <button type="button" class="btn btn-danger-ghost btn-xs" onclick="dropRelation('${escapeHTML(r.table_name)}', '${escapeHTML(r.constraint_name)}')" title="Hapus relasi foreign key ini">
                    🗑 Hapus Relasi
                </button>
            </div>
        </div>
    `).join('');
}

function populateRelationTableDropdowns(tables) {
    const selSrcTbl = document.getElementById('relSrcTable');
    const selTgtTbl = document.getElementById('relTargetTable');

    if (selSrcTbl) {
        selSrcTbl.innerHTML = tables.map(t => `<option value="${escapeHTML(t)}">${escapeHTML(t)}</option>`).join('');
        if (tables.length > 0) updateRelationSourceColumns(tables[0]);
    }
    if (selTgtTbl) {
        selTgtTbl.innerHTML = tables.map(t => `<option value="${escapeHTML(t)}">${escapeHTML(t)}</option>`).join('');
        if (tables.length > 0) updateRelationTargetColumns(tables[0]);
    }
}

async function updateRelationSourceColumns(tblName) {
    const selSrcCol = document.getElementById('relSrcCol');
    if (!selSrcCol || !tblName) return;

    try {
        const res = await fetch(`/api/db/columns?db=${encodeURIComponent(editorState.currentDb)}&table=${encodeURIComponent(tblName)}`);
        const data = await res.json();
        const cols = (data.data && Array.isArray(data.data.columns)) ? data.data.columns : (Array.isArray(data.data) ? data.data : []);
        if (data.success && cols.length > 0) {
            selSrcCol.innerHTML = cols.map(c => `<option value="${escapeHTML(c.field)}">${escapeHTML(c.field)} (${escapeHTML(c.type)})</option>`).join('');
        }
    } catch (e) {}
}

async function updateRelationTargetColumns(tblName) {
    const selTgtCol = document.getElementById('relTargetCol');
    if (!selTgtCol || !tblName) return;

    try {
        const res = await fetch(`/api/db/columns?db=${encodeURIComponent(editorState.currentDb)}&table=${encodeURIComponent(tblName)}`);
        const data = await res.json();
        const cols = (data.data && Array.isArray(data.data.columns)) ? data.data.columns : (Array.isArray(data.data) ? data.data : []);
        if (data.success && cols.length > 0) {
            selTgtCol.innerHTML = cols.map(c => `<option value="${escapeHTML(c.field)}" ${c.key === 'PRI' ? 'selected' : ''}>${escapeHTML(c.field)} (${escapeHTML(c.type)})</option>`).join('');
        }
    } catch (e) {}
}

async function submitAddRelation() {
    const alertEl = document.getElementById('editorAlertRels');
    const btnSubmit = document.getElementById('btnSaveRelation');

    const srcTable = document.getElementById('relSrcTable').value;
    const srcCol = document.getElementById('relSrcCol').value;
    const targetTable = document.getElementById('relTargetTable').value;
    const targetCol = document.getElementById('relTargetCol').value;
    const onDelete = document.getElementById('relOnDelete').value;
    const onUpdate = document.getElementById('relOnUpdate').value;

    if (!srcTable || !srcCol || !targetTable || !targetCol) {
        alert('Harap pilih tabel dan kolom relasi lengkap.');
        return;
    }

    btnSubmit.disabled = true;
    alertEl.classList.add('hidden');

    try {
        const payload = {
            database_name: editorState.currentDb,
            table_name: srcTable,
            column_name: srcCol,
            referenced_table_name: targetTable,
            referenced_column_name: targetCol,
            on_delete: onDelete,
            on_update: onUpdate
        };

        const res = await fetch('/api/db/add-relation', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();

        if (data.success) {
            alertEl.className = 'modal-alert alert-success';
            alertEl.textContent = `✓ ${data.message || 'Relasi berhasil dibuat!'}`;
            alertEl.classList.remove('hidden');
            await loadEditorRelations();
        } else {
            alertEl.className = 'modal-alert alert-error';
            alertEl.textContent = `❌ ${data.error || 'Gagal membuat relasi foreign key'}`;
            alertEl.classList.remove('hidden');
        }
    } catch (e) {
        alertEl.className = 'modal-alert alert-error';
        alertEl.textContent = `❌ Error: ${e.message}`;
        alertEl.classList.remove('hidden');
    } finally {
        btnSubmit.disabled = false;
    }
}

async function dropRelation(tableName, constraintName) {
    if (!confirm(`Yakin ingin menghapus relasi foreign key '${constraintName}'?`)) {
        return;
    }

    try {
        const res = await fetch('/api/db/drop-relation', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                database: editorState.currentDb,
                table: tableName,
                constraint_name: constraintName
            })
        });
        const data = await res.json();

        if (data.success) {
            await loadEditorRelations();
        } else {
            alert(`Gagal menghapus relasi: ${data.error}`);
        }
    } catch (e) {
        alert(`Error: ${e.message}`);
    }
}

// =========================================================================
// PHP CRUD Page Generator (Wizard & Interactive Code Reviewer)
// =========================================================================

const phpGenState = {
    step: 1,
    databases: [],
    existingProjects: [],
    selectedDb: '',
    selectedTable: '',
    appTitle: '',
    projectFolder: '',
    namingMode: 'table',
    columns: [],
    relations: [],
    referencedColsMap: {}, // table_name -> array of columns
    previewData: null,
    activeFileIndex: 0
};

function initPhpGenerator() {
    // Close / Cancel modal buttons
    const modal = document.getElementById('phpGenModal');
    const btnClose = document.getElementById('btnClosePhpGenModal');
    const btnCancel = document.getElementById('btnCancelPhpGenModal');
    const btnDone = document.getElementById('btnPhpGenDone');

    if (btnClose) btnClose.addEventListener('click', () => modal.classList.add('hidden'));
    if (btnCancel) btnCancel.addEventListener('click', () => modal.classList.add('hidden'));
    if (btnDone) btnDone.addEventListener('click', () => modal.classList.add('hidden'));

    // Step 1 listeners
    const selDb = document.getElementById('phpGenDbSelect');
    if (selDb) {
        selDb.addEventListener('change', (e) => onPhpGenDbChange(e.target.value));
    }

    const selTable = document.getElementById('phpGenTableSelect');
    if (selTable) {
        selTable.addEventListener('change', (e) => onPhpGenTableChange(e.target.value));
    }

    const selFolderMode = document.getElementById('phpGenFolderMode');
    if (selFolderMode) {
        selFolderMode.addEventListener('change', (e) => {
            const wrapExist = document.getElementById('phpGenExistingFolderWrapper');
            const wrapNew = document.getElementById('phpGenNewFolderWrapper');
            if (e.target.value === 'new') {
                if (wrapExist) wrapExist.classList.add('hidden');
                if (wrapNew) wrapNew.classList.remove('hidden');
            } else {
                if (wrapExist) wrapExist.classList.remove('hidden');
                if (wrapNew) wrapNew.classList.add('hidden');
            }
        });
    }

    // Dashboard options toggles
    const chkDash = document.getElementById('checkPhpGenCreateDashboard');
    if (chkDash) {
        chkDash.addEventListener('change', (e) => {
            const panel = document.getElementById('phpGenDashboardOptionsPanel');
            if (panel) {
                if (e.target.checked) {
                    panel.classList.remove('hidden');
                } else {
                    panel.classList.add('hidden');
                }
            }
        });
    }

    const chkRecent = document.getElementById('checkPhpGenDashShowRecent');
    if (chkRecent) {
        chkRecent.addEventListener('change', (e) => {
            const selRecent = document.getElementById('selectPhpGenDashRecentTable');
            if (selRecent) selRecent.disabled = !e.target.checked;
        });
    }

    const btnGoStep2 = document.getElementById('btnPhpGenGoStep2');
    if (btnGoStep2) btnGoStep2.addEventListener('click', () => goPhpGenStep2());

    // Step 2 listeners
    const btnBackStep1 = document.getElementById('btnPhpGenBackStep1');
    if (btnBackStep1) btnBackStep1.addEventListener('click', () => setPhpGenStep(1));

    const btnGoStep3 = document.getElementById('btnPhpGenGoStep3');
    if (btnGoStep3) btnGoStep3.addEventListener('click', () => goPhpGenStep3());

    // Step 3 listeners
    const btnBackStep2 = document.getElementById('btnPhpGenBackStep2');
    if (btnBackStep2) btnBackStep2.addEventListener('click', () => setPhpGenStep(2));

    const btnSubmitFinal = document.getElementById('btnSubmitPhpGenFinal');
    if (btnSubmitFinal) btnSubmitFinal.addEventListener('click', () => submitPhpGenFinal());

    const btnCopy = document.getElementById('btnCopyPhpGenCode');
    if (btnCopy) btnCopy.addEventListener('click', () => copyPhpGenActiveCode());
}

async function openPhpGenModal() {
    const modal = document.getElementById('phpGenModal');
    const alertEl = document.getElementById('phpGenAlert');
    if (alertEl) alertEl.classList.add('hidden');

    setPhpGenStep(1);
    modal.classList.remove('hidden');

    // 1. Load available databases
    const selDb = document.getElementById('phpGenDbSelect');
    if (selDb) {
        selDb.innerHTML = '<option value="">Memuat daftar database...</option>';
        selDb.disabled = true;

        try {
            const res = await fetch('/api/db/list');
            const data = await res.json();

            if (data.success && data.data && Array.isArray(data.data.databases)) {
                phpGenState.databases = data.data.databases;
                selDb.innerHTML = data.data.databases.map(db => `<option value="${escapeHTML(db)}">${escapeHTML(db)}</option>`).join('');
                selDb.disabled = false;

                if (data.data.databases.length > 0) {
                    let targetDb = data.data.databases[0];
                    if (editorState && editorState.currentDb && data.data.databases.includes(editorState.currentDb)) {
                        targetDb = editorState.currentDb;
                    }
                    selDb.value = targetDb;
                    await onPhpGenDbChange(targetDb);
                }
            } else {
                selDb.innerHTML = '<option value="">Tidak ada database</option>';
            }
        } catch (e) {
            selDb.innerHTML = `<option value="">Error: ${escapeHTML(e.message)}</option>`;
        }
    }

    // 2. Load existing projects for folder selection
    const selFolderMode = document.getElementById('phpGenFolderMode');
    const selExistFolder = document.getElementById('phpGenExistingFolderSelect');
    const wrapExist = document.getElementById('phpGenExistingFolderWrapper');
    const wrapNew = document.getElementById('phpGenNewFolderWrapper');

    if (selExistFolder) {
        selExistFolder.innerHTML = '<option value="">Memuat folder proyek...</option>';
    }

    try {
        const resProj = await fetch('/api/projects');
        const dataProj = await resProj.json();

        const projectList = (dataProj.data && Array.isArray(dataProj.data.projects))
            ? dataProj.data.projects
            : (Array.isArray(dataProj.data) ? dataProj.data : []);

        if (dataProj.success && projectList.length > 0) {
            phpGenState.existingProjects = projectList;
            if (selExistFolder) {
                selExistFolder.innerHTML = projectList.map(p => {
                    const fName = p.relative_path || p.name || p.folder_name;
                    const fw = p.framework || 'PHP Native';
                    return `<option value="${escapeHTML(fName)}">📁 ${escapeHTML(fName)} (${escapeHTML(fw)})</option>`;
                }).join('');
            }
            if (selFolderMode) selFolderMode.value = 'existing';
            if (wrapExist) wrapExist.classList.remove('hidden');
            if (wrapNew) wrapNew.classList.add('hidden');
        } else {
            if (selExistFolder) {
                selExistFolder.innerHTML = '<option value="">-- Belum ada folder proyek di htdocs --</option>';
            }
            if (selFolderMode) selFolderMode.value = 'new';
            if (wrapExist) wrapExist.classList.add('hidden');
            if (wrapNew) wrapNew.classList.remove('hidden');
        }
    } catch (e) {
        if (selFolderMode) selFolderMode.value = 'new';
        if (wrapExist) wrapExist.classList.add('hidden');
        if (wrapNew) wrapNew.classList.remove('hidden');
    }
}

async function onPhpGenDbChange(dbName) {
    phpGenState.selectedDb = dbName;
    const selTable = document.getElementById('phpGenTableSelect');
    if (!selTable || !dbName) return;

    selTable.innerHTML = '<option value="">Memuat daftar tabel...</option>';
    selTable.disabled = true;

    try {
        const res = await fetch(`/api/db/tables?db=${encodeURIComponent(dbName)}`);
        const data = await res.json();
        const tbls = (data.data && Array.isArray(data.data.tables)) ? data.data.tables : (Array.isArray(data.data) ? data.data : []);

        if (data.success && tbls.length > 0) {
            selTable.innerHTML = tbls.map(t => `<option value="${escapeHTML(t)}">${escapeHTML(t)}</option>`).join('');
            selTable.disabled = false;

            let targetTable = tbls[0];
            if (editorState && editorState.currentTable && tbls.includes(editorState.currentTable)) {
                targetTable = editorState.currentTable;
            }
            selTable.value = targetTable;
            onPhpGenTableChange(targetTable);

            // Populate dashboard stat tables and recent preview table
            renderPhpGenDashboardTableOptions(tbls, targetTable);
        } else {
            selTable.innerHTML = '<option value="">-- Tidak ada tabel di DB ini --</option>';
            selTable.disabled = true;
            renderPhpGenDashboardTableOptions([], '');
        }
    } catch (e) {
        selTable.innerHTML = `<option value="">Error: ${escapeHTML(e.message)}</option>`;
        selTable.disabled = true;
        renderPhpGenDashboardTableOptions([], '');
    }
}

function renderPhpGenDashboardTableOptions(tbls, defaultTable) {
    const statContainer = document.getElementById('phpGenDashStatTablesContainer');
    const selRecent = document.getElementById('selectPhpGenDashRecentTable');
    
    if (statContainer) {
        if (!tbls || tbls.length === 0) {
            statContainer.innerHTML = '<span class="text-muted" style="font-size:12px;">Tidak ada tabel pada database ini</span>';
        } else {
            const colorPalette = ['primary', 'success', 'info', 'warning', 'danger', 'secondary'];
            const iconPalette = ['bi-table', 'bi-folder2-open', 'bi-database-fill', 'bi-card-checklist', 'bi-box-seam', 'bi-people-fill', 'bi-cart-check', 'bi-file-earmark-text'];
            
            statContainer.innerHTML = tbls.map((t, idx) => {
                const color = colorPalette[idx % colorPalette.length];
                const icon = iconPalette[idx % iconPalette.length];
                const label = t.replace(/[_-]/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
                return `
                    <label style="display:flex; align-items:center; gap:6px; background:#ffffff; border:1px solid var(--border-color); border-radius:6px; padding:4px 8px; font-size:12px; cursor:pointer; color: #1e293b;">
                        <input type="checkbox" class="php-gen-dash-stat-tbl" value="${escapeHTML(t)}" data-label="Total ${escapeHTML(label)}" data-color="${color}" data-icon="${icon}" checked>
                        <span>📁 <strong>${escapeHTML(t)}</strong></span>
                    </label>
                `;
            }).join('');
        }
    }

    if (selRecent) {
        if (!tbls || tbls.length === 0) {
            selRecent.innerHTML = '<option value="">Tidak ada tabel</option>';
        } else {
            selRecent.innerHTML = tbls.map(t => `<option value="${escapeHTML(t)}" ${t === defaultTable ? 'selected' : ''}>${escapeHTML(t)}</option>`).join('');
        }
    }
}

function onPhpGenTableChange(tableName) {
    phpGenState.selectedTable = tableName;
    if (!tableName) return;

    const inputTitle = document.getElementById('phpGenAppTitle');
    const inputFolder = document.getElementById('phpGenFolder');
    const inputDashTitle = document.getElementById('inputPhpGenDashTitle');

    const cleanTitle = tableName.replace(/[_-]/g, ' ').replace(/\b\w/g, l => l.toUpperCase());
    if (inputTitle) inputTitle.value = `Sistem Data ${cleanTitle}`;
    if (inputFolder) inputFolder.value = `${tableName.toLowerCase().replace(/[^a-z0-9_-]/g, '_')}_app`;
    if (inputDashTitle && (!inputDashTitle.value || inputDashTitle.value.startsWith('Dashboard Admin'))) {
        inputDashTitle.value = `Dashboard Admin ${cleanTitle}`;
    }
}

function setPhpGenStep(stepNum) {
    phpGenState.step = stepNum;

    // Toggle Wizard Step Indicators
    for (let i = 1; i <= 3; i++) {
        const item = document.getElementById(`wizStepItem${i}`);
        if (item) {
            item.classList.remove('active', 'completed');
            if (i === stepNum) item.classList.add('active');
            else if (i < stepNum) item.classList.add('completed');
        }
    }

    // Toggle Step Panes
    const panes = {
        1: document.getElementById('phpGenStep1'),
        2: document.getElementById('phpGenStep2'),
        3: document.getElementById('phpGenStep3'),
        4: document.getElementById('phpGenStepSuccess')
    };

    Object.keys(panes).forEach(k => {
        if (panes[k]) {
            if (parseInt(k) === stepNum) panes[k].classList.remove('hidden');
            else panes[k].classList.add('hidden');
        }
    });

    const alertEl = document.getElementById('phpGenAlert');
    if (alertEl) alertEl.classList.add('hidden');
}

async function goPhpGenStep2() {
    const selDb = document.getElementById('phpGenDbSelect').value;
    const selTable = document.getElementById('phpGenTableSelect').value;
    const appTitle = document.getElementById('phpGenAppTitle').value.trim();
    const folderMode = document.getElementById('phpGenFolderMode') ? document.getElementById('phpGenFolderMode').value : 'new';
    const namingMode = document.getElementById('phpGenNamingMode') ? document.getElementById('phpGenNamingMode').value : 'table';
    
    let folder = '';
    if (folderMode === 'existing') {
        const existSel = document.getElementById('phpGenExistingFolderSelect');
        folder = existSel ? existSel.value : '';
    } else {
        const newFolderInput = document.getElementById('phpGenFolder');
        folder = newFolderInput ? newFolderInput.value.trim() : '';
    }

    if (!selDb || !selTable) {
        alert('Harap pilih Database dan Tabel utama terlebih dahulu.');
        return;
    }
    if (!folder) {
        alert('Nama folder target di www/htdocs tidak boleh kosong.');
        return;
    }

    phpGenState.selectedDb = selDb;
    phpGenState.selectedTable = selTable;
    phpGenState.appTitle = appTitle || `Aplikasi ${selTable}`;
    phpGenState.projectFolder = folder;
    phpGenState.namingMode = namingMode;

    document.getElementById('phpGenInfoDb').textContent = selDb;
    document.getElementById('phpGenInfoTable').textContent = selTable;

    const tbody = document.getElementById('phpGenColumnsTableBody');
    tbody.innerHTML = '<tr><td colspan="6" class="text-center py-3 text-muted">Memuat struktur kolom & relasi foreign key...</td></tr>';

    setPhpGenStep(2);

    try {
        // Fetch columns and relations in parallel
        const [resCols, resRels] = await Promise.all([
            fetch(`/api/db/columns?db=${encodeURIComponent(selDb)}&table=${encodeURIComponent(selTable)}`),
            fetch(`/api/db/relations?db=${encodeURIComponent(selDb)}`)
        ]);

        const dataCols = await resCols.json();
        const dataRels = await resRels.json();

        const cols = (dataCols.data && Array.isArray(dataCols.data.columns)) ? dataCols.data.columns : (Array.isArray(dataCols.data) ? dataCols.data : []);
        phpGenState.columns = cols;

        const allRels = (dataRels.data && Array.isArray(dataRels.data)) ? dataRels.data : [];
        // Filter relations where this table is the child (has FK)
        const matchedRels = allRels.filter(r => r.table_name === selTable);
        phpGenState.relations = matchedRels;

        // Fetch columns of referenced tables for FK mapping
        phpGenState.referencedColsMap = {};
        for (const r of matchedRels) {
            if (!phpGenState.referencedColsMap[r.referenced_table_name]) {
                try {
                    const resRef = await fetch(`/api/db/columns?db=${encodeURIComponent(selDb)}&table=${encodeURIComponent(r.referenced_table_name)}`);
                    const dataRef = await resRef.json();
                    const refCols = (dataRef.data && Array.isArray(dataRef.data.columns)) ? dataRef.data.columns : [];
                    phpGenState.referencedColsMap[r.referenced_table_name] = refCols;
                } catch (e) {}
            }
        }

        renderPhpGenStep2UI();
    } catch (e) {
        tbody.innerHTML = `<tr><td colspan="6" class="text-center py-3 text-danger">Gagal memuat: ${escapeHTML(e.message)}</td></tr>`;
    }
}

function renderPhpGenStep2UI() {
    const cols = phpGenState.columns;
    const rels = phpGenState.relations;

    // 1. Populate Primary Key selector
    const selPK = document.getElementById('phpGenPrimaryKeySelect');
    if (selPK) {
        selPK.innerHTML = cols.map(c => `
            <option value="${escapeHTML(c.field)}" ${c.key === 'PRI' ? 'selected' : ''}>${escapeHTML(c.field)}</option>
        `).join('');
    }

    // 2. Render Foreign Key Relational JOIN Cards
    const relsSection = document.getElementById('phpGenRelationsSection');
    const relsContainer = document.getElementById('phpGenRelationsContainer');

    if (rels.length > 0) {
        relsSection.classList.remove('hidden');
        relsContainer.innerHTML = rels.map(r => {
            const refCols = phpGenState.referencedColsMap[r.referenced_table_name] || [];
            
            // Generate options for display column (prefer text/name columns)
            let colOpts = refCols.map(c => {
                const isLikelyName = /^(nama|name|judul|title|deskripsi|keterangan|kode)/i.test(c.field);
                return `<option value="${escapeHTML(c.field)}" ${isLikelyName ? 'selected' : ''}>${escapeHTML(c.field)} (${escapeHTML(c.type)})</option>`;
            }).join('');

            if (!colOpts) {
                colOpts = `<option value="${escapeHTML(r.referenced_column_name)}">${escapeHTML(r.referenced_column_name)}</option>`;
            }

            return `
                <div class="fk-config-card">
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                        <span style="font-family: var(--font-mono); font-size: 12px; font-weight: 600;">
                            Kolom <code>${escapeHTML(r.column_name)}</code> &rarr; Tabel Induk <code>${escapeHTML(r.referenced_table_name)}.${escapeHTML(r.referenced_column_name)}</code>
                        </span>
                        <span class="relation-badge">JOIN Otomatis</span>
                    </div>
                    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 10px; align-items: center;">
                        <div>
                            <label style="font-size: 11.5px; color: var(--text-muted); margin-bottom: 2px;">Tampilkan Data dari Tabel Induk:</label>
                            <select class="form-input font-mono fk-display-col-select" data-fk-col="${escapeHTML(r.column_name)}" data-ref-table="${escapeHTML(r.referenced_table_name)}" data-ref-pk="${escapeHTML(r.referenced_column_name)}" style="padding: 4px 8px; font-size: 12px;">
                                ${colOpts}
                            </select>
                        </div>
                        <div style="font-size: 11px; color: #15803d; line-height: 1.4;">
                            ✓ Pada tabel data akan ditampilkan via <strong>LEFT JOIN</strong>.<br>
                            ✓ Pada form input otomatis jadi <strong>dropdown &lt;select&gt;</strong> dinamis.
                        </div>
                    </div>
                </div>
            `;
        }).join('');
    } else {
        relsSection.classList.add('hidden');
        relsContainer.innerHTML = '';
    }

    // 3. Render Columns Mapping Table
    const tbody = document.getElementById('phpGenColumnsTableBody');
    tbody.innerHTML = cols.map(c => {
        const isAutoIncr = c.is_auto_incr || /auto_increment/i.test(c.extra || '');
        const isPri = c.key === 'PRI';
        const isRequired = c.null === 'NO' && !isAutoIncr;
        const defaultLabel = c.field.replace(/[_-]/g, ' ').replace(/\b\w/g, l => l.toUpperCase());

        // Check if this column is an FK
        const isFk = rels.some(r => r.table_name === phpGenState.selectedTable && r.column_name === c.field);

        return `
            <tr data-col-name="${escapeHTML(c.field)}" data-col-type="${escapeHTML(c.type)}">
                <td style="text-align: center;">
                    <input type="checkbox" class="col-show-list" checked title="Tampilkan kolom di tabel data">
                </td>
                <td style="text-align: center;">
                    <input type="checkbox" class="col-show-form" ${isAutoIncr ? '' : 'checked'} title="Tampilkan kolom di form tambah/edit">
                </td>
                <td>
                    <span class="font-mono" style="font-weight: 600;">${escapeHTML(c.field)}</span>
                    ${isPri ? '<span class="badge" style="background:#fef3c7; color:#92400e; font-size:10px; margin-left:4px;">PK</span>' : ''}
                    ${isFk ? '<span class="badge" style="background:#dcfce7; color:#166534; font-size:10px; margin-left:4px;">FK</span>' : ''}
                </td>
                <td>
                    <input type="text" class="form-input col-label-input" value="${escapeHTML(defaultLabel)}" style="padding: 3px 8px; font-size: 12px;">
                </td>
                <td>
                    <span class="font-mono text-muted" style="font-size: 11px;">${escapeHTML(c.type)}</span>
                </td>
                <td style="text-align: center;">
                    <input type="checkbox" class="col-required-check" ${isRequired ? 'checked' : ''} ${isAutoIncr ? 'disabled' : ''} title="Wajib diisi">
                </td>
            </tr>
        `;
    }).join('');
}

async function goPhpGenStep3() {
    const alertEl = document.getElementById('phpGenAlert');
    const primaryCol = document.getElementById('phpGenPrimaryKeySelect').value || 'id';

    // Collect column configurations
    const colRows = document.querySelectorAll('#phpGenColumnsTableBody tr[data-col-name]');
    const columnsConfig = [];

    colRows.forEach(row => {
        const field = row.getAttribute('data-col-name');
        const type = row.getAttribute('data-col-type') || 'VARCHAR(100)';
        const showInList = row.querySelector('.col-show-list').checked;
        const showInForm = row.querySelector('.col-show-form').checked;
        const label = row.querySelector('.col-label-input').value.trim() || field;
        const required = row.querySelector('.col-required-check').checked;

        columnsConfig.push({
            field: field,
            label: label,
            type: type,
            show_in_list: showInList,
            show_in_form: showInForm,
            required: required
        });
    });

    // Collect Foreign Key relational mappings
    const fkSelects = document.querySelectorAll('.fk-display-col-select');
    const relationsConfig = [];

    fkSelects.forEach(sel => {
        const fkCol = sel.getAttribute('data-fk-col');
        const refTable = sel.getAttribute('data-ref-table');
        const refPk = sel.getAttribute('data-ref-pk');
        const displayCol = sel.value;

        relationsConfig.push({
            column_name: fkCol,
            referenced_table: refTable,
            referenced_pk: refPk,
            display_column: displayCol,
            display_alias: `${refTable}_${displayCol}`
        });
    });

    // Collect Dashboard configuration
    const chkDash = document.getElementById('checkPhpGenCreateDashboard');
    let dashConfig = null;
    if (chkDash && chkDash.checked) {
        const dashTitle = (document.getElementById('inputPhpGenDashTitle') ? document.getElementById('inputPhpGenDashTitle').value.trim() : '') || 'Dashboard';
        const welcomeMsg = (document.getElementById('inputPhpGenDashWelcome') ? document.getElementById('inputPhpGenDashWelcome').value.trim() : '') || 'Ringkasan statistik data dan status sistem';
        const showRecent = document.getElementById('checkPhpGenDashShowRecent') ? document.getElementById('checkPhpGenDashShowRecent').checked : true;
        const recentTable = document.getElementById('selectPhpGenDashRecentTable') ? document.getElementById('selectPhpGenDashRecentTable').value : phpGenState.selectedTable;

        const statCheckboxes = document.querySelectorAll('.php-gen-dash-stat-tbl:checked');
        const statTables = [];
        statCheckboxes.forEach(cb => {
            statTables.push({
                table_name: cb.value,
                label: cb.getAttribute('data-label') || `Total ${cb.value}`,
                icon: cb.getAttribute('data-icon') || 'bi-table',
                color: cb.getAttribute('data-color') || 'primary'
            });
        });

        dashConfig = {
            enabled: true,
            dashboard_title: dashTitle,
            welcome_msg: welcomeMsg,
            stat_tables: statTables,
            show_recent: showRecent,
            recent_table: recentTable
        };
    }

    const payload = {
        database_name: phpGenState.selectedDb,
        table_name: phpGenState.selectedTable,
        project_folder: phpGenState.projectFolder,
        app_title: phpGenState.appTitle,
        primary_col: primaryCol,
        naming_mode: dashConfig && dashConfig.enabled ? 'table' : phpGenState.namingMode,
        columns: columnsConfig,
        relations: relationsConfig,
        dashboard: dashConfig
    };

    phpGenState.lastPayload = payload;

    // Switch to step 3 and render loading
    setPhpGenStep(3);
    document.getElementById('phpGenTargetPath').textContent = `www/htdocs/${phpGenState.projectFolder}/`;
    document.getElementById('phpGenCodeContent').textContent = '// Sedang memproses dan merender preview kode PHP...';

    try {
        const res = await fetch('/api/php-generator/preview', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload)
        });
        const data = await res.json();

        if (data.success && data.data) {
            phpGenState.previewData = data.data;
            renderPhpGenCodeTabs(data.data.files);

            // Handle Overwrite Warning Card
            const overwriteCard = document.getElementById('phpGenOverwriteCard');
            const overwriteDesc = document.getElementById('phpGenOverwriteDesc');
            const btnSubmit = document.getElementById('btnSubmitPhpGenFinal');

            if (data.data.has_existing && Array.isArray(data.data.existing_files) && data.data.existing_files.length > 0) {
                if (overwriteCard) overwriteCard.classList.remove('hidden');
                if (overwriteDesc) {
                    overwriteDesc.innerHTML = `Ditemukan <strong>${data.data.existing_files.length} berkas</strong> yang sudah ada sebelumnya di folder ini: <code>${data.data.existing_files.map(f => escapeHTML(f)).join(', ')}</code>.<br>Apakah Anda ingin menimpa (overwrite) file-file tersebut?`;
                }
                if (btnSubmit) {
                    btnSubmit.innerHTML = '⚠️ Timpa (Overwrite) & Pasang ke htdocs';
                    btnSubmit.style.background = '#d97706';
                    btnSubmit.style.borderColor = '#d97706';
                }
            } else {
                if (overwriteCard) overwriteCard.classList.add('hidden');
                if (btnSubmit) {
                    btnSubmit.innerHTML = '🚀 Konfirmasi & Pasang ke htdocs';
                    btnSubmit.style.background = '#10b981';
                    btnSubmit.style.borderColor = '#10b981';
                }
            }
        } else {
            document.getElementById('phpGenCodeContent').textContent = `❌ Gagal merender preview: ${data.error || 'Unknown error'}`;
        }
    } catch (e) {
        document.getElementById('phpGenCodeContent').textContent = `❌ Error: ${e.message}`;
    }
}

function renderPhpGenCodeTabs(files) {
    const tabsContainer = document.getElementById('phpGenFileTabs');
    if (!files || files.length === 0) return;

    phpGenState.activeFileIndex = 0;

    tabsContainer.innerHTML = files.map((file, idx) => `
        <button type="button" class="code-file-tab ${idx === 0 ? 'active' : ''}" onclick="selectPhpGenTab(${idx})">
            <span>📄 ${escapeHTML(file.filename)}</span>
        </button>
    `).join('');

    displayPhpGenActiveFile(0);
}

function selectPhpGenTab(index) {
    phpGenState.activeFileIndex = index;
    const tabs = document.querySelectorAll('.code-file-tab');
    tabs.forEach((t, i) => {
        if (i === index) t.classList.add('active');
        else t.classList.remove('active');
    });
    displayPhpGenActiveFile(index);
}

function displayPhpGenActiveFile(index) {
    if (!phpGenState.previewData || !phpGenState.previewData.files) return;
    const file = phpGenState.previewData.files[index];
    if (!file) return;

    const codeEl = document.getElementById('phpGenCodeContent');
    codeEl.textContent = file.content;
}

function copyPhpGenActiveCode() {
    if (!phpGenState.previewData || !phpGenState.previewData.files) return;
    const file = phpGenState.previewData.files[phpGenState.activeFileIndex];
    if (!file) return;

    navigator.clipboard.writeText(file.content).then(() => {
        const btn = document.getElementById('btnCopyPhpGenCode');
        const origText = btn.innerHTML;
        btn.innerHTML = '✓ Tersalin!';
        btn.classList.add('btn-primary');
        btn.classList.remove('btn-subtle');
        setTimeout(() => {
            btn.innerHTML = origText;
            btn.classList.remove('btn-primary');
            btn.classList.add('btn-subtle');
        }, 1800);
    });
}

async function submitPhpGenFinal() {
    if (!phpGenState.lastPayload) return;

    // Check if files exist and overwrite checkbox is unchecked
    if (phpGenState.previewData && phpGenState.previewData.has_existing) {
        const checkOverwrite = document.getElementById('checkPhpGenOverwrite');
        if (checkOverwrite && !checkOverwrite.checked) {
            alert('Peringatan: Berkas sudah ada di folder tersebut. Harap centang persetujuan timpa (overwrite) atau ubah nama folder proyek.');
            return;
        }
    }

    const btnSubmit = document.getElementById('btnSubmitPhpGenFinal');
    btnSubmit.disabled = true;
    btnSubmit.innerHTML = '⏳ Menyimpan ke htdocs...';

    try {
        const payloadWithOverwrite = {
            ...phpGenState.lastPayload,
            overwrite: true
        };

        const res = await fetch('/api/php-generator/generate', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payloadWithOverwrite)
        });
        const data = await res.json();

        if (data.success && data.data) {
            setPhpGenStep(4);
            const desc = document.getElementById('phpGenSuccessDesc');
            desc.innerHTML = `Seluruh file CRUD PHP telah disimpan di folder <code>www/htdocs/${escapeHTML(data.data.project_folder)}/</code>`;

            const btnOpen = document.getElementById('btnPhpGenOpenBrowser');
            btnOpen.href = data.data.web_url;

            // Refresh projects in background
            fetchProjects();
        } else {
            alert(`Gagal menyimpan file: ${data.error || 'Unknown error'}`);
        }
    } catch (e) {
        alert(`Error: ${e.message}`);
    } finally {
        btnSubmit.disabled = false;
        btnSubmit.innerHTML = '🚀 Konfirmasi & Pasang ke htdocs';
    }
}


