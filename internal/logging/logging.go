package logging

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// AuditEvent represents an audit log event
type AuditEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	EventType   string                 `json:"event_type"`
	UserID      string                 `json:"user_id,omitempty"`
	AgentID     string                 `json:"agent_id"`
	Source      string                 `json:"source"`
	Action      string                 `json:"action"`
	Resource    string                 `json:"resource,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Success     bool                   `json:"success"`
	ErrorMsg    string                 `json:"error_msg,omitempty"`
	SessionID   string                 `json:"session_id,omitempty"`
	RemoteAddr  string                 `json:"remote_addr,omitempty"`
	RequestID   string                 `json:"request_id,omitempty"`
	Duration    time.Duration          `json:"duration,omitempty"`
	Severity    string                 `json:"severity"`
	Category    string                 `json:"category"`
	Tags        []string               `json:"tags,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AuditLogger handles audit logging functionality
type AuditLogger struct {
	logger     *logrus.Logger
	file       *lumberjack.Logger
	agentID    string
	buffer     chan AuditEvent
	bufferSize int
	mu         sync.RWMutex
	closed     bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// LogStreamer handles streaming logs to backend
type LogStreamer struct {
	endpoint   string
	apiToken   string
	buffer     chan LogEntry
	bufferSize int
	batchSize  int
	mu         sync.RWMutex
	closed     bool
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
}

// LogEntry represents a log entry to be streamed
type LogEntry struct {
	Timestamp   time.Time              `json:"timestamp"`
	Level       string                 `json:"level"`
	Message     string                 `json:"message"`
	Source      string                 `json:"source"`
	AgentID     string                 `json:"agent_id"`
	ContainerID string                 `json:"container_id,omitempty"`
	ServiceName string                 `json:"service_name,omitempty"`
	Fields      map[string]interface{} `json:"fields,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
}

// ContainerLogReader handles reading container logs
type ContainerLogReader struct {
	containerID string
	reader      io.ReadCloser
	scanner     *bufio.Scanner
	streamer    *LogStreamer
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logPath string) (*AuditLogger, error) {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logPath), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// Create rotating file logger
	fileLogger := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    100, // MB
		MaxBackups: 10,
		MaxAge:     30, // days
		Compress:   true,
	}

	// Create logrus logger
	logger := logrus.New()
	logger.SetOutput(fileLogger)
	logger.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339Nano,
	})
	logger.SetLevel(logrus.InfoLevel)

	// Create context for shutdown
	ctx, cancel := context.WithCancel(context.Background())

	// Create audit logger
	auditLogger := &AuditLogger{
		logger:     logger,
		file:       fileLogger,
		agentID:    getAgentID(),
		buffer:     make(chan AuditEvent, 1000),
		bufferSize: 1000,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start background processor
	auditLogger.wg.Add(1)
	go auditLogger.processEvents()

	return auditLogger, nil
}

// LogEvent logs an audit event
func (al *AuditLogger) LogEvent(eventType string, details map[string]interface{}) {
	al.LogEventWithContext(context.Background(), eventType, details)
}

// LogEventWithContext logs an audit event with context
func (al *AuditLogger) LogEventWithContext(ctx context.Context, eventType string, details map[string]interface{}) {
	al.mu.RLock()
	if al.closed {
		al.mu.RUnlock()
		return
	}
	al.mu.RUnlock()

	event := AuditEvent{
		Timestamp: time.Now().UTC(),
		EventType: eventType,
		AgentID:   al.agentID,
		Source:    "deployment-agent",
		Details:   details,
		Success:   true,
		Severity:  "info",
		Category:  "security",
	}

	// Extract common fields from context
	if requestID := ctx.Value("request_id"); requestID != nil {
		if reqID, ok := requestID.(string); ok {
			event.RequestID = reqID
		}
	}

	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			event.UserID = uid
		}
	}

	if sessionID := ctx.Value("session_id"); sessionID != nil {
		if sid, ok := sessionID.(string); ok {
			event.SessionID = sid
		}
	}

	if remoteAddr := ctx.Value("remote_addr"); remoteAddr != nil {
		if addr, ok := remoteAddr.(string); ok {
			event.RemoteAddr = addr
		}
	}

	// Try to send to buffer, drop if full
	select {
	case al.buffer <- event:
	default:
		// Buffer full, log warning and drop event
		al.logger.Warn("Audit log buffer full, dropping event")
	}
}

// LogError logs an error event
func (al *AuditLogger) LogError(eventType string, err error, details map[string]interface{}) {
	al.LogErrorWithContext(context.Background(), eventType, err, details)
}

// LogErrorWithContext logs an error event with context
func (al *AuditLogger) LogErrorWithContext(ctx context.Context, eventType string, err error, details map[string]interface{}) {
	al.mu.RLock()
	if al.closed {
		al.mu.RUnlock()
		return
	}
	al.mu.RUnlock()

	event := AuditEvent{
		Timestamp: time.Now().UTC(),
		EventType: eventType,
		AgentID:   al.agentID,
		Source:    "deployment-agent",
		Details:   details,
		Success:   false,
		ErrorMsg:  err.Error(),
		Severity:  "error",
		Category:  "security",
	}

	// Extract common fields from context
	if requestID := ctx.Value("request_id"); requestID != nil {
		if reqID, ok := requestID.(string); ok {
			event.RequestID = reqID
		}
	}

	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			event.UserID = uid
		}
	}

	if sessionID := ctx.Value("session_id"); sessionID != nil {
		if sid, ok := sessionID.(string); ok {
			event.SessionID = sid
		}
	}

	if remoteAddr := ctx.Value("remote_addr"); remoteAddr != nil {
		if addr, ok := remoteAddr.(string); ok {
			event.RemoteAddr = addr
		}
	}

	// Try to send to buffer, drop if full
	select {
	case al.buffer <- event:
	default:
		// Buffer full, log warning and drop event
		al.logger.Warn("Audit log buffer full, dropping event")
	}
}

// LogDeploymentEvent logs a deployment-related event
func (al *AuditLogger) LogDeploymentEvent(action, resource string, success bool, details map[string]interface{}) {
	al.LogDeploymentEventWithContext(context.Background(), action, resource, success, details)
}

// LogDeploymentEventWithContext logs a deployment-related event with context
func (al *AuditLogger) LogDeploymentEventWithContext(ctx context.Context, action, resource string, success bool, details map[string]interface{}) {
	event := AuditEvent{
		Timestamp: time.Now().UTC(),
		EventType: "DEPLOYMENT_EVENT",
		AgentID:   al.agentID,
		Source:    "deployment-agent",
		Action:    action,
		Resource:  resource,
		Details:   details,
		Success:   success,
		Severity:  "info",
		Category:  "deployment",
	}

	if !success {
		event.Severity = "error"
	}

	// Extract common fields from context
	if requestID := ctx.Value("request_id"); requestID != nil {
		if reqID, ok := requestID.(string); ok {
			event.RequestID = reqID
		}
	}

	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			event.UserID = uid
		}
	}

	// Try to send to buffer, drop if full
	select {
	case al.buffer <- event:
	default:
		// Buffer full, log warning and drop event
		al.logger.Warn("Audit log buffer full, dropping event")
	}
}

// LogSecurityEvent logs a security-related event
func (al *AuditLogger) LogSecurityEvent(action string, success bool, details map[string]interface{}) {
	al.LogSecurityEventWithContext(context.Background(), action, success, details)
}

// LogSecurityEventWithContext logs a security-related event with context
func (al *AuditLogger) LogSecurityEventWithContext(ctx context.Context, action string, success bool, details map[string]interface{}) {
	event := AuditEvent{
		Timestamp: time.Now().UTC(),
		EventType: "SECURITY_EVENT",
		AgentID:   al.agentID,
		Source:    "deployment-agent",
		Action:    action,
		Details:   details,
		Success:   success,
		Severity:  "warning",
		Category:  "security",
	}

	if !success {
		event.Severity = "error"
	}

	// Extract common fields from context
	if requestID := ctx.Value("request_id"); requestID != nil {
		if reqID, ok := requestID.(string); ok {
			event.RequestID = reqID
		}
	}

	if userID := ctx.Value("user_id"); userID != nil {
		if uid, ok := userID.(string); ok {
			event.UserID = uid
		}
	}

	if remoteAddr := ctx.Value("remote_addr"); remoteAddr != nil {
		if addr, ok := remoteAddr.(string); ok {
			event.RemoteAddr = addr
		}
	}

	// Try to send to buffer, drop if full
	select {
	case al.buffer <- event:
	default:
		// Buffer full, log warning and drop event
		al.logger.Warn("Audit log buffer full, dropping event")
	}
}

// processEvents processes audit events from the buffer
func (al *AuditLogger) processEvents() {
	defer al.wg.Done()

	for {
		select {
		case event := <-al.buffer:
			al.writeEvent(event)
		case <-al.ctx.Done():
			// Process remaining events in buffer
			for {
				select {
				case event := <-al.buffer:
					al.writeEvent(event)
				default:
					return
				}
			}
		}
	}
}

// writeEvent writes an event to the log file
func (al *AuditLogger) writeEvent(event AuditEvent) {
	// Convert to JSON
	eventJSON, err := json.Marshal(event)
	if err != nil {
		al.logger.Errorf("Failed to marshal audit event: %v", err)
		return
	}

	// Write to logger
	al.logger.Info(string(eventJSON))
}

// Close closes the audit logger
func (al *AuditLogger) Close() error {
	al.mu.Lock()
	if al.closed {
		al.mu.Unlock()
		return nil
	}
	al.closed = true
	al.mu.Unlock()

	// Cancel context and wait for goroutines
	al.cancel()
	al.wg.Wait()

	// Close file logger
	if al.file != nil {
		return al.file.Close()
	}

	return nil
}

// NewLogStreamer creates a new log streamer
func NewLogStreamer(endpoint, apiToken string) *LogStreamer {
	ctx, cancel := context.WithCancel(context.Background())

	streamer := &LogStreamer{
		endpoint:   endpoint,
		apiToken:   apiToken,
		buffer:     make(chan LogEntry, 10000),
		bufferSize: 10000,
		batchSize:  100,
		ctx:        ctx,
		cancel:     cancel,
	}

	// Start background processor
	streamer.wg.Add(1)
	go streamer.processLogs()

	return streamer
}

// StreamLog streams a log entry
func (ls *LogStreamer) StreamLog(entry LogEntry) {
	ls.mu.RLock()
	if ls.closed {
		ls.mu.RUnlock()
		return
	}
	ls.mu.RUnlock()

	// Try to send to buffer, drop if full
	select {
	case ls.buffer <- entry:
	default:
		// Buffer full, drop log entry
		logrus.Warn("Log streaming buffer full, dropping entry")
	}
}

// processLogs processes log entries from the buffer
func (ls *LogStreamer) processLogs() {
	defer ls.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var batch []LogEntry

	for {
		select {
		case entry := <-ls.buffer:
			batch = append(batch, entry)
			if len(batch) >= ls.batchSize {
				ls.sendBatch(batch)
				batch = batch[:0]
			}
		case <-ticker.C:
			if len(batch) > 0 {
				ls.sendBatch(batch)
				batch = batch[:0]
			}
		case <-ls.ctx.Done():
			// Send remaining logs
			if len(batch) > 0 {
				ls.sendBatch(batch)
			}
			return
		}
	}
}

// sendBatch sends a batch of log entries to the backend
func (ls *LogStreamer) sendBatch(batch []LogEntry) {
	// TODO: Implement HTTP client to send logs to backend
	// For now, just log locally
	logrus.Debugf("Sending %d log entries to backend", len(batch))
}

// Close closes the log streamer
func (ls *LogStreamer) Close() error {
	ls.mu.Lock()
	if ls.closed {
		ls.mu.Unlock()
		return nil
	}
	ls.closed = true
	ls.mu.Unlock()

	// Cancel context and wait for goroutines
	ls.cancel()
	ls.wg.Wait()

	return nil
}

// NewContainerLogReader creates a new container log reader
func NewContainerLogReader(containerID string, reader io.ReadCloser, streamer *LogStreamer) *ContainerLogReader {
	ctx, cancel := context.WithCancel(context.Background())

	return &ContainerLogReader{
		containerID: containerID,
		reader:      reader,
		scanner:     bufio.NewScanner(reader),
		streamer:    streamer,
		ctx:         ctx,
		cancel:      cancel,
	}
}

// Start starts reading container logs
func (clr *ContainerLogReader) Start() {
	clr.wg.Add(1)
	go clr.readLogs()
}

// readLogs reads logs from the container
func (clr *ContainerLogReader) readLogs() {
	defer clr.wg.Done()
	defer clr.reader.Close()

	for {
		select {
		case <-clr.ctx.Done():
			return
		default:
			if clr.scanner.Scan() {
				line := clr.scanner.Text()
				if line != "" {
					entry := LogEntry{
						Timestamp:   time.Now().UTC(),
						Level:       "info",
						Message:     line,
						Source:      "container",
						ContainerID: clr.containerID,
						AgentID:     getAgentID(),
					}
					clr.streamer.StreamLog(entry)
				}
			} else {
				// Scanner error or EOF
				if err := clr.scanner.Err(); err != nil {
					logrus.Errorf("Error reading container logs: %v", err)
				}
				return
			}
		}
	}
}

// Stop stops reading container logs
func (clr *ContainerLogReader) Stop() {
	clr.cancel()
	clr.wg.Wait()
}

// getAgentID returns the agent ID (hostname by default)
func getAgentID() string {
	if hostname, err := os.Hostname(); err == nil {
		return hostname
	}
	return "unknown"
}

// SetupApplicationLogging sets up application logging
func SetupApplicationLogging(logFile string, level string, format string) error {
	// Create log directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(logFile), 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Set up file rotation
	fileLogger := &lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100, // MB
		MaxBackups: 10,
		MaxAge:     30, // days
		Compress:   true,
	}

	// Configure logrus
	logrus.SetOutput(fileLogger)

	// Set log level
	logLevel, err := logrus.ParseLevel(level)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', using info", level)
		logLevel = logrus.InfoLevel
	}
	logrus.SetLevel(logLevel)

	// Set log format
	switch format {
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	case "text":
		logrus.SetFormatter(&logrus.TextFormatter{
			TimestampFormat: time.RFC3339,
			FullTimestamp:   true,
		})
	default:
		logrus.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339,
		})
	}

	return nil
}