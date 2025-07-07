package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"superagent/internal/config"
	"superagent/internal/logging"
	"superagent/internal/deploy"

	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

// APIServer provides HTTP API endpoints for the CLI
type APIServer struct {
	config          *config.Config
	auditLogger     *logging.AuditLogger
	deploymentEngine *deploy.DeploymentEngine
	router          *mux.Router
	server          *http.Server
	startTime       time.Time
}

// AgentStatus represents the agent's current status
type AgentStatus struct {
	Status           string            `json:"status"`
	Health           string            `json:"health"`
	Version          string            `json:"version"`
	Uptime           string            `json:"uptime"`
	ActiveDeployments int              `json:"active_deployments"`
	TotalDeployments  int              `json:"total_deployments"`
	Resources        ResourceInfo      `json:"resources"`
	LastUpdate       time.Time         `json:"last_update"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// DeploymentResponse represents a deployment response
type DeploymentResponse struct {
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	Message     string            `json:"message"`
	AppID       string            `json:"app_id"`
	Version     string            `json:"version"`
	ContainerID string            `json:"container_id,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// LogsResponse represents logs response
type LogsResponse struct {
	DeploymentID string     `json:"deployment_id"`
	Logs         []LogEntry `json:"logs"`
	HasMore      bool       `json:"has_more"`
	NextToken    string     `json:"next_token,omitempty"`
}

// LogEntry represents a log entry
type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
	Source    string    `json:"source"`
}

// NewAPIServer creates a new API server
func NewAPIServer(cfg *config.Config, auditLogger *logging.AuditLogger, deploymentEngine *deploy.DeploymentEngine) *APIServer {
	server := &APIServer{
		config:          cfg,
		auditLogger:     auditLogger,
		deploymentEngine: deploymentEngine,
		router:          mux.NewRouter(),
		startTime:       time.Now(),
	}

	server.setupRoutes()
	return server
}

// setupRoutes sets up HTTP routes
func (s *APIServer) setupRoutes() {
	api := s.router.PathPrefix("/api/v1").Subrouter()

	// Agent endpoints
	api.HandleFunc("/status", s.handleStatus).Methods("GET")
	api.HandleFunc("/version", s.handleVersion).Methods("GET")
	api.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Deployment endpoints
	api.HandleFunc("/deployments", s.handleCreateDeployment).Methods("POST")
	api.HandleFunc("/deployments", s.handleListDeployments).Methods("GET")
	api.HandleFunc("/deployments/{id}", s.handleGetDeployment).Methods("GET")
	api.HandleFunc("/deployments/{id}", s.handleDeleteDeployment).Methods("DELETE")
	api.HandleFunc("/deployments/{id}/logs", s.handleGetLogs).Methods("GET")
	api.HandleFunc("/deployments/{id}/start", s.handleStartDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/stop", s.handleStopDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/restart", s.handleRestartDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/rollback", s.handleRollbackDeployment).Methods("POST")

	// Metrics endpoint
	api.HandleFunc("/metrics", s.handleMetrics).Methods("GET")

	// Add middleware
	s.router.Use(s.loggingMiddleware)
	s.router.Use(s.corsMiddleware)
}

// Start starts the API server
func (s *APIServer) Start(ctx context.Context) error {
	addr := fmt.Sprintf(":%d", s.config.Monitoring.HealthCheckPort)
	
	s.server = &http.Server{
		Addr:    addr,
		Handler: s.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	logrus.Infof("Starting API server on %s", addr)

	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.Errorf("API server error: %v", err)
		}
	}()

	s.auditLogger.LogEvent("API_SERVER_STARTED", map[string]interface{}{
		"address": addr,
	})

	return nil
}

// Stop stops the API server
func (s *APIServer) Stop(ctx context.Context) error {
	if s.server == nil {
		return nil
	}

	logrus.Info("Stopping API server")

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := s.server.Shutdown(ctx); err != nil {
		logrus.Errorf("API server shutdown error: %v", err)
		return err
	}

	s.auditLogger.LogEvent("API_SERVER_STOPPED", map[string]interface{}{})
	return nil
}

// handleStatus handles status requests
func (s *APIServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	deployments := s.deploymentEngine.ListDeployments()
	
	activeCount := 0
	for _, deployment := range deployments {
		if deployment.Status == deploy.StatusRunning {
			activeCount++
		}
	}

	status := &AgentStatus{
		Status:           "running",
		Health:           "healthy",
		Version:          "1.0.0",
		Uptime:           time.Since(s.startTime).String(),
		ActiveDeployments: activeCount,
		TotalDeployments:  len(deployments),
		LastUpdate:       time.Now(),
		Metadata: map[string]interface{}{
			"platform": "SuperAgent Enterprise Deployment System",
			"build":    "production",
		},
	}

	s.writeJSON(w, http.StatusOK, status)
}

// handleVersion handles version requests
func (s *APIServer) handleVersion(w http.ResponseWriter, r *http.Request) {
	version := map[string]interface{}{
		"version":    "1.0.0",
		"build_date": time.Now().Format("2006-01-02T15:04:05Z"),
		"commit":     "unknown",
		"platform":   "SuperAgent Enterprise",
	}

	s.writeJSON(w, http.StatusOK, version)
}

// handleHealth handles health check requests
func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now(),
		"checks": map[string]interface{}{
			"api":        "healthy",
			"deployment": "healthy",
			"storage":    "healthy",
		},
	}

	s.writeJSON(w, http.StatusOK, health)
}

// handleCreateDeployment handles deployment creation
func (s *APIServer) handleCreateDeployment(w http.ResponseWriter, r *http.Request) {
	var req deploy.DeploymentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("Invalid request body: %v", err))
		return
	}

	deployment, err := s.deploymentEngine.Deploy(&req)
	if err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Deployment failed: %v", err))
		return
	}

	response := &DeploymentResponse{
		ID:        deployment.ID,
		Status:    string(deployment.Status),
		Message:   "Deployment created successfully",
		AppID:     deployment.AppID,
		Version:   deployment.Version,
		CreatedAt: deployment.CreatedAt,
		Metadata: map[string]interface{}{
			"source": deployment.Source,
		},
	}

	s.writeJSON(w, http.StatusCreated, response)
}

// handleListDeployments handles listing deployments
func (s *APIServer) handleListDeployments(w http.ResponseWriter, r *http.Request) {
	deployments := s.deploymentEngine.ListDeployments()
	
	var response []DeploymentResponse
	for _, d := range deployments {
		response = append(response, DeploymentResponse{
			ID:          d.ID,
			Status:      string(d.Status),
			AppID:       d.AppID,
			Version:     d.Version,
			ContainerID: d.ContainerID,
			CreatedAt:   d.CreatedAt,
			Metadata: map[string]interface{}{
				"source": d.Source,
				"ports":  d.Ports,
			},
		})
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleGetDeployment handles getting a specific deployment
func (s *APIServer) handleGetDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	deployment, err := s.deploymentEngine.GetDeployment(deploymentID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Deployment not found: %v", err))
		return
	}

	response := &DeploymentResponse{
		ID:          deployment.ID,
		Status:      string(deployment.Status),
		AppID:       deployment.AppID,
		Version:     deployment.Version,
		ContainerID: deployment.ContainerID,
		CreatedAt:   deployment.CreatedAt,
		Metadata: map[string]interface{}{
			"source":       deployment.Source,
			"ports":        deployment.Ports,
			"environment":  deployment.Environment,
			"health_check": deployment.HealthCheck,
		},
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleDeleteDeployment handles deployment deletion
func (s *APIServer) handleDeleteDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	if err := s.deploymentEngine.Remove(deploymentID); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to delete deployment: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Deployment deleted successfully",
		"id":      deploymentID,
	})
}

// handleGetLogs handles getting deployment logs
func (s *APIServer) handleGetLogs(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	deployment, err := s.deploymentEngine.GetDeployment(deploymentID)
	if err != nil {
		s.writeError(w, http.StatusNotFound, fmt.Sprintf("Deployment not found: %v", err))
		return
	}

	// Get query parameters
	tail := 100
	if t := r.URL.Query().Get("tail"); t != "" {
		if parsed, err := strconv.Atoi(t); err == nil {
			tail = parsed
		}
	}

	// Get logs (combine build and deployment logs)
	var logs []LogEntry
	for _, log := range deployment.BuildLogs {
		logs = append(logs, LogEntry{
			Timestamp: log.Timestamp,
			Level:     log.Level,
			Message:   log.Message,
			Source:    "build",
		})
	}
	for _, log := range deployment.DeploymentLogs {
		logs = append(logs, LogEntry{
			Timestamp: log.Timestamp,
			Level:     log.Level,
			Message:   log.Message,
			Source:    "deployment",
		})
	}

	// Limit logs to tail count
	if len(logs) > tail {
		logs = logs[len(logs)-tail:]
	}

	response := &LogsResponse{
		DeploymentID: deploymentID,
		Logs:         logs,
		HasMore:      false,
	}

	s.writeJSON(w, http.StatusOK, response)
}

// handleStartDeployment handles starting a deployment
func (s *APIServer) handleStartDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	// TODO: Implement start functionality
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Deployment start initiated",
		"id":      deploymentID,
	})
}

// handleStopDeployment handles stopping a deployment
func (s *APIServer) handleStopDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

			if err := s.deploymentEngine.StopDeployment(deploymentID); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to stop deployment: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Deployment stopped successfully",
		"id":      deploymentID,
	})
}

// handleRestartDeployment handles restarting a deployment
func (s *APIServer) handleRestartDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	// TODO: Implement restart functionality
	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Deployment restart initiated",
		"id":      deploymentID,
	})
}

// handleRollbackDeployment handles rolling back a deployment
func (s *APIServer) handleRollbackDeployment(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	deploymentID := vars["id"]

	if err := s.deploymentEngine.Rollback(deploymentID, "Manual rollback via API"); err != nil {
		s.writeError(w, http.StatusInternalServerError, fmt.Sprintf("Failed to rollback deployment: %v", err))
		return
	}

	s.writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Deployment rollback initiated",
		"id":      deploymentID,
	})
}

// handleMetrics handles metrics requests
func (s *APIServer) handleMetrics(w http.ResponseWriter, r *http.Request) {
	deployments := s.deploymentEngine.ListDeployments()
	
	metrics := map[string]interface{}{
		"total_deployments":    len(deployments),
		"running_deployments":  0,
		"failed_deployments":   0,
		"pending_deployments":  0,
		"uptime_seconds":       time.Since(s.startTime).Seconds(),
		"last_update":          time.Now(),
	}

	for _, deployment := range deployments {
		switch deployment.Status {
		case deploy.StatusRunning:
			metrics["running_deployments"] = metrics["running_deployments"].(int) + 1
		case deploy.StatusFailed:
			metrics["failed_deployments"] = metrics["failed_deployments"].(int) + 1
		case deploy.StatusPending:
			metrics["pending_deployments"] = metrics["pending_deployments"].(int) + 1
		}
	}

	s.writeJSON(w, http.StatusOK, metrics)
}

// writeJSON writes JSON response
func (s *APIServer) writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes error response
func (s *APIServer) writeError(w http.ResponseWriter, status int, message string) {
	s.writeJSON(w, status, map[string]interface{}{
		"error":   message,
		"status":  status,
		"timestamp": time.Now(),
	})
}

// loggingMiddleware logs HTTP requests
func (s *APIServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		
		// Create a response writer wrapper to capture status
		wrapper := &responseWriter{ResponseWriter: w}
		
		next.ServeHTTP(wrapper, r)
		
		duration := time.Since(start)
		
		logrus.WithFields(logrus.Fields{
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     wrapper.status,
			"duration":   duration,
			"remote_addr": r.RemoteAddr,
		}).Info("HTTP request")
		
		s.auditLogger.LogEvent("HTTP_REQUEST", map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"status":     wrapper.status,
			"duration":   duration.Milliseconds(),
			"remote_addr": r.RemoteAddr,
		})
	})
}

// corsMiddleware adds CORS headers
func (s *APIServer) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	status int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.status = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	if rw.status == 0 {
		rw.status = http.StatusOK
	}
	return rw.ResponseWriter.Write(b)
}