package agent

import (
	"context"
	"fmt"
	"sync"
	"time"

	"deployment-agent/internal/api"
	"deployment-agent/internal/config"
	"deployment-agent/internal/docker"
	"deployment-agent/internal/git"
	"deployment-agent/internal/logging"

	"github.com/sirupsen/logrus"
)

// Agent represents the main deployment agent
type Agent struct {
	config            *config.Config
	auditLogger       *logging.AuditLogger
	logStreamer       *logging.LogStreamer
	backendClient     *api.BackendClient
	containerManager  *docker.ContainerManager
	gitManager        *git.GitManager
	commandQueue      chan *api.DeploymentCommand
	activeCommands    map[string]*CommandExecution
	mu                sync.RWMutex
	ctx               context.Context
	cancel            context.CancelFunc
	wg                sync.WaitGroup
	startTime         time.Time
}

// CommandExecution represents an executing command
type CommandExecution struct {
	Command   *api.DeploymentCommand
	StartTime time.Time
	Status    string
	Error     error
	Result    map[string]interface{}
	Context   context.Context
	Cancel    context.CancelFunc
}

// New creates a new deployment agent
func New(cfg *config.Config, auditLogger *logging.AuditLogger) (*Agent, error) {
	logrus.Info("Creating deployment agent")

	// Create log streamer if enabled
	var logStreamer *logging.LogStreamer
	if cfg.Monitoring.LogStreamingEnabled && cfg.Monitoring.LogStreamingEndpoint != "" {
		logStreamer = logging.NewLogStreamer(cfg.Monitoring.LogStreamingEndpoint, cfg.Backend.APIToken)
	}

	// Create backend client
	backendClient, err := api.NewBackendClient(cfg, auditLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create backend client: %w", err)
	}

	// Create container manager
	containerManager, err := docker.NewContainerManager(cfg, auditLogger, logStreamer)
	if err != nil {
		return nil, fmt.Errorf("failed to create container manager: %w", err)
	}

	// Create Git manager
	gitManager, err := git.NewGitManager(cfg, auditLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to create Git manager: %w", err)
	}

	// Create context for lifecycle management
	ctx, cancel := context.WithCancel(context.Background())

	agent := &Agent{
		config:           cfg,
		auditLogger:      auditLogger,
		logStreamer:      logStreamer,
		backendClient:    backendClient,
		containerManager: containerManager,
		gitManager:       gitManager,
		commandQueue:     make(chan *api.DeploymentCommand, 100),
		activeCommands:   make(map[string]*CommandExecution),
		ctx:              ctx,
		cancel:           cancel,
		startTime:        time.Now(),
	}

	return agent, nil
}

// Start starts the deployment agent
func (a *Agent) Start(ctx context.Context) error {
	logrus.Info("Starting deployment agent")

	// Start backend client
	if err := a.backendClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start backend client: %w", err)
	}

	// Start command processor
	a.wg.Add(1)
	go a.processCommands()

	// Start command poller
	a.wg.Add(1)
	go a.pollCommands()

	// Start status reporter
	a.wg.Add(1)
	go a.reportStatus()

	// Start token rotation
	if a.config.Security.TokenRotationInterval > 0 {
		a.wg.Add(1)
		go a.rotateTokens()
	}

	a.auditLogger.LogEvent("AGENT_STARTED", map[string]interface{}{
		"start_time": a.startTime,
		"config":     a.config.Agent.ID,
	})

	logrus.Info("Deployment agent started successfully")

	// Wait for context cancellation
	<-a.ctx.Done()
	return nil
}

// Shutdown gracefully shuts down the agent
func (a *Agent) Shutdown(ctx context.Context) error {
	logrus.Info("Shutting down deployment agent")

	// Cancel internal context
	a.cancel()

	// Wait for all goroutines to finish with timeout
	done := make(chan struct{})
	go func() {
		a.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		logrus.Info("All goroutines stopped")
	case <-ctx.Done():
		logrus.Warn("Shutdown timeout reached")
	}

	// Close components
	if a.containerManager != nil {
		if err := a.containerManager.Close(); err != nil {
			logrus.Errorf("Failed to close container manager: %v", err)
		}
	}

	if a.gitManager != nil {
		if err := a.gitManager.Close(); err != nil {
			logrus.Errorf("Failed to close Git manager: %v", err)
		}
	}

	if a.backendClient != nil {
		if err := a.backendClient.Close(); err != nil {
			logrus.Errorf("Failed to close backend client: %v", err)
		}
	}

	if a.logStreamer != nil {
		if err := a.logStreamer.Close(); err != nil {
			logrus.Errorf("Failed to close log streamer: %v", err)
		}
	}

	a.auditLogger.LogEvent("AGENT_SHUTDOWN", map[string]interface{}{
		"uptime": time.Since(a.startTime),
	})

	logrus.Info("Deployment agent shutdown completed")
	return nil
}

// processCommands processes deployment commands from the queue
func (a *Agent) processCommands() {
	defer a.wg.Done()

	semaphore := make(chan struct{}, a.config.Agent.MaxConcurrentOps)

	for {
		select {
		case <-a.ctx.Done():
			return
		case command := <-a.commandQueue:
			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				// Process command in goroutine
				go func(cmd *api.DeploymentCommand) {
					defer func() { <-semaphore }()
					a.executeCommand(cmd)
				}(command)
			default:
				// Queue is full, send error response
				response := &api.CommandResponse{
					CommandID: command.ID,
					Status:    "rejected",
					Success:   false,
					Message:   "Agent is at maximum concurrent operations capacity",
					Error:     "capacity_exceeded",
					Timestamp: time.Now(),
				}
				if err := a.backendClient.SendCommandResponse(context.Background(), response); err != nil {
					logrus.Errorf("Failed to send rejection response: %v", err)
				}
			}
		}
	}
}

// executeCommand executes a deployment command
func (a *Agent) executeCommand(command *api.DeploymentCommand) {
	startTime := time.Now()
	logrus.Infof("Executing command: %s (%s)", command.ID, command.Action)

	// Create command execution context
	cmdCtx, cmdCancel := context.WithCancel(a.ctx)
	if command.Timeout > 0 {
		cmdCtx, cmdCancel = context.WithTimeout(a.ctx, command.Timeout)
	}
	defer cmdCancel()

	// Track command execution
	execution := &CommandExecution{
		Command:   command,
		StartTime: startTime,
		Status:    "running",
		Context:   cmdCtx,
		Cancel:    cmdCancel,
	}

	a.mu.Lock()
	a.activeCommands[command.ID] = execution
	a.mu.Unlock()

	defer func() {
		a.mu.Lock()
		delete(a.activeCommands, command.ID)
		a.mu.Unlock()
	}()

	// Send initial response
	response := &api.CommandResponse{
		CommandID: command.ID,
		Status:    "started",
		Success:   true,
		Message:   "Command execution started",
		Timestamp: time.Now(),
	}
	a.backendClient.SendCommandResponse(context.Background(), response)

	// Execute command based on type
	var err error
	var result map[string]interface{}

	switch command.Type {
	case "deployment":
		result, err = a.handleDeploymentCommand(cmdCtx, command)
	case "container":
		result, err = a.handleContainerCommand(cmdCtx, command)
	case "git":
		result, err = a.handleGitCommand(cmdCtx, command)
	case "system":
		result, err = a.handleSystemCommand(cmdCtx, command)
	default:
		err = fmt.Errorf("unsupported command type: %s", command.Type)
	}

	// Update execution status
	execution.Status = "completed"
	execution.Error = err
	execution.Result = result

	// Send final response
	finalResponse := &api.CommandResponse{
		CommandID: command.ID,
		Status:    "completed",
		Success:   err == nil,
		Data:      result,
		Duration:  time.Since(startTime),
		Timestamp: time.Now(),
	}

	if err != nil {
		finalResponse.Success = false
		finalResponse.Error = err.Error()
		finalResponse.Status = "failed"
	}

	if err := a.backendClient.SendCommandResponse(context.Background(), finalResponse); err != nil {
		logrus.Errorf("Failed to send final command response: %v", err)
	}

	a.auditLogger.LogDeploymentEvent(command.Action, command.Target, err == nil, map[string]interface{}{
		"command_id": command.ID,
		"type":       command.Type,
		"duration":   time.Since(startTime),
		"error":      err,
	})

	if err != nil {
		logrus.Errorf("Command execution failed: %s (%s): %v", command.ID, command.Action, err)
	} else {
		logrus.Infof("Command executed successfully: %s (%s)", command.ID, command.Action)
	}
}

// handleDeploymentCommand handles deployment commands
func (a *Agent) handleDeploymentCommand(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	switch command.Action {
	case "deploy":
		return a.deployApplication(ctx, command)
	case "update":
		return a.updateApplication(ctx, command)
	case "rollback":
		return a.rollbackApplication(ctx, command)
	case "scale":
		return a.scaleApplication(ctx, command)
	default:
		return nil, fmt.Errorf("unsupported deployment action: %s", command.Action)
	}
}

// handleContainerCommand handles container commands
func (a *Agent) handleContainerCommand(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	switch command.Action {
	case "start":
		return a.startContainer(ctx, command)
	case "stop":
		return a.stopContainer(ctx, command)
	case "restart":
		return a.restartContainer(ctx, command)
	case "delete":
		return a.deleteContainer(ctx, command)
	case "logs":
		return a.getContainerLogs(ctx, command)
	case "stats":
		return a.getContainerStats(ctx, command)
	default:
		return nil, fmt.Errorf("unsupported container action: %s", command.Action)
	}
}

// handleGitCommand handles Git commands
func (a *Agent) handleGitCommand(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	switch command.Action {
	case "clone":
		return a.cloneRepository(ctx, command)
	case "pull":
		return a.pullRepository(ctx, command)
	case "build":
		return a.buildRepository(ctx, command)
	default:
		return nil, fmt.Errorf("unsupported Git action: %s", command.Action)
	}
}

// handleSystemCommand handles system commands
func (a *Agent) handleSystemCommand(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	switch command.Action {
	case "status":
		return a.getSystemStatus(ctx, command)
	case "health":
		return a.getHealthStatus(ctx, command)
	case "cleanup":
		return a.performCleanup(ctx, command)
	case "update":
		return a.updateAgent(ctx, command)
	default:
		return nil, fmt.Errorf("unsupported system action: %s", command.Action)
	}
}

// deployApplication deploys a new application
func (a *Agent) deployApplication(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	// Extract deployment specification from command
	spec, err := a.parseDeploymentSpec(command.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment spec: %w", err)
	}

	// If deploying from Git repository
	if spec.Source == "git" {
		repoInfo, err := a.cloneAndBuildRepository(ctx, spec)
		if err != nil {
			return nil, fmt.Errorf("failed to clone and build repository: %w", err)
		}
		spec.Image = repoInfo.Image // Use built image
	}

	// Deploy container
	containerInfo, err := a.containerManager.DeployContainer(ctx, spec.ContainerSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy container: %w", err)
	}

	return map[string]interface{}{
		"container_id": containerInfo.ID,
		"name":         containerInfo.Name,
		"image":        containerInfo.Image,
		"status":       containerInfo.Status,
		"ports":        containerInfo.Ports,
	}, nil
}

// updateApplication updates an existing application
func (a *Agent) updateApplication(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	containerID := command.Target
	if containerID == "" {
		return nil, fmt.Errorf("container ID is required for update")
	}

	// Extract deployment specification
	spec, err := a.parseDeploymentSpec(command.Spec)
	if err != nil {
		return nil, fmt.Errorf("failed to parse deployment spec: %w", err)
	}

	// Update container with zero-downtime
	containerInfo, err := a.containerManager.UpdateContainer(ctx, containerID, spec.ContainerSpec)
	if err != nil {
		return nil, fmt.Errorf("failed to update container: %w", err)
	}

	return map[string]interface{}{
		"old_container_id": containerID,
		"new_container_id": containerInfo.ID,
		"name":             containerInfo.Name,
		"image":            containerInfo.Image,
		"status":           containerInfo.Status,
	}, nil
}

// rollbackApplication rolls back an application to a previous version
func (a *Agent) rollbackApplication(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	// TODO: Implement rollback logic
	return map[string]interface{}{
		"message": "Rollback functionality not yet implemented",
	}, nil
}

// scaleApplication scales an application
func (a *Agent) scaleApplication(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	// TODO: Implement scaling logic
	return map[string]interface{}{
		"message": "Scaling functionality not yet implemented",
	}, nil
}

// Container command implementations
func (a *Agent) startContainer(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	containerID := command.Target
	if containerID == "" {
		return nil, fmt.Errorf("container ID is required")
	}

	err := a.containerManager.RestartContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": containerID,
		"status":      "started",
	}, nil
}

func (a *Agent) stopContainer(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	containerID := command.Target
	if containerID == "" {
		return nil, fmt.Errorf("container ID is required")
	}

	err := a.containerManager.StopContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": containerID,
		"status":      "stopped",
	}, nil
}

func (a *Agent) restartContainer(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	containerID := command.Target
	if containerID == "" {
		return nil, fmt.Errorf("container ID is required")
	}

	err := a.containerManager.RestartContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": containerID,
		"status":      "restarted",
	}, nil
}

func (a *Agent) deleteContainer(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	containerID := command.Target
	if containerID == "" {
		return nil, fmt.Errorf("container ID is required")
	}

	err := a.containerManager.DeleteContainer(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": containerID,
		"status":      "deleted",
	}, nil
}

func (a *Agent) getContainerLogs(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	// TODO: Implement log retrieval
	return map[string]interface{}{
		"message": "Log retrieval not yet implemented",
	}, nil
}

func (a *Agent) getContainerStats(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	containerID := command.Target
	if containerID == "" {
		return nil, fmt.Errorf("container ID is required")
	}

	stats, err := a.containerManager.GetContainerStats(ctx, containerID)
	if err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"container_id": stats.ContainerID,
		"cpu_usage":   stats.CPUUsage,
		"memory_usage": stats.MemoryUsage,
		"network_io":  stats.NetworkIO,
		"block_io":    stats.BlockIO,
		"timestamp":   stats.Timestamp,
	}, nil
}

// Git command implementations
func (a *Agent) cloneRepository(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	// TODO: Parse clone options from command.Spec
	return map[string]interface{}{
		"message": "Repository cloning not yet fully implemented",
	}, nil
}

func (a *Agent) pullRepository(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	// TODO: Implement repository pulling
	return map[string]interface{}{
		"message": "Repository pulling not yet implemented",
	}, nil
}

func (a *Agent) buildRepository(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	// TODO: Implement repository building
	return map[string]interface{}{
		"message": "Repository building not yet implemented",
	}, nil
}

// System command implementations
func (a *Agent) getSystemStatus(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	containers, err := a.containerManager.ListContainers(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	repositories := a.gitManager.ListRepositories()

	return map[string]interface{}{
		"agent_id":       a.config.Agent.ID,
		"server_id":      a.config.Agent.ServerID,
		"location":       a.config.Agent.Location,
		"status":         "running",
		"uptime":         time.Since(a.startTime),
		"containers":     len(containers),
		"repositories":   len(repositories),
		"active_commands": len(a.activeCommands),
	}, nil
}

func (a *Agent) getHealthStatus(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	return map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"uptime":    time.Since(a.startTime),
	}, nil
}

func (a *Agent) performCleanup(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	// TODO: Implement system cleanup
	return map[string]interface{}{
		"message": "Cleanup completed",
	}, nil
}

func (a *Agent) updateAgent(ctx context.Context, command *api.DeploymentCommand) (map[string]interface{}, error) {
	// TODO: Implement agent self-update
	return map[string]interface{}{
		"message": "Agent update not yet implemented",
	}, nil
}

// pollCommands polls for new commands from the backend
func (a *Agent) pollCommands() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.Backend.RefreshInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			commands, err := a.backendClient.GetCommands(ctx)
			cancel()

			if err != nil {
				logrus.Errorf("Failed to poll commands: %v", err)
				continue
			}

			for _, command := range commands {
				select {
				case a.commandQueue <- command:
					logrus.Debugf("Queued command: %s", command.ID)
				default:
					logrus.Warnf("Command queue full, dropping command: %s", command.ID)
				}
			}
		}
	}
}

// reportStatus reports agent status to backend
func (a *Agent) reportStatus() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.Agent.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			
			// Get current containers
			containers, err := a.containerManager.ListContainers(ctx)
			var containerStatuses []api.ContainerStatus
			if err == nil {
				for _, container := range containers {
					containerStatuses = append(containerStatuses, api.ContainerStatus{
						ID:      container.ID,
						Name:    container.Name,
						Image:   container.Image,
						Status:  container.Status,
						Health:  container.Health.Status,
						Created: container.CreatedAt,
						Started: container.StartedAt,
					})
				}
			}

			// Send status report
			err = a.backendClient.SendStatusReport(ctx, containerStatuses, nil)
			cancel()

			if err != nil {
				logrus.Errorf("Failed to send status report: %v", err)
			}
		}
	}
}

// rotateTokens rotates API tokens periodically
func (a *Agent) rotateTokens() {
	defer a.wg.Done()

	ticker := time.NewTicker(a.config.Security.TokenRotationInterval)
	defer ticker.Stop()

	for {
		select {
		case <-a.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			err := a.backendClient.RefreshToken(ctx)
			cancel()

			if err != nil {
				logrus.Errorf("Failed to rotate token: %v", err)
			} else {
				logrus.Info("Token rotated successfully")
			}
		}
	}
}

// Helper types and functions

// DeploymentSpec represents a deployment specification
type DeploymentSpec struct {
	Source        string                    `json:"source"`      // git, docker
	Repository    *git.CloneOptions         `json:"repository"`
	Build         *git.BuildSpec            `json:"build"`
	ContainerSpec *docker.DeploymentSpec    `json:"container"`
	Image         string                    `json:"image"`
}

// parseDeploymentSpec parses deployment specification from command spec
func (a *Agent) parseDeploymentSpec(spec map[string]interface{}) (*DeploymentSpec, error) {
	// TODO: Implement proper spec parsing
	// For now, return a mock spec
	return &DeploymentSpec{
		Source: "docker",
		ContainerSpec: &docker.DeploymentSpec{
			Name:  "test-app",
			Image: "nginx",
			Tag:   "latest",
		},
	}, nil
}

// cloneAndBuildRepository clones and builds a repository
func (a *Agent) cloneAndBuildRepository(ctx context.Context, spec *DeploymentSpec) (map[string]interface{}, error) {
	// Clone repository
	repoInfo, err := a.gitManager.CloneRepository(ctx, spec.Repository)
	if err != nil {
		return nil, fmt.Errorf("failed to clone repository: %w", err)
	}

	// Build repository if build spec provided
	if spec.Build != nil {
		err = a.gitManager.BuildRepository(ctx, repoInfo.Path, spec.Build)
		if err != nil {
			return nil, fmt.Errorf("failed to build repository: %w", err)
		}
	}

	return map[string]interface{}{
		"path":   repoInfo.Path,
		"commit": repoInfo.CommitHash,
		"image":  "built-image:latest", // TODO: Actual image tag
	}, nil
}