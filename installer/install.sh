#!/usr/bin/env bash
# ==============================================================================
# Script Name   : install.sh
# Description   : 1-Click Master Installer for GenieACS Server + WireGuard + Go Dashboard
# Architecture  : x86_64 / amd64 Linux (Ubuntu 20.04, 22.04, 24.04, Debian 11/12)
# ==============================================================================

set -o pipefail

RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\133[1;33m'; BLUE='\033[0;34m'; PURPLE='\033[0;35m'; CYAN='\033[0;36m'; BOLD='\033[1m'; NC='\033[0m'
info()    { echo -e "${CYAN}[INFO]${NC} $1"; }
success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
warn()    { echo -e "${YELLOW}[WARNING]${NC} $1"; }
error()   { echo -e "${RED}[ERROR]${NC} $1"; }
title()   { echo -e "\n${BOLD}${BLUE}================================================================${NC}\n${BOLD}${PURPLE}  $1${NC}\n${BOLD}${BLUE}================================================================${NC}\n"; }

if [[ $EUID -ne 0 ]]; then
    error "Skrip ini harus dijalankan sebagai root (atau gunakan sudo)."
    exit 1
fi

. /etc/os-release
OS_ID=$ID
OS_CODENAME=$VERSION_CODENAME

# ------------------------------------------------------------------------------
# 1. DEPENDENSI DASAR
# ------------------------------------------------------------------------------
title "1. Memasang Dependensi Sistem"
export DEBIAN_FRONTEND=noninteractive
apt-get update -y
apt-get install -y curl wget git unzip zip tar ca-certificates apt-transport-https lsb-release gnupg ufw wireguard wireguard-tools

# ------------------------------------------------------------------------------
# 2. MONGODB 4.4 (NON-AVX COMPATIBLE)
# ------------------------------------------------------------------------------
title "2. Menginstal MongoDB Server (Versi 4.4 Ramah Semua Tipe VPS)"
if ! (command -v mongod >/dev/null 2>&1 && systemctl is-active --quiet mongod); then
    systemctl stop mongod 2>/dev/null || true
    rm -f /etc/apt/sources.list.d/mongodb-org-*.list

    if [[ "$OS_ID" == "ubuntu" && "$OS_CODENAME" == "jammy" ]]; then
        wget -q http://archive.ubuntu.com/ubuntu/pool/main/o/openssl/libssl1.1_1.1.0g-2ubuntu4_amd64.deb -O /tmp/libssl1.1.deb
        dpkg -i /tmp/libssl1.1.deb 2>/dev/null || true
        rm -f /tmp/libssl1.1.deb
    fi

    curl -fsSL https://www.mongodb.org/static/pgp/server-4.4.asc | gpg --dearmor -o /usr/share/keyrings/mongodb-server-4.4.gpg --yes 2>/dev/null || true
    echo "deb [ arch=amd64,arm64 signed-by=/usr/share/keyrings/mongodb-server-4.4.gpg ] https://repo.mongodb.org/apt/ubuntu focal/mongodb-org/4.4 multiverse" > /etc/apt/sources.list.d/mongodb-org-4.4.list

    apt-get update -y
    apt-get install -y mongodb-org mongodb-org-tools 2>/dev/null || apt-get install -y mongodb
    apt-mark hold mongodb-org mongodb-org-server mongodb-org-shell mongodb-org-mongos mongodb-org-tools 2>/dev/null || true

    systemctl daemon-reload
    systemctl start mongod 2>/dev/null || service mongod start
    systemctl enable mongod 2>/dev/null || true
fi
success "MongoDB Server aktif."

# ------------------------------------------------------------------------------
# 3. GENIEACS SERVER v1.2.13
# ------------------------------------------------------------------------------
title "3. Menginstal Node.js & GenieACS Server Engine"
if ! command -v node >/dev/null 2>&1; then
    curl -fsSL https://deb.nodesource.com/setup_18.x | bash -
    apt-get install -y nodejs
fi

if ! command -v genieacs-nbi >/dev/null 2>&1; then
    npm install -g genieacs@1.2.13
fi

id -u genieacs &>/dev/null || useradd --system --no-create-home --user-group genieacs
mkdir -p /opt/genieacs/ext /var/log/genieacs
chown -R genieacs:genieacs /opt/genieacs /var/log/genieacs

if [[ ! -f /opt/genieacs/genieacs.env ]]; then
    JWT_SECRET=$(node -e "console.log(require('crypto').randomBytes(128).toString('hex'))" 2>/dev/null || openssl rand -hex 128)
    cat << EOF > /opt/genieacs/genieacs.env
GENIEACS_CWMP_ACCESS_LOG_FILE=/var/log/genieacs/genieacs-cwmp-access.log
GENIEACS_NBI_ACCESS_LOG_FILE=/var/log/genieacs/genieacs-nbi-access.log
GENIEACS_FS_ACCESS_LOG_FILE=/var/log/genieacs/genieacs-fs-access.log
GENIEACS_UI_ACCESS_LOG_FILE=/var/log/genieacs/genieacs-ui-access.log
GENIEACS_DEBUG_FILE=/var/log/genieacs/genieacs-debug.yaml
NODE_OPTIONS=--enable-source-maps
GENIEACS_EXT_DIR=/opt/genieacs/ext
GENIEACS_UI_JWT_SECRET=${JWT_SECRET}
GENIEACS_NBI_IP=0.0.0.0
EOF
    chown genieacs:genieacs /opt/genieacs/genieacs.env
    chmod 600 /opt/genieacs/genieacs.env
fi

GENIEACS_BIN_DIR=$(dirname "$(command -v genieacs-cwmp || echo "/usr/bin/genieacs-cwmp")")
for service in cwmp nbi fs ui; do
    cat << EOF > /etc/systemd/system/genieacs-$service.service
[Unit]
Description=GenieACS $service
After=network.target mongod.service

[Service]
User=genieacs
EnvironmentFile=/opt/genieacs/genieacs.env
ExecStart=${GENIEACS_BIN_DIR}/genieacs-$service
Restart=always
RestartSec=3

[Install]
WantedBy=default.target
EOF
done

systemctl daemon-reload
for service in cwmp nbi fs ui; do
    systemctl enable genieacs-$service 2>/dev/null || true
    systemctl restart genieacs-$service 2>/dev/null || true
done
success "GenieACS Server aktif (CWMP:7547, NBI:7557, UI:3000)."

# Restore parameter presets jika belum ada
if ! mongosh --eval "db.getSiblingDB('genieacs').getCollection('presets').countDocuments()" 2>/dev/null | grep -q "[1-9]"; then
    rm -rf /tmp/gacs-param
    if git clone --depth=1 https://github.com/safrinnetwork/GACS-Ubuntu-22.04.git /tmp/gacs-param 2>/dev/null; then
        mongorestore --db genieacs --drop /tmp/gacs-param/parameter 2>/dev/null || true
        systemctl restart genieacs-cwmp genieacs-ui genieacs-nbi 2>/dev/null || true
        rm -rf /tmp/gacs-param
        success "Parameter Virtual & Presets berhasil diimpor!"
    fi
fi

# ------------------------------------------------------------------------------
# 4. WIREGUARD TUNNEL
# ------------------------------------------------------------------------------
title "4. Menyiapkan WireGuard VPN Tunnel"
sysctl -w net.ipv4.ip_forward=1 >/dev/null
sed -i 's/#net.ipv4.ip_forward=1/net.ipv4.ip_forward=1/' /etc/sysctl.conf 2>/dev/null || true

if [[ ! -f /etc/wireguard/wg0.conf ]]; then
    mkdir -p /etc/wireguard && chmod 700 /etc/wireguard
    SERVER_PRIV=$(wg genkey); SERVER_PUB=$(echo "$SERVER_PRIV" | wg pubkey)
    MIKROTIK_PRIV=$(wg genkey); MIKROTIK_PUB=$(echo "$MIKROTIK_PRIV" | wg pubkey)

    cat << EOF > /etc/wireguard/wg0.conf
[Interface]
Address = 10.10.10.1/24
ListenPort = 51820
PrivateKey = ${SERVER_PRIV}

[Peer]
PublicKey = ${MIKROTIK_PUB}
AllowedIPs = 10.10.10.2/32
EOF
    chmod 600 /etc/wireguard/wg0.conf
    systemctl enable wg-quick@wg0 2>/dev/null || true
    systemctl restart wg-quick@wg0
    success "WireGuard Server aktif pada IP: 10.10.10.1/24 (Port UDP 51820)."
else
    success "WireGuard sudah terkonfigurasi."
fi

# ------------------------------------------------------------------------------
# 5. INSTALASI GENIEACS-DASHBOARD (GOLANG NATIVE BINARY)
# ------------------------------------------------------------------------------
title "5. Memasang GenieACS Pro Dashboard (Golang Native Binary)"

# Pasang binary dari folder lokal atau repositori
INSTALL_DIR="/usr/local/bin"
if [[ -f "./genieacs-dashboard-linux-amd64" ]]; then
    cp ./genieacs-dashboard-linux-amd64 "${INSTALL_DIR}/genieacs-dashboard"
elif [[ -f "../genieacs-dashboard-linux-amd64" ]]; then
    cp ../genieacs-dashboard-linux-amd64 "${INSTALL_DIR}/genieacs-dashboard"
else
    info "Mengunduh binary rilis GenieACS Dashboard terbaru..."
    curl -sSL -o "${INSTALL_DIR}/genieacs-dashboard" https://raw.githubusercontent.com/aanmochtar80/GenieACS-Dashboard/main/genieacs-dashboard-linux-amd64
fi

chmod +x "${INSTALL_DIR}/genieacs-dashboard"

# Buat systemd service untuk Dashboard Go
cat << EOF > /etc/systemd/system/genieacs-dashboard.service
[Unit]
Description=GenieACS Pro Dashboard (Native Go)
After=network.target genieacs-nbi.service

[Service]
Type=simple
User=root
Environment="PORT=8080"
Environment="GENIEACS_HOST=127.0.0.1"
Environment="GENIEACS_PORT=7557"
ExecStart=${INSTALL_DIR}/genieacs-dashboard
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload
systemctl enable genieacs-dashboard
systemctl restart genieacs-dashboard

# Buka Firewall Ports
ufw allow 8080/tcp 2>/dev/null || true   # Dashboard Web UI
ufw allow 3000/tcp 2>/dev/null || true   # GenieACS Admin UI
ufw allow 7547/tcp 2>/dev/null || true   # CWMP TR-069
ufw allow 7557/tcp 2>/dev/null || true   # NBI REST API
ufw allow 51820/udp 2>/dev/null || true  # WireGuard VPN

SERVER_IP=$(curl -4 -s https://api.ipify.org || hostname -I | awk '{print $1}')

title "SEMUA SISTEM TELAH AKTIF SEMPURNA! 🎉"
echo -e "${BOLD}${CYAN}Akses Dashboard & Layanan:${NC}"
echo -e "----------------------------------------------------------------"
echo -e "⚡ ${BOLD}GenieACS Pro Dashboard (Go):${NC} ${GREEN}http://${SERVER_IP}:8080/${NC}"
echo -e "🛠️  ${BOLD}GenieACS Admin UI          :${NC} ${GREEN}http://${SERVER_IP}:3000/${NC}"
echo -e "🛡️  ${BOLD}WireGuard Gateway          :${NC} ${GREEN}10.10.10.1 (UDP 51820)${NC}"
echo -e "----------------------------------------------------------------"
echo -e "\n${BOLD}${YELLOW}Konfigurasi WireGuard MikroTik ROS7:${NC}"
echo -e "${CYAN}Buka menu WireGuard di Dashboard (http://${SERVER_IP}:8080/wireguard) untuk menyalin script MikroTik siap pakai!${NC}\n"
