package paas

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"superagent/internal/storage"
	"superagent/internal/logging"

	"github.com/sirupsen/logrus"
)

// AppCatalog manages the collection of approved applications
type AppCatalog struct {
	store       *storage.SecureStore
	auditLogger *logging.AuditLogger
	apps        map[string]*Application
	licenses    map[string]*AppLicense
	mu          sync.RWMutex
}

// Application represents an approved application in the catalog
type Application struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	Tags            []string               `json:"tags"`
	Publisher       string                 `json:"publisher"`
	Status          AppStatus              `json:"status"`
	Type            ApplicationType        `json:"type"`
	Source          ApplicationSource      `json:"source"`
	Versions        []*AppVersion          `json:"versions"`
	LatestVersion   string                 `json:"latest_version"`
	DefaultConfig   ApplicationConfig      `json:"default_config"`
	Pricing         PricingInfo            `json:"pricing"`
	Requirements    SystemRequirements     `json:"requirements"`
	Features        []string               `json:"features"`
	Screenshots     []string               `json:"screenshots"`
	Documentation   string                 `json:"documentation"`
	SupportEmail    string                 `json:"support_email"`
	Homepage        string                 `json:"homepage"`
	License         string                 `json:"license"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Downloads       int64                  `json:"downloads"`
	Rating          float64                `json:"rating"`
	Reviews         int                    `json:"reviews"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// ApplicationType defines the type of application
type ApplicationType string

const (
	AppTypeWebApp     ApplicationType = "webapp"
	AppTypeAPI        ApplicationType = "api"
	AppTypeMicroservice ApplicationType = "microservice"
	AppTypeDatabase   ApplicationType = "database"
	AppTypeWorker     ApplicationType = "worker"
	AppTypeCLI        ApplicationType = "cli"
	AppTypeOther      ApplicationType = "other"
)

// AppStatus represents application status in the catalog
type AppStatus string

const (
	AppStatusActive     AppStatus = "active"
	AppStatusDeprecated AppStatus = "deprecated"
	AppStatusBeta       AppStatus = "beta"
	AppStatusAlpha      AppStatus = "alpha"
	AppStatusDisabled   AppStatus = "disabled"
)

// ApplicationSource defines where the application code/image comes from
type ApplicationSource struct {
	Type         SourceType            `json:"type"`
	Repository   *GitRepository        `json:"repository,omitempty"`
	DockerImage  *DockerImageSource    `json:"docker_image,omitempty"`
	Archive      *ArchiveSource        `json:"archive,omitempty"`
	Credentials  map[string]string     `json:"credentials,omitempty"`
}

// SourceType defines the source type
type SourceType string

const (
	SourceTypeGit    SourceType = "git"
	SourceTypeDocker SourceType = "docker"
	SourceTypeArchive SourceType = "archive"
)

// GitRepository defines Git repository source
type GitRepository struct {
	URL        string `json:"url"`
	Branch     string `json:"branch"`
	Tag        string `json:"tag,omitempty"`
	Commit     string `json:"commit,omitempty"`
	Subfolder  string `json:"subfolder,omitempty"`
	Private    bool   `json:"private"`
	SSHKey     string `json:"ssh_key,omitempty"`
	Username   string `json:"username,omitempty"`
	Token      string `json:"token,omitempty"`
}

// DockerImageSource defines Docker image source
type DockerImageSource struct {
	Registry string `json:"registry"`
	Image    string `json:"image"`
	Tag      string `json:"tag"`
	Private  bool   `json:"private"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ArchiveSource defines archive file source
type ArchiveSource struct {
	URL      string            `json:"url"`
	Type     string            `json:"type"` // zip, tar.gz, etc.
	Checksum string            `json:"checksum"`
	Headers  map[string]string `json:"headers,omitempty"`
}

// AppVersion represents a specific version of an application
type AppVersion struct {
	Version     string                 `json:"version"`
	Description string                 `json:"description"`
	ReleaseDate time.Time              `json:"release_date"`
	Source      ApplicationSource      `json:"source"`
	Config      ApplicationConfig      `json:"config"`
	Status      VersionStatus          `json:"status"`
	Changelog   string                 `json:"changelog"`
	Breaking    bool                   `json:"breaking"`
	Security    bool                   `json:"security"`
	Downloads   int64                  `json:"downloads"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// VersionStatus represents version status
type VersionStatus string

const (
	VersionStatusStable     VersionStatus = "stable"
	VersionStatusBeta       VersionStatus = "beta"
	VersionStatusAlpha      VersionStatus = "alpha"
	VersionStatusDeprecated VersionStatus = "deprecated"
	VersionStatusYanked     VersionStatus = "yanked"
)

// ApplicationConfig defines application configuration
type ApplicationConfig struct {
	Port            int                    `json:"port"`
	Environment     map[string]EnvVar      `json:"environment"`
	Volumes         []VolumeMount          `json:"volumes"`
	HealthCheck     HealthCheckConfig      `json:"health_check"`
	Resources       ResourceConfig         `json:"resources"`
	Network         NetworkConfig          `json:"network"`
	Security        SecurityConfig         `json:"security"`
	Scaling         ScalingConfig          `json:"scaling"`
	BuildCommand    string                 `json:"build_command,omitempty"`
	StartCommand    string                 `json:"start_command"`
	StopCommand     string                 `json:"stop_command,omitempty"`
	Dockerfile      string                 `json:"dockerfile,omitempty"`
	WorkingDir      string                 `json:"working_dir,omitempty"`
	Dependencies    []string               `json:"dependencies"`
	Capabilities    []string               `json:"capabilities"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// EnvVar represents an environment variable
type EnvVar struct {
	Value       string `json:"value,omitempty"`
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Type        string `json:"type"` // string, int, bool, email, url, etc.
	Default     string `json:"default,omitempty"`
	Options     []string `json:"options,omitempty"` // For select type
	Sensitive   bool   `json:"sensitive"`
}

// VolumeMount represents a volume mount
type VolumeMount struct {
	Name        string `json:"name"`
	Path        string `json:"path"`
	Type        string `json:"type"` // persistent, temp, config
	Size        string `json:"size,omitempty"`
	Permissions string `json:"permissions"`
	Backup      bool   `json:"backup"`
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	Enabled         bool   `json:"enabled"`
	Path            string `json:"path"`
	Interval        int    `json:"interval"` // seconds
	Timeout         int    `json:"timeout"`  // seconds
	Retries         int    `json:"retries"`
	InitialDelay    int    `json:"initial_delay"` // seconds
	Command         string `json:"command,omitempty"`
	ExpectedStatus  int    `json:"expected_status"`
	ExpectedContent string `json:"expected_content,omitempty"`
}

// ResourceConfig defines resource requirements and limits
type ResourceConfig struct {
	CPU struct {
		Request string `json:"request"` // 0.1, 0.5, 1.0
		Limit   string `json:"limit"`
	} `json:"cpu"`
	Memory struct {
		Request string `json:"request"` // 128Mi, 256Mi, 512Mi
		Limit   string `json:"limit"`
	} `json:"memory"`
	Storage struct {
		Request string `json:"request"` // 1Gi, 5Gi, 10Gi
		Limit   string `json:"limit"`
	} `json:"storage"`
	GPU struct {
		Required bool   `json:"required"`
		Type     string `json:"type,omitempty"`
		Count    int    `json:"count,omitempty"`
	} `json:"gpu"`
}

// NetworkConfig defines network configuration
type NetworkConfig struct {
	PublicAccess bool     `json:"public_access"`
	CustomDomain bool     `json:"custom_domain"`
	SSL          bool     `json:"ssl"`
	Ports        []int    `json:"ports"`
	Protocols    []string `json:"protocols"` // http, https, tcp, udp
	IPWhitelist  []string `json:"ip_whitelist,omitempty"`
	RateLimit    struct {
		Enabled bool `json:"enabled"`
		RPM     int  `json:"rpm"` // Requests per minute
		Burst   int  `json:"burst"`
	} `json:"rate_limit"`
}

// SecurityConfig defines security configuration
type SecurityConfig struct {
	RunAsNonRoot     bool     `json:"run_as_non_root"`
	ReadOnlyRootFS   bool     `json:"read_only_root_fs"`
	AllowPrivileged  bool     `json:"allow_privileged"`
	Capabilities     []string `json:"capabilities"`
	SELinux          bool     `json:"selinux"`
	AppArmor         bool     `json:"apparmor"`
	Seccomp          bool     `json:"seccomp"`
	NetworkPolicies  []string `json:"network_policies"`
	PodSecurityLevel string   `json:"pod_security_level"` // privileged, baseline, restricted
}

// ScalingConfig defines scaling configuration
type ScalingConfig struct {
	AutoScale struct {
		Enabled    bool `json:"enabled"`
		MinReplicas int  `json:"min_replicas"`
		MaxReplicas int  `json:"max_replicas"`
		CPUTarget   int  `json:"cpu_target"`    // percentage
		MemoryTarget int `json:"memory_target"` // percentage
	} `json:"auto_scale"`
	Manual struct {
		Replicas int `json:"replicas"`
	} `json:"manual"`
}

// PricingInfo defines pricing information
type PricingInfo struct {
	Model    string             `json:"model"` // free, one-time, subscription
	Currency string             `json:"currency"`
	Tiers    []PricingTier      `json:"tiers"`
	Trial    *TrialInfo         `json:"trial,omitempty"`
	Metadata map[string]interface{} `json:"metadata"`
}

// PricingTier represents a pricing tier
type PricingTier struct {
	Name        string  `json:"name"`
	Price       float64 `json:"price"`
	Period      string  `json:"period"` // month, year, one-time
	Features    []string `json:"features"`
	Limitations map[string]interface{} `json:"limitations"`
	Popular     bool    `json:"popular"`
}

// TrialInfo defines trial information
type TrialInfo struct {
	Enabled    bool `json:"enabled"`
	Duration   int  `json:"duration"` // days
	Features   []string `json:"features"`
	Limitations map[string]interface{} `json:"limitations"`
}

// SystemRequirements defines system requirements
type SystemRequirements struct {
	MinCPU      float64 `json:"min_cpu"`
	MinMemory   int64   `json:"min_memory"`
	MinStorage  int64   `json:"min_storage"`
	MinBandwidth int64  `json:"min_bandwidth"`
	Architecture []string `json:"architecture"` // amd64, arm64
	OS          []string `json:"os"`           // linux, windows
	Dependencies []string `json:"dependencies"`
}

// AppLicense represents a license for an application
type AppLicense struct {
	ID          string                 `json:"id"`
	AppID       string                 `json:"app_id"`
	CustomerID  string                 `json:"customer_id"`
	Type        LicenseType            `json:"type"`
	Status      LicenseStatus          `json:"status"`
	ValidFrom   time.Time              `json:"valid_from"`
	ValidUntil  *time.Time             `json:"valid_until,omitempty"`
	Limitations LicenseLimitations     `json:"limitations"`
	Features    []string               `json:"features"`
	PurchaseInfo PurchaseInfo          `json:"purchase_info"`
	Usage       LicenseUsage           `json:"usage"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// LicenseType defines license types
type LicenseType string

const (
	LicenseTypeFree        LicenseType = "free"
	LicenseTypeTrial       LicenseType = "trial"
	LicenseTypeSubscription LicenseType = "subscription"
	LicenseTypeOneTime     LicenseType = "one-time"
	LicenseTypeEnterprise  LicenseType = "enterprise"
)

// LicenseStatus defines license status
type LicenseStatus string

const (
	LicenseStatusActive    LicenseStatus = "active"
	LicenseStatusExpired   LicenseStatus = "expired"
	LicenseStatusSuspended LicenseStatus = "suspended"
	LicenseStatusRevoked   LicenseStatus = "revoked"
)

// LicenseLimitations defines license limitations
type LicenseLimitations struct {
	MaxDeployments int                    `json:"max_deployments"`
	MaxInstances   int                    `json:"max_instances"`
	MaxUsers       int                    `json:"max_users"`
	MaxDomains     int                    `json:"max_domains"`
	MaxBandwidth   int64                  `json:"max_bandwidth"` // GB/month
	MaxStorage     int64                  `json:"max_storage"`   // GB
	Features       []string               `json:"features"`
	Restrictions   map[string]interface{} `json:"restrictions"`
}

// PurchaseInfo contains purchase information
type PurchaseInfo struct {
	OrderID       string    `json:"order_id"`
	TransactionID string    `json:"transaction_id"`
	Amount        float64   `json:"amount"`
	Currency      string    `json:"currency"`
	PaymentMethod string    `json:"payment_method"`
	PurchaseDate  time.Time `json:"purchase_date"`
	InvoiceURL    string    `json:"invoice_url,omitempty"`
}

// LicenseUsage tracks license usage
type LicenseUsage struct {
	Deployments    int       `json:"deployments"`
	Instances      int       `json:"instances"`
	Users          int       `json:"users"`
	Domains        int       `json:"domains"`
	BandwidthUsed  int64     `json:"bandwidth_used"`  // GB
	StorageUsed    int64     `json:"storage_used"`    // GB
	LastUsed       time.Time `json:"last_used"`
	LastUpdated    time.Time `json:"last_updated"`
}

// CreateApplicationRequest represents application creation request
type CreateApplicationRequest struct {
	Name            string                 `json:"name"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	Tags            []string               `json:"tags"`
	Publisher       string                 `json:"publisher"`
	Type            ApplicationType        `json:"type"`
	Source          ApplicationSource      `json:"source"`
	DefaultConfig   ApplicationConfig      `json:"default_config"`
	Pricing         PricingInfo            `json:"pricing"`
	Requirements    SystemRequirements     `json:"requirements"`
	Features        []string               `json:"features"`
	Documentation   string                 `json:"documentation"`
	SupportEmail    string                 `json:"support_email"`
	Homepage        string                 `json:"homepage"`
	License         string                 `json:"license"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// NewAppCatalog creates a new app catalog
func NewAppCatalog(store *storage.SecureStore, auditLogger *logging.AuditLogger) *AppCatalog {
	ac := &AppCatalog{
		store:       store,
		auditLogger: auditLogger,
		apps:        make(map[string]*Application),
		licenses:    make(map[string]*AppLicense),
	}

	// Load existing apps and licenses
	if err := ac.loadAppsAndLicenses(); err != nil {
		logrus.Warnf("Failed to load existing apps and licenses: %v", err)
	}

	return ac
}

// AddApplication adds a new application to the catalog
func (ac *AppCatalog) AddApplication(ctx context.Context, req *CreateApplicationRequest) (*Application, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Validate application name uniqueness
	for _, app := range ac.apps {
		if strings.EqualFold(app.Name, req.Name) {
			return nil, fmt.Errorf("application with name %s already exists", req.Name)
		}
	}

	// Generate application ID
	appID := ac.generateApplicationID()

	// Create initial version
	initialVersion := &AppVersion{
		Version:     "1.0.0",
		Description: "Initial version",
		ReleaseDate: time.Now(),
		Source:      req.Source,
		Config:      req.DefaultConfig,
		Status:      VersionStatusStable,
		Changelog:   "Initial release",
		Downloads:   0,
		Metadata:    make(map[string]interface{}),
	}

	// Create application
	app := &Application{
		ID:              appID,
		Name:            req.Name,
		Description:     req.Description,
		Category:        req.Category,
		Tags:            req.Tags,
		Publisher:       req.Publisher,
		Status:          AppStatusActive,
		Type:            req.Type,
		Source:          req.Source,
		Versions:        []*AppVersion{initialVersion},
		LatestVersion:   "1.0.0",
		DefaultConfig:   req.DefaultConfig,
		Pricing:         req.Pricing,
		Requirements:    req.Requirements,
		Features:        req.Features,
		Documentation:   req.Documentation,
		SupportEmail:    req.SupportEmail,
		Homepage:        req.Homepage,
		License:         req.License,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		Downloads:       0,
		Rating:          0.0,
		Reviews:         0,
		Metadata:        req.Metadata,
	}

	// Set default values
	ac.setDefaultValues(app)

	// Store application
	ac.apps[appID] = app
	if err := ac.saveApplication(app); err != nil {
		delete(ac.apps, appID)
		return nil, fmt.Errorf("failed to save application: %w", err)
	}

	ac.auditLogger.LogEvent("APPLICATION_ADDED", map[string]interface{}{
		"app_id":    appID,
		"name":      req.Name,
		"publisher": req.Publisher,
	})

	logrus.Infof("Application added: %s (%s)", app.Name, appID)
	return app, nil
}

// GetApplication retrieves an application by ID
func (ac *AppCatalog) GetApplication(appID string) (*Application, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	app, exists := ac.apps[appID]
	if !exists {
		return nil, fmt.Errorf("application not found: %s", appID)
	}

	return app, nil
}

// GetApplicationByName retrieves an application by name
func (ac *AppCatalog) GetApplicationByName(name string) (*Application, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	for _, app := range ac.apps {
		if strings.EqualFold(app.Name, name) {
			return app, nil
		}
	}

	return nil, fmt.Errorf("application not found with name: %s", name)
}

// ListApplications returns all applications
func (ac *AppCatalog) ListApplications() []*Application {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	apps := make([]*Application, 0, len(ac.apps))
	for _, app := range ac.apps {
		apps = append(apps, app)
	}

	return apps
}

// GetApplicationsByCategory returns applications in a specific category
func (ac *AppCatalog) GetApplicationsByCategory(category string) []*Application {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	var apps []*Application
	for _, app := range ac.apps {
		if app.Category == category {
			apps = append(apps, app)
		}
	}

	return apps
}

// AddVersion adds a new version to an application
func (ac *AppCatalog) AddVersion(appID string, version *AppVersion) error {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	app, exists := ac.apps[appID]
	if !exists {
		return fmt.Errorf("application not found: %s", appID)
	}

	// Check if version already exists
	for _, existing := range app.Versions {
		if existing.Version == version.Version {
			return fmt.Errorf("version %s already exists", version.Version)
		}
	}

	app.Versions = append(app.Versions, version)
	app.LatestVersion = version.Version
	app.UpdatedAt = time.Now()

	// Save application
	if err := ac.saveApplication(app); err != nil {
		return fmt.Errorf("failed to save application: %w", err)
	}

	ac.auditLogger.LogEvent("VERSION_ADDED", map[string]interface{}{
		"app_id":  appID,
		"version": version.Version,
	})

	return nil
}

// CreateLicense creates a new license for a customer
func (ac *AppCatalog) CreateLicense(appID, customerID string, licenseType LicenseType, validUntil *time.Time) (*AppLicense, error) {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Verify application exists
	app, exists := ac.apps[appID]
	if !exists {
		return nil, fmt.Errorf("application not found: %s", appID)
	}

	// Generate license ID
	licenseID := ac.generateLicenseID()

	// Set limitations based on license type
	limitations := ac.getDefaultLimitations(licenseType)

	// Create license
	license := &AppLicense{
		ID:          licenseID,
		AppID:       appID,
		CustomerID:  customerID,
		Type:        licenseType,
		Status:      LicenseStatusActive,
		ValidFrom:   time.Now(),
		ValidUntil:  validUntil,
		Limitations: limitations,
		Features:    app.Features,
		Usage: LicenseUsage{
			LastUpdated: time.Now(),
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// Store license
	ac.licenses[licenseID] = license
	if err := ac.saveLicense(license); err != nil {
		delete(ac.licenses, licenseID)
		return nil, fmt.Errorf("failed to save license: %w", err)
	}

	ac.auditLogger.LogEvent("LICENSE_CREATED", map[string]interface{}{
		"license_id":  licenseID,
		"app_id":      appID,
		"customer_id": customerID,
		"type":        licenseType,
	})

	logrus.Infof("License created: %s for app %s, customer %s", licenseID, appID, customerID)
	return license, nil
}

// ValidateLicense checks if a license is valid for deployment
func (ac *AppCatalog) ValidateLicense(licenseID string) (*AppLicense, error) {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	license, exists := ac.licenses[licenseID]
	if !exists {
		return nil, fmt.Errorf("license not found: %s", licenseID)
	}

	// Check license status
	if license.Status != LicenseStatusActive {
		return nil, fmt.Errorf("license is not active: %s", license.Status)
	}

	// Check expiration
	if license.ValidUntil != nil && time.Now().After(*license.ValidUntil) {
		return nil, fmt.Errorf("license has expired")
	}

	return license, nil
}

// Helper functions

func (ac *AppCatalog) generateApplicationID() string {
	bytes := make([]byte, 8)
	rand.Read(bytes)
	return fmt.Sprintf("app_%s", hex.EncodeToString(bytes))
}

func (ac *AppCatalog) generateLicenseID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return fmt.Sprintf("lic_%s", hex.EncodeToString(bytes))
}

func (ac *AppCatalog) setDefaultValues(app *Application) {
	if app.Category == "" {
		app.Category = "other"
	}
	if len(app.Tags) == 0 {
		app.Tags = []string{}
	}
	if len(app.Features) == 0 {
		app.Features = []string{}
	}
	if app.Metadata == nil {
		app.Metadata = make(map[string]interface{})
	}
}

func (ac *AppCatalog) getDefaultLimitations(licenseType LicenseType) LicenseLimitations {
	switch licenseType {
	case LicenseTypeFree:
		return LicenseLimitations{
			MaxDeployments: 1,
			MaxInstances:   1,
			MaxUsers:       1,
			MaxDomains:     0,
			MaxBandwidth:   5,  // 5GB
			MaxStorage:     1,  // 1GB
			Features:       []string{"basic"},
			Restrictions:   make(map[string]interface{}),
		}
	case LicenseTypeTrial:
		return LicenseLimitations{
			MaxDeployments: 3,
			MaxInstances:   2,
			MaxUsers:       5,
			MaxDomains:     1,
			MaxBandwidth:   25, // 25GB
			MaxStorage:     5,  // 5GB
			Features:       []string{"basic", "advanced"},
			Restrictions:   make(map[string]interface{}),
		}
	default:
		return LicenseLimitations{
			MaxDeployments: -1, // unlimited
			MaxInstances:   -1,
			MaxUsers:       -1,
			MaxDomains:     -1,
			MaxBandwidth:   -1,
			MaxStorage:     -1,
			Features:       []string{"basic", "advanced", "premium"},
			Restrictions:   make(map[string]interface{}),
		}
	}
}

func (ac *AppCatalog) saveApplication(app *Application) error {
	appData := map[string]interface{}{
		"application": app,
	}

	return ac.store.StoreDeploymentState(fmt.Sprintf("app_%s", app.ID), appData)
}

func (ac *AppCatalog) saveLicense(license *AppLicense) error {
	licenseData := map[string]interface{}{
		"license": license,
	}

	return ac.store.StoreDeploymentState(fmt.Sprintf("license_%s", license.ID), licenseData)
}

func (ac *AppCatalog) loadAppsAndLicenses() error {
	// This would load apps and licenses from storage
	// For now, we'll implement basic loading
	return nil
}