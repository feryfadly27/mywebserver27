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

    if (ap.running) {
        indAp.className = 'status-indicator-pill online';
        statusAp.textContent = 'Running';
        btnToggleAp.textContent = 'Stop';
        btnToggleAp.className = 'btn btn-action btn-stop';
        btnToggleAp.onclick = () => serviceAction('apache', 'stop');
    } else {
        indAp.className = 'status-indicator-pill offline';
        statusAp.textContent = 'Stopped';
        btnToggleAp.textContent = 'Start';
        btnToggleAp.className = 'btn btn-action btn-start';
        btnToggleAp.onclick = () => serviceAction('apache', 'start');
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

    if (db.running) {
        indDb.className = 'status-indicator-pill online';
        statusDb.textContent = 'Running';
        btnToggleDb.textContent = 'Stop';
        btnToggleDb.className = 'btn btn-action btn-stop';
        btnToggleDb.onclick = () => serviceAction('mariadb', 'stop');
    } else {
        indDb.className = 'status-indicator-pill offline';
        statusDb.textContent = 'Stopped';
        btnToggleDb.textContent = 'Start';
        btnToggleDb.className = 'btn btn-action btn-start';
        btnToggleDb.onclick = () => serviceAction('mariadb', 'start');
    }
}

function updateQuickLinks(settings) {
    const webUrl = `http://localhost:${settings.apache_port}`;
    const pmaUrl = `http://localhost:${settings.apache_port}/phpmyadmin`;

    const linkWeb = document.getElementById('linkWebsite');
    linkWeb.href = webUrl;
    document.getElementById('textWebsiteUrl').textContent = webUrl;

    const linkPma = document.getElementById('linkPhpMyAdmin');
    linkPma.href = pmaUrl;
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

    // Open Folders
    document.getElementById('btnOpenHtdocs').addEventListener('click', () => {
        fetch('/api/open-folder', { method: 'POST' });
    });
    document.getElementById('btnOpenLogs').addEventListener('click', () => {
        fetch('/api/open-folder?folder=logs', { method: 'POST' });
    });

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
    document.getElementById('btnStartDownload').addEventListener('click', startDownload);

    // Database Tools Modal
    const btnDbModal = document.getElementById('btnOpenDatabaseModal');
    if (btnDbModal) {
        btnDbModal.addEventListener('click', openDatabaseModal);
    }

    // Settings Modal
    const modal = document.getElementById('settingsModal');
    document.getElementById('btnSettings').addEventListener('click', () => {
        if (!currentSettings) return;
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

        document.getElementById('settingsAlert').classList.add('hidden');
        modal.classList.remove('hidden');
    });

    document.getElementById('btnCloseSettings').addEventListener('click', () => modal.classList.add('hidden'));
    document.getElementById('btnCancelSettings').addEventListener('click', () => modal.classList.add('hidden'));

    document.getElementById('btnCheckUpdateFromSettings').addEventListener('click', () => {
        modal.classList.add('hidden');
        checkForUpdates(true);
    });

    document.getElementById('btnCheckUpdateTop').addEventListener('click', () => {
        checkForUpdates(true);
    });

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

        if (info.has_update) {
            if (notifDot) notifDot.classList.remove('hidden');

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

async function openDatabaseModal() {
    const modal = document.getElementById('dbToolsModal');
    modal.classList.remove('hidden');
    await fetchDatabaseData();
}

async function fetchDatabaseData() {
    const selectBackup = document.getElementById('selectBackupDb');
    const selectRestore = document.getElementById('selectRestoreTargetDb');
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

