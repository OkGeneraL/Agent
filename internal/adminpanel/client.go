package adminpanel

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"superagent/internal/config"
	"superagent/internal/logging"
)

// Client represents an admin panel API client
type Client struct {
	config      *config.AdminPanelConfig
	httpClient  *http.Client
	auditLogger *logging.AuditLogger
}

// User represents a user from the admin panel
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Active   bool   `json:"active"`
}

// AuthResponse represents authentication response
type AuthResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      User      `json:"user"`
}

// ConnectionStatus represents the connection status with admin panel
type ConnectionStatus struct {
	Connected    bool      `json:"connected"`
	LastSync     time.Time `json:"last_sync"`
	URL          string    `json:"url"`
	Username     string    `json:"username"`
	TokenExpiry  time.Time `json:"token_expiry"`
	ErrorMessage string    `json:"error_message,omitempty"`
}

// NewClient creates a new admin panel client
func NewClient(cfg *config.AdminPanelConfig, auditLogger *logging.AuditLogger) *Client {
	return &Client{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.ConnectionTimeout,
		},
		auditLogger: auditLogger,
	}
}

// TestConnection tests the connection to the admin panel
func (c *Client) TestConnection(ctx context.Context) error {
	if c.config.BaseURL == "" {
		return fmt.Errorf("admin panel base URL not configured")
	}

	url := strings.TrimSuffix(c.config.BaseURL, "/") + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to connect to admin panel: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("admin panel returned status %d", resp.StatusCode)
	}

	c.auditLogger.Log("admin_panel_connection_test", map[string]interface{}{
		"url":    url,
		"status": "success",
	})

	return nil
}

// Authenticate authenticates with the admin panel
func (c *Client) Authenticate(ctx context.Context, username, password string) (*AuthResponse, error) {
	url := strings.TrimSuffix(c.config.BaseURL, "/") + c.config.APIEndpoint + "/auth/login"
	
	loginData := map[string]string{
		"username": username,
		"password": password,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal login data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("authentication failed: %s", string(body))
	}

	var authResp AuthResponse
	if err := json.Unmarshal(body, &authResp); err != nil {
		return nil, fmt.Errorf("failed to parse auth response: %w", err)
	}

	c.auditLogger.Log("admin_panel_authentication", map[string]interface{}{
		"username": username,
		"status":   "success",
	})

	return &authResp, nil
}

// GetUsers retrieves users from the admin panel
func (c *Client) GetUsers(ctx context.Context) ([]User, error) {
	url := strings.TrimSuffix(c.config.BaseURL, "/") + c.config.APIEndpoint + "/users"
	
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get users: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get users: %s", string(body))
	}

	var users []User
	if err := json.Unmarshal(body, &users); err != nil {
		return nil, fmt.Errorf("failed to parse users response: %w", err)
	}

	return users, nil
}

// SyncDeployment syncs deployment status with admin panel
func (c *Client) SyncDeployment(ctx context.Context, deployment map[string]interface{}) error {
	url := strings.TrimSuffix(c.config.BaseURL, "/") + c.config.APIEndpoint + "/deployments"
	
	jsonData, err := json.Marshal(deployment)
	if err != nil {
		return fmt.Errorf("failed to marshal deployment data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.config.Token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to sync deployment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to sync deployment: %s", string(body))
	}

	c.auditLogger.Log("admin_panel_deployment_sync", map[string]interface{}{
		"deployment_id": deployment["id"],
		"status":        "success",
	})

	return nil
}

// GetConnectionStatus returns the current connection status
func (c *Client) GetConnectionStatus() ConnectionStatus {
	status := ConnectionStatus{
		Connected: c.config.Enabled && c.config.Token != "",
		URL:       c.config.BaseURL,
		Username:  c.config.Username,
	}

	if c.config.Token != "" {
		// In a real implementation, you would decode the JWT token to get expiry
		// For now, we'll assume 24 hours from last login
		status.TokenExpiry = time.Now().Add(24 * time.Hour)
	}

	return status
}

// IsConnected returns true if connected to admin panel
func (c *Client) IsConnected() bool {
	return c.config.Enabled && c.config.Token != ""
}

// UpdateConfig updates the admin panel configuration
func (c *Client) UpdateConfig(baseURL, username, password, token string) {
	c.config.BaseURL = baseURL
	c.config.Username = username
	c.config.Password = password
	c.config.Token = token
	c.config.Enabled = baseURL != ""
}

// StartAutoSync starts automatic synchronization with admin panel
func (c *Client) StartAutoSync(ctx context.Context) {
	if !c.config.AutoSync || !c.IsConnected() {
		return
	}

	ticker := time.NewTicker(c.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Perform periodic sync operations here
			c.auditLogger.Log("admin_panel_auto_sync", map[string]interface{}{
				"status": "triggered",
			})
		}
	}
}