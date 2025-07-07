package monitoring

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"superagent/internal/logging"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Monitor handles metrics collection and health monitoring
type Monitor struct {
	auditLogger      *logging.AuditLogger
	registry         *prometheus.Registry
	httpServer       *http.Server
	metricsPort      int
	healthChecks     map[string]HealthChecker
	deploymentMetrics map[string]*DeploymentMetrics
	systemMetrics    *SystemMetrics
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	started          bool
}

// HealthChecker interface for health check implementations
type HealthChecker interface {
	Check(ctx context.Context) error
	Name() string
}

// DeploymentMetrics contains metrics for deployment monitoring
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

// SystemMetrics contains system-level metrics
type SystemMetrics struct {
	// Deployment counters
	deploymentsTotal     prometheus.Counter
	deploymentsSuccessful prometheus.Counter
	deploymentsFailed    prometheus.Counter
	deploymentsActive    prometheus.Gauge

	// Resource metrics
	cpuUsageGauge    prometheus.GaugeVec
	memoryUsageGauge prometheus.GaugeVec
	diskUsageGauge   prometheus.GaugeVec
	networkRxGauge   prometheus.GaugeVec
	networkTxGauge   prometheus.GaugeVec

	// Health check metrics
	healthCheckTotal     prometheus.CounterVec
	healthCheckSuccessful prometheus.CounterVec
	healthCheckDuration  prometheus.HistogramVec

	// API metrics
	apiRequestsTotal    prometheus.CounterVec
	apiRequestDuration prometheus.HistogramVec

	// Agent metrics
	agentUptime    prometheus.Gauge
	agentVersion   prometheus.GaugeVec
	gitOperations  prometheus.CounterVec
	dockerOperations prometheus.CounterVec
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Name      string                 `json:"name"`
	Status    string                 `json:"status"` // "healthy", "unhealthy", "unknown"
	Message   string                 `json:"message"`
	LastCheck time.Time              `json:"last_check"`
	Duration  time.Duration          `json:"duration"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// NewMonitor creates a new monitoring instance
func NewMonitor(auditLogger *logging.AuditLogger, metricsPort int) *Monitor {
	ctx, cancel := context.WithCancel(context.Background())

	registry := prometheus.NewRegistry()
	
	m := &Monitor{
		auditLogger:       auditLogger,
		registry:         registry,
		metricsPort:      metricsPort,
		healthChecks:     make(map[string]HealthChecker),
		deploymentMetrics: make(map[string]*DeploymentMetrics),
		ctx:              ctx,
		cancel:           cancel,
	}

	// Initialize system metrics
	m.initSystemMetrics()

	return m
}

// Start initializes and starts the monitoring system
func (m *Monitor) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.started {
		return fmt.Errorf("monitor already started")
	}

	logrus.Info("Starting monitoring system")

	// Start metrics HTTP server
	if err := m.startMetricsServer(); err != nil {
		return fmt.Errorf("failed to start metrics server: %w", err)
	}

	// Start health check goroutine
	m.wg.Add(1)
	go m.healthCheckWorker()

	// Start metrics collection goroutine
	m.wg.Add(1)
	go m.metricsCollectionWorker()

	m.started = true

	m.auditLogger.LogEvent("MONITOR_STARTED", map[string]interface{}{
		"metrics_port": m.metricsPort,
	})

	return nil
}

// Stop gracefully stops the monitoring system
func (m *Monitor) Stop() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.started {
		return nil
	}

	logrus.Info("Stopping monitoring system")

	// Cancel context to stop workers
	m.cancel()

	// Stop HTTP server
	if m.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := m.httpServer.Shutdown(ctx); err != nil {
			logrus.Warnf("Failed to gracefully shutdown metrics server: %v", err)
		}
	}

	// Wait for workers to finish
	m.wg.Wait()

	m.started = false

	m.auditLogger.LogEvent("MONITOR_STOPPED", map[string]interface{}{})

	return nil
}

// RegisterHealthCheck registers a health checker
func (m *Monitor) RegisterHealthCheck(checker HealthChecker) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.healthChecks[checker.Name()] = checker
	logrus.Infof("Registered health check: %s", checker.Name())
}

// UnregisterHealthCheck removes a health checker
func (m *Monitor) UnregisterHealthCheck(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.healthChecks, name)
	logrus.Infof("Unregistered health check: %s", name)
}

// GetHealthStatus returns the current health status of all components
func (m *Monitor) GetHealthStatus() map[string]*HealthStatus {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]*HealthStatus)

	for name, checker := range m.healthChecks {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		start := time.Now()
		
		err := checker.Check(ctx)
		duration := time.Since(start)
		cancel()

		healthStatus := &HealthStatus{
			Name:      name,
			LastCheck: time.Now(),
			Duration:  duration,
		}

		if err != nil {
			healthStatus.Status = "unhealthy"
			healthStatus.Message = err.Error()
		} else {
			healthStatus.Status = "healthy"
			healthStatus.Message = "OK"
		}

		status[name] = healthStatus

		// Record metrics
		labels := prometheus.Labels{"component": name}
		m.systemMetrics.healthCheckTotal.With(labels).Inc()
		if err == nil {
			m.systemMetrics.healthCheckSuccessful.With(labels).Inc()
		}
		m.systemMetrics.healthCheckDuration.With(labels).Observe(duration.Seconds())
	}

	return status
}

// RecordDeploymentStatus records deployment status metrics
func (m *Monitor) RecordDeploymentStatus(deploymentID, status string) {
	_ = prometheus.Labels{"deployment_id": deploymentID, "status": status} // labels not used for counter

	m.systemMetrics.deploymentsTotal.Inc()
	
	switch status {
	case "running":
		m.systemMetrics.deploymentsSuccessful.Inc()
	case "failed":
		m.systemMetrics.deploymentsFailed.Inc()
	}

	m.auditLogger.LogEvent("DEPLOYMENT_STATUS_RECORDED", map[string]interface{}{
		"deployment_id": deploymentID,
		"status":        status,
	})
}

// RecordDeploymentMetrics records deployment resource metrics
func (m *Monitor) RecordDeploymentMetrics(deploymentID string, metrics DeploymentMetrics) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Store metrics
	m.deploymentMetrics[deploymentID] = &metrics

	// Update Prometheus metrics
	labels := prometheus.Labels{"deployment_id": deploymentID}
	
	m.systemMetrics.cpuUsageGauge.With(labels).Set(metrics.CPUUsage)
	m.systemMetrics.memoryUsageGauge.With(labels).Set(float64(metrics.MemoryUsage))
	m.systemMetrics.networkRxGauge.With(labels).Set(float64(metrics.NetworkRx))
	m.systemMetrics.networkTxGauge.With(labels).Set(float64(metrics.NetworkTx))
	m.systemMetrics.diskUsageGauge.With(labels).Set(float64(metrics.DiskUsage))
}

// RecordAPIRequest records API request metrics
func (m *Monitor) RecordAPIRequest(method, endpoint, status string, duration time.Duration) {
	labels := prometheus.Labels{
		"method":   method,
		"endpoint": endpoint,
		"status":   status,
	}

	m.systemMetrics.apiRequestsTotal.With(labels).Inc()
	m.systemMetrics.apiRequestDuration.With(labels).Observe(duration.Seconds())
}

// RecordGitOperation records Git operation metrics
func (m *Monitor) RecordGitOperation(operation, status string) {
	labels := prometheus.Labels{
		"operation": operation,
		"status":    status,
	}

	m.systemMetrics.gitOperations.With(labels).Inc()
}

// RecordDockerOperation records Docker operation metrics
func (m *Monitor) RecordDockerOperation(operation, status string) {
	labels := prometheus.Labels{
		"operation": operation,
		"status":    status,
	}

	m.systemMetrics.dockerOperations.With(labels).Inc()
}

// GetDeploymentMetrics returns metrics for a specific deployment
func (m *Monitor) GetDeploymentMetrics(deploymentID string) *DeploymentMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.deploymentMetrics[deploymentID]
}

// GetAllDeploymentMetrics returns metrics for all deployments
func (m *Monitor) GetAllDeploymentMetrics() map[string]*DeploymentMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string]*DeploymentMetrics)
	for k, v := range m.deploymentMetrics {
		result[k] = v
	}

	return result
}

// UpdateActiveDeployments updates the count of active deployments
func (m *Monitor) UpdateActiveDeployments(count int) {
	m.systemMetrics.deploymentsActive.Set(float64(count))
}

// initSystemMetrics initializes Prometheus metrics
func (m *Monitor) initSystemMetrics() {
	m.systemMetrics = &SystemMetrics{
		// Deployment counters
		deploymentsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "superagent_deployments_total",
			Help: "Total number of deployments",
		}),
		deploymentsSuccessful: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "superagent_deployments_successful_total",
			Help: "Total number of successful deployments",
		}),
		deploymentsFailed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "superagent_deployments_failed_total",
			Help: "Total number of failed deployments",
		}),
		deploymentsActive: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "superagent_deployments_active",
			Help: "Number of currently active deployments",
		}),

		// Resource metrics
		cpuUsageGauge: *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "superagent_deployment_cpu_usage_percent",
			Help: "CPU usage percentage for deployments",
		}, []string{"deployment_id"}),
		
		memoryUsageGauge: *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "superagent_deployment_memory_usage_bytes",
			Help: "Memory usage in bytes for deployments",
		}, []string{"deployment_id"}),
		
		diskUsageGauge: *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "superagent_deployment_disk_usage_bytes",
			Help: "Disk usage in bytes for deployments",
		}, []string{"deployment_id"}),
		
		networkRxGauge: *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "superagent_deployment_network_rx_bytes",
			Help: "Network received bytes for deployments",
		}, []string{"deployment_id"}),
		
		networkTxGauge: *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "superagent_deployment_network_tx_bytes",
			Help: "Network transmitted bytes for deployments",
		}, []string{"deployment_id"}),

		// Health check metrics
		healthCheckTotal: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "superagent_health_checks_total",
			Help: "Total number of health checks performed",
		}, []string{"component"}),
		
		healthCheckSuccessful: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "superagent_health_checks_successful_total",
			Help: "Total number of successful health checks",
		}, []string{"component"}),
		
		healthCheckDuration: *prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "superagent_health_check_duration_seconds",
			Help: "Duration of health checks in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"component"}),

		// API metrics
		apiRequestsTotal: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "superagent_api_requests_total",
			Help: "Total number of API requests",
		}, []string{"method", "endpoint", "status"}),
		
		apiRequestDuration: *prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "superagent_api_request_duration_seconds",
			Help: "Duration of API requests in seconds",
			Buckets: prometheus.DefBuckets,
		}, []string{"method", "endpoint", "status"}),

		// Agent metrics
		agentUptime: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "superagent_uptime_seconds",
			Help: "Agent uptime in seconds",
		}),
		
		agentVersion: *prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "superagent_version_info",
			Help: "Agent version information",
		}, []string{"version", "commit", "build_date"}),

		// Operation metrics
		gitOperations: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "superagent_git_operations_total",
			Help: "Total number of Git operations",
		}, []string{"operation", "status"}),
		
		dockerOperations: *prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "superagent_docker_operations_total",
			Help: "Total number of Docker operations",
		}, []string{"operation", "status"}),
	}

	// Register all metrics
	m.registry.MustRegister(
		m.systemMetrics.deploymentsTotal,
		m.systemMetrics.deploymentsSuccessful,
		m.systemMetrics.deploymentsFailed,
		m.systemMetrics.deploymentsActive,
		m.systemMetrics.cpuUsageGauge,
		m.systemMetrics.memoryUsageGauge,
		m.systemMetrics.diskUsageGauge,
		m.systemMetrics.networkRxGauge,
		m.systemMetrics.networkTxGauge,
		m.systemMetrics.healthCheckTotal,
		m.systemMetrics.healthCheckSuccessful,
		m.systemMetrics.healthCheckDuration,
		m.systemMetrics.apiRequestsTotal,
		m.systemMetrics.apiRequestDuration,
		m.systemMetrics.agentUptime,
		m.systemMetrics.agentVersion,
		m.systemMetrics.gitOperations,
		m.systemMetrics.dockerOperations,
	)
}

// startMetricsServer starts the HTTP server for Prometheus metrics
func (m *Monitor) startMetricsServer() error {
	mux := http.NewServeMux()
	
	// Prometheus metrics endpoint
	mux.Handle("/metrics", promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{}))
	
	// Health check endpoint
	mux.HandleFunc("/health", m.handleHealthCheck)
	
	// Agent info endpoint
	mux.HandleFunc("/info", m.handleAgentInfo)

	m.httpServer = &http.Server{
		Addr:    fmt.Sprintf(":%d", m.metricsPort),
		Handler: mux,
	}

	go func() {
		if err := m.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("Metrics server error: %v", err)
		}
	}()

	logrus.Infof("Metrics server started on port %d", m.metricsPort)
	return nil
}

// handleHealthCheck handles health check HTTP requests
func (m *Monitor) handleHealthCheck(w http.ResponseWriter, r *http.Request) {
	status := m.GetHealthStatus()
	
	overallHealthy := true
	for _, health := range status {
		if health.Status != "healthy" {
			overallHealthy = false
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	
	if overallHealthy {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","components":` + fmt.Sprintf("%+v", status) + `}`))
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"status":"unhealthy","components":` + fmt.Sprintf("%+v", status) + `}`))
	}
}

// handleAgentInfo handles agent information HTTP requests
func (m *Monitor) handleAgentInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"name":        "SuperAgent",
		"version":     "1.0.0",
		"build_date":  time.Now().Format(time.RFC3339),
		"uptime":      time.Since(time.Now()).Seconds(), // This would be actual uptime in real implementation
		"deployments": len(m.deploymentMetrics),
		"health_checks": len(m.healthChecks),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("%+v", info)))
}

// healthCheckWorker runs periodic health checks
func (m *Monitor) healthCheckWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.runHealthChecks()
		}
	}
}

// metricsCollectionWorker runs periodic metrics collection
func (m *Monitor) metricsCollectionWorker() {
	defer m.wg.Done()

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	startTime := time.Now()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			// Update uptime
			m.systemMetrics.agentUptime.Set(time.Since(startTime).Seconds())
			
			// Update active deployments count
			m.mu.RLock()
			activeCount := len(m.deploymentMetrics)
			m.mu.RUnlock()
			m.systemMetrics.deploymentsActive.Set(float64(activeCount))
		}
	}
}

// runHealthChecks executes all registered health checks
func (m *Monitor) runHealthChecks() {
	m.mu.RLock()
	checkers := make(map[string]HealthChecker)
	for k, v := range m.healthChecks {
		checkers[k] = v
	}
	m.mu.RUnlock()

	for name, checker := range checkers {
		func(name string, checker HealthChecker) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			start := time.Now()
			err := checker.Check(ctx)
			duration := time.Since(start)

			labels := prometheus.Labels{"component": name}
			m.systemMetrics.healthCheckTotal.With(labels).Inc()
			
			if err == nil {
				m.systemMetrics.healthCheckSuccessful.With(labels).Inc()
				logrus.Debugf("Health check passed for %s in %v", name, duration)
			} else {
				logrus.Warnf("Health check failed for %s: %v (took %v)", name, err, duration)
			}
			
			m.systemMetrics.healthCheckDuration.With(labels).Observe(duration.Seconds())
		}(name, checker)
	}
}

// SetVersion sets the agent version information
func (m *Monitor) SetVersion(version, commit, buildDate string) {
	labels := prometheus.Labels{
		"version":    version,
		"commit":     commit,
		"build_date": buildDate,
	}
	m.systemMetrics.agentVersion.With(labels).Set(1)
}

// RemoveDeploymentMetrics removes metrics for a deployment
func (m *Monitor) RemoveDeploymentMetrics(deploymentID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.deploymentMetrics, deploymentID)

	// Remove from Prometheus metrics
	labels := prometheus.Labels{"deployment_id": deploymentID}
	m.systemMetrics.cpuUsageGauge.Delete(labels)
	m.systemMetrics.memoryUsageGauge.Delete(labels)
	m.systemMetrics.networkRxGauge.Delete(labels)
	m.systemMetrics.networkTxGauge.Delete(labels)
	m.systemMetrics.diskUsageGauge.Delete(labels)
}

// GetMetricsPort returns the metrics server port
func (m *Monitor) GetMetricsPort() int {
	return m.metricsPort
}

// IsHealthy returns true if all health checks are passing
func (m *Monitor) IsHealthy() bool {
	status := m.GetHealthStatus()
	
	for _, health := range status {
		if health.Status != "healthy" {
			return false
		}
	}
	
	return true
}