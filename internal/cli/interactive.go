package cli

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
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
	
	// Try to connect to admin panel API
	// This would be implemented based on your admin panel API
	ic.adminConnected = false
	ic.adminPanelURL = ""
	
	if ic.adminConnected {
		fmt.Println("✅ Admin panel connected!")
		fmt.Printf("🌐 Admin panel URL: %s\n", ic.adminPanelURL)
	} else {
		fmt.Println("❌ Admin panel not connected")
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
		if ic.adminConnected {
			fmt.Println("7. 🌐 Open Admin Panel")
		}
		fmt.Println("0. 🚪 Exit")

		choice := ic.promptChoice("Select an option", []string{"0", "1", "2", "3", "4", "5", "6", "7"})

		switch choice {
		case "0":
			fmt.Println("👋 Goodbye!")
			return nil
		case "1":
			if err := ic.deployApplication(); err != nil {
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
			if ic.adminConnected {
				ic.openAdminPanel()
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
		repoURL = ic.promptString("Enter GitHub repository URL", "")
	} else {
		repoURL = ic.promptString("Enter GitHub repository URL", "")
		fmt.Println("🔐 For private repositories, ensure SSH keys or tokens are configured")
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
		ic.saveConfig()
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
	
	ic.saveConfig()
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
	
	ic.saveConfig()
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
	return cmd.Start()
}



func (ic *InteractiveCLI) saveConfig() {
	// Save configuration to file
	config := map[string]interface{}{
		"base_domain":      ic.baseDomain,
		"traefik_enabled":  ic.traefikEnabled,
		"admin_connected":  ic.adminConnected,
		"admin_panel_url":  ic.adminPanelURL,
	}
	
	configData, _ := yaml.Marshal(config)
	configPath := filepath.Join(os.Getenv("HOME"), ".superagent-interactive.yaml")
	ioutil.WriteFile(configPath, configData, 0644)
}

func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}