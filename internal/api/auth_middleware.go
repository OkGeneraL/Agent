package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"superagent/internal/config"
	"superagent/internal/logging"

	"github.com/sirupsen/logrus"
)

// AuthMiddleware handles API authentication using Bearer tokens
type AuthMiddleware struct {
	config      *config.Config
	auditLogger *logging.AuditLogger
	httpClient  *http.Client
}

// TokenValidationRequest represents the request to validate a token
type TokenValidationRequest struct {
	Token string `json:"token"`
}

// TokenValidationResponse represents the response from token validation
type TokenValidationResponse struct {
	Valid     bool   `json:"valid"`
	ServerID  string `json:"server_id,omitempty"`
	ExpiresAt string `json:"expires_at,omitempty"`
	Error     string `json:"error,omitempty"`
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(cfg *config.Config, auditLogger *logging.AuditLogger) *AuthMiddleware {
	return &AuthMiddleware{
		config:      cfg,
		auditLogger: auditLogger,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// Middleware returns the HTTP middleware function
func (am *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health and version endpoints
		if am.isPublicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract Bearer token from Authorization header
		token := am.extractBearerToken(r)
		if token == "" {
			am.auditLogger.LogSecurityEvent("AUTH_MISSING_TOKEN", false, map[string]interface{}{
				"path":        r.URL.Path,
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
			})
			am.writeAuthError(w, "Authorization token required", http.StatusUnauthorized)
			return
		}

		// Validate token against admin panel
		valid, serverID, err := am.validateToken(token)
		if err != nil {
			logrus.Errorf("Token validation error: %v", err)
			am.auditLogger.LogSecurityEvent("AUTH_VALIDATION_ERROR", false, map[string]interface{}{
				"error":       err.Error(),
				"path":        r.URL.Path,
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
			})
			am.writeAuthError(w, "Authentication failed", http.StatusInternalServerError)
			return
		}

		if !valid {
			am.auditLogger.LogSecurityEvent("AUTH_INVALID_TOKEN", false, map[string]interface{}{
				"path":        r.URL.Path,
				"method":      r.Method,
				"remote_addr": r.RemoteAddr,
			})
			am.writeAuthError(w, "Invalid or expired token", http.StatusUnauthorized)
			return
		}

		// Add server ID to request context for use in handlers
		ctx := context.WithValue(r.Context(), "server_id", serverID)
		r = r.WithContext(ctx)

		// Log successful authentication
		am.auditLogger.LogSecurityEvent("AUTH_SUCCESS", true, map[string]interface{}{
			"server_id":   serverID,
			"path":        r.URL.Path,
			"method":      r.Method,
			"remote_addr": r.RemoteAddr,
		})

		next.ServeHTTP(w, r)
	})
}

// isPublicEndpoint checks if the endpoint should bypass authentication
func (am *AuthMiddleware) isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/health",
		"/api/v1/version",
		"/metrics", // Prometheus metrics (consider protecting this in production)
	}

	for _, endpoint := range publicEndpoints {
		if path == endpoint {
			return true
		}
	}
	return false
}

// extractBearerToken extracts the Bearer token from the Authorization header
func (am *AuthMiddleware) extractBearerToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return ""
	}

	return parts[1]
}

// validateToken validates the token against the admin panel
func (am *AuthMiddleware) validateToken(token string) (bool, string, error) {
	// Get validation URL from config
	validationURL := am.getValidationURL()
	if validationURL == "" {
		return false, "", fmt.Errorf("no validation URL configured")
	}

	// Create validation request
	reqBody := TokenValidationRequest{
		Token: token,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal validation request: %w", err)
	}

	// Make HTTP request to admin panel
	req, err := http.NewRequest("POST", validationURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return false, "", fmt.Errorf("failed to create validation request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "SuperAgent/1.0")

	resp, err := am.httpClient.Do(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to validate token: %w", err)
	}
	defer resp.Body.Close()

	// Parse response
	var validationResp TokenValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&validationResp); err != nil {
		return false, "", fmt.Errorf("failed to decode validation response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		if validationResp.Error != "" {
			return false, "", fmt.Errorf("validation failed: %s", validationResp.Error)
		}
		return false, "", fmt.Errorf("validation failed with status: %d", resp.StatusCode)
	}

	return validationResp.Valid, validationResp.ServerID, nil
}

// getValidationURL constructs the token validation URL
func (am *AuthMiddleware) getValidationURL() string {
	baseURL := am.config.Backend.BaseURL
	if baseURL == "" {
		return ""
	}

	// Ensure baseURL doesn't end with /
	baseURL = strings.TrimSuffix(baseURL, "/")
	return baseURL + "/api/auth/validate"
}

// writeAuthError writes an authentication error response
func (am *AuthMiddleware) writeAuthError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"error":     message,
		"status":    statusCode,
		"timestamp": time.Now(),
	}

	json.NewEncoder(w).Encode(response)
}

// GetServerIDFromContext extracts the server ID from request context
func GetServerIDFromContext(ctx context.Context) string {
	if serverID, ok := ctx.Value("server_id").(string); ok {
		return serverID
	}
	return ""
}