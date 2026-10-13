package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	ServerPort   string
	DataDir      string
	DBPath       string
	GenieACSHost string
	GenieACSPort string
	MikroTikHost string
	MikroTikPort string
	MikroTikUser string
	MikroTikPass string
	WireGuardIP  string
}

func LoadConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	dataDir := os.Getenv("DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(homeDir, ".genieacs-dashboard")
	}
	os.MkdirAll(dataDir, 0755)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	acsHost := os.Getenv("GENIEACS_HOST")
	if acsHost == "" {
		acsHost = "127.0.0.1"
	}

	acsPort := os.Getenv("GENIEACS_PORT")
	if acsPort == "" {
		acsPort = "7557"
	}

	return &Config{
		ServerPort:   port,
		DataDir:      dataDir,
		DBPath:       filepath.Join(dataDir, "dashboard.db"),
		GenieACSHost: acsHost,
		GenieACSPort: acsPort,
		MikroTikHost: os.Getenv("MIKROTIK_HOST"),
		MikroTikPort: "8728",
		MikroTikUser: os.Getenv("MIKROTIK_USER"),
		MikroTikPass: os.Getenv("MIKROTIK_PASS"),
		WireGuardIP:  "10.10.10.1",
	}
}
