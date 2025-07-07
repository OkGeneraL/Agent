package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"superagent/internal/config"
	"superagent/internal/logging"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

// BackendClient handles communication with the backend API
type BackendClient struct {
	config      *config.Config
	auditLogger *logging.AuditLogger
	httpClient  *http.Client
	wsConn      *websocket.Conn
	wsConnMu    sync.RWMutex
	apiToken    string
	baseURL     string
	agentID     string
	serverID    string
	location    string
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// AgentRegistration represents agent registration data
type AgentRegistration struct {
	ID          string            `json:"id"`
	ServerID    string            `json:"server_id"`
	Location    string            `json:"location"`
	Version     string            `json:"version"`
	Capabilities []string         `json:"capabilities"`
	Resources   ResourceInfo      `json:"resources"`
	Status      string            `json:"status"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ResourceInfo represents agent resource information
type ResourceInfo struct {
	CPU      ResourceMetric `json:"cpu"`
	Memory   ResourceMetric `json:"memory"`
	Storage  ResourceMetric `json:"storage"`
	Network  ResourceMetric `json:"network"`
	Containers int          `json:"containers"`
	MaxContainers int       `json:"max_containers"`
}

// ResourceMetric represents a resource metric
type ResourceMetric struct {
	Used      float64 `json:"used"`
	Total     float64 `json:"total"`
	Available float64 `json:"available"`
	Unit      string  `json:"unit"`
}

// DeploymentCommand represents a deployment command from backend
type DeploymentCommand struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Action      string                 `json:"action"`
	Target      string                 `json:"target"`
	Spec        map[string]interface{} `json:"spec"`
	Environment map[string]string      `json:"environment"`
	Timeout     time.Duration          `json:"timeout"`
	Priority    int                    `json:"priority"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// CommandResponse represents a response to a deployment command
type CommandResponse struct {
	CommandID string                 `json:"command_id"`
	Status    string                 `json:"status"`
	Success   bool                   `json:"success"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data"`
	Error     string                 `json:"error,omitempty"`
	Duration  time.Duration          `json:"duration"`
	Timestamp time.Time              `json:"timestamp"`
}

// StatusReport represents agent status report
type StatusReport struct {
	AgentID     string                 `json:"agent_id"`
	ServerID    string                 `json:"server_id"`
	Location    string                 `json:"location"`
	Status      string                 `json:"status"`
	Health      string                 `json:"health"`
	Resources   ResourceInfo           `json:"resources"`
	Containers  []ContainerStatus      `json:"containers"`
	Deployments []DeploymentStatus     `json:"deployments"`
	Uptime      time.Duration          `json:"uptime"`
	LastSeen    time.Time              `json:"last_seen"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// ContainerStatus represents container status
type ContainerStatus struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Image    string            `json:"image"`
	Status   string            `json:"status"`
	Health   string            `json:"health"`
	Ports    []string          `json:"ports"`
	Resources ResourceInfo     `json:"resources"`
	Labels   map[string]string `json:"labels"`
	Created  time.Time         `json:"created"`
	Started  time.Time         `json:"started"`
}

// DeploymentStatus represents deployment status
type DeploymentStatus struct {
	ID       string            `json:"id"`
	Name     string            `json:"name"`
	Status   string            `json:"status"`
	Health   string            `json:"health"`
	Version  string            `json:"version"`
	Replicas int               `json:"replicas"`
	Labels   map[string]string `json:"labels"`
	Created  time.Time         `json:"created"`
	Updated  time.Time         `json:"updated"`
}

// NewBackendClient creates a new backend client
func NewBackendClient(cfg *config.Config, auditLogger *logging.AuditLogger) (*BackendClient, error) {
	// Load API token
	apiToken, err := loadAPIToken(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to load API token: %w", err)
	}

	// Create HTTP client with TLS configuration
	httpClient, err := createHTTPClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP client: %w", err)
	}

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	client := &BackendClient{
		config:      cfg,
		auditLogger: auditLogger,
		httpClient:  httpClient,
		apiToken:    apiToken,
		baseURL:     cfg.Backend.BaseURL,
		agentID:     cfg.Agent.ID,
		serverID:    cfg.Agent.ServerID,
		location:    cfg.Agent.Location,
		ctx:         ctx,
		cancel:      cancel,
	}

	// Generate IDs if not provided
	if client.agentID == "" {
		client.agentID = generateAgentID()
	}
	if client.serverID == "" {
		client.serverID = generateServerID()
	}

	return client, nil
}

// Start starts the backend client
func (bc *BackendClient) Start(ctx context.Context) error {
	logrus.Info("Starting backend client")

	// Register agent with backend
	if err := bc.RegisterAgent(ctx); err != nil {
		return fmt.Errorf("failed to register agent: %w", err)
	}

	// Start heartbeat
	bc.wg.Add(1)
	go bc.heartbeat()

	// Start WebSocket connection for real-time commands
	bc.wg.Add(1)
	go bc.handleWebSocketConnection()

	logrus.Info("Backend client started successfully")
	return nil
}

// RegisterAgent registers the agent with the backend
func (bc *BackendClient) RegisterAgent(ctx context.Context) error {
	logrus.Info("Registering agent with backend")

	registration := &AgentRegistration{
		ID:          bc.agentID,
		ServerID:    bc.serverID,
		Location:    bc.location,
		Version:     "1.0.0",
		Capabilities: []string{"docker", "git", "traefik", "monitoring"},
		Status:      "online",
		Metadata: map[string]interface{}{
			"start_time": time.Now(),
		},
	}

	// Get current resource info
	resourceInfo, err := bc.getResourceInfo()
	if err != nil {
		logrus.Warnf("Failed to get resource info: %v", err)
	} else {
		registration.Resources = resourceInfo
	}

	// Send registration request
	response, err := bc.sendRequest(ctx, "POST", "/agents/register", registration)
	if err != nil {
		bc.auditLogger.LogError("AGENT_REGISTRATION_FAILED", err, map[string]interface{}{
			"agent_id":  bc.agentID,
			"server_id": bc.serverID,
		})
		return fmt.Errorf("failed to send registration: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusCreated {
		bc.auditLogger.LogError("AGENT_REGISTRATION_REJECTED", fmt.Errorf("status: %d", response.StatusCode), map[string]interface{}{
			"agent_id":    bc.agentID,
			"server_id":   bc.serverID,
			"status_code": response.StatusCode,
		})
		return fmt.Errorf("registration rejected with status: %d", response.StatusCode)
	}

	bc.auditLogger.LogEvent("AGENT_REGISTRATION_SUCCESS", map[string]interface{}{
		"agent_id":  bc.agentID,
		"server_id": bc.serverID,
		"location":  bc.location,
	})

	logrus.Info("Agent registered successfully")
	return nil
}

// SendStatusReport sends a status report to the backend
func (bc *BackendClient) SendStatusReport(ctx context.Context, containers []ContainerStatus, deployments []DeploymentStatus) error {
	resourceInfo, err := bc.getResourceInfo()
	if err != nil {
		logrus.Warnf("Failed to get resource info: %v", err)
	}

	report := &StatusReport{
		AgentID:     bc.agentID,
		ServerID:    bc.serverID,
		Location:    bc.location,
		Status:      "online",
		Health:      "healthy",
		Resources:   resourceInfo,
		Containers:  containers,
		Deployments: deployments,
		LastSeen:    time.Now(),
		Metadata: map[string]interface{}{
			"version": "1.0.0",
		},
	}

	response, err := bc.sendRequest(ctx, "POST", "/agents/status", report)
	if err != nil {
		return fmt.Errorf("failed to send status report: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("status report rejected with status: %d", response.StatusCode)
	}

	return nil
}

// SendCommandResponse sends a command response to the backend
func (bc *BackendClient) SendCommandResponse(ctx context.Context, response *CommandResponse) error {
	httpResponse, err := bc.sendRequest(ctx, "POST", "/commands/response", response)
	if err != nil {
		return fmt.Errorf("failed to send command response: %w", err)
	}
	defer httpResponse.Body.Close()

	if httpResponse.StatusCode != http.StatusOK {
		return fmt.Errorf("command response rejected with status: %d", httpResponse.StatusCode)
	}

	bc.auditLogger.LogEvent("COMMAND_RESPONSE_SENT", map[string]interface{}{
		"command_id": response.CommandID,
		"status":     response.Status,
		"success":    response.Success,
	})

	return nil
}

// GetCommands retrieves pending commands from the backend
func (bc *BackendClient) GetCommands(ctx context.Context) ([]*DeploymentCommand, error) {
	response, err := bc.sendRequest(ctx, "GET", fmt.Sprintf("/agents/%s/commands", bc.agentID), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get commands: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get commands failed with status: %d", response.StatusCode)
	}

	var commands []*DeploymentCommand
	if err := json.NewDecoder(response.Body).Decode(&commands); err != nil {
		return nil, fmt.Errorf("failed to decode commands: %w", err)
	}

	return commands, nil
}

// RefreshToken refreshes the API token
func (bc *BackendClient) RefreshToken(ctx context.Context) error {
	response, err := bc.sendRequest(ctx, "POST", "/auth/refresh", map[string]string{
		"agent_id": bc.agentID,
	})
	if err != nil {
		return fmt.Errorf("failed to refresh token: %w", err)
	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("token refresh failed with status: %d", response.StatusCode)
	}

	var tokenResponse struct {
		Token     string    `json:"token"`
		ExpiresAt time.Time `json:"expires_at"`
	}

	if err := json.NewDecoder(response.Body).Decode(&tokenResponse); err != nil {
		return fmt.Errorf("failed to decode token response: %w", err)
	}

	bc.apiToken = tokenResponse.Token

	bc.auditLogger.LogEvent("TOKEN_REFRESHED", map[string]interface{}{
		"agent_id":   bc.agentID,
		"expires_at": tokenResponse.ExpiresAt,
	})

	return nil
}

// sendRequest sends an HTTP request to the backend
func (bc *BackendClient) sendRequest(ctx context.Context, method, path string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	req, err := http.NewRequestWithContext(ctx, method, bc.baseURL+path, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+bc.apiToken)
	req.Header.Set("User-Agent", "superagent/1.0.0")
	req.Header.Set("X-Agent-ID", bc.agentID)
	req.Header.Set("X-Server-ID", bc.serverID)
	req.Header.Set("X-Location", bc.location)

	// Add custom headers
	for key, value := range bc.config.Backend.Headers {
		req.Header.Set(key, value)
	}

	// Send request with retry logic
	return bc.sendWithRetry(req)
}

// sendWithRetry sends a request with retry logic
func (bc *BackendClient) sendWithRetry(req *http.Request) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt < bc.config.Backend.RetryAttempts; attempt++ {
		if attempt > 0 {
			time.Sleep(bc.config.Backend.RetryDelay * time.Duration(attempt))
		}

		resp, err := bc.httpClient.Do(req)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		lastErr = err
		if err == nil {
			lastErr = fmt.Errorf("server error: %d", resp.StatusCode)
		}

		logrus.Warnf("Request attempt %d failed: %v", attempt+1, lastErr)
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", bc.config.Backend.RetryAttempts, lastErr)
}

// heartbeat sends periodic heartbeat to backend
func (bc *BackendClient) heartbeat() {
	defer bc.wg.Done()

	ticker := time.NewTicker(bc.config.Agent.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bc.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if err := bc.SendStatusReport(ctx, nil, nil); err != nil {
				logrus.Errorf("Failed to send heartbeat: %v", err)
			}
			cancel()
		}
	}
}

// handleWebSocketConnection handles WebSocket connection for real-time commands
func (bc *BackendClient) handleWebSocketConnection() {
	defer bc.wg.Done()

	for {
		select {
		case <-bc.ctx.Done():
			return
		default:
			if err := bc.connectWebSocket(); err != nil {
				logrus.Errorf("WebSocket connection failed: %v", err)
				time.Sleep(30 * time.Second)
				continue
			}

			bc.handleWebSocketMessages()
		}
	}
}

// connectWebSocket establishes WebSocket connection
func (bc *BackendClient) connectWebSocket() error {
	dialer := websocket.Dialer{
		HandshakeTimeout: 30 * time.Second,
		TLSClientConfig:  bc.httpClient.Transport.(*http.Transport).TLSClientConfig,
	}

	headers := http.Header{}
	headers.Set("Authorization", "Bearer "+bc.apiToken)
	headers.Set("X-Agent-ID", bc.agentID)
	headers.Set("X-Server-ID", bc.serverID)

	wsURL := strings.Replace(bc.baseURL, "http://", "ws://", 1)
	wsURL = strings.Replace(wsURL, "https://", "wss://", 1)
	wsURL += "/agents/ws"

	conn, _, err := dialer.Dial(wsURL, headers)
	if err != nil {
		return fmt.Errorf("failed to dial WebSocket: %w", err)
	}

	bc.wsConnMu.Lock()
	bc.wsConn = conn
	bc.wsConnMu.Unlock()

	bc.auditLogger.LogEvent("WEBSOCKET_CONNECTED", map[string]interface{}{
		"agent_id": bc.agentID,
		"url":      wsURL,
	})

	return nil
}

// handleWebSocketMessages handles incoming WebSocket messages
func (bc *BackendClient) handleWebSocketMessages() {
	bc.wsConnMu.RLock()
	conn := bc.wsConn
	bc.wsConnMu.RUnlock()

	if conn == nil {
		return
	}

	defer func() {
		bc.wsConnMu.Lock()
		if bc.wsConn == conn {
			bc.wsConn.Close()
			bc.wsConn = nil
		}
		bc.wsConnMu.Unlock()
	}()

	for {
		var command DeploymentCommand
		if err := conn.ReadJSON(&command); err != nil {
			logrus.Errorf("Failed to read WebSocket message: %v", err)
			break
		}

		bc.auditLogger.LogEvent("COMMAND_RECEIVED", map[string]interface{}{
			"command_id": command.ID,
			"type":       command.Type,
			"action":     command.Action,
		})

		// TODO: Handle command via deployment manager
		// For now, just acknowledge receipt
		response := &CommandResponse{
			CommandID: command.ID,
			Status:    "received",
			Success:   true,
			Message:   "Command received and queued for processing",
			Timestamp: time.Now(),
		}

		if err := bc.SendCommandResponse(context.Background(), response); err != nil {
			logrus.Errorf("Failed to send command acknowledgment: %v", err)
		}
	}
}

// getResourceInfo gets current resource information
func (bc *BackendClient) getResourceInfo() (ResourceInfo, error) {
	// TODO: Implement actual resource monitoring
	// For now, return mock data
	return ResourceInfo{
		CPU: ResourceMetric{
			Used:      50.0,
			Total:     100.0,
			Available: 50.0,
			Unit:      "percent",
		},
		Memory: ResourceMetric{
			Used:      4096,
			Total:     8192,
			Available: 4096,
			Unit:      "MB",
		},
		Storage: ResourceMetric{
			Used:      100,
			Total:     500,
			Available: 400,
			Unit:      "GB",
		},
		Network: ResourceMetric{
			Used:      10,
			Total:     1000,
			Available: 990,
			Unit:      "Mbps",
		},
		Containers:    0,
		MaxContainers: bc.config.Resources.MaxContainers,
	}, nil
}

// Close closes the backend client
func (bc *BackendClient) Close() error {
	bc.cancel()

	// Close WebSocket connection
	bc.wsConnMu.Lock()
	if bc.wsConn != nil {
		bc.wsConn.Close()
		bc.wsConn = nil
	}
	bc.wsConnMu.Unlock()

	bc.wg.Wait()

	bc.auditLogger.LogEvent("BACKEND_CLIENT_CLOSED", map[string]interface{}{
		"agent_id": bc.agentID,
	})

	return nil
}

// loadAPIToken loads the API token from config or file
func loadAPIToken(cfg *config.Config) (string, error) {
	if cfg.Backend.APIToken != "" {
		return cfg.Backend.APIToken, nil
	}

	if cfg.Backend.TokenFile != "" {
		tokenData, err := os.ReadFile(cfg.Backend.TokenFile)
		if err != nil {
			return "", fmt.Errorf("failed to read token file: %w", err)
		}
		return strings.TrimSpace(string(tokenData)), nil
	}

	return "", fmt.Errorf("no API token configured")
}

// createHTTPClient creates an HTTP client with TLS configuration
func createHTTPClient(cfg *config.Config) (*http.Client, error) {
	transport := &http.Transport{
		MaxIdleConns:       10,
		IdleConnTimeout:    30 * time.Second,
		DisableCompression: false,
	}

	// Configure TLS
	tlsConfig := &tls.Config{
		InsecureSkipVerify: cfg.Backend.InsecureSkipTLS,
	}

	// Load CA certificate
	if cfg.Backend.CACertFile != "" {
		caCert, err := os.ReadFile(cfg.Backend.CACertFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, fmt.Errorf("failed to parse CA certificate")
		}

		tlsConfig.RootCAs = caCertPool
	}

	// Configure client certificates for mTLS
	if cfg.Backend.ClientCertFile != "" && cfg.Backend.ClientKeyFile != "" {
		clientCert, err := tls.LoadX509KeyPair(cfg.Backend.ClientCertFile, cfg.Backend.ClientKeyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load client certificate: %w", err)
		}

		tlsConfig.Certificates = []tls.Certificate{clientCert}
	}

	transport.TLSClientConfig = tlsConfig

	return &http.Client{
		Transport: transport,
		Timeout:   cfg.Backend.Timeout,
	}, nil
}

// generateAgentID generates a unique agent ID
func generateAgentID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("agent-%s-%d", hostname, time.Now().Unix())
}

// generateServerID generates a unique server ID
func generateServerID() string {
	hostname, _ := os.Hostname()
	return fmt.Sprintf("server-%s", hostname)
}