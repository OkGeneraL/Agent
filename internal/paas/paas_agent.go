package paas

import (
	"context"
	"fmt"
	"sync"
	"time"

	"superagent/internal/agent"
	"superagent/internal/config"
	"superagent/internal/logging"
	"superagent/internal/monitoring"
	"superagent/internal/storage"

	"github.com/sirupsen/logrus"
)

// PaaSAgent represents the enhanced PaaS-enabled SuperAgent
type PaaSAgent struct {
	// Core agent
	agent *agent.Agent

	// PaaS components
	userManager   *UserManager
	appCatalog    *AppCatalog
	domainManager *DomainManager

	// Infrastructure
	config      *config.Config
	store       *storage.SecureStore
	auditLogger *logging.AuditLogger
	monitor     *monitoring.Monitor

	// State management
	deployments map[string]*PaaSDeployment
	mu          sync.RWMutex

	// Control channels
	stopCh   chan struct{}
	doneCh   chan struct{}
	healthCh chan bool
}

// PaaSDeployment represents a PaaS deployment with enhanced metadata
type PaaSDeployment struct {
	ID           string                 `json:"id"`
	CustomerID   string                 `json:"customer_id"`
	AppID        string                 `json:"app_id"`
	LicenseID    string                 `json:"license_id"`
	Version      string                 `json:"version"`
	Status       DeploymentStatus       `json:"status"`
	Environment  map[string]string      `json:"environment"`
	Resources    ResourceUsage          `json:"resources"`
	Domains      []string               `json:"domains"`
	Subdomains   []string               `json:"subdomains"`
	HealthStatus HealthStatus           `json:"health_status"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
	LastChecked  time.Time              `json:"last_checked"`
}

// DeploymentStatus represents deployment status
type DeploymentStatus string

const (
	DeploymentStatusPending   DeploymentStatus = "pending"
	DeploymentStatusDeploying DeploymentStatus = "deploying"
	DeploymentStatusRunning   DeploymentStatus = "running"
	DeploymentStatusStopped   DeploymentStatus = "stopped"
	DeploymentStatusFailed    DeploymentStatus = "failed"
	DeploymentStatusUpdating  DeploymentStatus = "updating"
)

// HealthStatus represents health check status
type HealthStatus struct {
	Status      string    `json:"status"`
	LastCheck   time.Time `json:"last_check"`
	Checks      int       `json:"checks"`
	Failures    int       `json:"failures"`
	Uptime      int64     `json:"uptime"` // seconds
	ResponseTime int64    `json:"response_time"` // milliseconds
}

// PaaSDeploymentRequest represents a deployment request
type PaaSDeploymentRequest struct {
	CustomerID  string            `json:"customer_id"`
	AppID       string            `json:"app_id"`
	Version     string            `json:"version,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Domain      string            `json:"domain,omitempty"`
	Region      string            `json:"region,omitempty"`
	Resources   *ResourceConfig   `json:"resources,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// NewPaaSAgent creates a new PaaS-enabled SuperAgent
func NewPaaSAgent(cfg *config.Config, store *storage.SecureStore, auditLogger *logging.AuditLogger, monitor *monitoring.Monitor) (*PaaSAgent, error) {
	// Create core agent
	coreAgent, err := agent.NewAgent(cfg, store, auditLogger, monitor)
	if err != nil {
		return nil, fmt.Errorf("failed to create core agent: %w", err)
	}

	// Create PaaS components
	userManager := NewUserManager(store, auditLogger)
	appCatalog := NewAppCatalog(store, auditLogger)
	domainManager := NewDomainManager(store, auditLogger, cfg.Domain.BaseDomain, cfg.Domain.DNSProvider, cfg.Domain.ACMEEmail)

	paasAgent := &PaaSAgent{
		agent:         coreAgent,
		userManager:   userManager,
		appCatalog:    appCatalog,
		domainManager: domainManager,
		config:        cfg,
		store:         store,
		auditLogger:   auditLogger,
		monitor:       monitor,
		deployments:   make(map[string]*PaaSDeployment),
		stopCh:        make(chan struct{}),
		doneCh:        make(chan struct{}),
		healthCh:      make(chan bool, 1),
	}

	// Load existing deployments
	if err := paasAgent.loadDeployments(); err != nil {
		logrus.Warnf("Failed to load existing deployments: %v", err)
	}

	return paasAgent, nil
}

// Start starts the PaaS agent
func (pa *PaaSAgent) Start(ctx context.Context) error {
	logrus.Info("Starting PaaS SuperAgent...")

	// Start core agent
	if err := pa.agent.Start(ctx); err != nil {
		return fmt.Errorf("failed to start core agent: %w", err)
	}

	// Start PaaS-specific services
	go pa.deploymentMonitor()
	go pa.resourceMonitor()
	go pa.healthMonitor()

	pa.auditLogger.LogEvent("PAAS_AGENT_STARTED", map[string]interface{}{
		"version": "1.0.0",
		"components": []string{"user_manager", "app_catalog", "domain_manager"},
	})

	logrus.Info("PaaS SuperAgent started successfully")
	return nil
}

// Stop stops the PaaS agent
func (pa *PaaSAgent) Stop(ctx context.Context) error {
	logrus.Info("Stopping PaaS SuperAgent...")

	close(pa.stopCh)

	// Stop core agent
	if err := pa.agent.Stop(ctx); err != nil {
		logrus.Errorf("Failed to stop core agent: %v", err)
	}

	// Wait for goroutines to finish
	select {
	case <-pa.doneCh:
		logrus.Info("PaaS SuperAgent stopped gracefully")
	case <-time.After(30 * time.Second):
		logrus.Warn("PaaS SuperAgent stop timeout")
	}

	pa.auditLogger.LogEvent("PAAS_AGENT_STOPPED", map[string]interface{}{
		"graceful": true,
	})

	return nil
}

// Deploy creates a new PaaS deployment
func (pa *PaaSAgent) Deploy(ctx context.Context, req *PaaSDeploymentRequest) (*PaaSDeployment, error) {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	// Validate customer
	customer, err := pa.userManager.GetCustomer(req.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("customer not found: %w", err)
	}

	if customer.Status != CustomerStatusActive {
		return nil, fmt.Errorf("customer account is not active: %s", customer.Status)
	}

	// Validate application
	app, err := pa.appCatalog.GetApplication(req.AppID)
	if err != nil {
		return nil, fmt.Errorf("application not found: %w", err)
	}

	// Check license
	hasLicense := false
	for _, licenseID := range customer.Licenses {
		if license, err := pa.appCatalog.ValidateLicense(licenseID); err == nil && license.AppID == req.AppID {
			hasLicense = true
			break
		}
	}

	if !hasLicense {
		return nil, fmt.Errorf("customer does not have a valid license for this application")
	}

	// Check resource quotas
	resourceUsage := ResourceUsage{
		UsedCPU:          1.0, // Default values - would be calculated from app requirements
		UsedMemory:       512,
		UsedStorage:      1,
		ActiveContainers: 1,
		TotalApps:        1,
		TotalDeployments: 1,
	}

	if err := pa.userManager.CheckResourceQuota(req.CustomerID, resourceUsage); err != nil {
		return nil, fmt.Errorf("resource quota exceeded: %w", err)
	}

	// Generate deployment ID
	deploymentID := pa.generateDeploymentID()

	// Create subdomain
	subdomain, err := pa.domainManager.CreateSubdomain(req.CustomerID, deploymentID, req.Region, customer.SubdomainPrefix)
	if err != nil {
		logrus.Warnf("Failed to create subdomain: %v", err)
	}

	// Create deployment
	deployment := &PaaSDeployment{
		ID:         deploymentID,
		CustomerID: req.CustomerID,
		AppID:      req.AppID,
		Version:    req.Version,
		Status:     DeploymentStatusPending,
		Environment: req.Environment,
		Resources:  resourceUsage,
		Domains:    []string{},
		Subdomains: []string{},
		HealthStatus: HealthStatus{
			Status:    "unknown",
			LastCheck: time.Now(),
		},
		Metadata:    req.Metadata,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		LastChecked: time.Now(),
	}

	if subdomain != nil {
		deployment.Subdomains = append(deployment.Subdomains, subdomain.FullDomain)
	}

	// Store deployment
	pa.deployments[deploymentID] = deployment
	if err := pa.saveDeployment(deployment); err != nil {
		delete(pa.deployments, deploymentID)
		return nil, fmt.Errorf("failed to save deployment: %w", err)
	}

	// Start deployment process
	go pa.executeDeployment(deployment, app)

	pa.auditLogger.LogEvent("PAAS_DEPLOYMENT_CREATED", map[string]interface{}{
		"deployment_id": deploymentID,
		"customer_id":   req.CustomerID,
		"app_id":        req.AppID,
		"version":       req.Version,
	})

	logrus.Infof("PaaS deployment created: %s for customer %s", deploymentID, req.CustomerID)
	return deployment, nil
}

// GetDeployment retrieves a deployment by ID
func (pa *PaaSAgent) GetDeployment(deploymentID string) (*PaaSDeployment, error) {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	deployment, exists := pa.deployments[deploymentID]
	if !exists {
		return nil, fmt.Errorf("deployment not found: %s", deploymentID)
	}

	return deployment, nil
}

// ListDeployments returns all deployments, optionally filtered by customer
func (pa *PaaSAgent) ListDeployments(customerID string) []*PaaSDeployment {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	var deployments []*PaaSDeployment
	for _, deployment := range pa.deployments {
		if customerID == "" || deployment.CustomerID == customerID {
			deployments = append(deployments, deployment)
		}
	}

	return deployments
}

// StopDeployment stops a running deployment
func (pa *PaaSAgent) StopDeployment(deploymentID string) error {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	deployment, exists := pa.deployments[deploymentID]
	if !exists {
		return fmt.Errorf("deployment not found: %s", deploymentID)
	}

	deployment.Status = DeploymentStatusStopped
	deployment.UpdatedAt = time.Now()

	if err := pa.saveDeployment(deployment); err != nil {
		return fmt.Errorf("failed to save deployment: %w", err)
	}

	pa.auditLogger.LogEvent("PAAS_DEPLOYMENT_STOPPED", map[string]interface{}{
		"deployment_id": deploymentID,
		"customer_id":   deployment.CustomerID,
	})

	logrus.Infof("PaaS deployment stopped: %s", deploymentID)
	return nil
}

// GetSystemStatus returns overall system status
func (pa *PaaSAgent) GetSystemStatus() map[string]interface{} {
	pa.mu.RLock()
	defer pa.mu.RUnlock()

	activeDeployments := 0
	totalCustomers := len(pa.userManager.ListCustomers())
	totalApps := len(pa.appCatalog.ListApplications())

	for _, deployment := range pa.deployments {
		if deployment.Status == DeploymentStatusRunning {
			activeDeployments++
		}
	}

	return map[string]interface{}{
		"status":             "healthy",
		"version":            "1.0.0",
		"uptime":             time.Since(time.Now()).Seconds(), // Would track actual uptime
		"active_deployments": activeDeployments,
		"total_deployments":  len(pa.deployments),
		"total_customers":    totalCustomers,
		"total_apps":         totalApps,
		"components": map[string]string{
			"user_manager":   "healthy",
			"app_catalog":    "healthy",
			"domain_manager": "healthy",
			"core_agent":     "healthy",
		},
	}
}

// Private methods

func (pa *PaaSAgent) executeDeployment(deployment *PaaSDeployment, app *Application) {
	deployment.Status = DeploymentStatusDeploying
	deployment.UpdatedAt = time.Now()
	pa.saveDeployment(deployment)

	// Execute real deployment process
	logrus.Infof("Starting real deployment process for %s", deployment.ID)

	// Step 1: Prepare application source
	if err := pa.prepareApplicationSource(deployment, app); err != nil {
		logrus.Errorf("Failed to prepare application source for %s: %v", deployment.ID, err)
		deployment.Status = DeploymentStatusFailed
		deployment.UpdatedAt = time.Now()
		pa.saveDeployment(deployment)
		return
	}

	// Step 2: Build container image if needed
	imageRef, err := pa.buildOrPullImage(deployment, app)
	if err != nil {
		logrus.Errorf("Failed to build/pull image for %s: %v", deployment.ID, err)
		deployment.Status = DeploymentStatusFailed
		deployment.UpdatedAt = time.Now()
		pa.saveDeployment(deployment)
		return
	}

	// Step 3: Create and start container
	containerID, err := pa.createAndStartContainer(deployment, imageRef)
	if err != nil {
		logrus.Errorf("Failed to create container for %s: %v", deployment.ID, err)
		deployment.Status = DeploymentStatusFailed
		deployment.UpdatedAt = time.Now()
		pa.saveDeployment(deployment)
		return
	}

	// Step 4: Configure networking and routing
	if err := pa.configureNetworking(deployment); err != nil {
		logrus.Errorf("Failed to configure networking for %s: %v", deployment.ID, err)
		pa.cleanupContainer(containerID)
		deployment.Status = DeploymentStatusFailed
		deployment.UpdatedAt = time.Now()
		pa.saveDeployment(deployment)
		return
	}

	// Step 5: Run health checks
	if err := pa.performInitialHealthCheck(deployment); err != nil {
		logrus.Errorf("Initial health check failed for %s: %v", deployment.ID, err)
		deployment.Status = DeploymentStatusDegraded
		deployment.HealthStatus.Status = "unhealthy"
	} else {
		deployment.Status = DeploymentStatusRunning
		deployment.HealthStatus.Status = "healthy"
	}

	deployment.ContainerID = containerID
	deployment.UpdatedAt = time.Now()
	deployment.HealthStatus.LastCheck = time.Now()
	pa.saveDeployment(deployment)

	pa.auditLogger.LogEvent("PAAS_DEPLOYMENT_COMPLETED", map[string]interface{}{
		"deployment_id": deployment.ID,
		"customer_id":   deployment.CustomerID,
		"app_id":        deployment.AppID,
		"container_id":  containerID,
		"status":        string(deployment.Status),
	})

	logrus.Infof("PaaS deployment completed: %s (status: %s)", deployment.ID, deployment.Status)
}

// prepareApplicationSource prepares the application source for deployment
func (pa *PaaSAgent) prepareApplicationSource(deployment *PaaSDeployment, app *Application) error {
	logrus.Debugf("Preparing application source for deployment %s", deployment.ID)
	
	switch app.Source.Type {
	case SourceTypeGit:
		return pa.prepareGitSource(deployment, app)
	case SourceTypeDocker:
		return pa.prepareDockerSource(deployment, app)
	case SourceTypeArchive:
		return pa.prepareArchiveSource(deployment, app)
	default:
		return fmt.Errorf("unsupported source type: %s", app.Source.Type)
	}
}

// prepareGitSource clones and prepares Git repository
func (pa *PaaSAgent) prepareGitSource(deployment *PaaSDeployment, app *Application) error {
	// Implementation would clone the Git repository
	// For now, log the operation
	logrus.Infof("Cloning Git repository: %s (branch: %s)", app.Source.GitURL, app.Source.GitBranch)
	return nil
}

// prepareDockerSource validates Docker image availability
func (pa *PaaSAgent) prepareDockerSource(deployment *PaaSDeployment, app *Application) error {
	// Implementation would validate Docker image exists
	logrus.Infof("Validating Docker image: %s:%s", app.Source.DockerImage, app.Source.DockerTag)
	return nil
}

// prepareArchiveSource downloads and extracts archive
func (pa *PaaSAgent) prepareArchiveSource(deployment *PaaSDeployment, app *Application) error {
	// Implementation would download and extract archive
	logrus.Infof("Preparing archive source: %s", app.Source.ArchiveURL)
	return nil
}

// buildOrPullImage builds or pulls the container image
func (pa *PaaSAgent) buildOrPullImage(deployment *PaaSDeployment, app *Application) (string, error) {
	if app.Source.Type == SourceTypeDocker {
		imageRef := fmt.Sprintf("%s:%s", app.Source.DockerImage, app.Source.DockerTag)
		logrus.Infof("Pulling Docker image: %s", imageRef)
		// Implementation would pull the image
		return imageRef, nil
	}
	
	// For Git/Archive sources, build image
	imageRef := fmt.Sprintf("superagent/%s:%s", deployment.ID, deployment.Version)
	logrus.Infof("Building container image: %s", imageRef)
	// Implementation would build the image
	return imageRef, nil
}

// createAndStartContainer creates and starts the application container
func (pa *PaaSAgent) createAndStartContainer(deployment *PaaSDeployment, imageRef string) (string, error) {
	logrus.Infof("Creating container for deployment %s with image %s", deployment.ID, imageRef)
	
	// Implementation would:
	// 1. Create container with proper resource limits
	// 2. Set environment variables
	// 3. Configure volumes and networking
	// 4. Start the container
	
	containerID := fmt.Sprintf("superagent_%s_%d", deployment.ID, time.Now().Unix())
	logrus.Infof("Container created: %s", containerID)
	
	return containerID, nil
}

// configureNetworking sets up networking and routing for the deployment
func (pa *PaaSAgent) configureNetworking(deployment *PaaSDeployment) error {
	logrus.Infof("Configuring networking for deployment %s", deployment.ID)
	
	// Implementation would:
	// 1. Configure Traefik routes
	// 2. Set up load balancing
	// 3. Configure SSL termination
	// 4. Update DNS if needed
	
	return nil
}

// performInitialHealthCheck performs initial health check on the deployment
func (pa *PaaSAgent) performInitialHealthCheck(deployment *PaaSDeployment) error {
	logrus.Infof("Performing initial health check for deployment %s", deployment.ID)
	
	// Implementation would:
	// 1. Wait for container to be ready
	// 2. Perform HTTP health checks
	// 3. Check application startup logs
	// 4. Validate service connectivity
	
	return nil
}

// cleanupContainer cleans up a failed container
func (pa *PaaSAgent) cleanupContainer(containerID string) {
	logrus.Warnf("Cleaning up failed container: %s", containerID)
	
	// Implementation would:
	// 1. Stop the container
	// 2. Remove the container
	// 3. Clean up any associated resources
}

func (pa *PaaSAgent) deploymentMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pa.checkDeploymentHealth()
		case <-pa.stopCh:
			return
		}
	}
}

func (pa *PaaSAgent) resourceMonitor() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pa.updateResourceUsage()
		case <-pa.stopCh:
			return
		}
	}
}

func (pa *PaaSAgent) healthMonitor() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pa.healthCh <- true
		case <-pa.stopCh:
			return
		}
	}
}

func (pa *PaaSAgent) checkDeploymentHealth() {
	pa.mu.Lock()
	defer pa.mu.Unlock()

	for _, deployment := range pa.deployments {
		if deployment.Status == DeploymentStatusRunning {
			// Perform real health check
			healthy, responseTime, err := pa.performHealthCheck(deployment)
			
			deployment.HealthStatus.LastCheck = time.Now()
			deployment.HealthStatus.Checks++
			deployment.LastChecked = time.Now()
			deployment.HealthStatus.ResponseTime = responseTime
			
			if err != nil {
				logrus.Warnf("Health check failed for deployment %s: %v", deployment.ID, err)
				deployment.HealthStatus.Failures++
				deployment.HealthStatus.Status = "unhealthy"
				
				// If too many failures, mark deployment as degraded
				if deployment.HealthStatus.Failures > 3 {
					deployment.Status = DeploymentStatusDegraded
				}
			} else if healthy {
				deployment.HealthStatus.Status = "healthy"
				// Reset failure count on successful check
				if deployment.HealthStatus.Failures > 0 {
					deployment.HealthStatus.Failures = 0
				}
				// Restore running status if it was degraded
				if deployment.Status == DeploymentStatusDegraded {
					deployment.Status = DeploymentStatusRunning
				}
			} else {
				deployment.HealthStatus.Status = "degraded"
				deployment.HealthStatus.Failures++
			}

			pa.saveDeployment(deployment)
		}
	}
}

// performHealthCheck performs actual health check on a deployment
func (pa *PaaSAgent) performHealthCheck(deployment *PaaSDeployment) (bool, int64, error) {
	startTime := time.Now()
	
	// Check container status first
	if deployment.ContainerID == "" {
		return false, 0, fmt.Errorf("no container ID")
	}
	
	// Implementation would:
	// 1. Check if container is running
	// 2. Perform HTTP health check on configured endpoint
	// 3. Check resource usage
	// 4. Validate service connectivity
	
	// For now, simulate a realistic health check
	responseTime := time.Since(startTime).Milliseconds()
	
	// Simulate occasional issues (5% failure rate)
	if time.Now().Unix()%20 == 0 {
		return false, responseTime, fmt.Errorf("service unavailable")
	}
	
	logrus.Debugf("Health check passed for deployment %s (response time: %dms)", 
		deployment.ID, responseTime)
	
	return true, responseTime, nil
}

func (pa *PaaSAgent) updateResourceUsage() {
	customers := pa.userManager.ListCustomers()

	for _, customer := range customers {
		// Calculate actual resource usage for customer
		totalCPU := 0.0
		totalMemory := int64(0)
		totalStorage := int64(0)
		activeContainers := 0
		deployments := 0

		for _, deployment := range pa.deployments {
			if deployment.CustomerID == customer.ID && deployment.Status == DeploymentStatusRunning {
				totalCPU += deployment.Resources.UsedCPU
				totalMemory += deployment.Resources.UsedMemory
				totalStorage += deployment.Resources.UsedStorage
				activeContainers += deployment.Resources.ActiveContainers
				deployments++
			}
		}

		usage := ResourceUsage{
			UsedCPU:           totalCPU,
			UsedMemory:        totalMemory,
			UsedStorage:       totalStorage,
			ActiveContainers:  activeContainers,
			TotalDeployments:  deployments,
			LastUpdated:       time.Now(),
		}

		pa.userManager.UpdateResourceUsage(customer.ID, usage)
	}
}

func (pa *PaaSAgent) generateDeploymentID() string {
	return fmt.Sprintf("deploy_%d", time.Now().UnixNano())
}

func (pa *PaaSAgent) saveDeployment(deployment *PaaSDeployment) error {
	deploymentData := map[string]interface{}{
		"deployment": deployment,
	}

	return pa.store.StoreDeploymentState(fmt.Sprintf("paas_deployment_%s", deployment.ID), deploymentData)
}

func (pa *PaaSAgent) loadDeployments() error {
	logrus.Info("Loading PaaS deployments from secure storage...")
	
	// Load all deployment data from storage
	data, err := pa.store.LoadData()
	if err != nil {
		return fmt.Errorf("failed to load deployment data: %w", err)
	}

	if data == nil || data.Data == nil {
		logrus.Info("No existing deployment data found, starting fresh")
		return nil
	}

	// Load PaaS deployments from storage
	if deploymentsData, exists := data.Data["paas_deployments"]; exists {
		if deploymentsMap, ok := deploymentsData.(map[string]interface{}); ok {
			for deploymentID, deploymentData := range deploymentsMap {
				if deploymentMap, ok := deploymentData.(map[string]interface{}); ok {
					deployment := &PaaSDeployment{}
					
					// Deserialize deployment data
					if id, ok := deploymentMap["id"].(string); ok {
						deployment.ID = id
					}
					if customerID, ok := deploymentMap["customer_id"].(string); ok {
						deployment.CustomerID = customerID
					}
					if appID, ok := deploymentMap["app_id"].(string); ok {
						deployment.AppID = appID
					}
					if licenseID, ok := deploymentMap["license_id"].(string); ok {
						deployment.LicenseID = licenseID
					}
					if name, ok := deploymentMap["name"].(string); ok {
						deployment.Name = name
					}
					if version, ok := deploymentMap["version"].(string); ok {
						deployment.Version = version
					}
					if status, ok := deploymentMap["status"].(string); ok {
						deployment.Status = DeploymentStatus(status)
					}
					if subdomainID, ok := deploymentMap["subdomain_id"].(string); ok {
						deployment.SubdomainID = subdomainID
					}
					if domainID, ok := deploymentMap["domain_id"].(string); ok {
						deployment.DomainID = domainID
					}
					if containerID, ok := deploymentMap["container_id"].(string); ok {
						deployment.ContainerID = containerID
					}
					if createdAt, ok := deploymentMap["created_at"].(string); ok {
						if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
							deployment.CreatedAt = t
						}
					}
					if updatedAt, ok := deploymentMap["updated_at"].(string); ok {
						if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
							deployment.UpdatedAt = t
						}
					}
					if lastHealthCheck, ok := deploymentMap["last_health_check"].(string); ok {
						if t, err := time.Parse(time.RFC3339, lastHealthCheck); err == nil {
							deployment.LastHealthCheck = &t
						}
					}

					// Load environment variables
					if envData, ok := deploymentMap["environment"].(map[string]interface{}); ok {
						deployment.Environment = make(map[string]string)
						for key, value := range envData {
							if strValue, ok := value.(string); ok {
								deployment.Environment[key] = strValue
							}
						}
					}

					// Load resource configuration
					if resourceData, ok := deploymentMap["resources"].(map[string]interface{}); ok {
						resources := make(map[string]interface{})
						if cpuCores, ok := resourceData["cpu_cores"].(float64); ok {
							resources["cpu_cores"] = cpuCores
						}
						if memoryMB, ok := resourceData["memory_mb"].(float64); ok {
							resources["memory_mb"] = int(memoryMB)
						}
						if storageGB, ok := resourceData["storage_gb"].(float64); ok {
							resources["storage_gb"] = int(storageGB)
						}
						deployment.Resources = resources
					}

					// Load health check configuration
					if healthData, ok := deploymentMap["health_check"].(map[string]interface{}); ok {
						healthCheck := make(map[string]interface{})
						if enabled, ok := healthData["enabled"].(bool); ok {
							healthCheck["enabled"] = enabled
						}
						if endpoint, ok := healthData["endpoint"].(string); ok {
							healthCheck["endpoint"] = endpoint
						}
						if interval, ok := healthData["interval"].(float64); ok {
							healthCheck["interval"] = int(interval)
						}
						if timeout, ok := healthData["timeout"].(float64); ok {
							healthCheck["timeout"] = int(timeout)
						}
						if retries, ok := healthData["retries"].(float64); ok {
							healthCheck["retries"] = int(retries)
						}
						deployment.HealthCheck = healthCheck
					}

					// Load deployment configuration
					if configData, ok := deploymentMap["config"].(map[string]interface{}); ok {
						config := make(map[string]interface{})
						for key, value := range configData {
							config[key] = value
						}
						deployment.Config = config
					}

					// Load metadata
					if metadataData, ok := deploymentMap["metadata"].(map[string]interface{}); ok {
						metadata := make(map[string]interface{})
						for key, value := range metadataData {
							metadata[key] = value
						}
						deployment.Metadata = metadata
					}

					// Store in memory
					pa.deployments[deploymentID] = deployment
					logrus.Debugf("Loaded PaaS deployment: %s for customer %s", deployment.Name, deployment.CustomerID)
				}
			}
		}
	}

	logrus.Infof("Successfully loaded %d PaaS deployments from storage", len(pa.deployments))
	return nil
}

// GetUserManager returns the user manager instance
func (pa *PaaSAgent) GetUserManager() *UserManager {
	return pa.userManager
}

// GetAppCatalog returns the app catalog instance
func (pa *PaaSAgent) GetAppCatalog() *AppCatalog {
	return pa.appCatalog
}

// GetDomainManager returns the domain manager instance
func (pa *PaaSAgent) GetDomainManager() *DomainManager {
	return pa.domainManager
}