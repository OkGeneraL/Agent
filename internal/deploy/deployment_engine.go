package deploy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"superagent/internal/deploy/git"
	"superagent/internal/deploy/docker"
	"superagent/internal/deploy/lifecycle"
	"superagent/internal/deploy/resources"
	"superagent/internal/storage"
	"superagent/internal/logging"
	"superagent/internal/monitoring"

	"github.com/sirupsen/logrus"
)

// DeploymentEngine orchestrates the complete deployment process
type DeploymentEngine struct {
	gitManager        *git.GitManager
	dockerManager     *docker.DockerManager
	lifecycleManager  *lifecycle.LifecycleManager
	resourceManager   *resources.ResourceManager
	store             *storage.SecureStore
	auditLogger       *logging.AuditLogger
	monitor           *monitoring.Monitor
	deployments       map[string]*Deployment
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
}

// Deployment represents a complete deployment with all its components
type Deployment struct {
	ID                string                 `json:"id"`
	AppID             string                 `json:"app_id"`
	Version           string                 `json:"version"`
	Status            DeploymentStatus       `json:"status"`
	Source            DeploymentSource       `json:"source"`
	Config            DeploymentConfig       `json:"config"`
	ResourceLimits    resources.ResourceLimits `json:"resource_limits"`
	HealthCheck       HealthCheckConfig      `json:"health_check"`
	Environment       map[string]string      `json:"environment"`
	CreatedAt         time.Time             `json:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at"`
	DeployedAt        *time.Time            `json:"deployed_at,omitempty"`
	LastHealthCheck   *time.Time            `json:"last_health_check,omitempty"`
	ContainerID       string                `json:"container_id,omitempty"`
	ContainerName     string                `json:"container_name,omitempty"`
	Ports             []PortMapping         `json:"ports"`
	Networks          []string              `json:"networks"`
	Volumes           []VolumeMapping       `json:"volumes"`
	Labels            map[string]string     `json:"labels"`
	BuildLogs         []LogEntry            `json:"build_logs"`
	DeploymentLogs    []LogEntry            `json:"deployment_logs"`
	Metrics           DeploymentMetrics     `json:"metrics"`
	Rollback          *RollbackInfo         `json:"rollback,omitempty"`
}

// DeploymentStatus represents the current state of a deployment
type DeploymentStatus string

const (
	StatusPending      DeploymentStatus = "pending"
	StatusBuilding     DeploymentStatus = "building"
	StatusDeploying    DeploymentStatus = "deploying"
	StatusRunning      DeploymentStatus = "running"
	StatusStopping     DeploymentStatus = "stopping"
	StatusStopped      DeploymentStatus = "stopped"
	StatusFailed       DeploymentStatus = "failed"
	StatusRollingBack  DeploymentStatus = "rolling_back"
	StatusHealthCheck  DeploymentStatus = "health_check"
	StatusUpdating     DeploymentStatus = "updating"
)

// DeploymentSource specifies where the deployment comes from
type DeploymentSource struct {
	Type       string            `json:"type"`       // "git" or "docker"
	Repository string            `json:"repository"` // Git repo URL or Docker image
	Branch     string            `json:"branch,omitempty"`
	Commit     string            `json:"commit,omitempty"`
	Tag        string            `json:"tag,omitempty"`
	BuildPath  string            `json:"build_path,omitempty"`
	Dockerfile string            `json:"dockerfile,omitempty"`
	Auth       map[string]string `json:"auth,omitempty"`
}

// DeploymentConfig holds deployment configuration
type DeploymentConfig struct {
	Replicas        int               `json:"replicas"`
	Strategy        string            `json:"strategy"`         // "rolling", "blue-green", "recreate"
	MaxUnavailable  int               `json:"max_unavailable"`
	MaxSurge        int               `json:"max_surge"`
	ProgressTimeout time.Duration     `json:"progress_timeout"`
	RestartPolicy   string            `json:"restart_policy"`
	Privileged      bool              `json:"privileged"`
	ReadOnlyRootFS  bool              `json:"read_only_root_fs"`
	User            string            `json:"user"`
	WorkingDir      string            `json:"working_dir"`
	Command         []string          `json:"command,omitempty"`
	Args            []string          `json:"args,omitempty"`
	Security        SecurityConfig    `json:"security"`
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RunAsNonRoot     bool     `json:"run_as_non_root"`
	RunAsUser        int      `json:"run_as_user"`
	RunAsGroup       int      `json:"run_as_group"`
	ReadOnlyRootFS   bool     `json:"read_only_root_fs"`
	AllowPrivileged  bool     `json:"allow_privileged"`
	Capabilities     []string `json:"capabilities"`
	SeccompProfile   string   `json:"seccomp_profile"`
	AppArmorProfile  string   `json:"apparmor_profile"`
	SELinuxOptions   map[string]string `json:"selinux_options"`
}

// HealthCheckConfig defines health check parameters
type HealthCheckConfig struct {
	Enabled             bool          `json:"enabled"`
	Type                string        `json:"type"`                // "http", "tcp", "cmd"
	Path                string        `json:"path,omitempty"`
	Port                int           `json:"port,omitempty"`
	Command             []string      `json:"command,omitempty"`
	InitialDelaySeconds int           `json:"initial_delay_seconds"`
	PeriodSeconds       int           `json:"period_seconds"`
	TimeoutSeconds      int           `json:"timeout_seconds"`
	FailureThreshold    int           `json:"failure_threshold"`
	SuccessThreshold    int           `json:"success_threshold"`
	Headers             map[string]string `json:"headers,omitempty"`
}

// PortMapping defines port configuration
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port,omitempty"`
	Protocol      string `json:"protocol"` // "tcp", "udp"
	HostIP        string `json:"host_ip,omitempty"`
}

// VolumeMapping defines volume configuration
type VolumeMapping struct {
	Source      string `json:"source"`
	Target      string `json:"target"`
	Type        string `json:"type"`        // "bind", "volume", "tmpfs"
	ReadOnly    bool   `json:"read_only"`
	Consistency string `json:"consistency,omitempty"`
	Options     []string `json:"options,omitempty"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// DeploymentMetrics holds metrics for a deployment
type DeploymentMetrics struct {
	CPUUsage         float64   `json:"cpu_usage"`
	MemoryUsage      int64     `json:"memory_usage"`
	MemoryLimit      int64     `json:"memory_limit"`
	NetworkRx        int64     `json:"network_rx"`
	NetworkTx        int64     `json:"network_tx"`
	DiskUsage        int64     `json:"disk_usage"`
	RestartCount     int       `json:"restart_count"`
	HealthCheckCount int       `json:"health_check_count"`
	LastUpdated      time.Time `json:"last_updated"`
}

// RollbackInfo holds rollback information
type RollbackInfo struct {
	PreviousVersion string    `json:"previous_version"`
	Reason          string    `json:"reason"`
	Timestamp       time.Time `json:"timestamp"`
	Status          string    `json:"status"`
}

// DeploymentRequest represents a deployment request
type DeploymentRequest struct {
	AppID          string                   `json:"app_id"`
	Version        string                   `json:"version"`
	Source         DeploymentSource         `json:"source"`
	Config         DeploymentConfig         `json:"config"`
	ResourceLimits resources.ResourceLimits `json:"resource_limits"`
	HealthCheck    HealthCheckConfig        `json:"health_check"`
	Environment    map[string]string        `json:"environment"`
	Ports          []PortMapping            `json:"ports"`
	Networks       []string                 `json:"networks"`
	Volumes        []VolumeMapping          `json:"volumes"`
	Labels         map[string]string        `json:"labels"`
}

// NewDeploymentEngine creates a new deployment engine
func NewDeploymentEngine(
	store *storage.SecureStore,
	auditLogger *logging.AuditLogger,
	monitor *monitoring.Monitor,
) (*DeploymentEngine, error) {
	ctx, cancel := context.WithCancel(context.Background())

	gitManager, err := git.NewGitManager(auditLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create git manager: %w", err)
	}

	dockerManager, err := docker.NewDockerManager(auditLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create docker manager: %w", err)
	}

	lifecycleManager, err := lifecycle.NewLifecycleManager(auditLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create lifecycle manager: %w", err)
	}

	resourceManager, err := resources.NewResourceManager(auditLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource manager: %w", err)
	}

	engine := &DeploymentEngine{
		gitManager:       gitManager,
		dockerManager:    dockerManager,
		lifecycleManager: lifecycleManager,
		resourceManager:  resourceManager,
		store:            store,
		auditLogger:      auditLogger,
		monitor:          monitor,
		deployments:      make(map[string]*Deployment),
		ctx:              ctx,
		cancel:           cancel,
	}

	return engine, nil
}

// Start initializes the deployment engine
func (de *DeploymentEngine) Start() error {
	logrus.Info("Starting deployment engine")

	// Load existing deployments from storage
	if err := de.loadDeployments(); err != nil {
		logrus.Warnf("Failed to load deployments: %v", err)
	}

	// Start monitoring goroutine
	de.wg.Add(1)
	go de.monitorDeployments()

	de.auditLogger.LogEvent("DEPLOYMENT_ENGINE_STARTED", map[string]interface{}{
		"deployment_count": len(de.deployments),
	})

	return nil
}

// Stop gracefully stops the deployment engine
func (de *DeploymentEngine) Stop() error {
	logrus.Info("Stopping deployment engine")

	de.cancel()
	de.wg.Wait()

	de.auditLogger.LogEvent("DEPLOYMENT_ENGINE_STOPPED", map[string]interface{}{})

	return nil
}

// Deploy creates and starts a new deployment
func (de *DeploymentEngine) Deploy(request *DeploymentRequest) (*Deployment, error) {
	de.mu.Lock()
	defer de.mu.Unlock()

	// Generate deployment ID
	deploymentID := fmt.Sprintf("%s-%s-%d", request.AppID, request.Version, time.Now().Unix())

	// Create deployment
	deployment := &Deployment{
		ID:             deploymentID,
		AppID:          request.AppID,
		Version:        request.Version,
		Status:         StatusPending,
		Source:         request.Source,
		Config:         request.Config,
		ResourceLimits: request.ResourceLimits,
		HealthCheck:    request.HealthCheck,
		Environment:    request.Environment,
		Ports:          request.Ports,
		Networks:       request.Networks,
		Volumes:        request.Volumes,
		Labels:         request.Labels,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		BuildLogs:      []LogEntry{},
		DeploymentLogs: []LogEntry{},
		Metrics:        DeploymentMetrics{},
	}

	// Store deployment
	de.deployments[deploymentID] = deployment

	// Start deployment process asynchronously
	de.wg.Add(1)
	go de.deployAsync(deployment)

	de.auditLogger.LogEvent("DEPLOYMENT_CREATED", map[string]interface{}{
		"deployment_id": deploymentID,
		"app_id":        request.AppID,
		"version":       request.Version,
		"source_type":   request.Source.Type,
	})

	return deployment, nil
}

// deployAsync handles the complete deployment process
func (de *DeploymentEngine) deployAsync(deployment *Deployment) {
	defer de.wg.Done()

	ctx, cancel := context.WithTimeout(de.ctx, deployment.Config.ProgressTimeout)
	defer cancel()

	// Update status to building
	de.updateDeploymentStatus(deployment, StatusBuilding)

	// Step 1: Build or pull image
	var imageID string
	var err error

	if deployment.Source.Type == "git" {
		imageID, err = de.buildFromGit(ctx, deployment)
	} else if deployment.Source.Type == "docker" {
		imageID, err = de.pullDockerImage(ctx, deployment)
	} else {
		err = fmt.Errorf("unsupported source type: %s", deployment.Source.Type)
	}

	if err != nil {
		de.handleDeploymentError(deployment, fmt.Errorf("failed to prepare image: %w", err))
		return
	}

	// Update status to deploying
	de.updateDeploymentStatus(deployment, StatusDeploying)

	// Step 2: Deploy container
	containerID, err := de.deployContainer(ctx, deployment, imageID)
	if err != nil {
		de.handleDeploymentError(deployment, fmt.Errorf("failed to deploy container: %w", err))
		return
	}

	deployment.ContainerID = containerID
	deployment.ContainerName = fmt.Sprintf("superagent-%s", deployment.ID)

	// Step 3: Health check
	if deployment.HealthCheck.Enabled {
		de.updateDeploymentStatus(deployment, StatusHealthCheck)
		if err := de.performHealthCheck(ctx, deployment); err != nil {
			de.handleDeploymentError(deployment, fmt.Errorf("health check failed: %w", err))
			return
		}
	}

	// Step 4: Mark as running
	now := time.Now()
	deployment.DeployedAt = &now
	de.updateDeploymentStatus(deployment, StatusRunning)

	// Step 5: Start monitoring
	de.startDeploymentMonitoring(deployment)

	de.auditLogger.LogEvent("DEPLOYMENT_COMPLETED", map[string]interface{}{
		"deployment_id": deployment.ID,
		"container_id":  containerID,
		"duration":      time.Since(deployment.CreatedAt).Seconds(),
	})
}

// buildFromGit builds a Docker image from a Git repository
func (de *DeploymentEngine) buildFromGit(ctx context.Context, deployment *Deployment) (string, error) {
	// Clone repository
	repoPath, err := de.gitManager.CloneRepository(ctx, deployment.Source.Repository, deployment.Source.Branch, deployment.Source.Auth)
	if err != nil {
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}
	defer de.gitManager.CleanupRepository(repoPath)

	// Build Docker image
	buildContext := docker.BuildContext{
		ContextPath:  repoPath,
		Dockerfile:   deployment.Source.Dockerfile,
		BuildPath:    deployment.Source.BuildPath,
		ImageTag:     fmt.Sprintf("superagent/%s:%s", deployment.AppID, deployment.Version),
		BuildArgs:    deployment.Environment,
		Labels:       deployment.Labels,
		NoCache:      false,
		Pull:         true,
		Target:       "",
		Platform:     "linux/amd64",
	}

	imageID, err := de.dockerManager.BuildImage(ctx, buildContext, func(log string) {
		de.addBuildLog(deployment, "info", log)
	})

	if err != nil {
		return "", fmt.Errorf("failed to build image: %w", err)
	}

	return imageID, nil
}

// pullDockerImage pulls a Docker image from a registry
func (de *DeploymentEngine) pullDockerImage(ctx context.Context, deployment *Deployment) (string, error) {
	imageName := deployment.Source.Repository
	if deployment.Source.Tag != "" {
		imageName = fmt.Sprintf("%s:%s", imageName, deployment.Source.Tag)
	}

	imageID, err := de.dockerManager.PullImage(ctx, imageName, deployment.Source.Auth, func(log string) {
		de.addBuildLog(deployment, "info", log)
	})

	if err != nil {
		return "", fmt.Errorf("failed to pull image: %w", err)
	}

	return imageID, nil
}

// deployContainer creates and starts a container
func (de *DeploymentEngine) deployContainer(ctx context.Context, deployment *Deployment, imageID string) (string, error) {
	containerConfig := docker.ContainerConfig{
		Image:        imageID,
		Name:         deployment.ContainerName,
		Environment:  deployment.Environment,
		Ports:        convertPortMappings(deployment.Ports),
		Volumes:      convertVolumeMappings(deployment.Volumes),
		Networks:     deployment.Networks,
		Labels:       deployment.Labels,
		Command:      deployment.Config.Command,
		Args:         deployment.Config.Args,
		WorkingDir:   deployment.Config.WorkingDir,
		User:         deployment.Config.User,
		Privileged:   deployment.Config.Privileged,
		ReadOnlyRootFS: deployment.Config.ReadOnlyRootFS,
		RestartPolicy: deployment.Config.RestartPolicy,
		ResourceLimits: docker.ResourceLimits{
			CPULimit:      deployment.ResourceLimits.CPULimit,
			MemoryLimit:   deployment.ResourceLimits.MemoryLimit,
			DiskLimit:     deployment.ResourceLimits.DiskLimit,
			NetworkLimit:  deployment.ResourceLimits.NetworkLimit,
			ProcessLimit:  deployment.ResourceLimits.ProcessLimit,
		},
		SecurityOpts: buildSecurityOpts(deployment.Config.Security),
	}

	containerID, err := de.dockerManager.CreateContainer(ctx, containerConfig)
	if err != nil {
		return "", fmt.Errorf("failed to create container: %w", err)
	}

	if err := de.dockerManager.StartContainer(ctx, containerID); err != nil {
		// Cleanup container if start fails
		de.dockerManager.RemoveContainer(ctx, containerID, true)
		return "", fmt.Errorf("failed to start container: %w", err)
	}

	return containerID, nil
}

// performHealthCheck performs health checks on the deployment
func (de *DeploymentEngine) performHealthCheck(ctx context.Context, deployment *Deployment) error {
	return de.lifecycleManager.PerformHealthCheck(ctx, deployment.ContainerID, lifecycle.HealthCheckConfig{
		Type:                deployment.HealthCheck.Type,
		Path:                deployment.HealthCheck.Path,
		Port:                deployment.HealthCheck.Port,
		Command:             deployment.HealthCheck.Command,
		InitialDelaySeconds: deployment.HealthCheck.InitialDelaySeconds,
		PeriodSeconds:       deployment.HealthCheck.PeriodSeconds,
		TimeoutSeconds:      deployment.HealthCheck.TimeoutSeconds,
		FailureThreshold:    deployment.HealthCheck.FailureThreshold,
		SuccessThreshold:    deployment.HealthCheck.SuccessThreshold,
		Headers:             deployment.HealthCheck.Headers,
	})
}

// StopDeployment stops a deployment
func (de *DeploymentEngine) StopDeployment(deploymentID string) error {
	de.mu.Lock()
	defer de.mu.Unlock()

	deployment, exists := de.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	if deployment.Status == StatusStopped {
		return nil // Already stopped
	}

	de.updateDeploymentStatus(deployment, StatusStopping)

	// Stop container
	if deployment.ContainerID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := de.dockerManager.StopContainer(ctx, deployment.ContainerID, 10); err != nil {
			logrus.Warnf("Failed to stop container %s: %v", deployment.ContainerID, err)
		}
	}

	de.updateDeploymentStatus(deployment, StatusStopped)

	de.auditLogger.LogEvent("DEPLOYMENT_STOPPED", map[string]interface{}{
		"deployment_id": deploymentID,
		"container_id":  deployment.ContainerID,
	})

	return nil
}

// Remove removes a deployment completely
func (de *DeploymentEngine) Remove(deploymentID string) error {
	de.mu.Lock()
	defer de.mu.Unlock()

	deployment, exists := de.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	// Stop if running
	if deployment.Status == StatusRunning {
		if err := de.StopDeployment(deploymentID); err != nil {
			return fmt.Errorf("failed to stop deployment: %w", err)
		}
	}

	// Remove container
	if deployment.ContainerID != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := de.dockerManager.RemoveContainer(ctx, deployment.ContainerID, true); err != nil {
			logrus.Warnf("Failed to remove container %s: %v", deployment.ContainerID, err)
		}
	}

	// Remove from storage
	if err := de.store.DeleteDeploymentState(deploymentID); err != nil {
		logrus.Warnf("Failed to delete deployment state: %v", err)
	}

	// Remove from memory
	delete(de.deployments, deploymentID)

	de.auditLogger.LogEvent("DEPLOYMENT_REMOVED", map[string]interface{}{
		"deployment_id": deploymentID,
	})

	return nil
}

// GetDeployment returns deployment information
func (de *DeploymentEngine) GetDeployment(deploymentID string) (*Deployment, error) {
	de.mu.RLock()
	defer de.mu.RUnlock()

	deployment, exists := de.deployments[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	return deployment, nil
}

// ListDeployments returns all deployments
func (de *DeploymentEngine) ListDeployments() []*Deployment {
	de.mu.RLock()
	defer de.mu.RUnlock()

	deployments := make([]*Deployment, 0, len(de.deployments))
	for _, deployment := range de.deployments {
		deployments = append(deployments, deployment)
	}

	return deployments
}

// Rollback rolls back a deployment to a previous version
func (de *DeploymentEngine) Rollback(deploymentID string, reason string) error {
	de.mu.Lock()
	defer de.mu.Unlock()

	deployment, exists := de.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	if deployment.Rollback == nil {
		return fmt.Errorf("no rollback information available for deployment: %s", deploymentID)
	}

	de.updateDeploymentStatus(deployment, StatusRollingBack)

	// Implement rollback logic here
	// This is a simplified version - in production you'd want more sophisticated rollback

	rollbackInfo := &RollbackInfo{
		PreviousVersion: deployment.Version,
		Reason:          reason,
		Timestamp:       time.Now(),
		Status:          "completed",
	}

	deployment.Rollback = rollbackInfo
	deployment.UpdatedAt = time.Now()

	de.auditLogger.LogEvent("DEPLOYMENT_ROLLBACK", map[string]interface{}{
		"deployment_id":     deploymentID,
		"previous_version":  rollbackInfo.PreviousVersion,
		"reason":            reason,
	})

	return nil
}

// Helper functions

func (de *DeploymentEngine) updateDeploymentStatus(deployment *Deployment, status DeploymentStatus) {
	deployment.Status = status
	deployment.UpdatedAt = time.Now()

	// Save to storage
	stateData := map[string]interface{}{
		"status":     string(status),
		"updated_at": deployment.UpdatedAt,
	}

	if err := de.store.StoreDeploymentState(deployment.ID, stateData); err != nil {
		logrus.Warnf("Failed to store deployment state: %v", err)
	}

	// Send metrics
	if de.monitor != nil {
		de.monitor.RecordDeploymentStatus(deployment.ID, string(status))
	}
}

func (de *DeploymentEngine) handleDeploymentError(deployment *Deployment, err error) {
	de.updateDeploymentStatus(deployment, StatusFailed)

	de.addDeploymentLog(deployment, "error", err.Error())

	de.auditLogger.LogEvent("DEPLOYMENT_FAILED", map[string]interface{}{
		"deployment_id": deployment.ID,
		"error":         err.Error(),
	})
}

func (de *DeploymentEngine) addBuildLog(deployment *Deployment, level, message string) {
	logEntry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    "build",
	}

	deployment.BuildLogs = append(deployment.BuildLogs, logEntry)
}

func (de *DeploymentEngine) addDeploymentLog(deployment *Deployment, level, message string) {
	logEntry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Source:    "deployment",
	}

	deployment.DeploymentLogs = append(deployment.DeploymentLogs, logEntry)
}

func (de *DeploymentEngine) loadDeployments() error {
	deploymentIDs, err := de.store.ListDeployments()
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}

	for _, deploymentID := range deploymentIDs {
		state, err := de.store.LoadDeploymentState(deploymentID)
		if err != nil {
			logrus.Warnf("Failed to load deployment state for %s: %v", deploymentID, err)
			continue
		}

		// Reconstruct deployment from state
		// This is a simplified version - in production you'd want more complete state restoration
		deployment := &Deployment{
			ID:        deploymentID,
			Status:    DeploymentStatus(state["status"].(string)),
			UpdatedAt: state["updated_at"].(time.Time),
		}

		de.deployments[deploymentID] = deployment
	}

	return nil
}

func (de *DeploymentEngine) monitorDeployments() {
	defer de.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-de.ctx.Done():
			return
		case <-ticker.C:
			de.updateDeploymentMetrics()
		}
	}
}

func (de *DeploymentEngine) updateDeploymentMetrics() {
	de.mu.RLock()
	defer de.mu.RUnlock()

	for _, deployment := range de.deployments {
		if deployment.Status == StatusRunning && deployment.ContainerID != "" {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			stats, err := de.dockerManager.GetContainerStats(ctx, deployment.ContainerID)
			cancel()

			if err != nil {
				logrus.Warnf("Failed to get stats for container %s: %v", deployment.ContainerID, err)
				continue
			}

			deployment.Metrics = DeploymentMetrics{
				CPUUsage:      stats.CPUUsage,
				MemoryUsage:   stats.MemoryUsage,
				MemoryLimit:   stats.MemoryLimit,
				NetworkRx:     stats.NetworkRx,
				NetworkTx:     stats.NetworkTx,
				DiskUsage:     stats.DiskUsage,
				RestartCount:  stats.RestartCount,
				LastUpdated:   time.Now(),
			}

			// Send metrics to monitoring system
			if de.monitor != nil {
				monitoringMetrics := monitoring.DeploymentMetrics{
					CPUUsage:         deployment.Metrics.CPUUsage,
					MemoryUsage:      deployment.Metrics.MemoryUsage,
					MemoryLimit:      deployment.Metrics.MemoryLimit,
					NetworkRx:        deployment.Metrics.NetworkRx,
					NetworkTx:        deployment.Metrics.NetworkTx,
					DiskUsage:        deployment.Metrics.DiskUsage,
					RestartCount:     deployment.Metrics.RestartCount,
					HealthCheckCount: deployment.Metrics.HealthCheckCount,
					LastUpdated:      deployment.Metrics.LastUpdated,
				}
				de.monitor.RecordDeploymentMetrics(deployment.ID, monitoringMetrics)
			}
		}
	}
}

func (de *DeploymentEngine) startDeploymentMonitoring(deployment *Deployment) {
	// Start continuous health checking if enabled
	if deployment.HealthCheck.Enabled {
		de.wg.Add(1)
		go de.continuousHealthCheck(deployment)
	}
}

func (de *DeploymentEngine) continuousHealthCheck(deployment *Deployment) {
	defer de.wg.Done()

	ticker := time.NewTicker(time.Duration(deployment.HealthCheck.PeriodSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-de.ctx.Done():
			return
		case <-ticker.C:
			if deployment.Status != StatusRunning {
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), time.Duration(deployment.HealthCheck.TimeoutSeconds)*time.Second)
			err := de.performHealthCheck(ctx, deployment)
			cancel()

			now := time.Now()
			deployment.LastHealthCheck = &now

			if err != nil {
				logrus.Warnf("Health check failed for deployment %s: %v", deployment.ID, err)
				deployment.Metrics.HealthCheckCount++
				
				// If health check fails too many times, mark as failed
				if deployment.Metrics.HealthCheckCount >= deployment.HealthCheck.FailureThreshold {
					de.handleDeploymentError(deployment, fmt.Errorf("health check failed %d times", deployment.Metrics.HealthCheckCount))
					return
				}
			} else {
				deployment.Metrics.HealthCheckCount = 0 // Reset failure count on success
			}
		}
	}
}

// Helper functions for type conversion

func convertPortMappings(ports []PortMapping) []docker.PortMapping {
	dockerPorts := make([]docker.PortMapping, len(ports))
	for i, port := range ports {
		dockerPorts[i] = docker.PortMapping{
			ContainerPort: port.ContainerPort,
			HostPort:      port.HostPort,
			Protocol:      port.Protocol,
			HostIP:        port.HostIP,
		}
	}
	return dockerPorts
}

func convertVolumeMappings(volumes []VolumeMapping) []docker.VolumeMapping {
	dockerVolumes := make([]docker.VolumeMapping, len(volumes))
	for i, volume := range volumes {
		dockerVolumes[i] = docker.VolumeMapping{
			Source:      volume.Source,
			Target:      volume.Target,
			Type:        volume.Type,
			ReadOnly:    volume.ReadOnly,
			Consistency: volume.Consistency,
			Options:     volume.Options,
		}
	}
	return dockerVolumes
}

func buildSecurityOpts(security SecurityConfig) []string {
	var opts []string

	if security.SeccompProfile != "" {
		opts = append(opts, fmt.Sprintf("seccomp=%s", security.SeccompProfile))
	}

	if security.AppArmorProfile != "" {
		opts = append(opts, fmt.Sprintf("apparmor=%s", security.AppArmorProfile))
	}

	for key, value := range security.SELinuxOptions {
		opts = append(opts, fmt.Sprintf("label=%s:%s", key, value))
	}

	return opts
}