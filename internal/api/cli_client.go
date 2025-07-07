package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// CLIClient provides a client interface for the CLI to communicate with the API server
type CLIClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewCLIClient creates a new CLI client
func NewCLIClient(apiPort int) *CLIClient {
	return &CLIClient{
		baseURL: fmt.Sprintf("http://localhost:%d/api/v1", apiPort),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetStatus retrieves the agent status
func (c *CLIClient) GetStatus() (*AgentStatus, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/status")
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status request failed with status: %d", resp.StatusCode)
	}

	var status AgentStatus
	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to decode status response: %w", err)
	}

	return &status, nil
}

// GetVersion retrieves the agent version
func (c *CLIClient) GetVersion() (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/version")
	if err != nil {
		return nil, fmt.Errorf("failed to get version: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("version request failed with status: %d", resp.StatusCode)
	}

	var version map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&version); err != nil {
		return nil, fmt.Errorf("failed to decode version response: %w", err)
	}

	return version, nil
}

// GetHealth retrieves the agent health
func (c *CLIClient) GetHealth() (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return nil, fmt.Errorf("failed to get health: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("health request failed with status: %d", resp.StatusCode)
	}

	var health map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		return nil, fmt.Errorf("failed to decode health response: %w", err)
	}

	return health, nil
}

// CreateDeployment creates a new deployment
func (c *CLIClient) CreateDeployment(request interface{}) (*DeploymentResponse, error) {
	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal deployment request: %w", err)
	}

	resp, err := c.httpClient.Post(c.baseURL+"/deployments", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create deployment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deployment creation failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	var deployment DeploymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&deployment); err != nil {
		return nil, fmt.Errorf("failed to decode deployment response: %w", err)
	}

	return &deployment, nil
}

// ListDeployments lists all deployments
func (c *CLIClient) ListDeployments() ([]DeploymentResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/deployments")
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list deployments failed with status: %d", resp.StatusCode)
	}

	var deployments []DeploymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&deployments); err != nil {
		return nil, fmt.Errorf("failed to decode deployments response: %w", err)
	}

	return deployments, nil
}

// GetDeployment retrieves a specific deployment
func (c *CLIClient) GetDeployment(deploymentID string) (*DeploymentResponse, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/deployments/" + deploymentID)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get deployment failed with status: %d", resp.StatusCode)
	}

	var deployment DeploymentResponse
	if err := json.NewDecoder(resp.Body).Decode(&deployment); err != nil {
		return nil, fmt.Errorf("failed to decode deployment response: %w", err)
	}

	return &deployment, nil
}

// GetDeploymentLogs retrieves logs for a deployment
func (c *CLIClient) GetDeploymentLogs(deploymentID string, tail int) (*LogsResponse, error) {
	url := fmt.Sprintf("%s/deployments/%s/logs?tail=%d", c.baseURL, deploymentID, tail)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment logs: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get deployment logs failed with status: %d", resp.StatusCode)
	}

	var logs LogsResponse
	if err := json.NewDecoder(resp.Body).Decode(&logs); err != nil {
		return nil, fmt.Errorf("failed to decode logs response: %w", err)
	}

	return &logs, nil
}

// StopDeployment stops a deployment
func (c *CLIClient) StopDeployment(deploymentID string) error {
	req, err := http.NewRequest("POST", c.baseURL+"/deployments/"+deploymentID+"/stop", nil)
	if err != nil {
		return fmt.Errorf("failed to create stop request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to stop deployment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("stop deployment failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// DeleteDeployment deletes a deployment
func (c *CLIClient) DeleteDeployment(deploymentID string) error {
	req, err := http.NewRequest("DELETE", c.baseURL+"/deployments/"+deploymentID, nil)
	if err != nil {
		return fmt.Errorf("failed to create delete request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete deployment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete deployment failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RollbackDeployment rolls back a deployment
func (c *CLIClient) RollbackDeployment(deploymentID string) error {
	req, err := http.NewRequest("POST", c.baseURL+"/deployments/"+deploymentID+"/rollback", nil)
	if err != nil {
		return fmt.Errorf("failed to create rollback request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to rollback deployment: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("rollback deployment failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetMetrics retrieves agent metrics
func (c *CLIClient) GetMetrics() (map[string]interface{}, error) {
	resp, err := c.httpClient.Get(c.baseURL + "/metrics")
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("metrics request failed with status: %d", resp.StatusCode)
	}

	var metrics map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&metrics); err != nil {
		return nil, fmt.Errorf("failed to decode metrics response: %w", err)
	}

	return metrics, nil
}

// IsAgentRunning checks if the agent is running by trying to connect to the API
func (c *CLIClient) IsAgentRunning() bool {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}