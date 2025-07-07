package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"sync"
	"time"

	"superagent/internal/logging"

	"github.com/sirupsen/logrus"
)

// DockerManager handles Docker operations
type DockerManager struct {
	auditLogger *logging.AuditLogger
	mu          sync.RWMutex
}

// BuildContext contains build configuration
type BuildContext struct {
	ContextPath  string            `json:"context_path"`
	Dockerfile   string            `json:"dockerfile"`
	BuildPath    string            `json:"build_path"`
	ImageTag     string            `json:"image_tag"`
	BuildArgs    map[string]string `json:"build_args"`
	Labels       map[string]string `json:"labels"`
	NoCache      bool              `json:"no_cache"`
	Pull         bool              `json:"pull"`
	Target       string            `json:"target"`
	Platform     string            `json:"platform"`
	NetworkMode  string            `json:"network_mode"`
	Isolation    string            `json:"isolation"`
	ShmSize      string            `json:"shm_size"`
	Ulimits      []string          `json:"ulimits"`
	CacheFrom    []string          `json:"cache_from"`
	ExtraHosts   []string          `json:"extra_hosts"`
	Squash       bool              `json:"squash"`
	Compress     bool              `json:"compress"`
	SecurityOpt  []string          `json:"security_opt"`
}

// ContainerConfig contains container configuration
type ContainerConfig struct {
	Image           string            `json:"image"`
	Name            string            `json:"name"`
	Command         []string          `json:"command"`
	Args            []string          `json:"args"`
	Environment     map[string]string `json:"environment"`
	Ports           []PortMapping     `json:"ports"`
	Volumes         []VolumeMapping   `json:"volumes"`
	Networks        []string          `json:"networks"`
	Labels          map[string]string `json:"labels"`
	WorkingDir      string            `json:"working_dir"`
	User            string            `json:"user"`
	Hostname        string            `json:"hostname"`
	Domainname      string            `json:"domainname"`
	MacAddress      string            `json:"mac_address"`
	Privileged      bool              `json:"privileged"`
	ReadOnlyRootFS  bool              `json:"read_only_root_fs"`
	RestartPolicy   string            `json:"restart_policy"`
	StopSignal      string            `json:"stop_signal"`
	StopTimeout     int               `json:"stop_timeout"`
	Init            bool              `json:"init"`
	Tty             bool              `json:"tty"`
	OpenStdin       bool              `json:"open_stdin"`
	StdinOnce       bool              `json:"stdin_once"`
	AttachStdout    bool              `json:"attach_stdout"`
	AttachStderr    bool              `json:"attach_stderr"`
	ResourceLimits  ResourceLimits    `json:"resource_limits"`
	SecurityOpts    []string          `json:"security_opts"`
	DNSOptions      []string          `json:"dns_options"`
	ExtraHosts      []string          `json:"extra_hosts"`
	LogDriver       string            `json:"log_driver"`
	LogOptions      map[string]string `json:"log_options"`
	Ulimits         []string          `json:"ulimits"`
	ShmSize         string            `json:"shm_size"`
	Runtime         string            `json:"runtime"`
	Isolation       string            `json:"isolation"`
	CgroupParent    string            `json:"cgroup_parent"`
	AutoRemove      bool              `json:"auto_remove"`
	ReadOnlyTmpfs   []string          `json:"read_only_tmpfs"`
	Tmpfs           []string          `json:"tmpfs"`
	StorageOpt      map[string]string `json:"storage_opt"`
	Sysctls         map[string]string `json:"sysctls"`
	GroupAdd        []string          `json:"group_add"`
	PidMode         string            `json:"pid_mode"`
	UTSMode         string            `json:"uts_mode"`
	UsernsMode      string            `json:"userns_mode"`
	IPCMode         string            `json:"ipc_mode"`
	PublishAll      bool              `json:"publish_all"`
	Entrypoint      []string          `json:"entrypoint"`
	OnBuild         []string          `json:"on_build"`
	HealthCheck     *HealthCheckConfig `json:"health_check"`
}

// PortMapping defines port mapping
type PortMapping struct {
	ContainerPort int    `json:"container_port"`
	HostPort      int    `json:"host_port"`
	Protocol      string `json:"protocol"`
	HostIP        string `json:"host_ip"`
}

// VolumeMapping defines volume mapping
type VolumeMapping struct {
	Source      string   `json:"source"`
	Target      string   `json:"target"`
	Type        string   `json:"type"`
	ReadOnly    bool     `json:"read_only"`
	Consistency string   `json:"consistency"`
	Options     []string `json:"options"`
}

// ResourceLimits defines resource constraints
type ResourceLimits struct {
	CPULimit      float64 `json:"cpu_limit"`
	MemoryLimit   int64   `json:"memory_limit"`
	SwapLimit     int64   `json:"swap_limit"`
	DiskLimit     int64   `json:"disk_limit"`
	NetworkLimit  int64   `json:"network_limit"`
	ProcessLimit  int     `json:"process_limit"`
	FileLimit     int     `json:"file_limit"`
	CPUShares     int     `json:"cpu_shares"`
	CPUPeriod     int     `json:"cpu_period"`
	CPUQuota      int     `json:"cpu_quota"`
	CPUSetCPUs    string  `json:"cpu_set_cpus"`
	CPUSetMems    string  `json:"cpu_set_mems"`
	BlkioWeight   int     `json:"blkio_weight"`
	MemorySwap    int64   `json:"memory_swap"`
	MemoryReservation int64 `json:"memory_reservation"`
	KernelMemory  int64   `json:"kernel_memory"`
	OomKillDisable bool   `json:"oom_kill_disable"`
	OomScoreAdj   int     `json:"oom_score_adj"`
	ShmSize       string  `json:"shm_size"`
	Ulimits       []string `json:"ulimits"`
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	Test        []string      `json:"test"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	Retries     int           `json:"retries"`
	StartPeriod time.Duration `json:"start_period"`
}

// ContainerStats contains container statistics
type ContainerStats struct {
	ContainerID  string    `json:"container_id"`
	Name         string    `json:"name"`
	CPUUsage     float64   `json:"cpu_usage"`
	MemoryUsage  int64     `json:"memory_usage"`
	MemoryLimit  int64     `json:"memory_limit"`
	NetworkRx    int64     `json:"network_rx"`
	NetworkTx    int64     `json:"network_tx"`
	DiskUsage    int64     `json:"disk_usage"`
	RestartCount int       `json:"restart_count"`
	Status       string    `json:"status"`
	State        string    `json:"state"`
	ExitCode     int       `json:"exit_code"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ContainerInfo contains container information
type ContainerInfo struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Image       string                 `json:"image"`
	ImageID     string                 `json:"image_id"`
	Command     string                 `json:"command"`
	Status      string                 `json:"status"`
	State       string                 `json:"state"`
	Ports       []PortMapping          `json:"ports"`
	Labels      map[string]string      `json:"labels"`
	Mounts      []VolumeMapping        `json:"mounts"`
	Networks    map[string]interface{} `json:"networks"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   time.Time              `json:"started_at"`
	FinishedAt  time.Time              `json:"finished_at"`
	RestartCount int                   `json:"restart_count"`
	ExitCode    int                    `json:"exit_code"`
	Platform    string                 `json:"platform"`
	Architecture string                `json:"architecture"`
	Size        int64                  `json:"size"`
	VirtualSize int64                  `json:"virtual_size"`
}

// ImageInfo contains image information
type ImageInfo struct {
	ID          string            `json:"id"`
	Repository  string            `json:"repository"`
	Tag         string            `json:"tag"`
	Digest      string            `json:"digest"`
	CreatedAt   time.Time         `json:"created_at"`
	Size        int64             `json:"size"`
	VirtualSize int64             `json:"virtual_size"`
	Labels      map[string]string `json:"labels"`
	Architecture string           `json:"architecture"`
	OS          string            `json:"os"`
	Author      string            `json:"author"`
	Comment     string            `json:"comment"`
	Config      map[string]interface{} `json:"config"`
	RootFS      map[string]interface{} `json:"root_fs"`
	History     []map[string]interface{} `json:"history"`
	Layers      []string          `json:"layers"`
}

// LogCallback is a callback function for build/pull logs
type LogCallback func(string)

// NewDockerManager creates a new Docker manager
func NewDockerManager(auditLogger *logging.AuditLogger) (*DockerManager, error) {
	// Check if Docker is available
	if err := checkDockerAvailable(); err != nil {
		return nil, fmt.Errorf("Docker is not available: %w", err)
	}

	dm := &DockerManager{
		auditLogger: auditLogger,
	}

	auditLogger.LogEvent("DOCKER_MANAGER_INITIALIZED", map[string]interface{}{})

	return dm, nil
}

// BuildImage builds a Docker image
func (dm *DockerManager) BuildImage(ctx context.Context, buildContext BuildContext, logCallback LogCallback) (string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	logrus.Infof("Building Docker image: %s", buildContext.ImageTag)

	// Prepare build command
	cmd := exec.CommandContext(ctx, "docker", "build")

	// Add build arguments
	if buildContext.Dockerfile != "" {
		cmd.Args = append(cmd.Args, "-f", buildContext.Dockerfile)
	}

	// Add image tag
	if buildContext.ImageTag != "" {
		cmd.Args = append(cmd.Args, "-t", buildContext.ImageTag)
	}

	// Add build args
	for key, value := range buildContext.BuildArgs {
		cmd.Args = append(cmd.Args, "--build-arg", fmt.Sprintf("%s=%s", key, value))
	}

	// Add labels
	for key, value := range buildContext.Labels {
		cmd.Args = append(cmd.Args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add additional options
	if buildContext.NoCache {
		cmd.Args = append(cmd.Args, "--no-cache")
	}

	if buildContext.Pull {
		cmd.Args = append(cmd.Args, "--pull")
	}

	if buildContext.Target != "" {
		cmd.Args = append(cmd.Args, "--target", buildContext.Target)
	}

	if buildContext.Platform != "" {
		cmd.Args = append(cmd.Args, "--platform", buildContext.Platform)
	}

	if buildContext.NetworkMode != "" {
		cmd.Args = append(cmd.Args, "--network", buildContext.NetworkMode)
	}

	if buildContext.Isolation != "" {
		cmd.Args = append(cmd.Args, "--isolation", buildContext.Isolation)
	}

	if buildContext.ShmSize != "" {
		cmd.Args = append(cmd.Args, "--shm-size", buildContext.ShmSize)
	}

	// Add ulimits
	for _, ulimit := range buildContext.Ulimits {
		cmd.Args = append(cmd.Args, "--ulimit", ulimit)
	}

	// Add cache from
	for _, cacheFrom := range buildContext.CacheFrom {
		cmd.Args = append(cmd.Args, "--cache-from", cacheFrom)
	}

	// Add extra hosts
	for _, extraHost := range buildContext.ExtraHosts {
		cmd.Args = append(cmd.Args, "--add-host", extraHost)
	}

	// Add security options
	for _, securityOpt := range buildContext.SecurityOpt {
		cmd.Args = append(cmd.Args, "--security-opt", securityOpt)
	}

	if buildContext.Squash {
		cmd.Args = append(cmd.Args, "--squash")
	}

	if buildContext.Compress {
		cmd.Args = append(cmd.Args, "--compress")
	}

	// Add context path
	contextPath := buildContext.ContextPath
	if contextPath == "" {
		contextPath = "."
	}
	cmd.Args = append(cmd.Args, contextPath)

	// Set up stdout/stderr capture
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start build command: %w", err)
	}

	// Read logs
	go dm.readLogs(stdout, logCallback)
	go dm.readLogs(stderr, logCallback)

	// Wait for completion
	err = cmd.Wait()
	if err != nil {
		dm.auditLogger.LogEvent("DOCKER_BUILD_FAILED", map[string]interface{}{
			"image_tag": buildContext.ImageTag,
			"error":     err.Error(),
		})
		return "", fmt.Errorf("build failed: %w", err)
	}

	// Get the built image ID
	imageID, err := dm.getImageID(ctx, buildContext.ImageTag)
	if err != nil {
		return "", fmt.Errorf("failed to get image ID: %w", err)
	}

	dm.auditLogger.LogEvent("DOCKER_BUILD_SUCCESS", map[string]interface{}{
		"image_tag": buildContext.ImageTag,
		"image_id":  imageID,
	})

	return imageID, nil
}

// PullImage pulls a Docker image
func (dm *DockerManager) PullImage(ctx context.Context, imageName string, auth map[string]string, logCallback LogCallback) (string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	logrus.Infof("Pulling Docker image: %s", imageName)

	// Prepare pull command
	cmd := exec.CommandContext(ctx, "docker", "pull", imageName)

	// Set up authentication if provided
	if err := dm.setupAuth(cmd, auth); err != nil {
		return "", fmt.Errorf("failed to setup authentication: %w", err)
	}

	// Set up stdout/stderr capture
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stderr pipe: %w", err)
	}

	// Start the command
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start pull command: %w", err)
	}

	// Read logs
	go dm.readLogs(stdout, logCallback)
	go dm.readLogs(stderr, logCallback)

	// Wait for completion
	err = cmd.Wait()
	if err != nil {
		dm.auditLogger.LogEvent("DOCKER_PULL_FAILED", map[string]interface{}{
			"image_name": imageName,
			"error":      err.Error(),
		})
		return "", fmt.Errorf("pull failed: %w", err)
	}

	// Get the pulled image ID
	imageID, err := dm.getImageID(ctx, imageName)
	if err != nil {
		return "", fmt.Errorf("failed to get image ID: %w", err)
	}

	dm.auditLogger.LogEvent("DOCKER_PULL_SUCCESS", map[string]interface{}{
		"image_name": imageName,
		"image_id":   imageID,
	})

	return imageID, nil
}

// CreateContainer creates a new container
func (dm *DockerManager) CreateContainer(ctx context.Context, config ContainerConfig) (string, error) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	logrus.Infof("Creating container: %s", config.Name)

	// Prepare create command
	cmd := exec.CommandContext(ctx, "docker", "create")

	// Add container name
	if config.Name != "" {
		cmd.Args = append(cmd.Args, "--name", config.Name)
	}

	// Add environment variables
	for key, value := range config.Environment {
		cmd.Args = append(cmd.Args, "-e", fmt.Sprintf("%s=%s", key, value))
	}

	// Add port mappings
	for _, port := range config.Ports {
		portMapping := fmt.Sprintf("%d:%d", port.HostPort, port.ContainerPort)
		if port.Protocol != "" {
			portMapping += "/" + port.Protocol
		}
		if port.HostIP != "" {
			portMapping = port.HostIP + ":" + portMapping
		}
		cmd.Args = append(cmd.Args, "-p", portMapping)
	}

	// Add volume mappings
	for _, volume := range config.Volumes {
		volumeMapping := fmt.Sprintf("%s:%s", volume.Source, volume.Target)
		if volume.ReadOnly {
			volumeMapping += ":ro"
		}
		cmd.Args = append(cmd.Args, "-v", volumeMapping)
	}

	// Add networks
	for _, network := range config.Networks {
		cmd.Args = append(cmd.Args, "--network", network)
	}

	// Add labels
	for key, value := range config.Labels {
		cmd.Args = append(cmd.Args, "--label", fmt.Sprintf("%s=%s", key, value))
	}

	// Add other options
	if config.WorkingDir != "" {
		cmd.Args = append(cmd.Args, "-w", config.WorkingDir)
	}

	if config.User != "" {
		cmd.Args = append(cmd.Args, "-u", config.User)
	}

	if config.Hostname != "" {
		cmd.Args = append(cmd.Args, "-h", config.Hostname)
	}

	if config.Privileged {
		cmd.Args = append(cmd.Args, "--privileged")
	}

	if config.ReadOnlyRootFS {
		cmd.Args = append(cmd.Args, "--read-only")
	}

	if config.RestartPolicy != "" {
		cmd.Args = append(cmd.Args, "--restart", config.RestartPolicy)
	}

	// Add resource limits
	if config.ResourceLimits.CPULimit > 0 {
		cmd.Args = append(cmd.Args, "--cpus", fmt.Sprintf("%.2f", config.ResourceLimits.CPULimit))
	}

	if config.ResourceLimits.MemoryLimit > 0 {
		cmd.Args = append(cmd.Args, "--memory", fmt.Sprintf("%d", config.ResourceLimits.MemoryLimit))
	}

	if config.ResourceLimits.SwapLimit > 0 {
		cmd.Args = append(cmd.Args, "--memory-swap", fmt.Sprintf("%d", config.ResourceLimits.SwapLimit))
	}

	if config.ResourceLimits.CPUShares > 0 {
		cmd.Args = append(cmd.Args, "--cpu-shares", fmt.Sprintf("%d", config.ResourceLimits.CPUShares))
	}

	if config.ResourceLimits.CPUSetCPUs != "" {
		cmd.Args = append(cmd.Args, "--cpuset-cpus", config.ResourceLimits.CPUSetCPUs)
	}

	if config.ResourceLimits.CPUSetMems != "" {
		cmd.Args = append(cmd.Args, "--cpuset-mems", config.ResourceLimits.CPUSetMems)
	}

	if config.ResourceLimits.ProcessLimit > 0 {
		cmd.Args = append(cmd.Args, "--pids-limit", fmt.Sprintf("%d", config.ResourceLimits.ProcessLimit))
	}

	// Add security options
	for _, securityOpt := range config.SecurityOpts {
		cmd.Args = append(cmd.Args, "--security-opt", securityOpt)
	}

	// Add ulimits
	for _, ulimit := range config.ResourceLimits.Ulimits {
		cmd.Args = append(cmd.Args, "--ulimit", ulimit)
	}

	// Add extra hosts
	for _, extraHost := range config.ExtraHosts {
		cmd.Args = append(cmd.Args, "--add-host", extraHost)
	}

	// Add DNS options
	for _, dnsOption := range config.DNSOptions {
		cmd.Args = append(cmd.Args, "--dns-option", dnsOption)
	}

	// Add log driver
	if config.LogDriver != "" {
		cmd.Args = append(cmd.Args, "--log-driver", config.LogDriver)
	}

	// Add log options
	for key, value := range config.LogOptions {
		cmd.Args = append(cmd.Args, "--log-opt", fmt.Sprintf("%s=%s", key, value))
	}

	// Add health check
	if config.HealthCheck != nil {
		if len(config.HealthCheck.Test) > 0 {
			cmd.Args = append(cmd.Args, "--health-cmd", strings.Join(config.HealthCheck.Test, " "))
		}
		if config.HealthCheck.Interval > 0 {
			cmd.Args = append(cmd.Args, "--health-interval", config.HealthCheck.Interval.String())
		}
		if config.HealthCheck.Timeout > 0 {
			cmd.Args = append(cmd.Args, "--health-timeout", config.HealthCheck.Timeout.String())
		}
		if config.HealthCheck.Retries > 0 {
			cmd.Args = append(cmd.Args, "--health-retries", fmt.Sprintf("%d", config.HealthCheck.Retries))
		}
		if config.HealthCheck.StartPeriod > 0 {
			cmd.Args = append(cmd.Args, "--health-start-period", config.HealthCheck.StartPeriod.String())
		}
	}

	// Add image
	cmd.Args = append(cmd.Args, config.Image)

	// Add command and args
	if len(config.Command) > 0 {
		cmd.Args = append(cmd.Args, config.Command...)
	}
	if len(config.Args) > 0 {
		cmd.Args = append(cmd.Args, config.Args...)
	}

	// Execute command
	output, err := cmd.CombinedOutput()
	if err != nil {
		dm.auditLogger.LogEvent("DOCKER_CREATE_FAILED", map[string]interface{}{
			"container_name": config.Name,
			"image":          config.Image,
			"error":          err.Error(),
			"output":         string(output),
		})
		return "", fmt.Errorf("failed to create container: %w, output: %s", err, output)
	}

	containerID := strings.TrimSpace(string(output))

	dm.auditLogger.LogEvent("DOCKER_CREATE_SUCCESS", map[string]interface{}{
		"container_name": config.Name,
		"container_id":   containerID,
		"image":          config.Image,
	})

	return containerID, nil
}

// StartContainer starts a container
func (dm *DockerManager) StartContainer(ctx context.Context, containerID string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	logrus.Infof("Starting container: %s", containerID)

	cmd := exec.CommandContext(ctx, "docker", "start", containerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		dm.auditLogger.LogEvent("DOCKER_START_FAILED", map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
			"output":       string(output),
		})
		return fmt.Errorf("failed to start container: %w, output: %s", err, output)
	}

	dm.auditLogger.LogEvent("DOCKER_START_SUCCESS", map[string]interface{}{
		"container_id": containerID,
	})

	return nil
}

// StopContainer stops a container
func (dm *DockerManager) StopContainer(ctx context.Context, containerID string, timeout int) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	logrus.Infof("Stopping container: %s", containerID)

	cmd := exec.CommandContext(ctx, "docker", "stop")
	if timeout > 0 {
		cmd.Args = append(cmd.Args, "-t", fmt.Sprintf("%d", timeout))
	}
	cmd.Args = append(cmd.Args, containerID)

	output, err := cmd.CombinedOutput()
	if err != nil {
		dm.auditLogger.LogEvent("DOCKER_STOP_FAILED", map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
			"output":       string(output),
		})
		return fmt.Errorf("failed to stop container: %w, output: %s", err, output)
	}

	dm.auditLogger.LogEvent("DOCKER_STOP_SUCCESS", map[string]interface{}{
		"container_id": containerID,
	})

	return nil
}

// RemoveContainer removes a container
func (dm *DockerManager) RemoveContainer(ctx context.Context, containerID string, force bool) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	logrus.Infof("Removing container: %s", containerID)

	cmd := exec.CommandContext(ctx, "docker", "rm")
	if force {
		cmd.Args = append(cmd.Args, "-f")
	}
	cmd.Args = append(cmd.Args, containerID)

	output, err := cmd.CombinedOutput()
	if err != nil {
		dm.auditLogger.LogEvent("DOCKER_REMOVE_FAILED", map[string]interface{}{
			"container_id": containerID,
			"error":        err.Error(),
			"output":       string(output),
		})
		return fmt.Errorf("failed to remove container: %w, output: %s", err, output)
	}

	dm.auditLogger.LogEvent("DOCKER_REMOVE_SUCCESS", map[string]interface{}{
		"container_id": containerID,
	})

	return nil
}

// GetContainerStats gets container statistics
func (dm *DockerManager) GetContainerStats(ctx context.Context, containerID string) (*ContainerStats, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	cmd := exec.CommandContext(ctx, "docker", "stats", containerID, "--no-stream", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get container stats: %w", err)
	}

	var rawStats map[string]interface{}
	if err := json.Unmarshal(output, &rawStats); err != nil {
		return nil, fmt.Errorf("failed to parse stats JSON: %w", err)
	}

	stats := &ContainerStats{
		ContainerID: containerID,
		UpdatedAt:   time.Now(),
	}

	// Parse CPU usage
	if cpuPerc, ok := rawStats["CPUPerc"].(string); ok {
		if cpu, err := strconv.ParseFloat(strings.TrimSuffix(cpuPerc, "%"), 64); err == nil {
			stats.CPUUsage = cpu
		}
	}

	// Parse memory usage
	if memUsage, ok := rawStats["MemUsage"].(string); ok {
		parts := strings.Split(memUsage, " / ")
		if len(parts) == 2 {
			if usage, err := parseMemorySize(parts[0]); err == nil {
				stats.MemoryUsage = usage
			}
			if limit, err := parseMemorySize(parts[1]); err == nil {
				stats.MemoryLimit = limit
			}
		}
	}

	// Parse network I/O
	if netIO, ok := rawStats["NetIO"].(string); ok {
		parts := strings.Split(netIO, " / ")
		if len(parts) == 2 {
			if rx, err := parseMemorySize(parts[0]); err == nil {
				stats.NetworkRx = rx
			}
			if tx, err := parseMemorySize(parts[1]); err == nil {
				stats.NetworkTx = tx
			}
		}
	}

	return stats, nil
}

// GetContainerInfo gets container information
func (dm *DockerManager) GetContainerInfo(ctx context.Context, containerID string) (*ContainerInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	cmd := exec.CommandContext(ctx, "docker", "inspect", containerID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container: %w", err)
	}

	var rawInfo []map[string]interface{}
	if err := json.Unmarshal(output, &rawInfo); err != nil {
		return nil, fmt.Errorf("failed to parse inspect JSON: %w", err)
	}

	if len(rawInfo) == 0 {
		return nil, fmt.Errorf("no container info found")
	}

	containerData := rawInfo[0]
	info := &ContainerInfo{
		ID: containerID,
	}

	// Parse basic info
	if name, ok := containerData["Name"].(string); ok {
		info.Name = strings.TrimPrefix(name, "/")
	}

	if config, ok := containerData["Config"].(map[string]interface{}); ok {
		if image, ok := config["Image"].(string); ok {
			info.Image = image
		}
		if cmd, ok := config["Cmd"].([]interface{}); ok {
			if len(cmd) > 0 {
				info.Command = fmt.Sprintf("%v", cmd)
			}
		}
		if labels, ok := config["Labels"].(map[string]interface{}); ok {
			info.Labels = make(map[string]string)
			for k, v := range labels {
				if str, ok := v.(string); ok {
					info.Labels[k] = str
				}
			}
		}
	}

	if state, ok := containerData["State"].(map[string]interface{}); ok {
		if status, ok := state["Status"].(string); ok {
			info.Status = status
		}
		if running, ok := state["Running"].(bool); ok {
			if running {
				info.State = "running"
			} else {
				info.State = "stopped"
			}
		}
		if exitCode, ok := state["ExitCode"].(float64); ok {
			info.ExitCode = int(exitCode)
		}
		if startedAt, ok := state["StartedAt"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, startedAt); err == nil {
				info.StartedAt = t
			}
		}
		if finishedAt, ok := state["FinishedAt"].(string); ok {
			if t, err := time.Parse(time.RFC3339Nano, finishedAt); err == nil {
				info.FinishedAt = t
			}
		}
	}

	if created, ok := containerData["Created"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
			info.CreatedAt = t
		}
	}

	return info, nil
}

// ListContainers lists containers
func (dm *DockerManager) ListContainers(ctx context.Context, all bool) ([]*ContainerInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	cmd := exec.CommandContext(ctx, "docker", "ps", "--format", "json")
	if all {
		cmd.Args = append(cmd.Args, "-a")
	}

	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var containers []*ContainerInfo
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var rawContainer map[string]interface{}
		if err := json.Unmarshal([]byte(line), &rawContainer); err != nil {
			continue
		}

		container := &ContainerInfo{}
		if id, ok := rawContainer["ID"].(string); ok {
			container.ID = id
		}
		if name, ok := rawContainer["Names"].(string); ok {
			container.Name = name
		}
		if image, ok := rawContainer["Image"].(string); ok {
			container.Image = image
		}
		if command, ok := rawContainer["Command"].(string); ok {
			container.Command = command
		}
		if status, ok := rawContainer["Status"].(string); ok {
			container.Status = status
		}

		containers = append(containers, container)
	}

	return containers, nil
}

// Helper functions

// checkDockerAvailable checks if Docker is available
func checkDockerAvailable() error {
	cmd := exec.Command("docker", "version")
	return cmd.Run()
}

// setupAuth sets up authentication for Docker commands
func (dm *DockerManager) setupAuth(cmd *exec.Cmd, auth map[string]string) error {
	if auth == nil {
		return nil
	}

	// For Docker registry authentication, we would typically use docker login
	// or set up credential helpers. This is a simplified version.
	if username, ok := auth["username"]; ok {
		if password, ok := auth["password"]; ok {
			// Use docker login (this is synchronous and may not be ideal for all scenarios)
			loginCmd := exec.Command("docker", "login", "-u", username, "-p", password)
			if err := loginCmd.Run(); err != nil {
				return fmt.Errorf("failed to login to Docker registry: %w", err)
			}
		}
	}

	return nil
}

// readLogs reads logs from a pipe and calls the callback
func (dm *DockerManager) readLogs(pipe io.ReadCloser, callback LogCallback) {
	defer pipe.Close()

	scanner := bufio.NewScanner(pipe)
	for scanner.Scan() {
		line := scanner.Text()
		if callback != nil {
			callback(line)
		}
	}
}

// getImageID gets the ID of an image by name/tag
func (dm *DockerManager) getImageID(ctx context.Context, imageName string) (string, error) {
	cmd := exec.CommandContext(ctx, "docker", "images", imageName, "--format", "{{.ID}}")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get image ID: %w", err)
	}

	imageID := strings.TrimSpace(string(output))
	if imageID == "" {
		return "", fmt.Errorf("image not found: %s", imageName)
	}

	return imageID, nil
}

// parseMemorySize parses memory size string (e.g., "1.5GB", "512MB")
func parseMemorySize(sizeStr string) (int64, error) {
	sizeStr = strings.TrimSpace(sizeStr)
	if sizeStr == "" {
		return 0, nil
	}

	// Remove units and parse
	sizeStr = strings.ToUpper(sizeStr)
	
	var multiplier int64 = 1
	if strings.HasSuffix(sizeStr, "KB") {
		multiplier = 1024
		sizeStr = strings.TrimSuffix(sizeStr, "KB")
	} else if strings.HasSuffix(sizeStr, "MB") {
		multiplier = 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "MB")
	} else if strings.HasSuffix(sizeStr, "GB") {
		multiplier = 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "GB")
	} else if strings.HasSuffix(sizeStr, "TB") {
		multiplier = 1024 * 1024 * 1024 * 1024
		sizeStr = strings.TrimSuffix(sizeStr, "TB")
	} else if strings.HasSuffix(sizeStr, "B") {
		sizeStr = strings.TrimSuffix(sizeStr, "B")
	}

	size, err := strconv.ParseFloat(sizeStr, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse size: %w", err)
	}

	return int64(size * float64(multiplier)), nil
}

// RemoveImage removes a Docker image
func (dm *DockerManager) RemoveImage(ctx context.Context, imageID string, force bool) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	logrus.Infof("Removing image: %s", imageID)

	cmd := exec.CommandContext(ctx, "docker", "rmi")
	if force {
		cmd.Args = append(cmd.Args, "-f")
	}
	cmd.Args = append(cmd.Args, imageID)

	output, err := cmd.CombinedOutput()
	if err != nil {
		dm.auditLogger.LogEvent("DOCKER_IMAGE_REMOVE_FAILED", map[string]interface{}{
			"image_id": imageID,
			"error":    err.Error(),
			"output":   string(output),
		})
		return fmt.Errorf("failed to remove image: %w, output: %s", err, output)
	}

	dm.auditLogger.LogEvent("DOCKER_IMAGE_REMOVE_SUCCESS", map[string]interface{}{
		"image_id": imageID,
	})

	return nil
}

// GetImageInfo gets information about an image
func (dm *DockerManager) GetImageInfo(ctx context.Context, imageID string) (*ImageInfo, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	cmd := exec.CommandContext(ctx, "docker", "inspect", imageID)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to inspect image: %w", err)
	}

	var rawInfo []map[string]interface{}
	if err := json.Unmarshal(output, &rawInfo); err != nil {
		return nil, fmt.Errorf("failed to parse inspect JSON: %w", err)
	}

	if len(rawInfo) == 0 {
		return nil, fmt.Errorf("no image info found")
	}

	imageData := rawInfo[0]
	info := &ImageInfo{
		ID: imageID,
	}

	// Parse basic info
	if repoTags, ok := imageData["RepoTags"].([]interface{}); ok && len(repoTags) > 0 {
		if repoTag, ok := repoTags[0].(string); ok {
			parts := strings.Split(repoTag, ":")
			if len(parts) >= 2 {
				info.Repository = strings.Join(parts[:len(parts)-1], ":")
				info.Tag = parts[len(parts)-1]
			} else {
				info.Repository = repoTag
				info.Tag = "latest"
			}
		}
	}

	if created, ok := imageData["Created"].(string); ok {
		if t, err := time.Parse(time.RFC3339Nano, created); err == nil {
			info.CreatedAt = t
		}
	}

	if size, ok := imageData["Size"].(float64); ok {
		info.Size = int64(size)
	}

	if virtualSize, ok := imageData["VirtualSize"].(float64); ok {
		info.VirtualSize = int64(virtualSize)
	}

	if config, ok := imageData["Config"].(map[string]interface{}); ok {
		info.Config = config
		if labels, ok := config["Labels"].(map[string]interface{}); ok {
			info.Labels = make(map[string]string)
			for k, v := range labels {
				if str, ok := v.(string); ok {
					info.Labels[k] = str
				}
			}
		}
	}

	if rootFS, ok := imageData["RootFS"].(map[string]interface{}); ok {
		info.RootFS = rootFS
	}

	return info, nil
}

// PruneImages removes unused images
func (dm *DockerManager) PruneImages(ctx context.Context, dangling bool) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	logrus.Info("Pruning unused images")

	cmd := exec.CommandContext(ctx, "docker", "image", "prune", "-f")
	if !dangling {
		cmd.Args = append(cmd.Args, "-a")
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		dm.auditLogger.LogEvent("DOCKER_IMAGE_PRUNE_FAILED", map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
		return fmt.Errorf("failed to prune images: %w, output: %s", err, output)
	}

	dm.auditLogger.LogEvent("DOCKER_IMAGE_PRUNE_SUCCESS", map[string]interface{}{
		"output": string(output),
	})

	return nil
}

// PruneContainers removes stopped containers
func (dm *DockerManager) PruneContainers(ctx context.Context) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	logrus.Info("Pruning stopped containers")

	cmd := exec.CommandContext(ctx, "docker", "container", "prune", "-f")
	output, err := cmd.CombinedOutput()
	if err != nil {
		dm.auditLogger.LogEvent("DOCKER_CONTAINER_PRUNE_FAILED", map[string]interface{}{
			"error":  err.Error(),
			"output": string(output),
		})
		return fmt.Errorf("failed to prune containers: %w, output: %s", err, output)
	}

	dm.auditLogger.LogEvent("DOCKER_CONTAINER_PRUNE_SUCCESS", map[string]interface{}{
		"output": string(output),
	})

	return nil
}

// ExecuteInContainer executes a command in a running container
func (dm *DockerManager) ExecuteInContainer(ctx context.Context, containerID string, command []string) (string, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	cmd := exec.CommandContext(ctx, "docker", "exec", containerID)
	cmd.Args = append(cmd.Args, command...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("failed to execute command in container: %w", err)
	}

	return string(output), nil
}

// GetContainerLogs gets logs from a container
func (dm *DockerManager) GetContainerLogs(ctx context.Context, containerID string, follow bool, tail int) (io.ReadCloser, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	cmd := exec.CommandContext(ctx, "docker", "logs", containerID)
	if follow {
		cmd.Args = append(cmd.Args, "-f")
	}
	if tail > 0 {
		cmd.Args = append(cmd.Args, "--tail", fmt.Sprintf("%d", tail))
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start logs command: %w", err)
	}

	return stdout, nil
}