# ⚡ GenieACS Pro Dashboard (Native Go Engine)

Aplikasi **Dashboard GenieACS & Network Monitoring** mandiri berperforma tinggi yang dibangun dari 0 menggunakan **Golang**, dirancang khusus untuk memonitor perangkat TR-069 CPE/ONT serta terintegrasi langsung dengan **MikroTik RouterOS v7** melalui tunnel **WireGuard**.

---

## ✨ Fitur Utama

- **🚀 Ultra Ringan & Hemat RAM:** Hanya mengonsumsi **~15 MB – 25 MB RAM** (dibandingkan 150+ MB pada PHP/Nginx).
- **📦 Single Native Binary:** Seluruh antarmuka web (HTML, CSS, JS) di-*embed* langsung ke dalam 1 file binary mandiri (`embed.FS`). Tanpa perlu PHP, Apache, Nginx, ataupun Composer!
- **🎨 Modern Cyber-Glassmorphism UI:** Desain antarmuka gelap modern (*dark mode*) yang responsif dengan efek kaca, animasi halus, dan visualisasi metrik real-time.
- **⚡ GenieACS NBI Client:** Komunikasi langsung ke GenieACS Northbound Interface (Port `7557`) untuk mengambil data ONT, status online/offline, dan redaman optik (*Optical Rx Power*).
- **🌐 MikroTik RouterOS v7 API & WireGuard:** Memantau CPU load, memory, uptime, dan status tunnel WireGuard secara real-time.
- **🗺️ Topologi Jaringan Visual:** Peta kanvas interaktif yang memetakan jalur konektivitas `MikroTik -> OLT -> Splitter ODP -> ONT Pelanggan`.
- **🔄 Reboot ONT Instan:** Kirim task reboot langsung ke modem pelanggan dari dashboard.
- **💾 SQLite Terintegrasi:** Database internal otomatis tanpa perlu menginstal MySQL/MariaDB server terpisah.

---

## 🏗️ Struktur Direktori

```text
GenieACS-Dashboard/
├── cmd/
│   └── server/
│       └── main.go                 # Entrypoint server Golang
├── internal/
│   ├── config/                     # Konfigurasi aplikasi & port
│   ├── acs/                        # GenieACS NBI client (Port 7557)
│   ├── mikrotik/                   # MikroTik RouterOS API client
│   ├── db/                         # Database SQLite (Settings & Topologi)
│   └── handlers/                   # Web controllers & REST APIs
├── web/
│   ├── embed.go                    # Go embed filesystem
│   ├── static/
│   │   ├── css/style.css           # Glassmorphism design system
│   │   └── js/app.js               # Polling real-time & canvas topologi
│   └── templates/                  # Template HTML (Dashboard, Devices, Topology, dll)
├── installer/
│   └── install.sh                  # 1-Click Master VPS Auto Installer
├── genieacs-dashboard-linux-amd64  # Binary compiled siap pakai untuk Linux 64-bit
└── go.mod
```

---

## 💻 Menjalankan di Komputer Lokal (Windows)

```bash
cd GenieACS-Dashboard

# 1. Jalankan langsung dengan Go:
go run ./cmd/server

# Atau compile ke file .exe:
go build -o genieacs-dashboard.exe ./cmd/server
./genieacs-dashboard.exe
```

Akses di browser Anda: `http://localhost:8080/`

---

## 🐧 Compile Binary untuk VPS Linux (64-bit)

Karena menggunakan SQLite berbasis *Pure Go*, Anda dapat mengompilasi binary untuk Linux langsung dari Windows tanpa butuh GCC/CGO:

```powershell
cmd /c "set GOOS=linux&& set GOARCH=amd64&& set CGO_ENABLED=0&& go build -o genieacs-dashboard-linux-amd64 ./cmd/server"
```

---

## 🚀 1-Click Auto Installer di VPS Linux

Jalankan perintah ini di VPS Anda untuk menginstal **GenieACS Server + MongoDB 4.4 + WireGuard + Dashboard Go** sekaligus:

```bash
cd ~
git clone https://github.com/aanmochtar80/Mikrotik-Script.git
cd Mikrotik-Script/GenieACS-Dashboard
sudo chmod +x installer/install.sh
sudo ./installer/install.sh
```

### Port Layanan:
* **Web Dashboard:** `http://IP_VPS:8080/`
* **GenieACS Admin UI:** `http://IP_VPS:3000/`
* **CWMP TR-069:** `Port 7547 / TCP`
* **WireGuard Tunnel:** `Port 51820 / UDP` (Gateway `10.10.10.1`)
