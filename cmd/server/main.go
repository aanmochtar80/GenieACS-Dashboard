package main

import (
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/aanmochtar80/genieacs-dashboard/internal/config"
	"github.com/aanmochtar80/genieacs-dashboard/internal/db"
	"github.com/aanmochtar80/genieacs-dashboard/internal/handlers"
	"github.com/aanmochtar80/genieacs-dashboard/web"
)

func main() {
	cfg := config.LoadConfig()

	database, err := db.InitDB(cfg.DBPath)
	if err != nil {
		log.Fatalf("Inisialisasi Database SQLite gagal: %v", err)
	}

	server, err := handlers.NewServer(cfg, database, web.Files)
	if err != nil {
		log.Fatalf("Inisialisasi Server gagal: %v", err)
	}

	mux := http.NewServeMux()

	// Static assets from embedded FS
	staticFS, err := fs.Sub(web.Files, "static")
	if err != nil {
		log.Fatalf("Gagal memuat sub-filesystem static: %v", err)
	}
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// Web Pages
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		server.HandleDashboard(w, r)
	})
	mux.HandleFunc("/devices", server.HandleDevices)
	mux.HandleFunc("/topology", server.HandleTopology)
	mux.HandleFunc("/wireguard", server.HandleWireGuard)
	mux.HandleFunc("/settings", server.HandleSettings)

	// REST APIs
	mux.HandleFunc("/api/summary", server.HandleAPISummary)
	mux.HandleFunc("/api/devices", server.HandleAPIDevices)
	mux.HandleFunc("/api/devices/reboot/", server.HandleAPIDeviceReboot)
	mux.HandleFunc("/api/mikrotik/status", server.HandleAPIMikrotik)
	mux.HandleFunc("/api/settings", server.HandleAPISettings)
	mux.HandleFunc("/api/test-acs", server.HandleAPITestACS)
	mux.HandleFunc("/api/test-mikrotik", server.HandleAPITestMikrotik)
	mux.HandleFunc("/api/topology", server.HandleAPITopology)

	listenAddr := fmt.Sprintf("0.0.0.0:%s", cfg.ServerPort)

	fmt.Println("================================================================")
	fmt.Printf("⚡ GenieACS Pro Dashboard (Native Go Engine)\n")
	fmt.Printf("   Web UI: http://localhost:%s\n", cfg.ServerPort)
	fmt.Printf("   SQLite DB: %s\n", cfg.DBPath)
	fmt.Println("================================================================")

	httpServer := &http.Server{
		Addr:         listenAddr,
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Server berhenti dengan error: %v", err)
	}
}
