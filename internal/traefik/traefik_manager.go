package traefik

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
)

// TraefikManager manages Traefik configuration and routes
type TraefikManager struct {
	apiURL      string
	configPath  string
	httpClient  *http.Client
	baseDomain  string
}

// TraefikConfig represents Traefik configuration
type TraefikConfig struct {
	API     APIConfig     `yaml:"api"`
	EntryPoints EntryPointsConfig `yaml:"entryPoints"`
	Providers ProvidersConfig `yaml:"providers"`
	CertificatesResolvers map[string]CertificatesResolverConfig `yaml:"certificatesResolvers,omitempty"`
}

// APIConfig represents Traefik API configuration
type APIConfig struct {
	Dashboard bool `yaml:"dashboard"`
	Insecure bool `yaml:"insecure"`
}

// EntryPointsConfig represents Traefik entry points
type EntryPointsConfig struct {
	Web    EntryPointConfig `yaml:"web"`
	WebSecure EntryPointConfig `yaml:"websecure"`
}

// EntryPointConfig represents an entry point configuration
type EntryPointConfig struct {
	Address string `yaml:"address"`
}

// ProvidersConfig represents Traefik providers
type ProvidersConfig struct {
	Docker DockerProviderConfig `yaml:"docker"`
	File   FileProviderConfig   `yaml:"file"`
}

// DockerProviderConfig represents Docker provider configuration
type DockerProviderConfig struct {
	Endpoint  string `yaml:"endpoint"`
	ExposedByDefault bool `yaml:"exposedByDefault"`
	Network   string `yaml:"network"`
}

// FileProviderConfig represents file provider configuration
type FileProviderConfig struct {
	Directory string `yaml:"directory"`
	Watch     bool   `yaml:"watch"`
}

// CertificatesResolverConfig represents certificates resolver configuration
type CertificatesResolverConfig struct {
	Acme AcmeConfig `yaml:"acme"`
}

// AcmeConfig represents ACME configuration
type AcmeConfig struct {
	Email  string `yaml:"email"`
	Storage string `yaml:"storage"`
	HTTPChallenge HTTPChallengeConfig `yaml:"httpChallenge"`
}

// HTTPChallengeConfig represents HTTP challenge configuration
type HTTPChallengeConfig struct {
	EntryPoint string `yaml:"entryPoint"`
}

// Route represents a Traefik route
type Route struct {
	Rule    string            `json:"rule"`
	Service string            `json:"service"`
	Middlewares []string      `json:"middlewares,omitempty"`
	TLS     *TLSConfig        `json:"tls,omitempty"`
}

// Service represents a Traefik service
type Service struct {
	LoadBalancer LoadBalancerConfig `json:"loadBalancer"`
}

// LoadBalancerConfig represents load balancer configuration
type LoadBalancerConfig struct {
	Servers []ServerConfig `json:"servers"`
}

// ServerConfig represents a server configuration
type ServerConfig struct {
	URL string `json:"url"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	CertResolver string `json:"certResolver,omitempty"`
}

// NewTraefikManager creates a new Traefik manager
func NewTraefikManager(baseDomain string) *TraefikManager {
	return &TraefikManager{
		apiURL:     "http://localhost:8080/api",
		configPath: "/etc/traefik/traefik.yml",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		baseDomain: baseDomain,
	}
}

// InstallTraefik installs Traefik
func (tm *TraefikManager) InstallTraefik() error {
	logrus.Info("Installing Traefik...")

	// Check if Traefik is already installed
	if tm.isInstalled() {
		logrus.Info("Traefik is already installed")
		return nil
	}

	// Download and install Traefik
	if err := tm.downloadTraefik(); err != nil {
		return fmt.Errorf("failed to download Traefik: %w", err)
	}

	// Create configuration directory
	if err := os.MkdirAll("/etc/traefik", 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Generate configuration
	if err := tm.generateConfig(); err != nil {
		return fmt.Errorf("failed to generate config: %w", err)
	}

	// Create systemd service
	if err := tm.createSystemdService(); err != nil {
		return fmt.Errorf("failed to create systemd service: %w", err)
	}

	// Start Traefik service
	if err := tm.startService(); err != nil {
		return fmt.Errorf("failed to start Traefik service: %w", err)
	}

	logrus.Info("Traefik installed and started successfully")
	return nil
}

// IsInstalled checks if Traefik is installed
func (tm *TraefikManager) IsInstalled() bool {
	_, err := os.Stat("/usr/local/bin/traefik")
	return err == nil
}

// isInstalled checks if Traefik is installed (private method)
func (tm *TraefikManager) isInstalled() bool {
	return tm.IsInstalled()
}

// downloadTraefik downloads Traefik binary
func (tm *TraefikManager) downloadTraefik() error {
	// This would download Traefik binary
	// For now, we'll assume it's installed via package manager
	logrus.Info("Traefik binary should be installed via package manager")
	return nil
}

// generateConfig generates Traefik configuration
func (tm *TraefikManager) generateConfig() error {
	config := TraefikConfig{
		API: APIConfig{
			Dashboard: true,
			Insecure: true,
		},
		EntryPoints: EntryPointsConfig{
			Web: EntryPointConfig{
				Address: ":80",
			},
			WebSecure: EntryPointConfig{
				Address: ":443",
			},
		},
		Providers: ProvidersConfig{
			Docker: DockerProviderConfig{
				Endpoint:        "unix:///var/run/docker.sock",
				ExposedByDefault: false,
				Network:         "superagent",
			},
			File: FileProviderConfig{
				Directory: "/etc/traefik/dynamic",
				Watch:     true,
			},
		},
	}

	// Add SSL configuration if base domain is set
	if tm.baseDomain != "" {
		config.CertificatesResolvers = map[string]CertificatesResolverConfig{
			"letsencrypt": {
				Acme: AcmeConfig{
					Email:  "admin@" + tm.baseDomain,
					Storage: "/etc/traefik/acme.json",
					HTTPChallenge: HTTPChallengeConfig{
						EntryPoint: "web",
					},
				},
			},
		}
	}

	// Marshal to YAML
	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(tm.configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config: %w", err)
	}

	// Create dynamic config directory
	if err := os.MkdirAll("/etc/traefik/dynamic", 0755); err != nil {
		return fmt.Errorf("failed to create dynamic config directory: %w", err)
	}

	logrus.Info("Traefik configuration generated")
	return nil
}

// createSystemdService creates systemd service for Traefik
func (tm *TraefikManager) createSystemdService() error {
	serviceContent := `[Unit]
Description=Traefik
Documentation=https://doc.traefik.io/traefik/
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
ExecStart=/usr/local/bin/traefik --configfile=/etc/traefik/traefik.yml
Restart=on-failure
RestartSec=5
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`

	servicePath := "/etc/systemd/system/traefik.service"
	if err := ioutil.WriteFile(servicePath, []byte(serviceContent), 0644); err != nil {
		return fmt.Errorf("failed to write service file: %w", err)
	}

	// Reload systemd
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %w", err)
	}

	// Enable service
	cmd = exec.Command("systemctl", "enable", "traefik")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to enable service: %w", err)
	}

	logrus.Info("Traefik systemd service created")
	return nil
}

// startService starts Traefik service
func (tm *TraefikManager) startService() error {
	cmd := exec.Command("systemctl", "start", "traefik")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start Traefik service: %w", err)
	}

	// Wait for service to be ready
	time.Sleep(5 * time.Second)

	logrus.Info("Traefik service started")
	return nil
}

// AddRoute adds a route for an application
func (tm *TraefikManager) AddRoute(appID, containerName string, port int) error {
	subdomain := tm.generateSubdomain(appID)
	
	// Create dynamic configuration
	config := map[string]interface{}{
		"http": map[string]interface{}{
			"routers": map[string]interface{}{
				appID: map[string]interface{}{
					"rule":    fmt.Sprintf("Host(`%s.%s`)", subdomain, tm.baseDomain),
					"service": appID,
					"tls": map[string]interface{}{
						"certResolver": "letsencrypt",
					},
				},
			},
			"services": map[string]interface{}{
				appID: map[string]interface{}{
					"loadBalancer": map[string]interface{}{
						"servers": []map[string]interface{}{
							{
								"url": fmt.Sprintf("http://%s:%d", containerName, port),
							},
						},
					},
				},
			},
		},
	}

	// Write to dynamic config file
	configPath := filepath.Join("/etc/traefik/dynamic", fmt.Sprintf("%s.yml", appID))
	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal route config: %w", err)
	}

	if err := ioutil.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write route config: %w", err)
	}

	logrus.Infof("Route added for %s: %s.%s", appID, subdomain, tm.baseDomain)
	return nil
}

// RemoveRoute removes a route for an application
func (tm *TraefikManager) RemoveRoute(appID string) error {
	configPath := filepath.Join("/etc/traefik/dynamic", fmt.Sprintf("%s.yml", appID))
	
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove route config: %w", err)
	}

	logrus.Infof("Route removed for %s", appID)
	return nil
}

// TestConfiguration tests Traefik configuration
func (tm *TraefikManager) TestConfiguration() error {
	// Test Traefik API
	resp, err := tm.httpClient.Get(tm.apiURL + "/http/routers")
	if err != nil {
		return fmt.Errorf("failed to connect to Traefik API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Traefik API returned status: %d", resp.StatusCode)
	}

	logrus.Info("Traefik configuration is valid")
	return nil
}

// GetRoutes gets all routes from Traefik
func (tm *TraefikManager) GetRoutes() (map[string]interface{}, error) {
	resp, err := tm.httpClient.Get(tm.apiURL + "/http/routers")
	if err != nil {
		return nil, fmt.Errorf("failed to get routes: %w", err)
	}
	defer resp.Body.Close()

	var routes map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&routes); err != nil {
		return nil, fmt.Errorf("failed to decode routes: %w", err)
	}

	return routes, nil
}

// generateSubdomain generates a subdomain for the app
func (tm *TraefikManager) generateSubdomain(appID string) string {
	// Clean app ID for subdomain
	clean := strings.ToLower(appID)
	clean = strings.ReplaceAll(clean, "_", "-")
	clean = strings.ReplaceAll(clean, " ", "-")
	return clean
}

// GetDashboardURL returns Traefik dashboard URL
func (tm *TraefikManager) GetDashboardURL() string {
	return "http://localhost:8080"
}

// GetBaseDomain returns the base domain
func (tm *TraefikManager) GetBaseDomain() string {
	return tm.baseDomain
}

// SetBaseDomain sets the base domain
func (tm *TraefikManager) SetBaseDomain(domain string) {
	tm.baseDomain = domain
}