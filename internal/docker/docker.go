package docker

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"

	"superagent/internal/config"
	"superagent/internal/logging"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
	"github.com/sirupsen/logrus"
)

// ContainerManager manages Docker containers for deployments
type ContainerManager struct {
	client      *client.Client
	config      *config.Config
	auditLogger *logging.AuditLogger
	logStreamer *logging.LogStreamer
	containers  map[string]*ContainerInfo
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// ContainerInfo holds information about a running container
type ContainerInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	AppName     string                 `json:"app_name"`
	Version     string                 `json:"version"`
	Image       string                 `json:"image"`
	Status      string                 `json:"status"`
	State       string                 `json:"state"`
	Ports       []PortMapping          `json:"ports"`
	Environment map[string]string      `json:"environment"`
	Resources   ResourceLimits         `json:"resources"`
	Networks    []string               `json:"networks"`
	Volumes     []VolumeMount          `json:"volumes"`
	Labels      map[string]string      `json:"labels"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   time.Time              `json:"started_at"`
	FinishedAt  time.Time              `json:"finished_at"`
	Health      HealthStatus           `json:"health"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// PortMapping represents a port mapping
type PortMapping struct {
	HostPort      int    `json:"host_port"`
	ContainerPort int    `json:"container_port"`
	Protocol      string `json:"protocol"`
}

// ResourceLimits represents resource limits for a container
type ResourceLimits struct {
	CPUShares     int64  `json:"cpu_shares"`
	CPUQuota      int64  `json:"cpu_quota"`
	CPUPeriod     int64  `json:"cpu_period"`
	Memory        int64  `json:"memory"`
	MemorySwap    int64  `json:"memory_swap"`
	BlkioWeight   uint16 `json:"blkio_weight"`
	DiskQuota     int64  `json:"disk_quota"`
	NetworkLimit  int64  `json:"network_limit"`
	PidsLimit     int64  `json:"pids_limit"`
}

// VolumeMount represents a volume mount
type VolumeMount struct {
	Type     string `json:"type"`
	Source   string `json:"source"`
	Target   string `json:"target"`
	ReadOnly bool   `json:"read_only"`
}

// HealthStatus represents container health status
type HealthStatus struct {
	Status        string    `json:"status"`
	FailingStreak int       `json:"failing_streak"`
	Log           []string  `json:"log"`
	LastCheck     time.Time `json:"last_check"`
}

// DeploymentSpec represents a deployment specification
type DeploymentSpec struct {
	Name         string                 `json:"name"`
	Image        string                 `json:"image"`
	Tag          string                 `json:"tag"`
	Ports        []PortMapping          `json:"ports"`
	Environment  map[string]string      `json:"environment"`
	Resources    ResourceLimits         `json:"resources"`
	Volumes      []VolumeMount          `json:"volumes"`
	Networks     []string               `json:"networks"`
	Labels       map[string]string      `json:"labels"`
	HealthCheck  HealthCheckConfig      `json:"health_check"`
	RestartPolicy RestartPolicy         `json:"restart_policy"`
	SecurityOpts []string               `json:"security_opts"`
	Capabilities []string               `json:"capabilities"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Test         []string      `json:"test"`
	Interval     time.Duration `json:"interval"`
	Timeout      time.Duration `json:"timeout"`
	Retries      int           `json:"retries"`
	StartPeriod  time.Duration `json:"start_period"`
}

// RestartPolicy represents restart policy
type RestartPolicy struct {
	Name              string `json:"name"`
	MaximumRetryCount int    `json:"maximum_retry_count"`
}

// ContainerStats represents container statistics
type ContainerStats struct {
	ContainerID string                 `json:"container_id"`
	Name        string                 `json:"name"`
	CPUUsage    float64                `json:"cpu_usage"`
	MemoryUsage int64                  `json:"memory_usage"`
	NetworkIO   NetworkIOStats         `json:"network_io"`
	BlockIO     BlockIOStats           `json:"block_io"`
	Timestamp   time.Time              `json:"timestamp"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// NetworkIOStats represents network I/O statistics
type NetworkIOStats struct {
	RxBytes   int64 `json:"rx_bytes"`
	TxBytes   int64 `json:"tx_bytes"`
	RxPackets int64 `json:"rx_packets"`
	TxPackets int64 `json:"tx_packets"`
}

// BlockIOStats represents block I/O statistics
type BlockIOStats struct {
	ReadBytes  int64 `json:"read_bytes"`
	WriteBytes int64 `json:"write_bytes"`
	ReadOps    int64 `json:"read_ops"`
	WriteOps   int64 `json:"write_ops"`
}

// NewContainerManager creates a new container manager
func NewContainerManager(cfg *config.Config, auditLogger *logging.AuditLogger, logStreamer *logging.LogStreamer) (*ContainerManager, error) {
	// Create Docker client
	dockerClient, err := client.NewClientWithOpts(
		client.WithHost(cfg.Docker.Host),
		client.WithVersion(cfg.Docker.Version),
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Test Docker connection
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	_, err = dockerClient.Ping(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Docker daemon: %w", err)
	}

	// Create context for lifecycle management
	ctx, cancel = context.WithCancel(context.Background())

	cm := &ContainerManager{
		client:      dockerClient,
		config:      cfg,
		auditLogger: auditLogger,
		logStreamer: logStreamer,
		containers:  make(map[string]*ContainerInfo),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Start background tasks
	cm.wg.Add(2)
	go cm.monitorContainers()
	go cm.cleanupContainers()

	return cm, nil
}

// DeployContainer deploys a new container
func (cm *ContainerManager) DeployContainer(ctx context.Context, spec *DeploymentSpec) (*ContainerInfo, error) {
	logrus.Infof("Deploying container: %s", spec.Name)

	// Validate deployment spec
	if err := cm.validateDeploymentSpec(spec); err != nil {
		cm.auditLogger.LogDeploymentEventWithContext(ctx, "DEPLOY_VALIDATION_FAILED", spec.Name, false, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("deployment validation failed: %w", err)
	}

	// Check resource availability
	if err := cm.checkResourceAvailability(spec); err != nil {
		cm.auditLogger.LogDeploymentEventWithContext(ctx, "DEPLOY_RESOURCE_CHECK_FAILED", spec.Name, false, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("resource check failed: %w", err)
	}

	// Pull image if needed
	if err := cm.pullImage(ctx, spec.Image, spec.Tag); err != nil {
		cm.auditLogger.LogDeploymentEventWithContext(ctx, "DEPLOY_IMAGE_PULL_FAILED", spec.Name, false, map[string]interface{}{
			"image": spec.Image,
			"tag":   spec.Tag,
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to pull image: %w", err)
	}

	// Create container configuration
	containerConfig := cm.createContainerConfig(spec)
	hostConfig := cm.createHostConfig(spec)
	networkingConfig := cm.createNetworkingConfig(spec)

	// Create container
	containerName := fmt.Sprintf("%s-%s", spec.Name, generateRandomString(8))
	resp, err := cm.client.ContainerCreate(ctx, containerConfig, hostConfig, networkingConfig, nil, containerName)
	if err != nil {
		cm.auditLogger.LogDeploymentEventWithContext(ctx, "DEPLOY_CONTAINER_CREATE_FAILED", spec.Name, false, map[string]interface{}{
			"error": err.Error(),
		})
		return nil, fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := cm.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		// Clean up created container
		cm.client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{Force: true})
		cm.auditLogger.LogDeploymentEventWithContext(ctx, "DEPLOY_CONTAINER_START_FAILED", spec.Name, false, map[string]interface{}{
			"container_id": resp.ID,
			"error":        err.Error(),
		})
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	// Get container info
	containerInfo, err := cm.getContainerInfo(ctx, resp.ID)
	if err != nil {
		cm.auditLogger.LogDeploymentEventWithContext(ctx, "DEPLOY_CONTAINER_INFO_FAILED", spec.Name, false, map[string]interface{}{
			"container_id": resp.ID,
			"error":        err.Error(),
		})
		return nil, fmt.Errorf("failed to get container info: %w", err)
	}

	// Store container info
	cm.mu.Lock()
	cm.containers[resp.ID] = containerInfo
	cm.mu.Unlock()

	// Start log streaming
	if cm.logStreamer != nil {
		go cm.streamContainerLogs(ctx, resp.ID)
	}

	cm.auditLogger.LogDeploymentEventWithContext(ctx, "DEPLOY_CONTAINER_SUCCESS", spec.Name, true, map[string]interface{}{
		"container_id": resp.ID,
		"image":        spec.Image,
		"tag":          spec.Tag,
	})

	logrus.Infof("Container deployed successfully: %s (%s)", containerName, resp.ID)
	return containerInfo, nil
}

// StopContainer stops a running container
func (cm *ContainerManager) StopContainer(ctx context.Context, containerID string) error {
	logrus.Infof("Stopping container: %s", containerID)

	// Get container info
	containerInfo, exists := cm.getContainerFromCache(containerID)
	if !exists {
		return fmt.Errorf("container not found: %s", containerID)
	}

	// Stop container with timeout
	timeout := 30 * time.Second
	if err := cm.client.ContainerStop(ctx, containerID, &timeout); err != nil {
		cm.auditLogger.LogDeploymentEventWithContext(ctx, "STOP_CONTAINER_FAILED", containerInfo.Name, false, map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to stop container: %w", err)
	}

	// Update container info
	cm.mu.Lock()
	if info, exists := cm.containers[containerID]; exists {
		info.Status = "stopped"
		info.FinishedAt = time.Now()
	}
	cm.mu.Unlock()

	cm.auditLogger.LogDeploymentEventWithContext(ctx, "STOP_CONTAINER_SUCCESS", containerInfo.Name, true, map[string]interface{}{
		"container_id": containerID,
	})

	logrus.Infof("Container stopped successfully: %s", containerID)
	return nil
}

// RestartContainer restarts a container
func (cm *ContainerManager) RestartContainer(ctx context.Context, containerID string) error {
	logrus.Infof("Restarting container: %s", containerID)

	// Get container info
	containerInfo, exists := cm.getContainerFromCache(containerID)
	if !exists {
		return fmt.Errorf("container not found: %s", containerID)
	}

	// Restart container with timeout
	timeout := 30 * time.Second
	if err := cm.client.ContainerRestart(ctx, containerID, &timeout); err != nil {
		cm.auditLogger.LogDeploymentEventWithContext(ctx, "RESTART_CONTAINER_FAILED", containerInfo.Name, false, map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to restart container: %w", err)
	}

	// Update container info
	cm.mu.Lock()
	if info, exists := cm.containers[containerID]; exists {
		info.Status = "running"
		info.StartedAt = time.Now()
	}
	cm.mu.Unlock()

	cm.auditLogger.LogDeploymentEventWithContext(ctx, "RESTART_CONTAINER_SUCCESS", containerInfo.Name, true, map[string]interface{}{
		"container_id": containerID,
	})

	logrus.Infof("Container restarted successfully: %s", containerID)
	return nil
}

// DeleteContainer removes a container
func (cm *ContainerManager) DeleteContainer(ctx context.Context, containerID string) error {
	logrus.Infof("Deleting container: %s", containerID)

	// Get container info
	containerInfo, exists := cm.getContainerFromCache(containerID)
	if !exists {
		return fmt.Errorf("container not found: %s", containerID)
	}

	// Stop container first if running
	if containerInfo.Status == "running" {
		if err := cm.StopContainer(ctx, containerID); err != nil {
			logrus.Warnf("Failed to stop container before deletion: %v", err)
		}
	}

	// Remove container
	if err := cm.client.ContainerRemove(ctx, containerID, types.ContainerRemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	}); err != nil {
		cm.auditLogger.LogDeploymentEventWithContext(ctx, "DELETE_CONTAINER_FAILED", containerInfo.Name, false, map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
		})
		return fmt.Errorf("failed to remove container: %w", err)
	}

	// Remove from cache
	cm.mu.Lock()
	delete(cm.containers, containerID)
	cm.mu.Unlock()

	cm.auditLogger.LogDeploymentEventWithContext(ctx, "DELETE_CONTAINER_SUCCESS", containerInfo.Name, true, map[string]interface{}{
		"container_id": containerID,
	})

	logrus.Infof("Container deleted successfully: %s", containerID)
	return nil
}

// GetContainerInfo returns information about a container
func (cm *ContainerManager) GetContainerInfo(ctx context.Context, containerID string) (*ContainerInfo, error) {
	return cm.getContainerInfo(ctx, containerID)
}

// ListContainers lists all managed containers
func (cm *ContainerManager) ListContainers(ctx context.Context) ([]*ContainerInfo, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	containers := make([]*ContainerInfo, 0, len(cm.containers))
	for _, info := range cm.containers {
		containers = append(containers, info)
	}

	return containers, nil
}

// GetContainerStats returns statistics for a container
func (cm *ContainerManager) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	// Get container info
	containerInfo, exists := cm.getContainerFromCache(containerID)
	if !exists {
		return nil, fmt.Errorf("container not found: %s", containerID)
	}

	// Get stats from Docker
	stats, err := cm.client.ContainerStats(ctx, containerID, false)
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}
	defer stats.Body.Close()

	// Parse stats (simplified)
	containerStats := &ContainerStats{
		ContainerID: containerID,
		Name:        containerInfo.Name,
		Timestamp:   time.Now(),
	}

	// TODO: Parse actual stats from stats.Body

	return containerStats, nil
}

// UpdateContainer updates a container with zero-downtime
func (cm *ContainerManager) UpdateContainer(ctx context.Context, containerID string, spec *DeploymentSpec) (*ContainerInfo, error) {
	logrus.Infof("Updating container: %s", containerID)

	// Get current container info
	oldInfo, exists := cm.getContainerFromCache(containerID)
	if !exists {
		return nil, fmt.Errorf("container not found: %s", containerID)
	}

	// Deploy new container
	newInfo, err := cm.DeployContainer(ctx, spec)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy new container: %w", err)
	}

	// Wait for new container to be healthy
	if err := cm.waitForHealthy(ctx, newInfo.ID, 60*time.Second); err != nil {
		// Clean up new container
		cm.DeleteContainer(ctx, newInfo.ID)
		return nil, fmt.Errorf("new container failed health check: %w", err)
	}

	// Update routing (handled by Traefik integration)
	// TODO: Update Traefik configuration

	// Stop old container
	if err := cm.StopContainer(ctx, containerID); err != nil {
		logrus.Warnf("Failed to stop old container: %v", err)
	}

	// Clean up old container after delay
	go func() {
		time.Sleep(5 * time.Minute)
		cm.DeleteContainer(context.Background(), containerID)
	}()

	cm.auditLogger.LogDeploymentEventWithContext(ctx, "UPDATE_CONTAINER_SUCCESS", oldInfo.Name, true, map[string]interface{}{
		"old_container_id": containerID,
		"new_container_id": newInfo.ID,
	})

	logrus.Infof("Container updated successfully: %s -> %s", containerID, newInfo.ID)
	return newInfo, nil
}

// validateDeploymentSpec validates a deployment specification
func (cm *ContainerManager) validateDeploymentSpec(spec *DeploymentSpec) error {
	if spec.Name == "" {
		return fmt.Errorf("deployment name is required")
	}

	if spec.Image == "" {
		return fmt.Errorf("image is required")
	}

	// Validate image registry
	if err := cm.validateImageRegistry(spec.Image); err != nil {
		return fmt.Errorf("invalid image registry: %w", err)
	}

	// Validate resource limits
	if err := cm.validateResourceLimits(&spec.Resources); err != nil {
		return fmt.Errorf("invalid resource limits: %w", err)
	}

	// Validate ports
	for _, port := range spec.Ports {
		if port.ContainerPort <= 0 || port.ContainerPort > 65535 {
			return fmt.Errorf("invalid container port: %d", port.ContainerPort)
		}
		if port.HostPort <= 0 || port.HostPort > 65535 {
			return fmt.Errorf("invalid host port: %d", port.HostPort)
		}
	}

	return nil
}

// validateImageRegistry validates if the image registry is allowed
func (cm *ContainerManager) validateImageRegistry(image string) error {
	// Extract registry from image
	parts := strings.Split(image, "/")
	if len(parts) < 2 {
		return fmt.Errorf("invalid image format: %s", image)
	}

	registry := parts[0]

	// Check allowed registries
	if len(cm.config.Security.AllowedRegistries) > 0 {
		allowed := false
		for _, allowedRegistry := range cm.config.Security.AllowedRegistries {
			if registry == allowedRegistry {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("registry not allowed: %s", registry)
		}
	}

	// Check blocked registries
	for _, blockedRegistry := range cm.config.Security.BlockedRegistries {
		if registry == blockedRegistry {
			return fmt.Errorf("registry blocked: %s", registry)
		}
	}

	return nil
}

// validateResourceLimits validates resource limits
func (cm *ContainerManager) validateResourceLimits(limits *ResourceLimits) error {
	// Check memory limits
	if limits.Memory < 0 {
		return fmt.Errorf("memory limit cannot be negative")
	}

	// Check CPU limits
	if limits.CPUShares < 0 {
		return fmt.Errorf("CPU shares cannot be negative")
	}

	// Check disk quota
	if limits.DiskQuota < 0 {
		return fmt.Errorf("disk quota cannot be negative")
	}

	return nil
}

// checkResourceAvailability checks if resources are available for deployment
func (cm *ContainerManager) checkResourceAvailability(spec *DeploymentSpec) error {
	// Get current resource usage
	// TODO: Implement resource usage checking
	return nil
}

// pullImage pulls a Docker image
func (cm *ContainerManager) pullImage(ctx context.Context, image, tag string) error {
	imageRef := image
	if tag != "" {
		imageRef = fmt.Sprintf("%s:%s", image, tag)
	}

	logrus.Infof("Pulling image: %s", imageRef)

	// Check if image exists locally
	_, _, err := cm.client.ImageInspectWithRaw(ctx, imageRef)
	if err == nil {
		// Image exists locally
		return nil
	}

	// Pull image
	reader, err := cm.client.ImagePull(ctx, imageRef, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("failed to pull image: %w", err)
	}
	defer reader.Close()

	// Wait for pull to complete
	_, err = io.Copy(io.Discard, reader)
	if err != nil {
		return fmt.Errorf("failed to complete image pull: %w", err)
	}

	logrus.Infof("Image pulled successfully: %s", imageRef)
	return nil
}

// createContainerConfig creates container configuration
func (cm *ContainerManager) createContainerConfig(spec *DeploymentSpec) *container.Config {
	// Convert environment variables
	env := make([]string, 0, len(spec.Environment))
	for key, value := range spec.Environment {
		env = append(env, fmt.Sprintf("%s=%s", key, value))
	}

	// Convert ports
	exposedPorts := make(nat.PortSet)
	for _, port := range spec.Ports {
		portSpec := nat.Port(fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol))
		exposedPorts[portSpec] = struct{}{}
	}

	// Create health check
	var healthCheck *container.HealthConfig
	if len(spec.HealthCheck.Test) > 0 {
		healthCheck = &container.HealthConfig{
			Test:        spec.HealthCheck.Test,
			Interval:    spec.HealthCheck.Interval,
			Timeout:     spec.HealthCheck.Timeout,
			Retries:     spec.HealthCheck.Retries,
			StartPeriod: spec.HealthCheck.StartPeriod,
		}
	}

	imageRef := spec.Image
	if spec.Tag != "" {
		imageRef = fmt.Sprintf("%s:%s", spec.Image, spec.Tag)
	}

	return &container.Config{
		Image:        imageRef,
		Env:          env,
		ExposedPorts: exposedPorts,
		Labels:       spec.Labels,
		Healthcheck:  healthCheck,
		User:         "1000:1000", // Run as non-root user
	}
}

// createHostConfig creates host configuration
func (cm *ContainerManager) createHostConfig(spec *DeploymentSpec) *container.HostConfig {
	// Convert port bindings
	portBindings := make(nat.PortMap)
	for _, port := range spec.Ports {
		containerPort := nat.Port(fmt.Sprintf("%d/%s", port.ContainerPort, port.Protocol))
		portBindings[containerPort] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: strconv.Itoa(port.HostPort),
			},
		}
	}

	// Convert mounts
	mounts := make([]mount.Mount, 0, len(spec.Volumes))
	for _, volume := range spec.Volumes {
		mounts = append(mounts, mount.Mount{
			Type:     mount.Type(volume.Type),
			Source:   volume.Source,
			Target:   volume.Target,
			ReadOnly: volume.ReadOnly,
		})
	}

	// Create resource limits
	resources := container.Resources{
		Memory:     spec.Resources.Memory,
		CPUShares:  spec.Resources.CPUShares,
		CPUQuota:   spec.Resources.CPUQuota,
		CPUPeriod:  spec.Resources.CPUPeriod,
		BlkioWeight: spec.Resources.BlkioWeight,
		PidsLimit:  &spec.Resources.PidsLimit,
	}

	// Security options
	securityOpts := append([]string{}, spec.SecurityOpts...)
	if cm.config.Security.RunAsNonRoot {
		securityOpts = append(securityOpts, "no-new-privileges:true")
	}

	// Add seccomp profile
	if cm.config.Security.SeccompProfile != "" {
		securityOpts = append(securityOpts, fmt.Sprintf("seccomp:%s", cm.config.Security.SeccompProfile))
	}

	// Add AppArmor profile
	if cm.config.Security.AppArmorProfile != "" {
		securityOpts = append(securityOpts, fmt.Sprintf("apparmor:%s", cm.config.Security.AppArmorProfile))
	}

	return &container.HostConfig{
		PortBindings: portBindings,
		Mounts:       mounts,
		Resources:    resources,
		SecurityOpt:  securityOpts,
		ReadonlyRootfs: cm.config.Security.ReadOnlyRootFS,
		LogConfig: container.LogConfig{
			Type: cm.config.Docker.LogDriver,
			Config: cm.config.Docker.LogOptions,
		},
		RestartPolicy: container.RestartPolicy{
			Name:              spec.RestartPolicy.Name,
			MaximumRetryCount: spec.RestartPolicy.MaximumRetryCount,
		},
		NetworkMode: container.NetworkMode(cm.config.Docker.NetworkName),
	}
}

// createNetworkingConfig creates networking configuration
func (cm *ContainerManager) createNetworkingConfig(spec *DeploymentSpec) *network.NetworkingConfig {
	return &network.NetworkingConfig{
		EndpointsConfig: map[string]*network.EndpointSettings{
			cm.config.Docker.NetworkName: {
				NetworkID: cm.config.Docker.NetworkName,
			},
		},
	}
}

// getContainerInfo retrieves container information
func (cm *ContainerManager) getContainerInfo(ctx context.Context, containerID string) (*ContainerInfo, error) {
	// Get container details
	containerJSON, err := cm.client.ContainerInspect(ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	// Convert to ContainerInfo
	info := &ContainerInfo{
		ID:      containerJSON.ID,
		Name:    strings.TrimPrefix(containerJSON.Name, "/"),
		Image:   containerJSON.Image,
		Status:  containerJSON.State.Status,
		State:   containerJSON.State.Status,
		Labels:  containerJSON.Config.Labels,
	}

	// Parse creation time
	if createdTime, err := time.Parse(time.RFC3339, containerJSON.Created); err == nil {
		info.CreatedAt = createdTime
	}

	// Parse start time
	if containerJSON.State.StartedAt != "" {
		if startTime, err := time.Parse(time.RFC3339, containerJSON.State.StartedAt); err == nil {
			info.StartedAt = startTime
		}
	}

	// Parse finish time
	if containerJSON.State.FinishedAt != "" {
		if finishTime, err := time.Parse(time.RFC3339, containerJSON.State.FinishedAt); err == nil {
			info.FinishedAt = finishTime
		}
	}

	// Parse ports
	for containerPort, bindings := range containerJSON.NetworkSettings.Ports {
		for _, binding := range bindings {
			if hostPort, err := strconv.Atoi(binding.HostPort); err == nil {
				if contPort, err := strconv.Atoi(containerPort.Port()); err == nil {
					info.Ports = append(info.Ports, PortMapping{
						HostPort:      hostPort,
						ContainerPort: contPort,
						Protocol:      containerPort.Proto(),
					})
				}
			}
		}
	}

	// Parse environment variables
	info.Environment = make(map[string]string)
	for _, env := range containerJSON.Config.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			info.Environment[parts[0]] = parts[1]
		}
	}

	// Parse networks
	for networkName := range containerJSON.NetworkSettings.Networks {
		info.Networks = append(info.Networks, networkName)
	}

	return info, nil
}

// getContainerFromCache retrieves container info from cache
func (cm *ContainerManager) getContainerFromCache(containerID string) (*ContainerInfo, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	info, exists := cm.containers[containerID]
	return info, exists
}

// waitForHealthy waits for a container to become healthy
func (cm *ContainerManager) waitForHealthy(ctx context.Context, containerID string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for container to become healthy")
		case <-ticker.C:
			containerJSON, err := cm.client.ContainerInspect(ctx, containerID)
			if err != nil {
				return fmt.Errorf("failed to inspect container: %w", err)
			}

			if containerJSON.State.Health != nil {
				switch containerJSON.State.Health.Status {
				case "healthy":
					return nil
				case "unhealthy":
					return fmt.Errorf("container is unhealthy")
				}
			} else {
				// No health check defined, assume healthy if running
				if containerJSON.State.Running {
					return nil
				}
			}
		}
	}
}

// streamContainerLogs streams container logs
func (cm *ContainerManager) streamContainerLogs(ctx context.Context, containerID string) {
	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
		Timestamps: true,
	}

	reader, err := cm.client.ContainerLogs(ctx, containerID, logOptions)
	if err != nil {
		logrus.Errorf("Failed to get container logs: %v", err)
		return
	}

	logReader := logging.NewContainerLogReader(containerID, reader, cm.logStreamer)
	logReader.Start()

	// Wait for context cancellation
	<-ctx.Done()
	logReader.Stop()
}

// monitorContainers monitors container status
func (cm *ContainerManager) monitorContainers() {
	defer cm.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.updateContainerStatus()
		}
	}
}

// updateContainerStatus updates container status
func (cm *ContainerManager) updateContainerStatus() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// List all containers
	containers, err := cm.client.ContainerList(ctx, types.ContainerListOptions{
		All: true,
		Filters: filters.NewArgs(filters.Arg("label", "managed-by=deployment-agent")),
	})
	if err != nil {
		logrus.Errorf("Failed to list containers: %v", err)
		return
	}

	// Update container status
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, container := range containers {
		if info, exists := cm.containers[container.ID]; exists {
			info.Status = container.Status
			info.State = container.State
		}
	}
}

// cleanupContainers cleans up stopped containers
func (cm *ContainerManager) cleanupContainers() {
	defer cm.wg.Done()

	ticker := time.NewTicker(cm.config.Docker.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			return
		case <-ticker.C:
			cm.performCleanup()
		}
	}
}

// performCleanup performs container cleanup
func (cm *ContainerManager) performCleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Clean up stopped containers
	cm.cleanupStoppedContainers(ctx)

	// Clean up unused images
	cm.cleanupUnusedImages(ctx)

	// Clean up unused volumes
	cm.cleanupUnusedVolumes(ctx)
}

// cleanupStoppedContainers cleans up stopped containers
func (cm *ContainerManager) cleanupStoppedContainers(ctx context.Context) {
	// List stopped containers
	containers, err := cm.client.ContainerList(ctx, types.ContainerListOptions{
		All: true,
		Filters: filters.NewArgs(
			filters.Arg("label", "managed-by=deployment-agent"),
			filters.Arg("status", "exited"),
		),
	})
	if err != nil {
		logrus.Errorf("Failed to list stopped containers: %v", err)
		return
	}

	cutoff := time.Now().Add(-cm.config.Docker.CleanupRetention)

	for _, container := range containers {
		// Check if container is old enough to be cleaned up
		if container.Created < cutoff.Unix() {
			if err := cm.client.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
				Force:         true,
				RemoveVolumes: true,
			}); err != nil {
				logrus.Errorf("Failed to remove container %s: %v", container.ID, err)
			} else {
				logrus.Infof("Cleaned up stopped container: %s", container.ID)
				
				// Remove from cache
				cm.mu.Lock()
				delete(cm.containers, container.ID)
				cm.mu.Unlock()
			}
		}
	}
}

// cleanupUnusedImages cleans up unused images
func (cm *ContainerManager) cleanupUnusedImages(ctx context.Context) {
	// Prune unused images
	report, err := cm.client.ImagesPrune(ctx, filters.NewArgs())
	if err != nil {
		logrus.Errorf("Failed to prune images: %v", err)
		return
	}

	if report.SpaceReclaimed > 0 {
		logrus.Infof("Cleaned up unused images, reclaimed %d bytes", report.SpaceReclaimed)
	}
}

// cleanupUnusedVolumes cleans up unused volumes
func (cm *ContainerManager) cleanupUnusedVolumes(ctx context.Context) {
	// Prune unused volumes
	report, err := cm.client.VolumesPrune(ctx, filters.NewArgs())
	if err != nil {
		logrus.Errorf("Failed to prune volumes: %v", err)
		return
	}

	if report.SpaceReclaimed > 0 {
		logrus.Infof("Cleaned up unused volumes, reclaimed %d bytes", report.SpaceReclaimed)
	}
}

// Close closes the container manager
func (cm *ContainerManager) Close() error {
	cm.cancel()
	cm.wg.Wait()

	if cm.client != nil {
		return cm.client.Close()
	}

	return nil
}

// generateRandomString generates a random string
func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}