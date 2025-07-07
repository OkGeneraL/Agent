package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"superagent/internal/storage"
	"superagent/internal/logging"

	"github.com/sirupsen/logrus"
)

// TokenManager handles the lifecycle of API tokens
type TokenManager struct {
	store       *storage.SecureStore
	auditLogger *logging.AuditLogger
	currentToken *Token
	refreshChan  chan struct{}
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// Token represents an API authentication token
type Token struct {
	Value     string    `json:"value"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	TokenID   string    `json:"token_id"`
	Scope     []string  `json:"scope"`
}

// TokenInfo contains token metadata without the sensitive value
type TokenInfo struct {
	TokenID   string    `json:"token_id"`
	ExpiresAt time.Time `json:"expires_at"`
	IssuedAt  time.Time `json:"issued_at"`
	Scope     []string  `json:"scope"`
	IsValid   bool      `json:"is_valid"`
}

// NewTokenManager creates a new token manager instance
func NewTokenManager(store *storage.SecureStore, auditLogger *logging.AuditLogger) *TokenManager {
	ctx, cancel := context.WithCancel(context.Background())
	
	tm := &TokenManager{
		store:       store,
		auditLogger: auditLogger,
		refreshChan: make(chan struct{}, 1),
		ctx:         ctx,
		cancel:      cancel,
	}

	return tm
}

// Start initializes the token manager and starts background processes
func (tm *TokenManager) Start() error {
	logrus.Info("Starting token manager")

	// Load existing token from secure storage
	if err := tm.loadToken(); err != nil {
		logrus.Warnf("Failed to load existing token: %v", err)
		tm.auditLogger.LogSecurityEvent("TOKEN_LOAD_FAILED", false, map[string]interface{}{
			"error": err.Error(),
		})
	}

	// Start token refresh monitoring
	tm.wg.Add(1)
	go tm.tokenRefreshWorker()

	tm.auditLogger.LogSecurityEvent("TOKEN_MANAGER_STARTED", true, map[string]interface{}{
		"has_token": tm.currentToken != nil,
	})

	return nil
}

// GetToken returns the current valid token in a thread-safe manner
func (tm *TokenManager) GetToken() (string, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.currentToken == nil {
		return "", fmt.Errorf("no token available")
	}

	if tm.isTokenExpired(tm.currentToken) {
		return "", fmt.Errorf("token expired")
	}

	return tm.currentToken.Value, nil
}

// SetToken sets a new token and stores it securely
func (tm *TokenManager) SetToken(tokenValue string, expiresAt time.Time, scope []string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	tokenID, err := generateTokenID()
	if err != nil {
		return fmt.Errorf("failed to generate token ID: %w", err)
	}

	newToken := &Token{
		Value:     tokenValue,
		ExpiresAt: expiresAt,
		IssuedAt:  time.Now(),
		TokenID:   tokenID,
		Scope:     scope,
	}

	// Convert token to storage format
	tokenData := map[string]interface{}{
		"value":      newToken.Value,
		"expires_at": newToken.ExpiresAt.Format(time.RFC3339),
		"issued_at":  newToken.IssuedAt.Format(time.RFC3339),
		"token_id":   newToken.TokenID,
		"scope":      newToken.Scope,
	}

	// Store token securely
	if err := tm.store.StoreToken(tokenData); err != nil {
		tm.auditLogger.LogSecurityEvent("TOKEN_STORE_FAILED", false, map[string]interface{}{
			"token_id": tokenID,
			"error":    err.Error(),
		})
		return fmt.Errorf("failed to store token: %w", err)
	}

	// Update current token
	oldTokenID := ""
	if tm.currentToken != nil {
		oldTokenID = tm.currentToken.TokenID
	}
	tm.currentToken = newToken

	tm.auditLogger.LogSecurityEvent("TOKEN_SET", true, map[string]interface{}{
		"new_token_id": tokenID,
		"old_token_id": oldTokenID,
		"expires_at":   expiresAt,
		"scope":        scope,
	})

	logrus.Infof("Token updated successfully, expires at: %v", expiresAt)
	return nil
}

// RefreshToken triggers a token refresh
func (tm *TokenManager) RefreshToken() {
	select {
	case tm.refreshChan <- struct{}{}:
		logrus.Debug("Token refresh triggered")
	default:
		logrus.Debug("Token refresh already pending")
	}
}

// RevokeToken revokes the current token
func (tm *TokenManager) RevokeToken() error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	if tm.currentToken == nil {
		return fmt.Errorf("no token to revoke")
	}

	tokenID := tm.currentToken.TokenID

	// Remove from storage
	if err := tm.store.DeleteToken(tokenID); err != nil {
		tm.auditLogger.LogSecurityEvent("TOKEN_REVOKE_FAILED", false, map[string]interface{}{
			"token_id": tokenID,
			"error":    err.Error(),
		})
		return fmt.Errorf("failed to revoke token: %w", err)
	}

	tm.currentToken = nil

	tm.auditLogger.LogSecurityEvent("TOKEN_REVOKED", true, map[string]interface{}{
		"token_id": tokenID,
	})

	logrus.Info("Token revoked successfully")
	return nil
}

// GetTokenInfo returns metadata about the current token without exposing the value
func (tm *TokenManager) GetTokenInfo() *TokenInfo {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.currentToken == nil {
		return nil
	}

	return &TokenInfo{
		TokenID:   tm.currentToken.TokenID,
		ExpiresAt: tm.currentToken.ExpiresAt,
		IssuedAt:  tm.currentToken.IssuedAt,
		Scope:     tm.currentToken.Scope,
		IsValid:   !tm.isTokenExpired(tm.currentToken),
	}
}

// IsTokenValid checks if the current token is valid and not expired
func (tm *TokenManager) IsTokenValid() bool {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.currentToken == nil {
		return false
	}

	return !tm.isTokenExpired(tm.currentToken)
}

// TimeUntilExpiry returns the duration until the current token expires
func (tm *TokenManager) TimeUntilExpiry() time.Duration {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.currentToken == nil {
		return 0
	}

	return time.Until(tm.currentToken.ExpiresAt)
}

// Stop gracefully stops the token manager
func (tm *TokenManager) Stop() error {
	logrus.Info("Stopping token manager")

	tm.cancel()
	tm.wg.Wait()

	tm.auditLogger.LogSecurityEvent("TOKEN_MANAGER_STOPPED", true, map[string]interface{}{})

	return nil
}

// loadToken loads the token from secure storage
func (tm *TokenManager) loadToken() error {
	tokenData, err := tm.store.LoadToken()
	if err != nil {
		return fmt.Errorf("failed to load token from storage: %w", err)
	}

	if tokenData == nil {
		logrus.Info("No existing token found in storage")
		return nil
	}

	// Parse token data
	token := &Token{}
	
	if value, ok := tokenData["value"].(string); ok {
		token.Value = value
	}
	
	if tokenID, ok := tokenData["token_id"].(string); ok {
		token.TokenID = tokenID
	}
	
	if expiresAtStr, ok := tokenData["expires_at"].(string); ok {
		if expiresAt, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
			token.ExpiresAt = expiresAt
		}
	}
	
	if issuedAtStr, ok := tokenData["issued_at"].(string); ok {
		if issuedAt, err := time.Parse(time.RFC3339, issuedAtStr); err == nil {
			token.IssuedAt = issuedAt
		}
	}
	
	if scope, ok := tokenData["scope"].([]interface{}); ok {
		token.Scope = make([]string, len(scope))
		for i, s := range scope {
			if str, ok := s.(string); ok {
				token.Scope[i] = str
			}
		}
	}

	tm.mu.Lock()
	defer tm.mu.Unlock()

	tm.currentToken = token

	if tm.isTokenExpired(token) {
		logrus.Warn("Loaded token is expired")
		tm.auditLogger.LogSecurityEvent("TOKEN_LOADED_EXPIRED", false, map[string]interface{}{
			"token_id":   token.TokenID,
			"expires_at": token.ExpiresAt,
		})
		return fmt.Errorf("loaded token is expired")
	}

	tm.auditLogger.LogSecurityEvent("TOKEN_LOADED", true, map[string]interface{}{
		"token_id":   token.TokenID,
		"expires_at": token.ExpiresAt,
	})

	logrus.Infof("Token loaded successfully, expires at: %v", token.ExpiresAt)
	return nil
}

// isTokenExpired checks if a token is expired (with 5-minute buffer)
func (tm *TokenManager) isTokenExpired(token *Token) bool {
	if token == nil {
		return true
	}
	
	// Add 5-minute buffer to prevent last-minute failures
	expiryWithBuffer := token.ExpiresAt.Add(-5 * time.Minute)
	return time.Now().After(expiryWithBuffer)
}

// tokenRefreshWorker monitors token expiry and handles refresh requests
func (tm *TokenManager) tokenRefreshWorker() {
	defer tm.wg.Done()

	ticker := time.NewTicker(1 * time.Minute) // Check every minute
	defer ticker.Stop()

	for {
		select {
		case <-tm.ctx.Done():
			return

		case <-ticker.C:
			// Check if token needs refresh (1 hour before expiry)
			tm.mu.RLock()
			needsRefresh := tm.currentToken != nil && 
				time.Until(tm.currentToken.ExpiresAt) < 1*time.Hour
			tm.mu.RUnlock()

			if needsRefresh {
				tm.auditLogger.LogSecurityEvent("TOKEN_AUTO_REFRESH_NEEDED", true, map[string]interface{}{
					"expires_in": tm.TimeUntilExpiry().String(),
				})
				logrus.Info("Token approaching expiry, refresh needed")
			}

		case <-tm.refreshChan:
			tm.auditLogger.LogSecurityEvent("TOKEN_REFRESH_REQUESTED", true, map[string]interface{}{})
			logrus.Info("Token refresh requested")
			// The actual refresh will be handled by the API client
		}
	}
}

// generateTokenID generates a unique token ID
func generateTokenID() (string, error) {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateTokenScope checks if the token has the required scope
func (tm *TokenManager) ValidateTokenScope(requiredScope string) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	if tm.currentToken == nil {
		return fmt.Errorf("no token available")
	}

	// Check if token has required scope
	for _, scope := range tm.currentToken.Scope {
		if scope == requiredScope || scope == "*" {
			return nil
		}
	}

	tm.auditLogger.LogSecurityEvent("TOKEN_SCOPE_VALIDATION_FAILED", false, map[string]interface{}{
		"required_scope": requiredScope,
		"token_scope":    tm.currentToken.Scope,
	})

	return fmt.Errorf("token does not have required scope: %s", requiredScope)
}