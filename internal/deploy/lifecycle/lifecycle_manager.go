package lifecycle

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	"superagent/internal/logging"

	"github.com/sirupsen/logrus"
)

// LifecycleManager handles container lifecycle operations
type LifecycleManager struct {
	auditLogger *logging.AuditLogger
	mu          sync.RWMutex
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	Type                string            `json:"type"`                // "http", "tcp", "cmd"
	Path                string            `json:"path,omitempty"`
	Port                int               `json:"port,omitempty"`
	Command             []string          `json:"command,omitempty"`
	InitialDelaySeconds int               `json:"initial_delay_seconds"`
	PeriodSeconds       int               `json:"period_seconds"`
	TimeoutSeconds      int               `json:"timeout_seconds"`
	FailureThreshold    int               `json:"failure_threshold"`
	SuccessThreshold    int               `json:"success_threshold"`
	Headers             map[string]string `json:"headers,omitempty"`
}

// HealthCheckResult represents the result of a health check
type HealthCheckResult struct {
	Success   bool          `json:"success"`
	Message   string        `json:"message"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewLifecycleManager creates a new lifecycle manager
func NewLifecycleManager(auditLogger *logging.AuditLogger) (*LifecycleManager, error) {
	lm := &LifecycleManager{
		auditLogger: auditLogger,
	}

	auditLogger.LogEvent("LIFECYCLE_MANAGER_INITIALIZED", map[string]interface{}{})

	return lm, nil
}

// PerformHealthCheck performs a health check on a container
func (lm *LifecycleManager) PerformHealthCheck(ctx context.Context, containerID string, config HealthCheckConfig) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	logrus.Debugf("Performing health check for container %s", containerID)

	// Apply initial delay
	if config.InitialDelaySeconds > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(config.InitialDelaySeconds) * time.Second):
		}
	}

	// Perform health check based on type
	var result *HealthCheckResult
	var err error

	switch config.Type {
	case "http":
		result, err = lm.performHTTPHealthCheck(ctx, config)
	case "tcp":
		result, err = lm.performTCPHealthCheck(ctx, config)
	case "cmd":
		result, err = lm.performCommandHealthCheck(ctx, containerID, config)
	default:
		return fmt.Errorf("unsupported health check type: %s", config.Type)
	}

	if err != nil {
		lm.auditLogger.LogEvent("HEALTH_CHECK_ERROR", map[string]interface{}{
			"container_id": containerID,
			"type":         config.Type,
			"error":        err.Error(),
		})
		return fmt.Errorf("health check failed: %w", err)
	}

	// Log health check result
	lm.auditLogger.LogEvent("HEALTH_CHECK_COMPLETED", map[string]interface{}{
		"container_id": containerID,
		"type":         config.Type,
		"success":      result.Success,
		"duration":     result.Duration.Milliseconds(),
		"message":      result.Message,
	})

	if !result.Success {
		return fmt.Errorf("health check failed: %s", result.Message)
	}

	logrus.Debugf("Health check passed for container %s (%s)", containerID, result.Message)
	return nil
}

// PerformContinuousHealthCheck performs continuous health checks with retry logic
func (lm *LifecycleManager) PerformContinuousHealthCheck(ctx context.Context, containerID string, config HealthCheckConfig) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	successCount := 0
	failureCount := 0

	ticker := time.NewTicker(time.Duration(config.PeriodSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			// Create timeout context for individual health check
			checkCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
			
			var result *HealthCheckResult
			var err error

			switch config.Type {
			case "http":
				result, err = lm.performHTTPHealthCheck(checkCtx, config)
			case "tcp":
				result, err = lm.performTCPHealthCheck(checkCtx, config)
			case "cmd":
				result, err = lm.performCommandHealthCheck(checkCtx, containerID, config)
			default:
				cancel()
				return fmt.Errorf("unsupported health check type: %s", config.Type)
			}

			cancel()

			if err != nil || !result.Success {
				failureCount++
				successCount = 0
				
				logrus.Warnf("Health check failed for container %s: %v", containerID, err)
				
				if failureCount >= config.FailureThreshold {
					lm.auditLogger.LogEvent("HEALTH_CHECK_THRESHOLD_EXCEEDED", map[string]interface{}{
						"container_id":    containerID,
						"failure_count":   failureCount,
						"threshold":       config.FailureThreshold,
					})
					return fmt.Errorf("health check failed %d times, exceeding threshold", failureCount)
				}
			} else {
				successCount++
				failureCount = 0
				
				if successCount >= config.SuccessThreshold {
					logrus.Debugf("Health check successful for container %s", containerID)
					return nil
				}
			}
		}
	}
}

// performHTTPHealthCheck performs an HTTP health check
func (lm *LifecycleManager) performHTTPHealthCheck(ctx context.Context, config HealthCheckConfig) (*HealthCheckResult, error) {
	start := time.Now()
	
	// Build URL
	url := fmt.Sprintf("http://localhost:%d%s", config.Port, config.Path)
	if config.Path == "" {
		url = fmt.Sprintf("http://localhost:%d/health", config.Port)
	}

	// Create HTTP request
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Add custom headers
	for key, value := range config.Headers {
		req.Header.Set(key, value)
	}

	// Perform request
	client := &http.Client{
		Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
	}

	resp, err := client.Do(req)
	duration := time.Since(start)

	result := &HealthCheckResult{
		Duration:  duration,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"url": url,
		},
	}

	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("HTTP request failed: %v", err)
		return result, nil
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Success = true
		result.Message = fmt.Sprintf("HTTP %d OK", resp.StatusCode)
	} else {
		result.Success = false
		result.Message = fmt.Sprintf("HTTP %d %s", resp.StatusCode, resp.Status)
	}

	result.Metadata["status_code"] = resp.StatusCode
	return result, nil
}

// performTCPHealthCheck performs a TCP health check
func (lm *LifecycleManager) performTCPHealthCheck(ctx context.Context, config HealthCheckConfig) (*HealthCheckResult, error) {
	start := time.Now()
	
	address := fmt.Sprintf("localhost:%d", config.Port)
	
	// Create dialer with timeout
	dialer := &net.Dialer{
		Timeout: time.Duration(config.TimeoutSeconds) * time.Second,
	}

	conn, err := dialer.DialContext(ctx, "tcp", address)
	duration := time.Since(start)

	result := &HealthCheckResult{
		Duration:  duration,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"address": address,
		},
	}

	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("TCP connection failed: %v", err)
		return result, nil
	}

	conn.Close()
	result.Success = true
	result.Message = "TCP connection successful"
	return result, nil
}

// performCommandHealthCheck performs a command-based health check
func (lm *LifecycleManager) performCommandHealthCheck(ctx context.Context, containerID string, config HealthCheckConfig) (*HealthCheckResult, error) {
	start := time.Now()
	
	if len(config.Command) == 0 {
		return nil, fmt.Errorf("no command specified for health check")
	}

	// Execute command in container
	cmd := exec.CommandContext(ctx, "docker", "exec", containerID)
	cmd.Args = append(cmd.Args, config.Command...)

	output, err := cmd.CombinedOutput()
	duration := time.Since(start)

	result := &HealthCheckResult{
		Duration:  duration,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"command": strings.Join(config.Command, " "),
			"output":  string(output),
		},
	}

	if err != nil {
		result.Success = false
		result.Message = fmt.Sprintf("Command failed: %v", err)
		return result, nil
	}

	// Check exit code (if command exits with 0, it's considered successful)
	if cmd.ProcessState.ExitCode() == 0 {
		result.Success = true
		result.Message = "Command executed successfully"
	} else {
		result.Success = false
		result.Message = fmt.Sprintf("Command exited with code %d", cmd.ProcessState.ExitCode())
	}

	result.Metadata["exit_code"] = cmd.ProcessState.ExitCode()
	return result, nil
}

// WaitForContainerReady waits for a container to be ready based on health checks
func (lm *LifecycleManager) WaitForContainerReady(ctx context.Context, containerID string, config HealthCheckConfig) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	logrus.Infof("Waiting for container %s to be ready", containerID)

	// Apply initial delay
	if config.InitialDelaySeconds > 0 {
		logrus.Debugf("Waiting %d seconds before starting health checks", config.InitialDelaySeconds)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Duration(config.InitialDelaySeconds) * time.Second):
		}
	}

	successCount := 0
	maxAttempts := config.FailureThreshold * 3 // Allow more attempts for readiness

	for attempt := 0; attempt < maxAttempts; attempt++ {
		// Create timeout context for individual health check
		checkCtx, cancel := context.WithTimeout(ctx, time.Duration(config.TimeoutSeconds)*time.Second)
		
		var result *HealthCheckResult
		var err error

		switch config.Type {
		case "http":
			result, err = lm.performHTTPHealthCheck(checkCtx, config)
		case "tcp":
			result, err = lm.performTCPHealthCheck(checkCtx, config)
		case "cmd":
			result, err = lm.performCommandHealthCheck(checkCtx, containerID, config)
		default:
			cancel()
			return fmt.Errorf("unsupported health check type: %s", config.Type)
		}

		cancel()

		if err == nil && result.Success {
			successCount++
			logrus.Debugf("Health check %d/%d passed for container %s", successCount, config.SuccessThreshold, containerID)
			
			if successCount >= config.SuccessThreshold {
				lm.auditLogger.LogEvent("CONTAINER_READY", map[string]interface{}{
					"container_id": containerID,
					"attempts":     attempt + 1,
					"duration":     time.Since(time.Now().Add(-time.Duration(attempt*config.PeriodSeconds)*time.Second)),
				})
				logrus.Infof("Container %s is ready after %d attempts", containerID, attempt+1)
				return nil
			}
		} else {
			successCount = 0
			logrus.Debugf("Health check failed for container %s (attempt %d/%d): %v", containerID, attempt+1, maxAttempts, err)
		}

		// Wait before next attempt
		if attempt < maxAttempts-1 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(config.PeriodSeconds) * time.Second):
			}
		}
	}

	lm.auditLogger.LogEvent("CONTAINER_READINESS_TIMEOUT", map[string]interface{}{
		"container_id": containerID,
		"attempts":     maxAttempts,
	})

	return fmt.Errorf("container %s failed to become ready after %d attempts", containerID, maxAttempts)
}

// GracefulShutdown performs a graceful shutdown of a container
func (lm *LifecycleManager) GracefulShutdown(ctx context.Context, containerID string, shutdownTimeout time.Duration) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	logrus.Infof("Performing graceful shutdown of container %s", containerID)

	// Send SIGTERM to container
	cmd := exec.CommandContext(ctx, "docker", "kill", "--signal=SIGTERM", containerID)
	if err := cmd.Run(); err != nil {
		lm.auditLogger.LogEvent("GRACEFUL_SHUTDOWN_SIGNAL_FAILED", map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to send SIGTERM: %w", err)
	}

	// Wait for graceful shutdown or timeout
	shutdownCtx, cancel := context.WithTimeout(ctx, shutdownTimeout)
	defer cancel()

	// Check if container has stopped
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-shutdownCtx.Done():
			// Timeout reached, force kill
			logrus.Warnf("Graceful shutdown timeout for container %s, forcing termination", containerID)
			
			killCmd := exec.CommandContext(ctx, "docker", "kill", "--signal=SIGKILL", containerID)
			if err := killCmd.Run(); err != nil {
				lm.auditLogger.LogEvent("FORCE_KILL_FAILED", map[string]interface{}{
					"container_id": containerID,
					"error":        err.Error(),
				})
				return fmt.Errorf("failed to force kill container: %w", err)
			}
			
			lm.auditLogger.LogEvent("CONTAINER_FORCE_KILLED", map[string]interface{}{
				"container_id": containerID,
			})
			return nil

		case <-ticker.C:
			// Check if container is still running
			statusCmd := exec.CommandContext(ctx, "docker", "inspect", "--format={{.State.Running}}", containerID)
			output, err := statusCmd.Output()
			if err != nil {
				// Container might be removed, consider it stopped
				lm.auditLogger.LogEvent("GRACEFUL_SHUTDOWN_COMPLETED", map[string]interface{}{
					"container_id": containerID,
				})
				logrus.Infof("Container %s gracefully shut down", containerID)
				return nil
			}

			if strings.TrimSpace(string(output)) == "false" {
				lm.auditLogger.LogEvent("GRACEFUL_SHUTDOWN_COMPLETED", map[string]interface{}{
					"container_id": containerID,
				})
				logrus.Infof("Container %s gracefully shut down", containerID)
				return nil
			}
		}
	}
}

// RestartContainer performs a controlled restart of a container
func (lm *LifecycleManager) RestartContainer(ctx context.Context, containerID string, healthCheck HealthCheckConfig) error {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	logrus.Infof("Restarting container %s", containerID)

	// Restart container
	cmd := exec.CommandContext(ctx, "docker", "restart", containerID)
	if err := cmd.Run(); err != nil {
		lm.auditLogger.LogEvent("CONTAINER_RESTART_FAILED", map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to restart container: %w", err)
	}

	// Wait for container to be ready
	if err := lm.WaitForContainerReady(ctx, containerID, healthCheck); err != nil {
		lm.auditLogger.LogEvent("CONTAINER_RESTART_READINESS_FAILED", map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
		})
		return fmt.Errorf("container restart succeeded but readiness check failed: %w", err)
	}

	lm.auditLogger.LogEvent("CONTAINER_RESTART_COMPLETED", map[string]interface{}{
		"container_id": containerID,
	})

	logrus.Infof("Container %s restarted successfully", containerID)
	return nil
}

// GetContainerHealth returns the current health status of a container
func (lm *LifecycleManager) GetContainerHealth(ctx context.Context, containerID string, config HealthCheckConfig) (*HealthCheckResult, error) {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	switch config.Type {
	case "http":
		return lm.performHTTPHealthCheck(ctx, config)
	case "tcp":
		return lm.performTCPHealthCheck(ctx, config)
	case "cmd":
		return lm.performCommandHealthCheck(ctx, containerID, config)
	default:
		return nil, fmt.Errorf("unsupported health check type: %s", config.Type)
	}
}