package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/aanmochtar80/genieacs-dashboard/internal/acs"
	"github.com/aanmochtar80/genieacs-dashboard/internal/config"
	"github.com/aanmochtar80/genieacs-dashboard/internal/db"
	"github.com/aanmochtar80/genieacs-dashboard/internal/mikrotik"
)

type Server struct {
	cfg       *config.Config
	database  *db.Database
	templates *template.Template
}

func NewServer(cfg *config.Config, database *db.Database, webFS fs.FS) (*Server, error) {
	tmpl, err := template.ParseFS(webFS, "templates/*.html")
	if err != nil {
		return nil, fmt.Errorf("parsing templates gagal: %w", err)
	}

	return &Server{
		cfg:       cfg,
		database:  database,
		templates: tmpl,
	}, nil
}

func (s *Server) getACSClient() *acs.Client {
	host := s.database.GetSetting("acs_host")
	port := s.database.GetSetting("acs_port")
	if host == "" {
		host = s.cfg.GenieACSHost
	}
	if port == "" {
		port = s.cfg.GenieACSPort
	}
	return acs.NewClient(host, port)
}

func (s *Server) getMikroTikClient() *mikrotik.Client {
	host := s.database.GetSetting("mikrotik_host")
	port := s.database.GetSetting("mikrotik_port")
	user := s.database.GetSetting("mikrotik_user")
	pass := s.database.GetSetting("mikrotik_pass")
	return mikrotik.NewClient(host, port, user, pass)
}

// Pages
func (s *Server) HandleDashboard(w http.ResponseWriter, r *http.Request) {
	acsCli := s.getACSClient()
	summary, _ := acsCli.GetSummary()
	mkCli := s.getMikroTikClient()
	routerStatus := mkCli.GetStatus()

	data := map[string]interface{}{
		"Title":        "Dashboard Monitoring",
		"ActivePage":   "dashboard",
		"Summary":      summary,
		"RouterStatus": routerStatus,
	}
	s.templates.ExecuteTemplate(w, "dashboard.html", data)
}

func (s *Server) HandleDevices(w http.ResponseWriter, r *http.Request) {
	acsCli := s.getACSClient()
	devices, err := acsCli.GetDevices()
	if err != nil {
		devices = []acs.Device{}
	}

	data := map[string]interface{}{
		"Title":      "Daftar Perangkat CPE / ONT",
		"ActivePage": "devices",
		"Devices":    devices,
	}
	s.templates.ExecuteTemplate(w, "devices.html", data)
}

func (s *Server) HandleTopology(w http.ResponseWriter, r *http.Request) {
	nodes, _ := s.database.GetTopologyNodes()
	data := map[string]interface{}{
		"Title":      "Topologi Jaringan",
		"ActivePage": "topology",
		"Nodes":      nodes,
	}
	s.templates.ExecuteTemplate(w, "topology.html", data)
}

func (s *Server) HandleWireGuard(w http.ResponseWriter, r *http.Request) {
	mkCli := s.getMikroTikClient()
	st := mkCli.GetStatus()
	data := map[string]interface{}{
		"Title":        "WireGuard VPN Tunnel",
		"ActivePage":   "wireguard",
		"RouterStatus": st,
		"ServerIP":     s.database.GetSetting("wireguard_ip"),
		"MikroTikIP":   s.database.GetSetting("mikrotik_host"),
	}
	s.templates.ExecuteTemplate(w, "wireguard.html", data)
}

func (s *Server) HandleSettings(w http.ResponseWriter, r *http.Request) {
	settings := s.database.GetAllSettings()
	data := map[string]interface{}{
		"Title":      "Pengaturan Koneksi",
		"ActivePage": "settings",
		"Settings":   settings,
	}
	s.templates.ExecuteTemplate(w, "settings.html", data)
}

// REST APIs
func (s *Server) HandleAPISummary(w http.ResponseWriter, r *http.Request) {
	acsCli := s.getACSClient()
	summary, err := acsCli.GetSummary()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(summary)
}

func (s *Server) HandleAPIDevices(w http.ResponseWriter, r *http.Request) {
	acsCli := s.getACSClient()
	devices, err := acsCli.GetDevices()
	if err != nil {
		devices = []acs.Device{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(devices)
}

func (s *Server) HandleAPIDeviceReboot(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/devices/reboot/")
	if id == "" {
		http.Error(w, "device id required", http.StatusBadRequest)
		return
	}

	acsCli := s.getACSClient()
	if err := acsCli.RebootDevice(id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Perintah reboot berhasil dikirim"})
}

func (s *Server) HandleAPIMikrotik(w http.ResponseWriter, r *http.Request) {
	mkCli := s.getMikroTikClient()
	st := mkCli.GetStatus()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(st)
}

func (s *Server) HandleAPISettings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var payload map[string]string
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}

		for k, v := range payload {
			s.database.SetSetting(k, v)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "message": "Pengaturan berhasil disimpan"})
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(s.database.GetAllSettings())
}

func (s *Server) HandleAPITestACS(w http.ResponseWriter, r *http.Request) {
	acsCli := s.getACSClient()
	err := acsCli.TestConnection()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Berhasil terhubung ke GenieACS NBI (Port 7557)!"})
}

func (s *Server) HandleAPITestMikrotik(w http.ResponseWriter, r *http.Request) {
	mkCli := s.getMikroTikClient()
	err := mkCli.CheckPortReachable()
	w.Header().Set("Content-Type", "application/json")
	if err != nil {
		json.NewEncoder(w).Encode(map[string]interface{}{"success": false, "error": err.Error()})
		return
	}
	json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "message": "Berhasil terhubung ke MikroTik API (Port 8728)!"})
}

func (s *Server) HandleAPITopology(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		var node db.TopologyNode
		if err := json.NewDecoder(r.Body).Decode(&node); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		if err := s.database.SaveTopologyNode(node); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
		return
	}

	nodes, err := s.database.GetTopologyNodes()
	if err != nil {
		nodes = []db.TopologyNode{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(nodes)
}
