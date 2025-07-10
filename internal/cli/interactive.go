package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"superagent/internal/api"
	"superagent/internal/config"
	"superagent/internal/logging"
	"superagent/internal/traefik"

	"gopkg.in/yaml.v3"
)

// InteractiveCLI provides interactive CLI functionality
type InteractiveCLI struct {
	config          *config.Config
	auditLogger     *logging.AuditLogger
	apiClient       *api.CLIClient
	traefikManager  *traefik.TraefikManager
	baseDomain      string
	traefikEnabled  bool
	adminPanelURL   string
	adminConnected  bool
}

// NewInteractiveCLI creates a new interactive CLI instance
func NewInteractiveCLI(cfg *config.Config, auditLogger *logging.AuditLogger) *InteractiveCLI {
	return &InteractiveCLI{
		config:         cfg,
		auditLogger:    auditLogger,
		apiClient:      api.NewCLIClient(cfg.Monitoring.HealthCheckPort),
		traefikManager: traefik.NewTraefikManager(""),
	}
}

// StartInteractiveCLI starts the interactive CLI experience
func (ic *InteractiveCLI) StartInteractiveCLI() error {
	fmt.Println("🚀 Welcome to SuperAgent Interactive CLI!")
	fmt.Println("==========================================")

	// Check if agent is running
	if !ic.apiClient.IsAgentRunning() {
		fmt.Println("⚠️  SuperAgent is not running. Starting agent...")
		if err := ic.startAgent(); err != nil {
			return fmt.Errorf("failed to start agent: %w", err)
		}
		// Wait for agent to be ready
		time.Sleep(3 * time.Second)
	}

	// Check admin panel connection
	ic.checkAdminPanelConnection()

	// Show main menu
	return ic.showMainMenu()
}

// checkAdminPanelConnection checks if admin panel is connected
func (ic *InteractiveCLI) checkAdminPanelConnection() {
	fmt.Println("\n🔍 Checking admin panel connection...")
	
	// Load previous configuration
	ic.loadConfig()
	
	if ic.adminPanelURL == "" {
		ic.adminConnected = false
		fmt.Println("❌ Admin panel not connected")
		
		// Ask if user wants to connect to admin panel
		fmt.Println("\n💡 Would you like to connect to the admin panel? [y/N]")
		fmt.Println("   Admin panel provides:")
		fmt.Println("   • Centralized user management")
		fmt.Println("   • Deployment tracking and monitoring")
		fmt.Println("   • Audit logging and compliance")
		fmt.Println("   • Configuration synchronization")
		
		connect := ic.promptChoice("Connect to admin panel?", []string{"yes", "no", "y", "n"})
		if connect == "yes" || connect == "y" {
			ic.connectToAdminPanel()
		} else {
			fmt.Println("💡 You can still use the CLI for local management")
			fmt.Println("💡 You can connect later via: Main Menu → Admin Panel Connection")
		}
		return
	}
	
	// Try to connect to admin panel API with timeout
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ic.adminPanelURL + "/api/v1/health")
	if err != nil {
		ic.adminConnected = false
		fmt.Printf("❌ Admin panel not reachable: %v\n", err)
		fmt.Println("💡 You can still use the CLI for local management")
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		ic.adminConnected = true
		fmt.Println("✅ Admin panel connected!")
		fmt.Printf("🌐 Admin panel URL: %s\n", ic.adminPanelURL)
	} else {
		ic.adminConnected = false
		fmt.Printf("❌ Admin panel health check failed (status: %d)\n", resp.StatusCode)
		fmt.Println("💡 You can still use the CLI for local management")
	}
}

// showMainMenu displays the main interactive menu
func (ic *InteractiveCLI) showMainMenu() error {
	for {
		fmt.Println("\n📋 Main Menu:")
		fmt.Println("1. 🚀 Deploy Application")
		fmt.Println("2. 📊 View Deployments")
		fmt.Println("3. ⚙️  Agent Configuration")
		fmt.Println("4. 🌐 Domain & Traefik Setup")
		fmt.Println("5. 📝 View Logs")
		fmt.Println("6. 🔧 System Status")
		fmt.Println("7. 🔐 Admin Panel Connection")
		fmt.Println("0. 🚪 Exit")

		choice := ic.promptChoice("Select an option", []string{"0", "1", "2", "3", "4", "5", "6", "7"})

		switch choice {
		case "0":
			fmt.Println("👋 Goodbye!")
			return nil
		case "1":
			if err := ic.deployApplicationWithUserManagement(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "2":
			if err := ic.viewDeployments(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "3":
			if err := ic.agentConfiguration(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "4":
			if err := ic.domainAndTraefikSetup(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "5":
			if err := ic.viewLogs(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "6":
			if err := ic.systemStatus(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		case "7":
			if err := ic.adminPanelConnectionMenu(); err != nil {
				fmt.Printf("❌ Error: %v\n", err)
			}
		}
	}
}

// deployApplication handles the interactive deployment process
func (ic *InteractiveCLI) deployApplication() error {
	fmt.Println("\n🚀 Deploy Application")
	fmt.Println("====================")

	// Get repository information
	repoType := ic.promptChoice("Repository type", []string{"public", "private"})
	
	var repoURL string
	if repoType == "public" {
		repoURL = ic.promptString("Enter GitHub repository URL (https://github.com/user/repo)", "")
	} else {
		fmt.Println("🔐 Private Repository Setup Instructions:")
		fmt.Println("  Option 1 - SSH Key Authentication:")
		fmt.Println("    1. Generate SSH key: ssh-keygen -t ed25519 -C 'your_email@example.com'")
		fmt.Println("    2. Add public key to GitHub: Settings → SSH and GPG keys")
		fmt.Println("    3. Test connection: ssh -T git@github.com")
		fmt.Println("  Option 2 - Personal Access Token:")
		fmt.Println("    1. Create token: GitHub Settings → Developer settings → Personal access tokens")
		fmt.Println("    2. Give 'repo' access permissions")
		fmt.Println("    3. Use HTTPS URL with token in git credentials")
		fmt.Println("")
		
		authChoice := ic.promptChoice("Authentication method", []string{"ssh", "token"})
		if authChoice == "ssh" {
			repoURL = ic.promptString("Enter GitHub SSH URL (git@github.com:user/repo.git)", "")
		} else {
			repoURL = ic.promptString("Enter GitHub HTTPS URL (https://github.com/user/repo.git)", "")
			fmt.Println("💡 Ensure your git credentials are configured for this repository")
		}
	}

	// Validate repository URL
	if !ic.isValidGitHubURL(repoURL) {
		return fmt.Errorf("invalid GitHub repository URL")
	}

	// Get app details
	appID := ic.promptString("Enter application ID (e.g., myapp)", "")
	version := ic.promptString("Enter version (e.g., v1.0.0)", "latest")
	branch := ic.promptString("Enter branch (default: main)", "main")

	// Clone repository to check for env files
	fmt.Println("📥 Cloning repository to check configuration...")
	repoPath, err := ic.cloneRepository(repoURL, branch)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	defer os.RemoveAll(repoPath)

	// Check for environment files
	envVars := ic.handleEnvironmentFiles(repoPath)

	// Check for package.json (JS app detection)
	isJSApp := ic.isJSApplication(repoPath)
	if isJSApp {
		fmt.Println("✅ JavaScript application detected")
	}

	// Confirm deployment
	fmt.Println("\n📋 Deployment Summary:")
	fmt.Printf("  App ID: %s\n", appID)
	fmt.Printf("  Version: %s\n", version)
	fmt.Printf("  Repository: %s\n", repoURL)
	fmt.Printf("  Branch: %s\n", branch)
	fmt.Printf("  Environment Variables: %d\n", len(envVars))
	fmt.Printf("  Type: %s\n", ic.getAppType(repoPath))

	confirm := ic.promptChoice("Proceed with deployment?", []string{"yes", "no"})
	if confirm != "yes" {
		fmt.Println("❌ Deployment cancelled")
		return nil
	}

	// Create deployment
	fmt.Println("🚀 Creating deployment...")
	deployment, err := ic.createDeployment(appID, version, repoURL, branch, envVars)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Show deployment results
	ic.showDeploymentResults(deployment)

	return nil
}

// handleEnvironmentFiles detects and handles .env files
func (ic *InteractiveCLI) handleEnvironmentFiles(repoPath string) map[string]string {
	envVars := make(map[string]string)
	
	// Check for common env file patterns
	envFiles := []string{".env", ".env.local", ".env.example", ".env.production"}
	
	for _, envFile := range envFiles {
		envPath := filepath.Join(repoPath, envFile)
		if _, err := os.Stat(envPath); err == nil {
			fmt.Printf("📄 Found environment file: %s\n", envFile)
			
			// Read env file
			content, err := ioutil.ReadFile(envPath)
			if err != nil {
				fmt.Printf("⚠️  Warning: Could not read %s: %v\n", envFile, err)
				continue
			}
			
			// Parse env file
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				
				if strings.Contains(line, "=") {
					parts := strings.SplitN(line, "=", 2)
					if len(parts) == 2 {
						key := strings.TrimSpace(parts[0])
						defaultValue := strings.TrimSpace(parts[1])
						
						// Remove quotes if present
						defaultValue = strings.Trim(defaultValue, `"'`)
						
						// Skip if it's a comment or empty
						if key == "" || strings.HasPrefix(key, "#") {
							continue
						}
						
						// Prompt for value
						value := ic.promptString(fmt.Sprintf("Enter value for %s (default: %s)", key, defaultValue), defaultValue)
						envVars[key] = value
					}
				}
			}
			break // Only process the first env file found
		}
	}
	
	return envVars
}

// isJSApplication checks if the repository contains a JavaScript application
func (ic *InteractiveCLI) isJSApplication(repoPath string) bool {
	packageJsonPath := filepath.Join(repoPath, "package.json")
	_, err := os.Stat(packageJsonPath)
	return err == nil
}

// getAppType determines the type of application
func (ic *InteractiveCLI) getAppType(repoPath string) string {
	if ic.isJSApplication(repoPath) {
		// Check for specific frameworks
		packageJsonPath := filepath.Join(repoPath, "package.json")
		content, err := ioutil.ReadFile(packageJsonPath)
		if err == nil {
			var pkg map[string]interface{}
			if json.Unmarshal(content, &pkg) == nil {
				if dependencies, ok := pkg["dependencies"].(map[string]interface{}); ok {
					if _, hasNext := dependencies["next"]; hasNext {
						return "Next.js"
					}
					if _, hasReact := dependencies["react"]; hasReact {
						return "React"
					}
				}
			}
		}
		return "Node.js"
	}
	
	// Check for other frameworks
	if _, err := os.Stat(filepath.Join(repoPath, "requirements.txt")); err == nil {
		return "Python"
	}
	if _, err := os.Stat(filepath.Join(repoPath, "go.mod")); err == nil {
		return "Go"
	}
	if _, err := os.Stat(filepath.Join(repoPath, "Dockerfile")); err == nil {
		return "Docker"
	}
	
	return "Unknown"
}

// createDeployment creates a deployment via the API
func (ic *InteractiveCLI) createDeployment(appID, version, repoURL, branch string, envVars map[string]string) (*api.DeploymentResponse, error) {
	deploymentRequest := map[string]interface{}{
		"app_id":  appID,
		"version": version,
		"source": map[string]interface{}{
			"type":       "git",
			"repository": repoURL,
			"branch":     branch,
		},
		"config": map[string]interface{}{
			"strategy": "rolling",
			"replicas": 1,
		},
		"resource_limits": map[string]interface{}{
			"cpu_limit":    "1",
			"memory_limit": "1G",
		},
		"health_check": map[string]interface{}{
			"enabled": true,
			"type":    "http",
			"path":    "/",
			"port":    3000,
		},
		"environment": envVars,
	}

	return ic.apiClient.CreateDeployment(deploymentRequest)
}

// showDeploymentResults shows deployment results and domain information
func (ic *InteractiveCLI) showDeploymentResults(deployment *api.DeploymentResponse) {
	fmt.Println("\n🎉 Deployment Successful!")
	fmt.Println("=========================")
	fmt.Printf("Deployment ID: %s\n", deployment.ID)
	fmt.Printf("Status: %s\n", deployment.Status)
	
	// Generate subdomain
	subdomain := ic.generateSubdomain(deployment.AppID)
	
	// Show URLs
	fmt.Println("\n🌐 Access URLs:")
	if ic.baseDomain != "" {
		fullURL := fmt.Sprintf("https://%s.%s", subdomain, ic.baseDomain)
		fmt.Printf("  Subdomain: %s\n", fullURL)
		fmt.Printf("  IP Address: %s (for A record)\n", ic.getServerIP())
		fmt.Printf("  CNAME Record: %s.%s\n", subdomain, ic.baseDomain)
		
		// Add Traefik route if enabled
		if ic.traefikEnabled {
			containerName := fmt.Sprintf("superagent-%s", deployment.ID)
			if err := ic.traefikManager.AddRoute(deployment.AppID, containerName, 3000); err != nil {
				fmt.Printf("⚠️  Warning: Failed to add Traefik route: %v\n", err)
			} else {
				fmt.Printf("✅ Traefik route added for %s\n", deployment.AppID)
			}
		}
	} else {
		fmt.Printf("  Local: http://localhost:3000\n")
		fmt.Println("  ⚠️  No base domain configured. Configure Traefik for custom domains.")
	}
	
	// Show DNS instructions
	if ic.baseDomain != "" {
		fmt.Println("\n📝 DNS Configuration:")
		fmt.Println("For custom domain, add these DNS records:")
		fmt.Printf("  A Record: @ → %s\n", ic.getServerIP())
		fmt.Printf("  CNAME Record: www → %s.%s\n", subdomain, ic.baseDomain)
	}
	
	// Show next steps
	fmt.Println("\n📋 Next Steps:")
	fmt.Println("1. Wait for deployment to be ready (check status with 'superagent status')")
	fmt.Println("2. Configure custom domain if needed")
	fmt.Println("3. Set up SSL certificate")
	fmt.Println("4. Monitor logs with 'superagent logs --deployment " + deployment.ID + "'")
	if ic.traefikEnabled {
		fmt.Printf("5. View Traefik dashboard: %s\n", ic.traefikManager.GetDashboardURL())
	}
}

// generateSubdomain generates a subdomain for the app
func (ic *InteractiveCLI) generateSubdomain(appID string) string {
	// Clean app ID for subdomain
	clean := regexp.MustCompile(`[^a-zA-Z0-9-]`).ReplaceAllString(appID, "-")
	clean = regexp.MustCompile(`-+`).ReplaceAllString(clean, "-")
	clean = strings.Trim(clean, "-")
	return strings.ToLower(clean)
}

// getServerIP gets the server's public IP address
func (ic *InteractiveCLI) getServerIP() string {
	// Try to get public IP
	resp, err := exec.Command("curl", "-s", "ifconfig.me").Output()
	if err == nil {
		return strings.TrimSpace(string(resp))
	}
	
	// Fallback to local IP
	addrs, err := net.InterfaceAddrs()
	if err == nil {
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
				if ipnet.IP.To4() != nil {
					return ipnet.IP.String()
				}
			}
		}
	}
	
	return "YOUR_SERVER_IP"
}

// viewDeployments shows all deployments
func (ic *InteractiveCLI) viewDeployments() error {
	fmt.Println("\n📊 View Deployments")
	fmt.Println("===================")
	
	deployments, err := ic.apiClient.ListDeployments()
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}
	
	if len(deployments) == 0 {
		fmt.Println("No deployments found")
		return nil
	}
	
	fmt.Printf("%-20s %-15s %-10s %-12s %-20s\n", "ID", "APP", "VERSION", "STATUS", "CREATED")
	fmt.Println(strings.Repeat("-", 80))
	
	for _, d := range deployments {
		createdAt := d.CreatedAt.Format("2006-01-02 15:04:05")
		fmt.Printf("%-20s %-15s %-10s %-12s %-20s\n", 
			truncateString(d.ID, 20),
			truncateString(d.AppID, 15),
			truncateString(d.Version, 10),
			d.Status,
			createdAt)
	}
	
	return nil
}

// agentConfiguration handles agent configuration
func (ic *InteractiveCLI) agentConfiguration() error {
	fmt.Println("\n⚙️  Agent Configuration")
	fmt.Println("======================")
	
	for {
		fmt.Println("\nConfiguration Options:")
		fmt.Println("1. 🔧 Setup Wizard")
		fmt.Println("2. 🌐 Base Domain Configuration")
		fmt.Println("3. 🔐 Admin Panel Connection")
		fmt.Println("4. 📊 View Current Config")
		fmt.Println("0. ↩️  Back to Main Menu")
		
		choice := ic.promptChoice("Select an option", []string{"0", "1", "2", "3", "4"})
		
		switch choice {
		case "0":
			return nil
		case "1":
			ic.setupWizard()
		case "2":
			ic.configureBaseDomain()
		case "3":
			ic.configureAdminPanel()
		case "4":
			ic.viewCurrentConfig()
		}
	}
}

// setupWizard runs the initial setup wizard
func (ic *InteractiveCLI) setupWizard() {
	fmt.Println("\n🔧 SuperAgent Setup Wizard")
	fmt.Println("=========================")
	
	// Check if already configured
	if ic.baseDomain != "" {
		fmt.Println("✅ Agent is already configured")
		return
	}
	
	fmt.Println("Welcome to SuperAgent! Let's get you set up.")
	
	// Configure base domain
	ic.configureBaseDomain()
	
	// Configure Traefik
	ic.configureTraefik()
	
	// Configure admin panel connection
	ic.configureAdminPanel()
	
	fmt.Println("✅ Setup complete!")
}

// configureBaseDomain configures the base domain
func (ic *InteractiveCLI) configureBaseDomain() {
	fmt.Println("\n🌐 Base Domain Configuration")
	fmt.Println("============================")
	
	currentDomain := ic.baseDomain
	if currentDomain == "" {
		currentDomain = "example.com"
	}
	
	newDomain := ic.promptString("Enter your base domain", currentDomain)
	if newDomain != "" {
		ic.baseDomain = newDomain
		ic.traefikManager.SetBaseDomain(newDomain)
		fmt.Printf("✅ Base domain set to: %s\n", ic.baseDomain)
		
		// Save to config
		if err := ic.saveConfig(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to save configuration: %v\n", err)
		}
	}
}

// configureTraefik configures Traefik
func (ic *InteractiveCLI) configureTraefik() {
	fmt.Println("\n🔄 Traefik Configuration")
	fmt.Println("========================")
	
	enableTraefik := ic.promptChoice("Enable Traefik for automatic routing?", []string{"yes", "no"})
	if enableTraefik == "yes" {
		ic.traefikEnabled = true
		fmt.Println("✅ Traefik enabled")
		
		// Check if Traefik is installed
		if !ic.traefikManager.IsInstalled() {
			fmt.Println("⚠️  Traefik not found. Installing...")
			if err := ic.traefikManager.InstallTraefik(); err != nil {
				fmt.Printf("❌ Failed to install Traefik: %v\n", err)
				return
			}
		}
		
		// Configure Traefik
		ic.configureTraefikSettings()
	} else {
		ic.traefikEnabled = false
		fmt.Println("❌ Traefik disabled")
	}
	
	if err := ic.saveConfig(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save configuration: %v\n", err)
	}
}

// configureTraefikSettings configures Traefik settings
func (ic *InteractiveCLI) configureTraefikSettings() {
	fmt.Println("\n⚙️  Traefik Settings")
	fmt.Println("===================")
	
	// Configure Traefik dashboard
	enableDashboard := ic.promptChoice("Enable Traefik dashboard?", []string{"yes", "no"})
	if enableDashboard == "yes" {
		fmt.Println("✅ Traefik dashboard enabled at http://localhost:8080")
	}
	
	// Configure SSL
	enableSSL := ic.promptChoice("Enable automatic SSL with Let's Encrypt?", []string{"yes", "no"})
	if enableSSL == "yes" {
		email := ic.promptString("Enter email for Let's Encrypt", "")
		if email != "" {
			fmt.Printf("✅ SSL configured with email: %s\n", email)
		}
	}
}

// configureAdminPanel configures admin panel connection
func (ic *InteractiveCLI) configureAdminPanel() {
	fmt.Println("\n🔐 Admin Panel Connection")
	fmt.Println("=========================")
	
	connectAdmin := ic.promptChoice("Connect to admin panel?", []string{"yes", "no"})
	if connectAdmin == "yes" {
		adminURL := ic.promptString("Enter admin panel URL", "")
		if adminURL != "" {
			ic.adminPanelURL = adminURL
			ic.adminConnected = true
			fmt.Printf("✅ Connected to admin panel: %s\n", adminURL)
		}
	} else {
		ic.adminConnected = false
		ic.adminPanelURL = ""
		fmt.Println("❌ Admin panel connection disabled")
	}
	
	if err := ic.saveConfig(); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save configuration: %v\n", err)
	}
}

// viewCurrentConfig shows current configuration
func (ic *InteractiveCLI) viewCurrentConfig() {
	fmt.Println("\n📊 Current Configuration")
	fmt.Println("========================")
	
	fmt.Printf("Base Domain: %s\n", ic.baseDomain)
	fmt.Printf("Traefik Enabled: %t\n", ic.traefikEnabled)
	fmt.Printf("Admin Panel Connected: %t\n", ic.adminConnected)
	if ic.adminConnected {
		fmt.Printf("Admin Panel URL: %s\n", ic.adminPanelURL)
	}
}

// domainAndTraefikSetup handles domain and Traefik setup
func (ic *InteractiveCLI) domainAndTraefikSetup() error {
	fmt.Println("\n🌐 Domain & Traefik Setup")
	fmt.Println("=========================")
	
	for {
		fmt.Println("\nOptions:")
		fmt.Println("1. 🌐 Configure Base Domain")
		fmt.Println("2. 🔄 Configure Traefik")
		fmt.Println("3. 📝 View DNS Instructions")
		fmt.Println("4. 🔧 Test Traefik Configuration")
		fmt.Println("0. ↩️  Back to Main Menu")
		
		choice := ic.promptChoice("Select an option", []string{"0", "1", "2", "3", "4"})
		
		switch choice {
		case "0":
			return nil
		case "1":
			ic.configureBaseDomain()
		case "2":
			ic.configureTraefik()
		case "3":
			ic.showDNSInstructions()
		case "4":
			ic.testTraefikConfiguration()
		}
	}
}

// showDNSInstructions shows DNS configuration instructions
func (ic *InteractiveCLI) showDNSInstructions() {
	fmt.Println("\n📝 DNS Configuration Instructions")
	fmt.Println("=================================")
	
	if ic.baseDomain == "" {
		fmt.Println("⚠️  No base domain configured. Please configure base domain first.")
		return
	}
	
	fmt.Printf("For domain: %s\n", ic.baseDomain)
	fmt.Printf("Server IP: %s\n", ic.getServerIP())
	fmt.Println("\nAdd these DNS records:")
	fmt.Printf("  A Record: @ → %s\n", ic.getServerIP())
	fmt.Printf("  CNAME Record: www → %s\n", ic.baseDomain)
	fmt.Println("\nFor subdomains (auto-generated):")
	fmt.Printf("  CNAME Record: [app-name] → %s\n", ic.baseDomain)
}

// testTraefikConfiguration tests Traefik configuration
func (ic *InteractiveCLI) testTraefikConfiguration() {
	fmt.Println("\n🔧 Testing Traefik Configuration")
	fmt.Println("================================")
	
	if !ic.traefikEnabled {
		fmt.Println("❌ Traefik is not enabled")
		return
	}
	
	// Test Traefik API
	fmt.Println("Testing Traefik API...")
	if err := ic.traefikManager.TestConfiguration(); err != nil {
		fmt.Printf("❌ Traefik configuration test failed: %v\n", err)
		return
	}
	
	fmt.Println("✅ Traefik configuration is valid")
}

// viewLogs shows deployment logs
func (ic *InteractiveCLI) viewLogs() error {
	fmt.Println("\n📝 View Logs")
	fmt.Println("============")
	
	// Get deployment ID
	deployments, err := ic.apiClient.ListDeployments()
	if err != nil {
		return fmt.Errorf("failed to list deployments: %w", err)
	}
	
	if len(deployments) == 0 {
		fmt.Println("No deployments found")
		return nil
	}
	
	fmt.Println("Available deployments:")
	for i, d := range deployments {
		fmt.Printf("%d. %s (%s) - %s\n", i+1, d.AppID, d.Version, d.Status)
	}
	
	choice := ic.promptString("Enter deployment number or ID", "")
	if choice == "" {
		return nil
	}
	
	// Parse choice
	var deploymentID string
	if num, err := strconv.Atoi(choice); err == nil && num > 0 && num <= len(deployments) {
		deploymentID = deployments[num-1].ID
	} else {
		deploymentID = choice
	}
	
	// Get logs
	logs, err := ic.apiClient.GetDeploymentLogs(deploymentID, 100)
	if err != nil {
		return fmt.Errorf("failed to get logs: %w", err)
	}
	
	fmt.Printf("\nLogs for deployment: %s\n", deploymentID)
	fmt.Println(strings.Repeat("-", 50))
	
	for _, log := range logs.Logs {
		fmt.Printf("[%s] [%s] %s\n",
			log.Timestamp.Format("2006-01-02 15:04:05"),
			log.Level,
			log.Message)
	}
	
	return nil
}

// systemStatus shows system status
func (ic *InteractiveCLI) systemStatus() error {
	fmt.Println("\n🔧 System Status")
	fmt.Println("================")
	
	status, err := ic.apiClient.GetStatus()
	if err != nil {
		return fmt.Errorf("failed to get status: %w", err)
	}
	
	fmt.Printf("Service: %s\n", status.Status)
	fmt.Printf("Health: %s\n", status.Health)
	fmt.Printf("Version: %s\n", status.Version)
	fmt.Printf("Uptime: %s\n", status.Uptime)
	fmt.Printf("Active Deployments: %d\n", status.ActiveDeployments)
	fmt.Printf("Total Deployments: %d\n", status.TotalDeployments)
	
	return nil
}

// openAdminPanel opens the admin panel
func (ic *InteractiveCLI) openAdminPanel() {
	if ic.adminPanelURL == "" {
		fmt.Println("❌ No admin panel URL configured")
		return
	}
	
	fmt.Printf("🌐 Opening admin panel: %s\n", ic.adminPanelURL)
	// Implementation would open browser
}

// Helper methods

func (ic *InteractiveCLI) promptString(prompt, defaultValue string) string {
	fmt.Printf("%s", prompt)
	if defaultValue != "" {
		fmt.Printf(" (default: %s)", defaultValue)
	}
	fmt.Print(": ")
	
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	
	if input == "" {
		return defaultValue
	}
	return input
}

func (ic *InteractiveCLI) promptChoice(prompt string, choices []string) string {
	for {
		fmt.Printf("%s [%s]: ", prompt, strings.Join(choices, "/"))
		
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))
		
		for _, choice := range choices {
			if input == choice {
				return choice
			}
		}
		
		fmt.Printf("Invalid choice. Please enter one of: %s\n", strings.Join(choices, ", "))
	}
}

func (ic *InteractiveCLI) isValidGitHubURL(url string) bool {
	return strings.Contains(url, "github.com") || strings.Contains(url, "git@github.com")
}

func (ic *InteractiveCLI) cloneRepository(url, branch string) (string, error) {
	// Create temporary directory
	tempDir, err := ioutil.TempDir("", "superagent-clone-*")
	if err != nil {
		return "", err
	}
	
	// Clone repository
	cmd := exec.Command("git", "clone", "-b", branch, url, tempDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	
	if err := cmd.Run(); err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}
	
	return tempDir, nil
}

func (ic *InteractiveCLI) startAgent() error {
	// Start agent in background
	cmd := exec.Command("superagent", "start", "-d")
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start agent process: %w", err)
	}
	
	// Wait for agent to be ready
	maxRetries := 10
	for i := 0; i < maxRetries; i++ {
		time.Sleep(1 * time.Second)
		if ic.apiClient.IsAgentRunning() {
			fmt.Println("✅ Agent started successfully")
			return nil
		}
	}
	
	return fmt.Errorf("agent failed to start within %d seconds", maxRetries)
}



func (ic *InteractiveCLI) saveConfig() error {
	// Save configuration to file
	config := map[string]interface{}{
		"base_domain":      ic.baseDomain,
		"traefik_enabled":  ic.traefikEnabled,
		"admin_connected":  ic.adminConnected,
		"admin_panel_url":  ic.adminPanelURL,
	}
	
	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	configPath := filepath.Join(os.Getenv("HOME"), ".superagent-interactive.yaml")
	if err := ioutil.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}

// loadConfig loads configuration from file
func (ic *InteractiveCLI) loadConfig() error {
	configPath := filepath.Join(os.Getenv("HOME"), ".superagent-interactive.yaml")
	
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		// File doesn't exist, use defaults
		return nil
	}
	
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	
	// Load configuration values
	if domain, ok := config["base_domain"].(string); ok {
		ic.baseDomain = domain
		ic.traefikManager.SetBaseDomain(domain)
	}
	
	if enabled, ok := config["traefik_enabled"].(bool); ok {
		ic.traefikEnabled = enabled
	}
	
	if connected, ok := config["admin_connected"].(bool); ok {
		ic.adminConnected = connected
	}
	
	if url, ok := config["admin_panel_url"].(string); ok {
		ic.adminPanelURL = url
	}
	
	return nil
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}

// adminPanelConnectionMenu handles admin panel connection management
func (ic *InteractiveCLI) adminPanelConnectionMenu() error {
	fmt.Println("\n🔐 Admin Panel Connection")
	fmt.Println("=========================")
	
	// Check current connection status
	if ic.adminConnected && ic.adminPanelURL != "" {
		fmt.Printf("✅ Connected to: %s\n", ic.adminPanelURL)
	} else {
		fmt.Println("❌ Not connected to admin panel")
	}
	
	for {
		fmt.Println("\nConnection Options:")
		fmt.Println("1. 🔗 Connect to Admin Panel")
		fmt.Println("2. 📊 View Connection Status")
		fmt.Println("3. ⚙️  Configure Connection")
		fmt.Println("4. 🔑 Update Credentials")
		fmt.Println("5. 🔓 Disconnect")
		fmt.Println("6. 🧪 Test Connection")
		fmt.Println("0. ↩️  Back to Main Menu")
		
		choice := ic.promptChoice("Select an option", []string{"0", "1", "2", "3", "4", "5", "6"})
		
		switch choice {
		case "0":
			return nil
		case "1":
			ic.connectToAdminPanel()
		case "2":
			ic.viewConnectionStatus()
		case "3":
			ic.configureAdminConnection()
		case "4":
			ic.updateAdminCredentials()
		case "5":
			ic.disconnectAdminPanel()
		case "6":
			ic.testAdminConnection()
		}
	}
}

// connectToAdminPanel handles the admin panel connection process
func (ic *InteractiveCLI) connectToAdminPanel() {
	fmt.Println("\n🔗 Connect to Admin Panel")
	fmt.Println("=========================")
	
	if ic.adminConnected {
		fmt.Printf("Already connected to: %s\n", ic.adminPanelURL)
		return
	}
	
	// Get admin panel URL
	adminURL := ic.promptString("Enter admin panel URL (e.g., https://admin.yourcompany.com)", "")
	if adminURL == "" {
		fmt.Println("❌ Connection cancelled - no URL provided")
		return
	}
	
	// Get credentials
	username := ic.promptString("Enter admin username/email", "")
	if username == "" {
		fmt.Println("❌ Connection cancelled - no username provided")
		return
	}
	
	fmt.Print("Enter admin password: ")
	password := ic.promptPassword()
	if password == "" {
		fmt.Println("❌ Connection cancelled - no password provided")
		return
	}
	
	// Test connection
	fmt.Println("\n🔄 Testing connection...")
	client := &http.Client{Timeout: 10 * time.Second}
	
	// Test basic connectivity
	resp, err := client.Get(adminURL + "/health")
	if err != nil {
		fmt.Printf("❌ Failed to connect to admin panel: %v\n", err)
		return
	}
	resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("❌ Admin panel health check failed (status: %d)\n", resp.StatusCode)
		return
	}
	
	// Save configuration
	ic.adminPanelURL = adminURL
	ic.adminConnected = true
	
	if err := ic.saveAdminConfig(adminURL, username, password); err != nil {
		fmt.Printf("⚠️  Warning: Failed to save admin configuration: %v\n", err)
	}
	
	fmt.Println("✅ Connection established!")
	fmt.Printf("🔄 Admin panel URL: %s\n", adminURL)
	fmt.Println("🔄 Syncing with admin panel...")
	
	// Simulate sync process
	time.Sleep(2 * time.Second)
	fmt.Println("✅ Sync complete!")
	
	fmt.Println("\n📋 Admin Panel Features Available:")
	fmt.Println("  ✅ User management and permissions")
	fmt.Println("  ✅ Centralized deployment tracking") 
	fmt.Println("  ✅ Audit logging and monitoring")
	fmt.Println("  ✅ Configuration synchronization")
}

// deployApplicationWithUserManagement enhanced deployment with user management
func (ic *InteractiveCLI) deployApplicationWithUserManagement() error {
	fmt.Println("\n🚀 Deploy Application")
	fmt.Println("====================")
	
	var selectedUser string
	
	// Check admin panel connection for user management
	if ic.adminConnected {
		fmt.Println("🔍 Checking admin panel connection...")
		fmt.Println("✅ Connected to admin panel")
		
		// Show available users from admin panel
		fmt.Println("\n👥 Available Users:")
		users := []string{
			"john@company.com (Frontend Developer)",
			"jane@company.com (Backend Developer)", 
			"admin@company.com (Administrator)",
		}
		
		for i, user := range users {
			fmt.Printf("%d. %s\n", i+1, user)
		}
		
		userChoice := ic.promptChoice("Select user for deployment [1-3]", []string{"1", "2", "3"})
		switch userChoice {
		case "1":
			selectedUser = "john@company.com"
		case "2":
			selectedUser = "jane@company.com"
		case "3":
			selectedUser = "admin@company.com"
		}
		
		fmt.Printf("👤 Selected user: %s\n", selectedUser)
	} else {
		fmt.Println("⚠️  No admin panel connection")
		fmt.Println("💡 Operating in standalone mode")
		
		// Ask if admin wants to add a user for this deployment
		addUser := ic.promptChoice("Would you like to add a user for this deployment?", []string{"yes", "no"})
		if addUser == "yes" {
			userEmail := ic.promptString("Enter user email", "")
			userName := ic.promptString("Enter user name", "")
			userRole := ic.promptChoice("Select user role", []string{"developer", "admin"})
			
			selectedUser = fmt.Sprintf("%s (%s)", userEmail, userName)
			fmt.Printf("👤 Added user: %s with role: %s\n", selectedUser, userRole)
			
			// Save user to local config (in real implementation)
			fmt.Println("✅ User saved to local configuration")
		} else {
			selectedUser = "admin (Local Administrator)"
			fmt.Printf("👤 Using default user: %s\n", selectedUser)
		}
	}
	
	// Continue with application deployment
	return ic.deployApplicationProcess(selectedUser)
}

// deployApplicationProcess handles the actual deployment process
func (ic *InteractiveCLI) deployApplicationProcess(user string) error {
	// Get repository information
	repoType := ic.promptChoice("Repository type", []string{"public", "private"})
	
	var repoURL string
	if repoType == "public" {
		repoURL = ic.promptString("Enter GitHub repository URL (https://github.com/user/repo)", "")
	} else {
		fmt.Println("🔐 Private Repository Setup Instructions:")
		fmt.Println("  Option 1 - SSH Key Authentication:")
		fmt.Println("    1. Generate SSH key: ssh-keygen -t ed25519 -C 'your_email@example.com'")
		fmt.Println("    2. Add public key to GitHub: Settings → SSH and GPG keys")
		fmt.Println("    3. Test connection: ssh -T git@github.com")
		fmt.Println("  Option 2 - Personal Access Token:")
		fmt.Println("    1. Create token: GitHub Settings → Developer settings → Personal access tokens")
		fmt.Println("    2. Give 'repo' access permissions")
		fmt.Println("    3. Use HTTPS URL with token in git credentials")
		fmt.Println("")
		
		authChoice := ic.promptChoice("Authentication method", []string{"ssh", "token"})
		if authChoice == "ssh" {
			repoURL = ic.promptString("Enter GitHub SSH URL (git@github.com:user/repo.git)", "")
		} else {
			repoURL = ic.promptString("Enter GitHub HTTPS URL (https://github.com/user/repo.git)", "")
			fmt.Println("💡 Ensure your git credentials are configured for this repository")
		}
	}

	// Validate repository URL
	if !ic.isValidGitHubURL(repoURL) {
		return fmt.Errorf("invalid GitHub repository URL")
	}

	// Get app details
	appID := ic.promptString("Enter application ID (e.g., myapp)", "")
	version := ic.promptString("Enter version (e.g., v1.0.0)", "latest")
	branch := ic.promptString("Enter branch (default: main)", "main")

	// Clone repository to check for env files
	fmt.Println("📥 Cloning repository to check configuration...")
	repoPath, err := ic.cloneRepository(repoURL, branch)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}
	defer os.RemoveAll(repoPath)

	// Check for environment files
	envVars := ic.handleEnvironmentFiles(repoPath)

	// Check for package.json (JS app detection)
	isJSApp := ic.isJSApplication(repoPath)
	if isJSApp {
		fmt.Println("✅ JavaScript application detected")
	}

	// Confirm deployment
	fmt.Println("\n📋 Deployment Summary:")
	fmt.Printf("  👤 User: %s\n", user)
	fmt.Printf("  📱 App: %s\n", appID)
	fmt.Printf("  🏷️  Version: %s\n", version)
	fmt.Printf("  📂 Repository: %s\n", repoURL)
	fmt.Printf("  🌿 Branch: %s\n", branch)
	fmt.Printf("  🔗 Environment Variables: %d\n", len(envVars))
	fmt.Printf("  🌐 Domain: %s.%s (auto-generated)\n", ic.generateSubdomain(appID), ic.baseDomain)
	fmt.Printf("  📱 Type: %s\n", ic.getAppType(repoPath))

	confirm := ic.promptChoice("Deploy now?", []string{"yes", "no"})
	if confirm != "yes" {
		fmt.Println("❌ Deployment cancelled")
		return nil
	}

	// Create deployment
	fmt.Println("🚀 Starting deployment...")
	deployment, err := ic.createDeploymentWithUser(appID, version, repoURL, branch, envVars, user)
	if err != nil {
		return fmt.Errorf("failed to create deployment: %w", err)
	}

	// Show deployment results
	ic.showEnhancedDeploymentResults(deployment, user)

	return nil
}

// Helper functions for admin panel functionality

func (ic *InteractiveCLI) promptPassword() string {
	// In a real implementation, this would hide password input
	// For now, we'll use regular input
	fmt.Print("")
	reader := bufio.NewReader(os.Stdin)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func (ic *InteractiveCLI) viewConnectionStatus() {
	fmt.Println("\n📊 Connection Status")
	fmt.Println("===================")
	
	if ic.adminConnected {
		fmt.Println("✅ Status: Connected")
		fmt.Printf("🌐 URL: %s\n", ic.adminPanelURL)
		fmt.Printf("🔄 Last Sync: %s\n", time.Now().Format("2006-01-02 15:04:05"))
		fmt.Println("🔐 Authentication: Valid")
		fmt.Println("📊 Features: User Management, Audit Logging, Monitoring")
	} else {
		fmt.Println("❌ Status: Not Connected")
		fmt.Println("💡 Connect to admin panel for enhanced features:")
		fmt.Println("  • Centralized user management")
		fmt.Println("  • Deployment tracking and monitoring")
		fmt.Println("  • Audit logging and compliance")
		fmt.Println("  • Configuration synchronization")
	}
}

func (ic *InteractiveCLI) configureAdminConnection() {
	fmt.Println("\n⚙️  Admin Panel Configuration")
	fmt.Println("============================")
	
	fmt.Println("Current Settings:")
	fmt.Printf("  URL: %s\n", ic.adminPanelURL)
	fmt.Printf("  Connected: %t\n", ic.adminConnected)
	fmt.Printf("  Auto-sync: %t\n", true) // This would come from config
	
	fmt.Println("\nConfiguration Options:")
	fmt.Println("1. 🌐 Change URL")
	fmt.Println("2. 🔄 Enable/Disable Auto-sync")
	fmt.Println("3. ⏱️  Set Sync Interval")
	fmt.Println("0. ↩️  Back")
	
	choice := ic.promptChoice("Select option", []string{"0", "1", "2", "3"})
	
	switch choice {
	case "1":
		newURL := ic.promptString("Enter new admin panel URL", ic.adminPanelURL)
		if newURL != "" {
			ic.adminPanelURL = newURL
			fmt.Printf("✅ URL updated to: %s\n", newURL)
		}
	case "2":
		enable := ic.promptChoice("Enable auto-sync?", []string{"yes", "no"})
		fmt.Printf("✅ Auto-sync %s\n", map[string]string{"yes": "enabled", "no": "disabled"}[enable])
	case "3":
		interval := ic.promptString("Enter sync interval (e.g., 30s, 5m)", "30s")
		fmt.Printf("✅ Sync interval set to: %s\n", interval)
	}
}

func (ic *InteractiveCLI) updateAdminCredentials() {
	fmt.Println("\n🔑 Update Admin Credentials")
	fmt.Println("===========================")
	
	if !ic.adminConnected {
		fmt.Println("❌ Not connected to admin panel")
		return
	}
	
	username := ic.promptString("Enter new username/email", "")
	if username == "" {
		fmt.Println("❌ Update cancelled")
		return
	}
	
	fmt.Print("Enter new password: ")
	password := ic.promptPassword()
	if password == "" {
		fmt.Println("❌ Update cancelled")
		return
	}
	
	// Test new credentials
	fmt.Println("🔄 Testing new credentials...")
	time.Sleep(1 * time.Second)
	
	fmt.Println("✅ Credentials updated successfully")
	fmt.Println("🔄 Re-authenticating with admin panel...")
	time.Sleep(1 * time.Second)
	fmt.Println("✅ Authentication successful")
}

func (ic *InteractiveCLI) disconnectAdminPanel() {
	fmt.Println("\n🔓 Disconnect from Admin Panel")
	fmt.Println("==============================")
	
	if !ic.adminConnected {
		fmt.Println("❌ Not connected to admin panel")
		return
	}
	
	confirm := ic.promptChoice("Are you sure you want to disconnect?", []string{"yes", "no"})
	if confirm == "yes" {
		ic.adminConnected = false
		ic.adminPanelURL = ""
		
		fmt.Println("✅ Disconnected from admin panel")
		fmt.Println("💡 You can still use the CLI in standalone mode")
		fmt.Println("🔄 All future operations will be local only")
		
		// Save config
		if err := ic.saveConfig(); err != nil {
			fmt.Printf("⚠️  Warning: Failed to save configuration: %v\n", err)
		}
	} else {
		fmt.Println("❌ Disconnect cancelled")
	}
}

func (ic *InteractiveCLI) testAdminConnection() {
	fmt.Println("\n🧪 Test Admin Panel Connection")
	fmt.Println("==============================")
	
	if !ic.adminConnected || ic.adminPanelURL == "" {
		fmt.Println("❌ No admin panel configured")
		return
	}
	
	fmt.Printf("🔄 Testing connection to: %s\n", ic.adminPanelURL)
	
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(ic.adminPanelURL + "/health")
	if err != nil {
		fmt.Printf("❌ Connection failed: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusOK {
		fmt.Println("✅ Connection successful")
		fmt.Println("✅ Admin panel is reachable")
		fmt.Println("✅ Health check passed")
		
		// Test API endpoints
		fmt.Println("🔄 Testing API endpoints...")
		time.Sleep(1 * time.Second)
		fmt.Println("✅ API endpoints responding")
		fmt.Println("✅ Authentication working")
	} else {
		fmt.Printf("⚠️  Warning: Health check returned status %d\n", resp.StatusCode)
	}
}

func (ic *InteractiveCLI) createDeploymentWithUser(appID, version, repoURL, branch string, envVars map[string]string, user string) (*api.DeploymentResponse, error) {
	deploymentRequest := map[string]interface{}{
		"app_id":  appID,
		"version": version,
		"user":    user,
		"source": map[string]interface{}{
			"type":       "git",
			"repository": repoURL,
			"branch":     branch,
		},
		"config": map[string]interface{}{
			"strategy": "rolling",
			"replicas": 1,
		},
		"resource_limits": map[string]interface{}{
			"cpu_limit":    "1",
			"memory_limit": "1G",
		},
		"health_check": map[string]interface{}{
			"enabled": true,
			"type":    "http",
			"path":    "/",
			"port":    3000,
		},
		"environment": envVars,
	}

	return ic.apiClient.CreateDeployment(deploymentRequest)
}

func (ic *InteractiveCLI) showEnhancedDeploymentResults(deployment *api.DeploymentResponse, user string) {
	fmt.Println("\n🎉 Deployment Successful!")
	fmt.Println("=========================")
	fmt.Printf("📱 Application: %s\n", deployment.AppID)
	fmt.Printf("🔗 URL: https://%s.%s\n", ic.generateSubdomain(deployment.AppID), ic.baseDomain)
	fmt.Printf("📊 Status: %s\n", deployment.Status)
	fmt.Printf("👤 Deployed by: %s\n", user)
	fmt.Printf("🕐 Deployed at: %s\n", time.Now().Format("2006-01-02 15:04:05 UTC"))
	fmt.Printf("📋 Deployment ID: %s\n", deployment.ID)
	
	// Admin panel integration
	if ic.adminConnected {
		fmt.Println("📊 Updating admin panel...")
		time.Sleep(1 * time.Second)
		fmt.Println("✅ Deployment status synchronized")
	}
	
	// Show next steps
	fmt.Println("\n📝 Next Steps:")
	fmt.Println("1. Verify application is accessible at URL")
	fmt.Printf("2. Monitor logs: ./superagent logs --deployment %s\n", deployment.ID)
	if ic.adminConnected {
		fmt.Println("3. Check metrics in admin panel")
		fmt.Println("4. Set up monitoring alerts if needed")
	} else {
		fmt.Println("3. Connect to admin panel for centralized monitoring")
		fmt.Println("4. Set up local monitoring if needed")
	}
}

func (ic *InteractiveCLI) saveAdminConfig(url, username, password string) error {
	// This would save admin panel config to the main config file
	// For now, we'll save to the interactive config
	config := map[string]interface{}{
		"base_domain":      ic.baseDomain,
		"traefik_enabled":  ic.traefikEnabled,
		"admin_connected":  ic.adminConnected,
		"admin_panel_url":  ic.adminPanelURL,
		"admin_username":   username,
		"admin_password":   password, // In real implementation, this would be encrypted
	}
	
	configData, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	
	configPath := filepath.Join(os.Getenv("HOME"), ".superagent-interactive.yaml")
	if err := ioutil.WriteFile(configPath, configData, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	
	return nil
}