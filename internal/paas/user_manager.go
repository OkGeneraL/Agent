package paas

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

// UserManager handles customer management for the PaaS platform
type UserManager struct {
	store       *storage.SecureStore
	auditLogger *logging.AuditLogger
	users       map[string]*Customer
	mu          sync.RWMutex
}

// Customer represents a PaaS platform customer
type Customer struct {
	ID               string                 `json:"id"`
	Email            string                 `json:"email"`
	Name             string                 `json:"name"`
	Company          string                 `json:"company"`
	Plan             string                 `json:"plan"`
	Status           CustomerStatus         `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
	LastLoginAt      *time.Time             `json:"last_login_at,omitempty"`
	ResourceQuotas   ResourceQuotas         `json:"resource_quotas"`
	UsedResources    ResourceUsage          `json:"used_resources"`
	Licenses         []string               `json:"licenses"`
	Deployments      []string               `json:"deployments"`
	CustomDomains    []string               `json:"custom_domains"`
	SubdomainPrefix  string                 `json:"subdomain_prefix"`
	APIKey           string                 `json:"api_key"`
	Settings         CustomerSettings       `json:"settings"`
	BillingInfo      BillingInfo            `json:"billing_info"`
	Metadata         map[string]interface{} `json:"metadata"`
}

// CustomerStatus represents customer account status
type CustomerStatus string

const (
	CustomerStatusActive    CustomerStatus = "active"
	CustomerStatusSuspended CustomerStatus = "suspended"
	CustomerStatusPending   CustomerStatus = "pending"
	CustomerStatusCancelled CustomerStatus = "cancelled"
)

// ResourceQuotas defines per-customer resource limits
type ResourceQuotas struct {
	MaxCPU           float64 `json:"max_cpu"`           // CPU cores
	MaxMemory        int64   `json:"max_memory"`        // Memory in MB
	MaxStorage       int64   `json:"max_storage"`       // Storage in GB
	MaxBandwidth     int64   `json:"max_bandwidth"`     // Bandwidth in GB/month
	MaxContainers    int     `json:"max_containers"`    // Max concurrent containers
	MaxApps          int     `json:"max_apps"`          // Max different apps
	MaxDeployments   int     `json:"max_deployments"`   // Max total deployments
	MaxCustomDomains int     `json:"max_custom_domains"` // Max custom domains
}

// ResourceUsage tracks current resource usage
type ResourceUsage struct {
	UsedCPU           float64   `json:"used_cpu"`
	UsedMemory        int64     `json:"used_memory"`
	UsedStorage       int64     `json:"used_storage"`
	UsedBandwidth     int64     `json:"used_bandwidth"`
	ActiveContainers  int       `json:"active_containers"`
	TotalApps         int       `json:"total_apps"`
	TotalDeployments  int       `json:"total_deployments"`
	CustomDomains     int       `json:"custom_domains"`
	LastUpdated       time.Time `json:"last_updated"`
}

// CustomerSettings holds customer preferences
type CustomerSettings struct {
	DefaultRegion       string            `json:"default_region"`
	NotificationsEmail  bool              `json:"notifications_email"`
	NotificationsSlack  string            `json:"notifications_slack,omitempty"`
	AutoSSL             bool              `json:"auto_ssl"`
	AutoBackup          bool              `json:"auto_backup"`
	BackupRetention     int               `json:"backup_retention"` // Days
	Environment         map[string]string `json:"environment"`      // Default env vars
	DeploymentStrategy  string            `json:"deployment_strategy"`
	HealthCheckEnabled  bool              `json:"health_check_enabled"`
}

// BillingInfo holds customer billing information
type BillingInfo struct {
	PlanID          string    `json:"plan_id"`
	BillingCycle    string    `json:"billing_cycle"` // monthly, yearly
	NextBillingDate time.Time `json:"next_billing_date"`
	CurrentPeriod   time.Time `json:"current_period"`
	TotalSpent      float64   `json:"total_spent"`
	Currency        string    `json:"currency"`
	PaymentMethod   string    `json:"payment_method"`
	BillingEmail    string    `json:"billing_email"`
}

// CreateCustomerRequest represents customer creation request
type CreateCustomerRequest struct {
	Email           string                 `json:"email"`
	Name            string                 `json:"name"`
	Company         string                 `json:"company"`
	Plan            string                 `json:"plan"`
	SubdomainPrefix string                 `json:"subdomain_prefix,omitempty"`
	Settings        CustomerSettings       `json:"settings,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// NewUserManager creates a new user manager
func NewUserManager(store *storage.SecureStore, auditLogger *logging.AuditLogger) *UserManager {
	um := &UserManager{
		store:       store,
		auditLogger: auditLogger,
		users:       make(map[string]*Customer),
	}

	// Load existing users
	if err := um.loadUsers(); err != nil {
		logrus.Warnf("Failed to load existing users: %v", err)
	}

	return um
}

// CreateCustomer creates a new customer
func (um *UserManager) CreateCustomer(ctx context.Context, req *CreateCustomerRequest) (*Customer, error) {
	um.mu.Lock()
	defer um.mu.Unlock()

	// Validate email uniqueness
	for _, customer := range um.users {
		if customer.Email == req.Email {
			return nil, fmt.Errorf("customer with email %s already exists", req.Email)
		}
	}

	// Generate customer ID and API key
	customerID := um.generateCustomerID()
	apiKey, err := um.generateAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate API key: %w", err)
	}

	// Generate subdomain prefix if not provided
	subdomainPrefix := req.SubdomainPrefix
	if subdomainPrefix == "" {
		subdomainPrefix = um.generateSubdomainPrefix(req.Name, req.Company)
	}

	// Set default resource quotas based on plan
	quotas := um.getDefaultQuotas(req.Plan)

	// Create customer
	customer := &Customer{
		ID:              customerID,
		Email:           req.Email,
		Name:            req.Name,
		Company:         req.Company,
		Plan:            req.Plan,
		Status:          CustomerStatusActive,
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
		ResourceQuotas:  quotas,
		UsedResources:   ResourceUsage{LastUpdated: time.Now()},
		Licenses:        []string{},
		Deployments:     []string{},
		CustomDomains:   []string{},
		SubdomainPrefix: subdomainPrefix,
		APIKey:          apiKey,
		Settings:        req.Settings,
		BillingInfo: BillingInfo{
			PlanID:          req.Plan,
			BillingCycle:    "monthly",
			NextBillingDate: time.Now().AddDate(0, 1, 0),
			CurrentPeriod:   time.Now(),
			Currency:        "USD",
			BillingEmail:    req.Email,
		},
		Metadata: req.Metadata,
	}

	// Set default settings
	um.setDefaultSettings(customer)

	// Store customer
	um.users[customerID] = customer
	if err := um.saveCustomer(customer); err != nil {
		delete(um.users, customerID)
		return nil, fmt.Errorf("failed to save customer: %w", err)
	}

	um.auditLogger.LogEvent("CUSTOMER_CREATED", map[string]interface{}{
		"customer_id": customerID,
		"email":       req.Email,
		"plan":        req.Plan,
	})

	logrus.Infof("Customer created: %s (%s)", customer.Name, customer.Email)
	return customer, nil
}

// GetCustomer retrieves a customer by ID
func (um *UserManager) GetCustomer(customerID string) (*Customer, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	customer, exists := um.users[customerID]
	if !exists {
		return nil, fmt.Errorf("customer not found: %s", customerID)
	}

	return customer, nil
}

// GetCustomerByEmail retrieves a customer by email
func (um *UserManager) GetCustomerByEmail(email string) (*Customer, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	for _, customer := range um.users {
		if customer.Email == email {
			return customer, nil
		}
	}

	return nil, fmt.Errorf("customer not found with email: %s", email)
}

// GetCustomerByAPIKey retrieves a customer by API key
func (um *UserManager) GetCustomerByAPIKey(apiKey string) (*Customer, error) {
	um.mu.RLock()
	defer um.mu.RUnlock()

	for _, customer := range um.users {
		if customer.APIKey == apiKey {
			return customer, nil
		}
	}

	return nil, fmt.Errorf("customer not found with API key")
}

// ListCustomers returns all customers
func (um *UserManager) ListCustomers() []*Customer {
	um.mu.RLock()
	defer um.mu.RUnlock()

	customers := make([]*Customer, 0, len(um.users))
	for _, customer := range um.users {
		customers = append(customers, customer)
	}

	return customers
}

// UpdateCustomer updates customer information
func (um *UserManager) UpdateCustomer(customerID string, updates map[string]interface{}) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	customer, exists := um.users[customerID]
	if !exists {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	// Apply updates
	if name, ok := updates["name"].(string); ok {
		customer.Name = name
	}
	if company, ok := updates["company"].(string); ok {
		customer.Company = company
	}
	if plan, ok := updates["plan"].(string); ok {
		customer.Plan = plan
		customer.ResourceQuotas = um.getDefaultQuotas(plan)
	}
	if status, ok := updates["status"].(string); ok {
		customer.Status = CustomerStatus(status)
	}

	customer.UpdatedAt = time.Now()

	// Save customer
	if err := um.saveCustomer(customer); err != nil {
		return fmt.Errorf("failed to save customer: %w", err)
	}

	um.auditLogger.LogEvent("CUSTOMER_UPDATED", map[string]interface{}{
		"customer_id": customerID,
		"updates":     updates,
	})

	return nil
}

// DeleteCustomer deletes a customer
func (um *UserManager) DeleteCustomer(customerID string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	customer, exists := um.users[customerID]
	if !exists {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	// Check for active deployments
	if len(customer.Deployments) > 0 {
		return fmt.Errorf("cannot delete customer with active deployments")
	}

	// Remove from storage
	if err := um.store.DeleteDeploymentState(fmt.Sprintf("customer_%s", customerID)); err != nil {
		logrus.Warnf("Failed to delete customer from storage: %v", err)
	}

	// Remove from memory
	delete(um.users, customerID)

	um.auditLogger.LogEvent("CUSTOMER_DELETED", map[string]interface{}{
		"customer_id": customerID,
		"email":       customer.Email,
	})

	logrus.Infof("Customer deleted: %s (%s)", customer.Name, customer.Email)
	return nil
}

// UpdateResourceUsage updates customer resource usage
func (um *UserManager) UpdateResourceUsage(customerID string, usage ResourceUsage) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	customer, exists := um.users[customerID]
	if !exists {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	customer.UsedResources = usage
	customer.UsedResources.LastUpdated = time.Now()
	customer.UpdatedAt = time.Now()

	// Save customer
	if err := um.saveCustomer(customer); err != nil {
		return fmt.Errorf("failed to save customer: %w", err)
	}

	return nil
}

// CheckResourceQuota checks if customer can use requested resources
func (um *UserManager) CheckResourceQuota(customerID string, requestedResources ResourceUsage) error {
	um.mu.RLock()
	defer um.mu.RUnlock()

	customer, exists := um.users[customerID]
	if !exists {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	quotas := customer.ResourceQuotas
	current := customer.UsedResources

	// Check CPU quota
	if current.UsedCPU+requestedResources.UsedCPU > quotas.MaxCPU {
		return fmt.Errorf("CPU quota exceeded: %.2f/%.2f cores", current.UsedCPU+requestedResources.UsedCPU, quotas.MaxCPU)
	}

	// Check Memory quota
	if current.UsedMemory+requestedResources.UsedMemory > quotas.MaxMemory {
		return fmt.Errorf("memory quota exceeded: %d/%d MB", current.UsedMemory+requestedResources.UsedMemory, quotas.MaxMemory)
	}

	// Check Container quota
	if current.ActiveContainers+requestedResources.ActiveContainers > quotas.MaxContainers {
		return fmt.Errorf("container quota exceeded: %d/%d containers", current.ActiveContainers+requestedResources.ActiveContainers, quotas.MaxContainers)
	}

	// Check Apps quota
	if current.TotalApps+requestedResources.TotalApps > quotas.MaxApps {
		return fmt.Errorf("apps quota exceeded: %d/%d apps", current.TotalApps+requestedResources.TotalApps, quotas.MaxApps)
	}

	return nil
}

// AddLicense adds a license to a customer
func (um *UserManager) AddLicense(customerID, licenseID string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	customer, exists := um.users[customerID]
	if !exists {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	// Check if license already exists
	for _, existing := range customer.Licenses {
		if existing == licenseID {
			return fmt.Errorf("license already assigned to customer")
		}
	}

	customer.Licenses = append(customer.Licenses, licenseID)
	customer.UpdatedAt = time.Now()

	// Save customer
	if err := um.saveCustomer(customer); err != nil {
		return fmt.Errorf("failed to save customer: %w", err)
	}

	um.auditLogger.LogEvent("LICENSE_ASSIGNED", map[string]interface{}{
		"customer_id": customerID,
		"license_id":  licenseID,
	})

	return nil
}

// RemoveLicense removes a license from a customer
func (um *UserManager) RemoveLicense(customerID, licenseID string) error {
	um.mu.Lock()
	defer um.mu.Unlock()

	customer, exists := um.users[customerID]
	if !exists {
		return fmt.Errorf("customer not found: %s", customerID)
	}

	// Find and remove license
	for i, existing := range customer.Licenses {
		if existing == licenseID {
			customer.Licenses = append(customer.Licenses[:i], customer.Licenses[i+1:]...)
			customer.UpdatedAt = time.Now()

			// Save customer
			if err := um.saveCustomer(customer); err != nil {
				return fmt.Errorf("failed to save customer: %w", err)
			}

			um.auditLogger.LogEvent("LICENSE_REVOKED", map[string]interface{}{
				"customer_id": customerID,
				"license_id":  licenseID,
			})

			return nil
		}
	}

	return fmt.Errorf("license not found for customer")
}

// HasLicense checks if customer has a specific license
func (um *UserManager) HasLicense(customerID, licenseID string) bool {
	um.mu.RLock()
	defer um.mu.RUnlock()

	customer, exists := um.users[customerID]
	if !exists {
		return false
	}

	for _, existing := range customer.Licenses {
		if existing == licenseID {
			return true
		}
	}

	return false
}

// Helper functions

func (um *UserManager) generateCustomerID() string {
	return fmt.Sprintf("cust_%d", time.Now().Unix())
}

func (um *UserManager) generateAPIKey() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sa_" + hex.EncodeToString(bytes), nil
}

func (um *UserManager) generateSubdomainPrefix(name, company string) string {
	base := ""
	if company != "" {
		base = company
	} else if name != "" {
		base = name
	} else {
		base = "customer"
	}

	// Clean and format subdomain prefix
	prefix := ""
	for _, char := range base {
		if (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') {
			prefix += string(char)
		}
	}

	if len(prefix) == 0 {
		prefix = "customer"
	}

	if len(prefix) > 20 {
		prefix = prefix[:20]
	}

	return fmt.Sprintf("%s-%d", prefix, time.Now().Unix()%10000)
}

func (um *UserManager) getDefaultQuotas(plan string) ResourceQuotas {
	switch plan {
	case "free":
		return ResourceQuotas{
			MaxCPU:           1.0,
			MaxMemory:        512,
			MaxStorage:       5,
			MaxBandwidth:     10,
			MaxContainers:    2,
			MaxApps:          1,
			MaxDeployments:   5,
			MaxCustomDomains: 0,
		}
	case "starter":
		return ResourceQuotas{
			MaxCPU:           2.0,
			MaxMemory:        2048,
			MaxStorage:       20,
			MaxBandwidth:     50,
			MaxContainers:    5,
			MaxApps:          3,
			MaxDeployments:   15,
			MaxCustomDomains: 1,
		}
	case "professional":
		return ResourceQuotas{
			MaxCPU:           4.0,
			MaxMemory:        8192,
			MaxStorage:       100,
			MaxBandwidth:     200,
			MaxContainers:    20,
			MaxApps:          10,
			MaxDeployments:   50,
			MaxCustomDomains: 5,
		}
	case "enterprise":
		return ResourceQuotas{
			MaxCPU:           16.0,
			MaxMemory:        32768,
			MaxStorage:       500,
			MaxBandwidth:     1000,
			MaxContainers:    100,
			MaxApps:          50,
			MaxDeployments:   200,
			MaxCustomDomains: 25,
		}
	default:
		return um.getDefaultQuotas("free")
	}
}

func (um *UserManager) setDefaultSettings(customer *Customer) {
	if customer.Settings.DefaultRegion == "" {
		customer.Settings.DefaultRegion = "us-east-1"
	}
	if customer.Settings.DeploymentStrategy == "" {
		customer.Settings.DeploymentStrategy = "rolling"
	}
	customer.Settings.AutoSSL = true
	customer.Settings.HealthCheckEnabled = true
	customer.Settings.NotificationsEmail = true
	customer.Settings.BackupRetention = 7

	if customer.Settings.Environment == nil {
		customer.Settings.Environment = make(map[string]string)
	}
}

func (um *UserManager) saveCustomer(customer *Customer) error {
	customerData := map[string]interface{}{
		"customer": customer,
	}

	return um.store.StoreDeploymentState(fmt.Sprintf("customer_%s", customer.ID), customerData)
}

func (um *UserManager) loadUsers() error {
	logrus.Info("Loading customers from secure storage...")
	
	// Load all customer data from storage
	data, err := um.store.LoadData()
	if err != nil {
		return fmt.Errorf("failed to load customer data: %w", err)
	}

	if data == nil || data.Data == nil {
		logrus.Info("No existing customer data found, starting fresh")
		return nil
	}

	// Load customers from storage
	if customersData, exists := data.Data["customers"]; exists {
		if customersMap, ok := customersData.(map[string]interface{}); ok {
			for customerID, customerData := range customersMap {
				if customerMap, ok := customerData.(map[string]interface{}); ok {
					customer := &Customer{}
					
					// Deserialize customer data
					if id, ok := customerMap["id"].(string); ok {
						customer.ID = id
					}
					if email, ok := customerMap["email"].(string); ok {
						customer.Email = email
					}
					if name, ok := customerMap["name"].(string); ok {
						customer.Name = name
					}
					if company, ok := customerMap["company"].(string); ok {
						customer.Company = company
					}
					if plan, ok := customerMap["plan"].(string); ok {
						customer.Plan = plan
					}
					if status, ok := customerMap["status"].(string); ok {
						customer.Status = CustomerStatus(status)
					}
					if apiKey, ok := customerMap["api_key"].(string); ok {
						customer.APIKey = apiKey
					}
					if subdomainPrefix, ok := customerMap["subdomain_prefix"].(string); ok {
						customer.SubdomainPrefix = subdomainPrefix
					}
					if createdAt, ok := customerMap["created_at"].(string); ok {
						if t, err := time.Parse(time.RFC3339, createdAt); err == nil {
							customer.CreatedAt = t
						}
					}
					if updatedAt, ok := customerMap["updated_at"].(string); ok {
						if t, err := time.Parse(time.RFC3339, updatedAt); err == nil {
							customer.UpdatedAt = t
						}
					}

					// Load resource quotas
					if quotasData, ok := customerMap["resource_quotas"].(map[string]interface{}); ok {
						quotas := ResourceQuotas{}
						if maxCPU, ok := quotasData["max_cpu"].(float64); ok {
							quotas.MaxCPU = maxCPU
						}
						if maxMemory, ok := quotasData["max_memory"].(float64); ok {
							quotas.MaxMemory = int64(maxMemory)
						}
						if maxStorage, ok := quotasData["max_storage"].(float64); ok {
							quotas.MaxStorage = int64(maxStorage)
						}
						if maxBandwidth, ok := quotasData["max_bandwidth"].(float64); ok {
							quotas.MaxBandwidth = int64(maxBandwidth)
						}
						if maxContainers, ok := quotasData["max_containers"].(float64); ok {
							quotas.MaxContainers = int(maxContainers)
						}
						if maxApps, ok := quotasData["max_apps"].(float64); ok {
							quotas.MaxApps = int(maxApps)
						}
						if maxDeployments, ok := quotasData["max_deployments"].(float64); ok {
							quotas.MaxDeployments = int(maxDeployments)
						}
						if maxCustomDomains, ok := quotasData["max_custom_domains"].(float64); ok {
							quotas.MaxCustomDomains = int(maxCustomDomains)
						}
						customer.ResourceQuotas = quotas
					}

					// Load resource usage
					if usageData, ok := customerMap["used_resources"].(map[string]interface{}); ok {
						usage := ResourceUsage{}
						if usedCPU, ok := usageData["used_cpu"].(float64); ok {
							usage.UsedCPU = usedCPU
						}
						if usedMemory, ok := usageData["used_memory"].(float64); ok {
							usage.UsedMemory = int64(usedMemory)
						}
						if usedStorage, ok := usageData["used_storage"].(float64); ok {
							usage.UsedStorage = int64(usedStorage)
						}
						if usedBandwidth, ok := usageData["used_bandwidth"].(float64); ok {
							usage.UsedBandwidth = int64(usedBandwidth)
						}
						if activeContainers, ok := usageData["active_containers"].(float64); ok {
							usage.ActiveContainers = int(activeContainers)
						}
						if totalApps, ok := usageData["total_apps"].(float64); ok {
							usage.TotalApps = int(totalApps)
						}
						if totalDeployments, ok := usageData["total_deployments"].(float64); ok {
							usage.TotalDeployments = int(totalDeployments)
						}
						if customDomains, ok := usageData["custom_domains"].(float64); ok {
							usage.CustomDomains = int(customDomains)
						}
						if lastUpdated, ok := usageData["last_updated"].(string); ok {
							if t, err := time.Parse(time.RFC3339, lastUpdated); err == nil {
								usage.LastUpdated = t
							}
						}
						customer.UsedResources = usage
					}

					// Store in memory
					um.users[customerID] = customer
					logrus.Debugf("Loaded customer: %s (%s)", customer.Name, customer.Email)
				}
			}
		}
	}

	logrus.Infof("Successfully loaded %d customers from storage", len(um.users))
	return nil
}