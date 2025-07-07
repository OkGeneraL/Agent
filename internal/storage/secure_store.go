package storage

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"superagent/internal/logging"

	"golang.org/x/crypto/pbkdf2"
)

// SecureStore handles encrypted storage of sensitive data
type SecureStore struct {
	storePath   string
	encryptKey  []byte
	auditLogger *logging.AuditLogger
	mu          sync.RWMutex
}

// StoredData represents the structure of encrypted stored data
type StoredData struct {
	Version   int                    `json:"version"`
	Timestamp time.Time              `json:"timestamp"`
	Data      map[string]interface{} `json:"data"`
	Checksum  string                 `json:"checksum"`
}

// EncryptedFile represents an encrypted file on disk
type EncryptedFile struct {
	Version int    `json:"version"`
	Salt    string `json:"salt"`
	Nonce   string `json:"nonce"`
	Data    string `json:"data"`
}

// NewSecureStore creates a new secure storage instance
func NewSecureStore(storePath string, encryptionKey string, auditLogger *logging.AuditLogger) (*SecureStore, error) {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(filepath.Dir(storePath), 0700); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Generate encryption key from password
	salt := []byte("superagent-salt") // In production, use a random salt
	encryptKey := pbkdf2.Key([]byte(encryptionKey), salt, 10000, 32, sha256.New)

	store := &SecureStore{
		storePath:   storePath,
		encryptKey:  encryptKey,
		auditLogger: auditLogger,
	}

	// Initialize empty storage if file doesn't exist
	if _, err := os.Stat(storePath); os.IsNotExist(err) {
		if err := store.initialize(); err != nil {
			return nil, fmt.Errorf("failed to initialize storage: %w", err)
		}
	}

	auditLogger.LogSecurityEvent("SECURE_STORE_INITIALIZED", true, map[string]interface{}{
		"store_path": storePath,
	})

	return store, nil
}

// StoreToken stores an authentication token securely
func (s *SecureStore) StoreToken(tokenData map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.loadData()
	if err != nil {
		return fmt.Errorf("failed to load existing data: %w", err)
	}

	// Store token data
	data.Data["token"] = tokenData
	data.Timestamp = time.Now()

	if err := s.saveData(data); err != nil {
		tokenID := ""
		if id, ok := tokenData["token_id"].(string); ok {
			tokenID = id
		}
		s.auditLogger.LogSecurityEvent("TOKEN_STORE_FAILED", false, map[string]interface{}{
			"token_id": tokenID,
			"error":    err.Error(),
		})
		return fmt.Errorf("failed to save token: %w", err)
	}

	tokenID := ""
	if id, ok := tokenData["token_id"].(string); ok {
		tokenID = id
	}
	s.auditLogger.LogSecurityEvent("TOKEN_STORED", true, map[string]interface{}{
		"token_id": tokenID,
	})

	return nil
}

// LoadToken loads a stored authentication token
func (s *SecureStore) LoadToken() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.loadData()
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	tokenData, exists := data.Data["token"]
	if !exists {
		return nil, nil // No token stored
	}

	tokenMap, ok := tokenData.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid token data format")
	}

	tokenID := ""
	if id, ok := tokenMap["token_id"].(string); ok {
		tokenID = id
	}
	s.auditLogger.LogSecurityEvent("TOKEN_LOADED", true, map[string]interface{}{
		"token_id": tokenID,
	})

	return tokenMap, nil
}

// DeleteToken removes a stored token
func (s *SecureStore) DeleteToken(tokenID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.loadData()
	if err != nil {
		return fmt.Errorf("failed to load data: %w", err)
	}

	delete(data.Data, "token")
	data.Timestamp = time.Now()

	if err := s.saveData(data); err != nil {
		s.auditLogger.LogSecurityEvent("TOKEN_DELETE_FAILED", false, map[string]interface{}{
			"token_id": tokenID,
			"error":    err.Error(),
		})
		return fmt.Errorf("failed to delete token: %w", err)
	}

	s.auditLogger.LogSecurityEvent("TOKEN_DELETED", true, map[string]interface{}{
		"token_id": tokenID,
	})

	return nil
}

// StoreDeploymentState stores deployment state information
func (s *SecureStore) StoreDeploymentState(deploymentID string, state map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.loadData()
	if err != nil {
		return fmt.Errorf("failed to load data: %w", err)
	}

	if data.Data["deployments"] == nil {
		data.Data["deployments"] = make(map[string]interface{})
	}

	deployments := data.Data["deployments"].(map[string]interface{})
	deployments[deploymentID] = state
	data.Timestamp = time.Now()

	if err := s.saveData(data); err != nil {
		s.auditLogger.LogSecurityEvent("DEPLOYMENT_STATE_STORE_FAILED", false, map[string]interface{}{
			"deployment_id": deploymentID,
			"error":         err.Error(),
		})
		return fmt.Errorf("failed to store deployment state: %w", err)
	}

	s.auditLogger.LogEvent("DEPLOYMENT_STATE_STORED", map[string]interface{}{
		"deployment_id": deploymentID,
	})

	return nil
}

// LoadDeploymentState loads deployment state information
func (s *SecureStore) LoadDeploymentState(deploymentID string) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.loadData()
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	deployments, exists := data.Data["deployments"]
	if !exists {
		return nil, nil
	}

	deploymentsMap, ok := deployments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid deployments data format")
	}

	state, exists := deploymentsMap[deploymentID]
	if !exists {
		return nil, nil
	}

	stateMap, ok := state.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid deployment state format")
	}

	return stateMap, nil
}

// DeleteDeploymentState removes deployment state information
func (s *SecureStore) DeleteDeploymentState(deploymentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.loadData()
	if err != nil {
		return fmt.Errorf("failed to load data: %w", err)
	}

	deployments, exists := data.Data["deployments"]
	if !exists {
		return nil
	}

	deploymentsMap, ok := deployments.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid deployments data format")
	}

	delete(deploymentsMap, deploymentID)
	data.Timestamp = time.Now()

	if err := s.saveData(data); err != nil {
		s.auditLogger.LogSecurityEvent("DEPLOYMENT_STATE_DELETE_FAILED", false, map[string]interface{}{
			"deployment_id": deploymentID,
			"error":         err.Error(),
		})
		return fmt.Errorf("failed to delete deployment state: %w", err)
	}

	s.auditLogger.LogEvent("DEPLOYMENT_STATE_DELETED", map[string]interface{}{
		"deployment_id": deploymentID,
	})

	return nil
}

// ListDeployments returns all stored deployment IDs
func (s *SecureStore) ListDeployments() ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.loadData()
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	deployments, exists := data.Data["deployments"]
	if !exists {
		return []string{}, nil
	}

	deploymentsMap, ok := deployments.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid deployments data format")
	}

	var deploymentIDs []string
	for deploymentID := range deploymentsMap {
		deploymentIDs = append(deploymentIDs, deploymentID)
	}

	return deploymentIDs, nil
}

// StoreConfig stores configuration data
func (s *SecureStore) StoreConfig(config map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := s.loadData()
	if err != nil {
		return fmt.Errorf("failed to load data: %w", err)
	}

	data.Data["config"] = config
	data.Timestamp = time.Now()

	if err := s.saveData(data); err != nil {
		s.auditLogger.LogSecurityEvent("CONFIG_STORE_FAILED", false, map[string]interface{}{
			"error": err.Error(),
		})
		return fmt.Errorf("failed to store config: %w", err)
	}

	s.auditLogger.LogEvent("CONFIG_STORED", map[string]interface{}{})

	return nil
}

// LoadConfig loads configuration data
func (s *SecureStore) LoadConfig() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := s.loadData()
	if err != nil {
		return nil, fmt.Errorf("failed to load data: %w", err)
	}

	config, exists := data.Data["config"]
	if !exists {
		return make(map[string]interface{}), nil
	}

	configMap, ok := config.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid config data format")
	}

	return configMap, nil
}

// initialize creates an empty encrypted storage file
func (s *SecureStore) initialize() error {
	data := &StoredData{
		Version:   1,
		Timestamp: time.Now(),
		Data:      make(map[string]interface{}),
	}

	return s.saveData(data)
}

// loadData loads and decrypts data from storage
func (s *SecureStore) loadData() (*StoredData, error) {
	file, err := os.Open(s.storePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open storage file: %w", err)
	}
	defer file.Close()

	var encryptedFile EncryptedFile
	if err := json.NewDecoder(file).Decode(&encryptedFile); err != nil {
		return nil, fmt.Errorf("failed to decode encrypted file: %w", err)
	}

	// Decrypt data
	decryptedData, err := s.decrypt(encryptedFile.Data, encryptedFile.Nonce)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	var data StoredData
	if err := json.Unmarshal(decryptedData, &data); err != nil {
		return nil, fmt.Errorf("failed to unmarshal decrypted data: %w", err)
	}

	// Verify checksum
	expectedChecksum := s.calculateChecksum(data.Data)
	if data.Checksum != expectedChecksum {
		s.auditLogger.LogSecurityEvent("DATA_INTEGRITY_VIOLATION", false, map[string]interface{}{
			"expected_checksum": expectedChecksum,
			"actual_checksum":   data.Checksum,
		})
		return nil, fmt.Errorf("data integrity check failed")
	}

	return &data, nil
}

// saveData encrypts and saves data to storage
func (s *SecureStore) saveData(data *StoredData) error {
	// Calculate checksum
	data.Checksum = s.calculateChecksum(data.Data)

	// Marshal data
	plaintext, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal data: %w", err)
	}

	// Encrypt data
	encryptedData, nonce, err := s.encrypt(plaintext)
	if err != nil {
		return fmt.Errorf("failed to encrypt data: %w", err)
	}

	encryptedFile := EncryptedFile{
		Version: 1,
		Salt:    hex.EncodeToString([]byte("superagent-salt")),
		Nonce:   hex.EncodeToString(nonce),
		Data:    hex.EncodeToString(encryptedData),
	}

	// Write to temporary file first
	tempPath := s.storePath + ".tmp"
	file, err := os.OpenFile(tempPath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}

	if err := json.NewEncoder(file).Encode(encryptedFile); err != nil {
		file.Close()
		os.Remove(tempPath)
		return fmt.Errorf("failed to encode encrypted file: %w", err)
	}

	file.Close()

	// Atomic rename
	if err := os.Rename(tempPath, s.storePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// encrypt encrypts data using AES-GCM
func (s *SecureStore) encrypt(plaintext []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(s.encryptKey)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	return ciphertext, nonce, nil
}

// decrypt decrypts data using AES-GCM
func (s *SecureStore) decrypt(encryptedDataHex string, nonceHex string) ([]byte, error) {
	encryptedData, err := hex.DecodeString(encryptedDataHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode encrypted data: %w", err)
	}

	nonce, err := hex.DecodeString(nonceHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode nonce: %w", err)
	}

	block, err := aes.NewCipher(s.encryptKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, encryptedData, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt data: %w", err)
	}

	return plaintext, nil
}

// calculateChecksum calculates SHA256 checksum of data
func (s *SecureStore) calculateChecksum(data map[string]interface{}) string {
	// Convert to JSON for consistent hashing
	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// GetStats returns storage statistics
func (s *SecureStore) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]interface{})
	
	if fileInfo, err := os.Stat(s.storePath); err == nil {
		stats["file_size"] = fileInfo.Size()
		stats["last_modified"] = fileInfo.ModTime()
	}

	if data, err := s.loadData(); err == nil {
		stats["version"] = data.Version
		stats["timestamp"] = data.Timestamp
		stats["data_keys"] = len(data.Data)
		
		// Count deployments
		if deployments, exists := data.Data["deployments"]; exists {
			if deploymentsMap, ok := deployments.(map[string]interface{}); ok {
				stats["deployment_count"] = len(deploymentsMap)
			}
		}
	}

	return stats
}

// Backup creates a backup of the storage file
func (s *SecureStore) Backup(backupPath string) error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	sourceFile, err := os.Open(s.storePath)
	if err != nil {
		return fmt.Errorf("failed to open source file: %w", err)
	}
	defer sourceFile.Close()

	if err := os.MkdirAll(filepath.Dir(backupPath), 0700); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	destFile, err := os.Create(backupPath)
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	s.auditLogger.LogEvent("STORAGE_BACKUP_CREATED", map[string]interface{}{
		"backup_path": backupPath,
	})

	return nil
}

// Restore restores storage from a backup file
func (s *SecureStore) Restore(backupPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sourceFile, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer sourceFile.Close()

	tempPath := s.storePath + ".restore"
	destFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	destFile.Close()

	// Verify the restored file can be loaded
	tempStore := &SecureStore{
		storePath:  tempPath,
		encryptKey: s.encryptKey,
	}

	if _, err := tempStore.loadData(); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("backup file is corrupted: %w", err)
	}

	// Atomic replace
	if err := os.Rename(tempPath, s.storePath); err != nil {
		os.Remove(tempPath)
		return fmt.Errorf("failed to replace storage file: %w", err)
	}

	s.auditLogger.LogEvent("STORAGE_RESTORED", map[string]interface{}{
		"backup_path": backupPath,
	})

	return nil
}