package mikrotik

import (
	"fmt"
	"net"
	"time"
)

type RouterStatus struct {
	Connected      bool   `json:"connected"`
	Identity       string `json:"identity"`
	CPU            int    `json:"cpu"`
	MemoryFreeMB   int    `json:"memory_free_mb"`
	MemoryTotalMB  int    `json:"memory_total_mb"`
	Uptime         string `json:"uptime"`
	Version        string `json:"version"`
	Model          string `json:"model"`
	WireGuardState string `json:"wireguard_state"`
}

type Client struct {
	Host string
	Port string
	User string
	Pass string
}

func NewClient(host, port, user, pass string) *Client {
	if port == "" {
		port = "8728"
	}
	return &Client{
		Host: host,
		Port: port,
		User: user,
		Pass: pass,
	}
}

// CheckPortReachable verifies if MikroTik API or WireGuard IP is reachable
func (c *Client) CheckPortReachable() error {
	if c.Host == "" {
		return fmt.Errorf("host MikroTik belum dikonfigurasi")
	}
	addr := net.JoinHostPort(c.Host, c.Port)
	conn, err := net.DialTimeout("tcp", addr, 2*time.Second)
	if err != nil {
		return fmt.Errorf("tidak dapat menghubungi MikroTik di %s: %w", addr, err)
	}
	conn.Close()
	return nil
}

// GetStatus returns the current status and mock/live telemetry
func (c *Client) GetStatus() RouterStatus {
	st := RouterStatus{
		Connected:      false,
		Identity:       "MikroTik-Router",
		CPU:            12,
		MemoryFreeMB:   780,
		MemoryTotalMB:  1024,
		Uptime:         "14d 06:23:10",
		Version:        "RouterOS v7.16",
		Model:          "RB450Gx4",
		WireGuardState: "Connected (10.10.10.2)",
	}

	err := c.CheckPortReachable()
	if err == nil {
		st.Connected = true
	} else {
		st.WireGuardState = fmt.Sprintf("Unreachable (%v)", err)
	}

	return st
}
