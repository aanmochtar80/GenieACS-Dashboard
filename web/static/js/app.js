// ==============================================================================
// GenieACS Dashboard - Frontend Dynamic Logic
// ==============================================================================

document.addEventListener('DOMContentLoaded', () => {
    initLiveStats();
    initSearchFilter();
    initTopology();
});

// Toast Notification
function showToast(message, type = 'success') {
    let container = document.getElementById('toast-container');
    if (!container) {
        container = document.createElement('div');
        container.id = 'toast-container';
        container.style.cssText = 'position: fixed; top: 24px; right: 24px; z-index: 9999; display: flex; flex-direction: column; gap: 10px;';
        document.body.appendChild(container);
    }

    const toast = document.createElement('div');
    const color = type === 'success' ? '#10b981' : (type === 'error' ? '#ef4444' : '#06b6d4');
    toast.style.cssText = `
        background: rgba(15, 23, 42, 0.95);
        color: #fff;
        border-left: 4px solid ${color};
        padding: 14px 20px;
        border-radius: 8px;
        box-shadow: 0 10px 25px rgba(0,0,0,0.5);
        font-size: 0.9rem;
        backdrop-filter: blur(10px);
        transition: all 0.3s ease;
        opacity: 0;
        transform: translateY(-10px);
    `;
    toast.textContent = message;
    container.appendChild(toast);

    setTimeout(() => {
        toast.style.opacity = '1';
        toast.style.transform = 'translateY(0)';
    }, 10);

    setTimeout(() => {
        toast.style.opacity = '0';
        toast.style.transform = 'translateY(-10px)';
        setTimeout(() => toast.remove(), 300);
    }, 3500);
}

// Live stats polling
function initLiveStats() {
    const totalEl = document.getElementById('stat-total-devices');
    const onlineEl = document.getElementById('stat-online-devices');
    const offlineEl = document.getElementById('stat-offline-devices');

    if (!totalEl) return;

    setInterval(async () => {
        try {
            const res = await fetch('/api/summary');
            if (res.ok) {
                const data = await res.json();
                totalEl.textContent = data.total_devices || 0;
                if (onlineEl) onlineEl.textContent = data.online_devices || 0;
                if (offlineEl) offlineEl.textContent = data.offline_devices || 0;
            }
        } catch (e) {
            console.warn('Gagal memperbarui live stats:', e);
        }
    }, 5000);
}

// Search and filter in devices table
function initSearchFilter() {
    const searchInput = document.getElementById('device-search');
    if (!searchInput) return;

    searchInput.addEventListener('input', (e) => {
        const query = e.target.value.toLowerCase();
        const rows = document.querySelectorAll('#devices-table-body tr');

        rows.forEach(row => {
            const text = row.textContent.toLowerCase();
            if (text.includes(query)) {
                row.style.display = '';
            } else {
                row.style.display = 'none';
            }
        });
    });
}

// Reboot device handler
async function rebootDevice(id) {
    if (!confirm(`Apakah Anda yakin ingin me-reboot ONT dengan ID: ${id}?`)) {
        return;
    }

    try {
        const res = await fetch(`/api/devices/reboot/${encodeURIComponent(id)}`, { method: 'POST' });
        const data = await res.json();
        if (res.ok) {
            showToast(data.message || 'Perintah reboot berhasil dikirim!', 'success');
        } else {
            showToast(data.error || 'Gagal mengirim perintah reboot', 'error');
        }
    } catch (err) {
        showToast('Koneksi ke server gagal: ' + err, 'error');
    }
}

// Test ACS Connection
async function testACSConnection() {
    showToast('Menguji koneksi ke GenieACS NBI...', 'info');
    try {
        const res = await fetch('/api/test-acs', { method: 'POST' });
        const data = await res.json();
        if (data.success) {
            showToast(data.message, 'success');
        } else {
            showToast('Gagal: ' + data.error, 'error');
        }
    } catch (e) {
        showToast('Error: ' + e, 'error');
    }
}

// Test MikroTik Connection
async function testMikroTikConnection() {
    showToast('Menguji koneksi ke MikroTik API...', 'info');
    try {
        const res = await fetch('/api/test-mikrotik', { method: 'POST' });
        const data = await res.json();
        if (data.success) {
            showToast(data.message, 'success');
        } else {
            showToast('Gagal: ' + data.error, 'error');
        }
    } catch (e) {
        showToast('Error: ' + e, 'error');
    }
}

// Interactive Network Topology Map
function initTopology() {
    const canvas = document.getElementById('topology-canvas');
    if (!canvas) return;

    const ctx = canvas.getContext('2d');
    function resize() {
        canvas.width = canvas.parentElement.clientWidth;
        canvas.height = 480;
        drawMap();
    }

    const nodes = [
        { id: '1', label: 'MikroTik Core (ROS7)', type: 'mikrotik', x: 120, y: 240, status: 'online' },
        { id: '2', label: 'OLT EPON/GPON', type: 'olt', x: 380, y: 240, status: 'online' },
        { id: '3', label: 'ODP-01 (Kawasan A)', type: 'odp', x: 640, y: 140, status: 'online' },
        { id: '4', label: 'ODP-02 (Kawasan B)', type: 'odp', x: 640, y: 340, status: 'online' },
        { id: '5', label: 'ONT Pelanggan 01', type: 'ont', x: 880, y: 90, status: 'online', rx: '-18.4 dBm' },
        { id: '6', label: 'ONT Pelanggan 02', type: 'ont', x: 880, y: 180, status: 'online', rx: '-21.2 dBm' },
        { id: '7', label: 'ONT Pelanggan 03', type: 'ont', x: 880, y: 300, status: 'online', rx: '-19.1 dBm' },
        { id: '8', label: 'ONT Pelanggan 04', type: 'ont', x: 880, y: 390, status: 'offline', rx: 'LOS' },
    ];

    const links = [
        ['1', '2'],
        ['2', '3'],
        ['2', '4'],
        ['3', '5'],
        ['3', '6'],
        ['4', '7'],
        ['4', '8'],
    ];

    function drawMap() {
        ctx.clearRect(0, 0, canvas.width, canvas.height);

        // Draw Links
        links.forEach(([fromId, toId]) => {
            const from = nodes.find(n => n.id === fromId);
            const to = nodes.find(n => n.id === toId);
            if (!from || !to) return;

            ctx.beginPath();
            ctx.moveTo(from.x, from.y);
            ctx.lineTo(to.x, to.y);
            ctx.strokeStyle = to.status === 'offline' ? 'rgba(239, 68, 68, 0.5)' : 'rgba(6, 182, 212, 0.4)';
            ctx.lineWidth = 2;
            if (to.status === 'offline') {
                ctx.setLineDash([5, 5]);
            } else {
                ctx.setLineDash([]);
            }
            ctx.stroke();
            ctx.setLineDash([]);
        });

        // Draw Nodes
        nodes.forEach(n => {
            ctx.beginPath();
            ctx.arc(n.x, n.y, n.type === 'mikrotik' || n.type === 'olt' ? 22 : 16, 0, Math.PI * 2);
            ctx.fillStyle = n.status === 'offline' ? '#ef4444' : (n.type === 'mikrotik' ? '#6366f1' : (n.type === 'olt' ? '#06b6d4' : '#10b981'));
            ctx.shadowColor = ctx.fillStyle;
            ctx.shadowBlur = 12;
            ctx.fill();
            ctx.shadowBlur = 0;

            // Border
            ctx.strokeStyle = '#fff';
            ctx.lineWidth = 2;
            ctx.stroke();

            // Label
            ctx.fillStyle = '#f8fafc';
            ctx.font = '600 12px Outfit, sans-serif';
            ctx.textAlign = 'center';
            ctx.fillText(n.label, n.x, n.y + 34);

            if (n.rx) {
                ctx.fillStyle = n.rx === 'LOS' ? '#ef4444' : '#94a3b8';
                ctx.font = '500 11px JetBrains Mono, monospace';
                ctx.fillText(n.rx, n.x, n.y + 48);
            }
        });
    }

    window.addEventListener('resize', resize);
    resize();
}
