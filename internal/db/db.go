package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Database struct {
	conn *sql.DB
}

type Setting struct {
	Key   string
	Value string
}

type TopologyNode struct {
	ID        string    `json:"id"`
	Label     string    `json:"label"`
	Type      string    `json:"type"` // "mikrotik", "olt", "odp", "ont"
	IP        string    `json:"ip"`
	Status    string    `json:"status"` // "online", "offline", "warning"
	X         float64   `json:"x"`
	Y         float64   `json:"y"`
	ParentID  string    `json:"parent_id"`
	RxPower   string    `json:"rx_power"`
	UpdatedAt time.Time `json:"updated_at"`
}

func InitDB(dbPath string) (*Database, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("gagal membuka sqlite: %w", err)
	}

	d := &Database{conn: conn}
	if err := d.migrate(); err != nil {
		return nil, fmt.Errorf("gagal migrasi sqlite: %w", err)
	}

	return d, nil
}

func (d *Database) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS topology_nodes (
			id TEXT PRIMARY KEY,
			label TEXT NOT NULL,
			type TEXT NOT NULL,
			ip TEXT,
			status TEXT DEFAULT 'online',
			x REAL DEFAULT 0,
			y REAL DEFAULT 0,
			parent_id TEXT,
			rx_power TEXT,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
	}

	for _, q := range queries {
		if _, err := d.conn.Exec(q); err != nil {
			return err
		}
	}

	// Default settings
	defaults := map[string]string{
		"acs_host":      "127.0.0.1",
		"acs_port":      "7557",
		"mikrotik_host": "10.10.10.2",
		"mikrotik_port": "8728",
		"mikrotik_user": "admin",
		"mikrotik_pass": "",
		"wireguard_ip":  "10.10.10.1",
	}

	for k, v := range defaults {
		d.conn.Exec(`INSERT OR IGNORE INTO settings (key, value) VALUES (?, ?)`, k, v)
	}

	return nil
}

func (d *Database) GetSetting(key string) string {
	var val string
	row := d.conn.QueryRow(`SELECT value FROM settings WHERE key = ?`, key)
	_ = row.Scan(&val)
	return val
}

func (d *Database) SetSetting(key, val string) error {
	_, err := d.conn.Exec(`INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value`, key, val)
	return err
}

func (d *Database) GetAllSettings() map[string]string {
	rows, err := d.conn.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return make(map[string]string)
	}
	defer rows.Close()

	res := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err == nil {
			res[k] = v
		}
	}
	return res
}

func (d *Database) GetTopologyNodes() ([]TopologyNode, error) {
	rows, err := d.conn.Query(`SELECT id, label, type, ip, status, x, y, parent_id, rx_power, updated_at FROM topology_nodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []TopologyNode
	for rows.Next() {
		var n TopologyNode
		if err := rows.Scan(&n.ID, &n.Label, &n.Type, &n.IP, &n.Status, &n.X, &n.Y, &n.ParentID, &n.RxPower, &n.UpdatedAt); err == nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func (d *Database) SaveTopologyNode(n TopologyNode) error {
	_, err := d.conn.Exec(`INSERT INTO topology_nodes (id, label, type, ip, status, x, y, parent_id, rx_power, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(id) DO UPDATE SET
			label=excluded.label, type=excluded.type, ip=excluded.ip,
			status=excluded.status, x=excluded.x, y=excluded.y, parent_id=excluded.parent_id,
			rx_power=excluded.rx_power, updated_at=CURRENT_TIMESTAMP`,
		n.ID, n.Label, n.Type, n.IP, n.Status, n.X, n.Y, n.ParentID, n.RxPower)
	return err
}
