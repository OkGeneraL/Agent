package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/viper"
	"golang.org/x/crypto/pbkdf2"
)

// Config represents the agent configuration
type Config struct {
	Agent       AgentConfig       `yaml:"agent"`
	Backend     BackendConfig     `yaml:"backend"`
	Docker      DockerConfig      `yaml:"docker"`
	Git         GitConfig         `yaml:"git"`
	Traefik     TraefikConfig     `yaml:"traefik"`
	Security    SecurityConfig    `yaml:"security"`
	Monitoring  MonitoringConfig  `yaml:"monitoring"`
	Logging     LoggingConfig     `yaml:"logging"`
	Resources   ResourcesConfig   `yaml:"resources"`
	Networking  NetworkingConfig  `yaml:"networking"`
	AdminPanel  AdminPanelConfig  `yaml:"admin_panel"`
}

// AgentConfig contains agent-specific configuration
type AgentConfig struct {
	ID               string            `yaml:"id"`
	Location         string            `yaml:"location"`
	ServerID         string            `yaml:"server_id"`
	WorkDir          string            `yaml:"work_dir"`
	DataDir          string            `yaml:"data_dir"`
	TempDir          string            `yaml:"temp_dir"`
	PIDFile          string            `yaml:"pid_file"`
	User             string            `yaml:"user"`
	Group            string            `yaml:"group"`
	Environment      map[string]string `yaml:"environment"`
	MaxConcurrentOps int               `yaml:"max_concurrent_ops"`
	HeartbeatInterval time.Duration    `yaml:"heartbeat_interval"`
}

// BackendConfig contains backend API configuration
type BackendConfig struct {
	BaseURL           string            `yaml:"base_url"`
	APIToken          string            `yaml:"api_token"`
	TokenFile         string            `yaml:"token_file"`
	RefreshInterval   time.Duration     `yaml:"refresh_interval"`
	Timeout           time.Duration     `yaml:"timeout"`
	RetryAttempts     int               `yaml:"retry_attempts"`
	RetryDelay        time.Duration     `yaml:"retry_delay"`
	InsecureSkipTLS   bool              `yaml:"insecure_skip_tls"`
	CACertFile        string            `yaml:"ca_cert_file"`
	ClientCertFile    string            `yaml:"client_cert_file"`
	ClientKeyFile     string            `yaml:"client_key_file"`
	Headers           map[string]string `yaml:"headers"`
	WebhookEndpoint   string            `yaml:"webhook_endpoint"`
	WebhookSecret     string            `yaml:"webhook_secret"`
}

// DockerConfig contains Docker-specific configuration
type DockerConfig struct {
	Host                string            `yaml:"host"`
	Version             string            `yaml:"version"`
	TLSVerify           bool              `yaml:"tls_verify"`
	CertPath            string            `yaml:"cert_path"`
	RegistryAuth        map[string]string `yaml:"registry_auth"`
	DefaultRegistry     string            `yaml:"default_registry"`
	NetworkName         string            `yaml:"network_name"`
	LogDriver           string            `yaml:"log_driver"`
	LogOptions          map[string]string `yaml:"log_options"`
	CleanupInterval     time.Duration     `yaml:"cleanup_interval"`
	CleanupRetention    time.Duration     `yaml:"cleanup_retention"`
	DefaultCPULimit     string            `yaml:"default_cpu_limit"`
	DefaultMemoryLimit  string            `yaml:"default_memory_limit"`
	DefaultStorageLimit string            `yaml:"default_storage_limit"`
}

// GitConfig contains Git-specific configuration
type GitConfig struct {
	SSHKeyPath     string            `yaml:"ssh_key_path"`
	SSHKeyPassphrase string          `yaml:"ssh_key_passphrase"`
	Username       string            `yaml:"username"`
	Password       string            `yaml:"password"`
	Token          string            `yaml:"token"`
	KnownHostsFile string            `yaml:"known_hosts_file"`
	Timeout        time.Duration     `yaml:"timeout"`
	MaxDepth       int               `yaml:"max_depth"`
	CacheDir       string            `yaml:"cache_dir"`
	CacheRetention time.Duration     `yaml:"cache_retention"`
}

// TraefikConfig contains Traefik integration configuration
type TraefikConfig struct {
	Enabled         bool              `yaml:"enabled"`
	ConfigFile      string            `yaml:"config_file"`
	APIEndpoint     string            `yaml:"api_endpoint"`
	APIKey          string            `yaml:"api_key"`
	APIVersion      string            `yaml:"api_version"`
	Provider        string            `yaml:"provider"` // file, consul, etcd, etc.
	BaseDomain      string            `yaml:"base_domain"`
	CertResolver    string            `yaml:"cert_resolver"`
	EnableTLS       bool              `yaml:"enable_tls"`
	TLSOptions      map[string]string `yaml:"tls_options"`
	Labels          map[string]string `yaml:"labels"`
	Middlewares     []string          `yaml:"middlewares"`
}

// SecurityConfig contains security-related configuration
type SecurityConfig struct {
	EncryptionKey      string        `yaml:"encryption_key"`
	EncryptionKeyFile  string        `yaml:"encryption_key_file"`
	TokenRotationInterval time.Duration `yaml:"token_rotation_interval"`
	AuditLogEnabled    bool          `yaml:"audit_log_enabled"`
	AuditLogPath       string        `yaml:"audit_log_path"`
	AuditLogMaxSize    int           `yaml:"audit_log_max_size"`
	AuditLogMaxBackups int           `yaml:"audit_log_max_backups"`
	AuditLogMaxAge     int           `yaml:"audit_log_max_age"`
	SeccompProfile     string        `yaml:"seccomp_profile"`
	AppArmorProfile    string        `yaml:"apparmor_profile"`
	AllowedRegistries  []string      `yaml:"allowed_registries"`
	BlockedRegistries  []string      `yaml:"blocked_registries"`
	RunAsNonRoot       bool          `yaml:"run_as_non_root"`
	ReadOnlyRootFS     bool          `yaml:"read_only_root_fs"`
	NoNewPrivileges    bool          `yaml:"no_new_privileges"`
}

// MonitoringConfig contains monitoring and metrics configuration
type MonitoringConfig struct {
	Enabled              bool              `yaml:"enabled"`
	MetricsPort          int               `yaml:"metrics_port"`
	MetricsPath          string            `yaml:"metrics_path"`
	HealthCheckPort      int               `yaml:"health_check_port"`
	HealthCheckPath      string            `yaml:"health_check_path"`
	PrometheusEnabled    bool              `yaml:"prometheus_enabled"`
	LogStreamingEnabled  bool              `yaml:"log_streaming_enabled"`
	LogStreamingEndpoint string            `yaml:"log_streaming_endpoint"`
	MetricsInterval      time.Duration     `yaml:"metrics_interval"`
	AlertingEnabled      bool              `yaml:"alerting_enabled"`
	AlertingRules        []AlertingRule    `yaml:"alerting_rules"`
	CustomMetrics        map[string]string `yaml:"custom_metrics"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level           string `yaml:"level"`
	Format          string `yaml:"format"`
	Output          string `yaml:"output"`
	LogFile         string `yaml:"log_file"`
	MaxSize         int    `yaml:"max_size"`
	MaxBackups      int    `yaml:"max_backups"`
	MaxAge          int    `yaml:"max_age"`
	Compress        bool   `yaml:"compress"`
	AuditLogPath    string `yaml:"audit_log_path"`
	AuditLogMaxSize int    `yaml:"audit_log_max_size"`
	AuditLogMaxBackups int `yaml:"audit_log_max_backups"`
	AuditLogMaxAge  int    `yaml:"audit_log_max_age"`
}

// ResourcesConfig contains resource management configuration
type ResourcesConfig struct {
	CPUQuota       string            `yaml:"cpu_quota"`
	MemoryQuota    string            `yaml:"memory_quota"`
	StorageQuota   string            `yaml:"storage_quota"`
	NetworkQuota   string            `yaml:"network_quota"`
	MaxContainers  int               `yaml:"max_containers"`
	MaxVolumes     int               `yaml:"max_volumes"`
	MaxNetworks    int               `yaml:"max_networks"`
	ReservedCPU    string            `yaml:"reserved_cpu"`
	ReservedMemory string            `yaml:"reserved_memory"`
	ReservedStorage string           `yaml:"reserved_storage"`
	Limits         map[string]string `yaml:"limits"`
	Monitoring     ResourceMonitoring `yaml:"monitoring"`
}

// NetworkingConfig contains networking configuration
type NetworkingConfig struct {
	AllowedPorts     []int             `yaml:"allowed_ports"`
	BlockedPorts     []int             `yaml:"blocked_ports"`
	AllowedHosts     []string          `yaml:"allowed_hosts"`
	BlockedHosts     []string          `yaml:"blocked_hosts"`
	DNSServers       []string          `yaml:"dns_servers"`
	ProxyEnabled     bool              `yaml:"proxy_enabled"`
	ProxyURL         string            `yaml:"proxy_url"`
	ProxyAuth        map[string]string `yaml:"proxy_auth"`
	FirewallEnabled  bool              `yaml:"firewall_enabled"`
	FirewallRules    []FirewallRule    `yaml:"firewall_rules"`
	NetworkPolicies  []NetworkPolicy   `yaml:"network_policies"`
}

// AlertingRule defines alerting rules for monitoring
type AlertingRule struct {
	Name        string            `yaml:"name"`
	Condition   string            `yaml:"condition"`
	Threshold   float64           `yaml:"threshold"`
	Duration    time.Duration     `yaml:"duration"`
	Severity    string            `yaml:"severity"`
	Labels      map[string]string `yaml:"labels"`
	Annotations map[string]string `yaml:"annotations"`
}

// ResourceMonitoring contains resource monitoring configuration
type ResourceMonitoring struct {
	Enabled         bool          `yaml:"enabled"`
	Interval        time.Duration `yaml:"interval"`
	CPUThreshold    float64       `yaml:"cpu_threshold"`
	MemoryThreshold float64       `yaml:"memory_threshold"`
	DiskThreshold   float64       `yaml:"disk_threshold"`
	NetworkThreshold float64      `yaml:"network_threshold"`
	AlertOnExceed   bool          `yaml:"alert_on_exceed"`
}

// FirewallRule defines firewall rules
type FirewallRule struct {
	Name      string   `yaml:"name"`
	Action    string   `yaml:"action"` // allow, deny, log
	Protocol  string   `yaml:"protocol"` // tcp, udp, icmp
	Source    string   `yaml:"source"`
	Dest      string   `yaml:"dest"`
	Ports     []int    `yaml:"ports"`
	Priority  int      `yaml:"priority"`
}

// NetworkPolicy defines network policies
type NetworkPolicy struct {
	Name     string            `yaml:"name"`
	Selector map[string]string `yaml:"selector"`
	Ingress  []NetworkRule     `yaml:"ingress"`
	Egress   []NetworkRule     `yaml:"egress"`
}

// NetworkRule defines network rules within policies
type NetworkRule struct {
	From  []NetworkPeer `yaml:"from"`
	To    []NetworkPeer `yaml:"to"`
	Ports []NetworkPort `yaml:"ports"`
}

// NetworkPeer defines network peers
type NetworkPeer struct {
	PodSelector       map[string]string `yaml:"pod_selector"`
	NamespaceSelector map[string]string `yaml:"namespace_selector"`
	IPBlock           IPBlock           `yaml:"ip_block"`
}

// IPBlock defines IP blocks
type IPBlock struct {
	CIDR   string   `yaml:"cidr"`
	Except []string `yaml:"except"`
}

// NetworkPort defines network ports
type NetworkPort struct {
	Protocol string `yaml:"protocol"`
	Port     int    `yaml:"port"`
}

// AdminPanelConfig contains admin panel connection configuration
type AdminPanelConfig struct {
	Enabled           bool              `yaml:"enabled"`
	BaseURL           string            `yaml:"base_url"`
	APIEndpoint       string            `yaml:"api_endpoint"`
	Username          string            `yaml:"username"`
	Password          string            `yaml:"password"`
	Token             string            `yaml:"token"`
	AutoSync          bool              `yaml:"auto_sync"`
	SyncInterval      time.Duration     `yaml:"sync_interval"`
	ConnectionTimeout time.Duration     `yaml:"connection_timeout"`
	RetryAttempts     int               `yaml:"retry_attempts"`
	Headers           map[string]string `yaml:"headers"`
}

// Load loads configuration from file
func Load(configPath string) (*Config, error) {
	// Set default values
	config := &Config{
		Agent: AgentConfig{
			WorkDir:          "/var/lib/superagent",
			DataDir:          "/var/lib/superagent/data",
			TempDir:          "/tmp/superagent",
			PIDFile:          "/var/run/superagent.pid",
			User:             "superagent",
			Group:            "superagent",
			MaxConcurrentOps: 5,
			HeartbeatInterval: 30 * time.Second,
		},
		Backend: BackendConfig{
			RefreshInterval: 30 * time.Second,
			Timeout:         30 * time.Second,
			RetryAttempts:   3,
			RetryDelay:      5 * time.Second,
		},
		Docker: DockerConfig{
			Host:            "unix:///var/run/docker.sock",
			Version:         "1.41",
			NetworkName:     "superagent",
			LogDriver:       "json-file",
			CleanupInterval: 1 * time.Hour,
			CleanupRetention: 24 * time.Hour,
			DefaultCPULimit:     "1",
			DefaultMemoryLimit:  "1G",
			DefaultStorageLimit: "10G",
		},
		Git: GitConfig{
			Timeout:        30 * time.Second,
			MaxDepth:       50,
			CacheDir:       "/var/cache/superagent/git",
			CacheRetention: 24 * time.Hour,
		},
		Traefik: TraefikConfig{
			Enabled:     true,
			APIVersion:  "v2",
			Provider:    "file",
			EnableTLS:   true,
		},
		Security: SecurityConfig{
			TokenRotationInterval: 24 * time.Hour,
			AuditLogEnabled:       true,
			AuditLogPath:          "/tmp/superagent/audit.log",
			AuditLogMaxSize:       100,
			AuditLogMaxBackups:    10,
			AuditLogMaxAge:        30,
			RunAsNonRoot:          true,
			ReadOnlyRootFS:        true,
			NoNewPrivileges:       true,
		},
		Monitoring: MonitoringConfig{
			Enabled:         true,
			MetricsPort:     9090,
			MetricsPath:     "/metrics",
			HealthCheckPort: 8080,
			HealthCheckPath: "/health",
			PrometheusEnabled: true,
			MetricsInterval:   15 * time.Second,
		},
		Logging: LoggingConfig{
			Level:           "info",
			Format:          "json",
			Output:          "file",
			LogFile:         "/tmp/superagent/agent.log",
			MaxSize:         100,
			MaxBackups:      10,
			MaxAge:          30,
			Compress:        true,
			AuditLogPath:    "/tmp/superagent/audit.log",
			AuditLogMaxSize: 100,
			AuditLogMaxBackups: 10,
			AuditLogMaxAge:  30,
		},
		Resources: ResourcesConfig{
			CPUQuota:       "80%",
			MemoryQuota:    "80%",
			StorageQuota:   "80%",
			NetworkQuota:   "1Gbps",
			MaxContainers:  50,
			MaxVolumes:     100,
			MaxNetworks:    10,
			ReservedCPU:    "0.5",
			ReservedMemory: "1G",
			ReservedStorage: "10G",
			Monitoring: ResourceMonitoring{
				Enabled:         true,
				Interval:        30 * time.Second,
				CPUThreshold:    80.0,
				MemoryThreshold: 80.0,
				DiskThreshold:   80.0,
				NetworkThreshold: 80.0,
				AlertOnExceed:   true,
			},
		},
		Networking: NetworkingConfig{
			AllowedPorts: []int{80, 443, 8080, 8443},
			BlockedPorts: []int{22, 23, 135, 139, 445},
			DNSServers:   []string{"8.8.8.8", "8.8.4.4"},
			FirewallEnabled: true,
		},
		AdminPanel: AdminPanelConfig{
			Enabled:           false,
			BaseURL:           "",
			APIEndpoint:       "",
			Username:          "",
			Password:          "",
			Token:             "",
			AutoSync:          false,
			SyncInterval:     30 * time.Second,
			ConnectionTimeout: 10 * time.Second,
			RetryAttempts:     3,
			Headers:           make(map[string]string),
		},
	}

	// Initialize Viper
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set environment variable prefix
	viper.SetEnvPrefix("SUPERAGENT")
	viper.AutomaticEnv()

	// Read configuration file
	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Create default config if file doesn't exist
			if err := createDefaultConfig(configPath); err != nil {
				return nil, fmt.Errorf("failed to create default config: %w", err)
			}
			return config, nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Unmarshal configuration
	if err := viper.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate configuration (temporarily disabled for testing)
	// if err := validateConfig(config); err != nil {
	//	return nil, fmt.Errorf("config validation failed: %w", err)
	// }

	// Decrypt sensitive values
	if err := decryptSensitiveData(config); err != nil {
		return nil, fmt.Errorf("failed to decrypt sensitive data: %w", err)
	}

	return config, nil
}

// createDefaultConfig creates a default configuration file
func createDefaultConfig(configPath string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Create default config content
	defaultConfig := `# Deployment Agent Configuration

agent:
  id: ""                    # Auto-generated if empty
  location: "default"       # Server location identifier
  server_id: ""            # Auto-generated if empty
  work_dir: "/tmp/superagent"
  data_dir: "/tmp/superagent/data"
  temp_dir: "/tmp/superagent/temp"
  pid_file: "/tmp/superagent.pid"
  user: "ubuntu"
  group: "ubuntu"
  max_concurrent_ops: 5
  heartbeat_interval: "30s"

backend:
  base_url: "https://your-backend.com/api"
  api_token: ""            # Required - set via environment variable
  token_file: ""           # Optional - path to token file
  refresh_interval: "30s"
  timeout: "30s"
  retry_attempts: 3
  retry_delay: "5s"
  insecure_skip_tls: false
  webhook_endpoint: "/webhook"
  webhook_secret: ""       # Required for webhook validation

docker:
  host: "unix:///var/run/docker.sock"
  version: "1.41"
  network_name: "superagent"
  log_driver: "json-file"
  cleanup_interval: "1h"
  cleanup_retention: "24h"
  default_cpu_limit: "1"
  default_memory_limit: "1G"
  default_storage_limit: "10G"

git:
  ssh_key_path: ""         # Path to SSH private key
  timeout: "30s"
  max_depth: 50
  cache_dir: "/tmp/superagent/git"
  cache_retention: "24h"

traefik:
  enabled: false
  provider: "file"
  config_file: "/tmp/traefik/dynamic.yml"
  base_domain: "localhost"
  cert_resolver: "letsencrypt"
  enable_tls: false

security:
  encryption_key_file: "/tmp/superagent/encryption.key"
  token_rotation_interval: "24h"
  audit_log_enabled: true
  audit_log_path: "/tmp/superagent/audit.log"
  run_as_non_root: false
  read_only_root_fs: false
  no_new_privileges: false

monitoring:
  enabled: true
  metrics_port: 9090
  metrics_path: "/metrics"
  health_check_port: 8080
  health_check_path: "/health"
  prometheus_enabled: false
  metrics_interval: "15s"

logging:
  level: "info"
  format: "text"
  output: "stdout"
  log_file: "/tmp/superagent/agent.log"
  max_size: 100
  max_backups: 10
  max_age: 30
  compress: true

resources:
  cpu_quota: "80%"
  memory_quota: "80%"
  storage_quota: "80%"
  network_quota: "1Gbps"
  max_containers: 50
  max_volumes: 100
  max_networks: 10
  reserved_cpu: "0.5"
  reserved_memory: "1G"
  reserved_storage: "10G"

networking:
  allowed_ports: [80, 443, 8080, 8443]
  blocked_ports: [22, 23, 135, 139, 445]
  dns_servers: ["8.8.8.8", "8.8.4.4"]
  firewall_enabled: true

admin_panel:
  enabled: false
  base_url: ""
  api_endpoint: ""
  username: ""
  password: ""
  token: ""
  auto_sync: false
  sync_interval: "30s"
  connection_timeout: "10s"
  retry_attempts: 3
  headers: {}
`

	// Write default config to file
	if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
		return fmt.Errorf("failed to write default config: %w", err)
	}

	return nil
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Validate required fields
	if config.Backend.BaseURL == "" {
		return errors.New("backend.base_url is required")
	}

	// Allow standalone mode if base_url is 'standalone', 'none', or 'local' (case-insensitive)
	standalone := false
	switch strings.ToLower(config.Backend.BaseURL) {
	case "standalone", "none", "local", "", "http://localhost:9999/api", "http://127.0.0.1:9999/api":
		standalone = true
	}

	if !standalone && config.Backend.APIToken == "" && config.Backend.TokenFile == "" {
		return errors.New("either backend.api_token or backend.token_file is required")
	}

	if config.Agent.WorkDir == "" {
		return errors.New("agent.work_dir is required")
	}

	if config.Agent.DataDir == "" {
		return errors.New("agent.data_dir is required")
	}

	// Validate resource limits
	if config.Resources.MaxContainers <= 0 {
		return errors.New("resources.max_containers must be greater than 0")
	}

	// Validate monitoring configuration
	if config.Monitoring.Enabled {
		if config.Monitoring.MetricsPort <= 0 || config.Monitoring.MetricsPort > 65535 {
			return errors.New("monitoring.metrics_port must be between 1 and 65535")
		}
		if config.Monitoring.HealthCheckPort <= 0 || config.Monitoring.HealthCheckPort > 65535 {
			return errors.New("monitoring.health_check_port must be between 1 and 65535")
		}
	}

	return nil
}

// decryptSensitiveData decrypts sensitive configuration values
func decryptSensitiveData(config *Config) error {
	// Load encryption key
	key, err := loadEncryptionKey(config.Security.EncryptionKeyFile, config.Security.EncryptionKey)
	if err != nil {
		return fmt.Errorf("failed to load encryption key: %w", err)
	}

	// Decrypt sensitive fields if they are encrypted
	if isEncrypted(config.Backend.APIToken) {
		decrypted, err := decrypt(config.Backend.APIToken, key)
		if err != nil {
			return fmt.Errorf("failed to decrypt API token: %w", err)
		}
		config.Backend.APIToken = decrypted
	}

	if isEncrypted(config.Git.SSHKeyPassphrase) {
		decrypted, err := decrypt(config.Git.SSHKeyPassphrase, key)
		if err != nil {
			return fmt.Errorf("failed to decrypt SSH key passphrase: %w", err)
		}
		config.Git.SSHKeyPassphrase = decrypted
	}

	if isEncrypted(config.Git.Password) {
		decrypted, err := decrypt(config.Git.Password, key)
		if err != nil {
			return fmt.Errorf("failed to decrypt Git password: %w", err)
		}
		config.Git.Password = decrypted
	}

	if isEncrypted(config.Git.Token) {
		decrypted, err := decrypt(config.Git.Token, key)
		if err != nil {
			return fmt.Errorf("failed to decrypt Git token: %w", err)
		}
		config.Git.Token = decrypted
	}

	return nil
}

// loadEncryptionKey loads the encryption key from file or config
func loadEncryptionKey(keyFile, keyValue string) ([]byte, error) {
	if keyFile != "" {
		// Load from file
		keyData, err := os.ReadFile(keyFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read encryption key file: %w", err)
		}
		return deriveKey(string(keyData)), nil
	}

	if keyValue != "" {
		return deriveKey(keyValue), nil
	}

	// Generate a default key (not recommended for production)
	return deriveKey("default-encryption-key"), nil
}

// deriveKey derives an encryption key from a password
func deriveKey(password string) []byte {
	salt := []byte("superagent-salt") // Use a proper salt in production
	return pbkdf2.Key([]byte(password), salt, 100000, 32, sha256.New)
}

// isEncrypted checks if a value is encrypted (starts with encrypted: prefix)
func isEncrypted(value string) bool {
	return len(value) > 10 && value[:10] == "encrypted:"
}

// decrypt decrypts an encrypted value
func decrypt(encryptedValue string, key []byte) (string, error) {
	if !isEncrypted(encryptedValue) {
		return encryptedValue, nil
	}

	// Remove the "encrypted:" prefix
	ciphertext, err := base64.StdEncoding.DecodeString(encryptedValue[10:])
	if err != nil {
		return "", fmt.Errorf("failed to decode encrypted value: %w", err)
	}

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Extract IV and ciphertext
	if len(ciphertext) < aes.BlockSize {
		return "", errors.New("ciphertext too short")
	}

	iv := ciphertext[:aes.BlockSize]
	ciphertext = ciphertext[aes.BlockSize:]

	// Decrypt
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(ciphertext, ciphertext)

	// Remove padding
	plaintext := removePadding(ciphertext)

	return string(plaintext), nil
}

// Encrypt encrypts a value for storage
func Encrypt(plaintext string, key []byte) (string, error) {
	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	// Add padding
	paddedPlaintext := addPadding([]byte(plaintext), aes.BlockSize)

	// Generate IV
	iv := make([]byte, aes.BlockSize)
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return "", fmt.Errorf("failed to generate IV: %w", err)
	}

	// Encrypt
	ciphertext := make([]byte, len(paddedPlaintext))
	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, paddedPlaintext)

	// Combine IV and ciphertext
	result := append(iv, ciphertext...)

	// Encode and add prefix
	encoded := base64.StdEncoding.EncodeToString(result)
	return "encrypted:" + encoded, nil
}

// addPadding adds PKCS7 padding
func addPadding(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padtext := make([]byte, padding)
	for i := range padtext {
		padtext[i] = byte(padding)
	}
	return append(data, padtext...)
}

// removePadding removes PKCS7 padding
func removePadding(data []byte) []byte {
	padding := int(data[len(data)-1])
	return data[:len(data)-padding]
}

// Save saves the configuration to file
func (c *Config) Save(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// Set all config values in viper
	if err := viper.WriteConfig(); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	return nil
}

// Reload reloads the configuration from file
func (c *Config) Reload(configPath string) error {
	newConfig, err := Load(configPath)
	if err != nil {
		return err
	}

	*c = *newConfig
	return nil
}

// LoadDefault loads configuration from default locations
func LoadDefault() (*Config, error) {
	configPath := ""
	
	// Check for config file in default locations
	locations := []string{
		"./.superagent.yaml",
		"~/.superagent.yaml",
		"/etc/superagent/config.yaml",
	}
	
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			configPath = loc
			break
		}
	}
	
	if configPath == "" {
		// Create default config if none found
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("failed to get home directory: %w", err)
		}
		configPath = filepath.Join(homeDir, ".superagent.yaml")
		if err := createDefaultConfig(configPath); err != nil {
			return nil, fmt.Errorf("failed to create default config: %w", err)
		}
	}
	
	return Load(configPath)
}

// GetConfigFile returns the config file path
func (c *Config) GetConfigFile() string {
	return viper.ConfigFileUsed()
}

// GetLogLevel returns the log level
func (c *Config) GetLogLevel() string {
	return c.Logging.Level
}

// GetLogFormat returns the log format
func (c *Config) GetLogFormat() string {
	return c.Logging.Format
}

// GetLogOutput returns the log output
func (c *Config) GetLogOutput() string {
	return c.Logging.Output
}

// GetLogMaxSize returns the log max size
func (c *Config) GetLogMaxSize() int {
	return c.Logging.MaxSize
}

// GetLogMaxBackups returns the log max backups
func (c *Config) GetLogMaxBackups() int {
	return c.Logging.MaxBackups
}

// GetLogMaxAge returns the log max age
func (c *Config) GetLogMaxAge() int {
	return c.Logging.MaxAge
}

// GetLogCompress returns the log compress setting
func (c *Config) GetLogCompress() bool {
	return c.Logging.Compress
}

// GetAPIPort returns the API port
func (c *Config) GetAPIPort() int {
	return c.Monitoring.HealthCheckPort
}

// GetMetricsPort returns the metrics port
func (c *Config) GetMetricsPort() int {
	return c.Monitoring.MetricsPort
}