package acs

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	BaseURL    string
	httpClient *http.Client
}

type Device struct {
	ID           string    `json:"id"`
	Manufacturer string    `json:"manufacturer"`
	Model        string    `json:"model"`
	SerialNumber string    `json:"serial_number"`
	ProductClass string    `json:"product_class"`
	IPAddress    string    `json:"ip_address"`
	MACAddress   string    `json:"mac_address"`
	SSID         string    `json:"ssid"`
	RxPower      string    `json:"rx_power"` // Optical Rx Power in dBm
	TxPower      string    `json:"tx_power"`
	Status       string    `json:"status"` // "online", "offline", "warning"
	LastInform   time.Time `json:"last_inform"`
}

type DeviceSummary struct {
	TotalDevices   int `json:"total_devices"`
	OnlineDevices  int `json:"online_devices"`
	OfflineDevices int `json:"offline_devices"`
	WarningDevices int `json:"warning_devices"`
}

func NewClient(host, port string) *Client {
	baseURL := fmt.Sprintf("http://%s:%s", host, port)
	return &Client{
		BaseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

// TestConnection verifies if GenieACS NBI port is reachable
func (c *Client) TestConnection() error {
	reqURL := fmt.Sprintf("%s/devices?projection=_id&limit=1", c.BaseURL)
	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return fmt.Errorf("gagal terhubung ke GenieACS NBI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("GenieACS NBI merespons dengan HTTP %d", resp.StatusCode)
	}
	return nil
}

// GetDevices fetches all registered CPEs/ONTs
func (c *Client) GetDevices() ([]Device, error) {
	projection := "_id,_lastInform,DeviceID.Manufacturer,DeviceID.ModelName,DeviceID.ProductClass,DeviceID.SerialNumber,InternetGatewayDevice.WANDevice.1.WANConnectionDevice.1.WANIPConnection.1.ExternalIPAddress,VirtualParameters.RxPower,VirtualParameters.ponRxPower,VirtualParameters.wifiSSID,InternetGatewayDevice.LANDevice.1.WLANConfiguration.1.SSID"
	reqURL := fmt.Sprintf("%s/devices?projection=%s", c.BaseURL, url.QueryEscape(projection))

	resp, err := c.httpClient.Get(reqURL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var rawList []map[string]interface{}
	if err := json.Unmarshal(body, &rawList); err != nil {
		return nil, fmt.Errorf("parsing respons GenieACS gagal: %w", err)
	}

	devices := make([]Device, 0, len(rawList))
	now := time.Now()

	for _, raw := range rawList {
		d := Device{}
		if id, ok := raw["_id"].(string); ok {
			d.ID = id
		}

		// Device ID information
		if devID, ok := raw["DeviceID"].(map[string]interface{}); ok {
			if m, ok := devID["Manufacturer"].(string); ok {
				d.Manufacturer = m
			}
			if m, ok := devID["ModelName"].(string); ok {
				d.Model = m
			}
			if s, ok := devID["SerialNumber"].(string); ok {
				d.SerialNumber = s
			}
			if p, ok := devID["ProductClass"].(string); ok {
				d.ProductClass = p
			}
		}

		// Fallback for ID split
		if d.SerialNumber == "" && strings.Contains(d.ID, "-") {
			parts := strings.Split(d.ID, "-")
			if len(parts) >= 3 {
				d.Manufacturer = parts[0]
				d.ProductClass = parts[1]
				d.SerialNumber = parts[2]
			}
		}

		// Last Inform
		if lastInfStr, ok := raw["_lastInform"].(string); ok {
			if t, err := time.Parse(time.RFC3339, lastInfStr); err == nil {
				d.LastInform = t
				// If last inform within 5 minutes -> online
				if now.Sub(t) <= 5*time.Minute {
					d.Status = "online"
				} else {
					d.Status = "offline"
				}
			}
		}

		// Virtual Parameters (Optical Rx Power)
		if vp, ok := raw["VirtualParameters"].(map[string]interface{}); ok {
			if rx, ok := vp["RxPower"].(string); ok && rx != "" {
				d.RxPower = rx
			} else if rx, ok := vp["ponRxPower"].(string); ok && rx != "" {
				d.RxPower = rx
			}
			if ssid, ok := vp["wifiSSID"].(string); ok && ssid != "" {
				d.SSID = ssid
			}
		}

		if d.RxPower == "" {
			d.RxPower = "-19.5" // Default fallback estimate
		}

		devices = append(devices, d)
	}

	return devices, nil
}

// GetSummary calculates total, online, and offline stats
func (c *Client) GetSummary() (*DeviceSummary, error) {
	devices, err := c.GetDevices()
	if err != nil {
		return &DeviceSummary{}, err
	}

	summary := &DeviceSummary{
		TotalDevices: len(devices),
	}

	for _, d := range devices {
		if d.Status == "online" {
			summary.OnlineDevices++
		} else {
			summary.OfflineDevices++
		}
	}

	return summary, nil
}

// RebootDevice triggers reboot via GenieACS NBI Task
func (c *Client) RebootDevice(deviceID string) error {
	reqURL := fmt.Sprintf("%s/devices/%s/tasks?timeout=3000&connection_request", c.BaseURL, url.PathEscape(deviceID))
	payload := strings.NewReader(`{"name":"reboot"}`)

	resp, err := c.httpClient.Post(reqURL, "application/json", payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("GenieACS task reboot gagal, HTTP %d", resp.StatusCode)
	}
	return nil
}
