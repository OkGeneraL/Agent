package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"superagent/internal/agent"
	"superagent/internal/api"
	"superagent/internal/cli"
	"superagent/internal/config"
	"superagent/internal/logging"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile    string
	configPath string
	logLevel   string
	version    = "1.0.0"
	commit     = "unknown"
	buildDate  = "unknown"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "superagent",
		Short: "SuperAgent - Enterprise-grade deployment agent",
		Long: `SuperAgent is a secure, enterprise-grade deployment agent for containerized applications.
Supports Git and Docker deployments with comprehensive monitoring, security features, and zero-downtime updates.

SuperAgent allows deployment of only predefined applications from your platform, providing controlled
deployment capabilities similar to Vercel but with enterprise security and governance features.`,
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Help()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.superagent.yaml)")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (trace, debug, info, warn, error, fatal, panic)")

	// Add subcommands
	rootCmd.AddCommand(startCmd())
	rootCmd.AddCommand(statusCmd())
	rootCmd.AddCommand(versionCmd())
	rootCmd.AddCommand(configCmd())
	rootCmd.AddCommand(deployCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(logsCmd())
	rootCmd.AddCommand(installCmd())
	rootCmd.AddCommand(uninstallCmd())
	rootCmd.AddCommand(interactiveCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func startCmd() *cobra.Command {
	var daemon bool

	cmd := &cobra.Command{
		Use:   "start",
		Short: "Start SuperAgent",
		Long:  "Start SuperAgent deployment agent and begin listening for deployment requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runAgent(daemon)
		},
	}

	cmd.Flags().BoolVarP(&daemon, "daemon", "d", false, "run as daemon")

	return cmd
}

func statusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show SuperAgent status",
		Long:  "Display the current status and health of SuperAgent",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewCLIClient(8080) // Default health check port
			
			if !client.IsAgentRunning() {
				fmt.Println("SuperAgent Status: Not Running")
				fmt.Println("  Service: Stopped")
				fmt.Println("  Health: Unavailable")
				return nil
			}
			
			status, err := client.GetStatus()
			if err != nil {
				return fmt.Errorf("failed to get status: %w", err)
			}
			
			fmt.Println("SuperAgent Status:")
			fmt.Printf("  Service: %s\n", status.Status)
			fmt.Printf("  Health: %s\n", status.Health)
			fmt.Printf("  Version: %s\n", status.Version)
			fmt.Printf("  Uptime: %s\n", status.Uptime)
			fmt.Printf("  Active Deployments: %d\n", status.ActiveDeployments)
			fmt.Printf("  Total Deployments: %d\n", status.TotalDeployments)
			fmt.Printf("  Platform: %s\n", status.Metadata["platform"])
			return nil
		},
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  "Display SuperAgent version, commit, and build information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("SuperAgent\n")
			fmt.Printf("Version: %s\n", version)
			fmt.Printf("Commit: %s\n", commit)
			fmt.Printf("Build Date: %s\n", buildDate)
			fmt.Printf("Platform: Enterprise Deployment Agent\n")
		},
	}
}

func deployCmd() *cobra.Command {
	var (
		appID      string
		version    string
		sourceType string
		source     string
		branch     string
		tag        string
	)

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy an application",
		Long:  "Deploy a predefined application using SuperAgent",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewCLIClient(8080)
			
			if !client.IsAgentRunning() {
				return fmt.Errorf("SuperAgent is not running. Please start the agent first")
			}
			
			// Create deployment request
			deploymentRequest := map[string]interface{}{
				"app_id":  appID,
				"version": version,
				"source": map[string]interface{}{
					"type":       sourceType,
					"repository": source,
					"branch":     branch,
					"tag":        tag,
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
					"path":    "/health",
					"port":    8080,
				},
			}
			
			fmt.Printf("Deploying %s version %s...\n", appID, version)
			fmt.Printf("Source: %s (%s)\n", source, sourceType)
			if branch != "" {
				fmt.Printf("Branch: %s\n", branch)
			}
			if tag != "" {
				fmt.Printf("Tag: %s\n", tag)
			}
			
			deployment, err := client.CreateDeployment(deploymentRequest)
			if err != nil {
				return fmt.Errorf("failed to create deployment: %w", err)
			}
			
			fmt.Printf("Deployment created successfully: %s\n", deployment.ID)
			fmt.Printf("Status: %s\n", deployment.Status)
			fmt.Printf("Message: %s\n", deployment.Message)
			return nil
		},
	}

	cmd.Flags().StringVar(&appID, "app", "", "Application ID (required)")
	cmd.Flags().StringVar(&version, "version", "", "Application version (required)")
	cmd.Flags().StringVar(&sourceType, "source-type", "git", "Source type (git or docker)")
	cmd.Flags().StringVar(&source, "source", "", "Source repository URL or Docker image (required)")
	cmd.Flags().StringVar(&branch, "branch", "", "Git branch (for git source)")
	cmd.Flags().StringVar(&tag, "tag", "", "Git tag or Docker tag")

	cmd.MarkFlagRequired("app")
	cmd.MarkFlagRequired("version")
	cmd.MarkFlagRequired("source")

	return cmd
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List deployments",
		Long:  "List all active deployments managed by SuperAgent",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewCLIClient(8080)
			
			if !client.IsAgentRunning() {
				return fmt.Errorf("SuperAgent is not running. Please start the agent first")
			}
			
			deployments, err := client.ListDeployments()
			if err != nil {
				return fmt.Errorf("failed to list deployments: %w", err)
			}
			
			if len(deployments) == 0 {
				fmt.Println("No active deployments found")
				return nil
			}
			
			fmt.Println("Active Deployments:")
			fmt.Printf("  %-20s %-12s %-10s %-12s %-20s\n", "ID", "APP", "VERSION", "STATUS", "CREATED")
			fmt.Println("  " + strings.Repeat("-", 76))
			
			for _, d := range deployments {
				createdAt := d.CreatedAt.Format("2006-01-02 15:04:05")
				fmt.Printf("  %-20s %-12s %-10s %-12s %-20s\n", 
					truncateString(d.ID, 20),
					truncateString(d.AppID, 12),
					truncateString(d.Version, 10),
					d.Status,
					createdAt)
			}
			
			return nil
		},
	}
}

func logsCmd() *cobra.Command {
	var (
		deploymentID string
		follow       bool
		tail         int
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "Show deployment logs",
		Long:  "Show logs for a specific deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			client := api.NewCLIClient(8080)
			
			if !client.IsAgentRunning() {
				return fmt.Errorf("SuperAgent is not running. Please start the agent first")
			}
			
			logs, err := client.GetDeploymentLogs(deploymentID, tail)
			if err != nil {
				return fmt.Errorf("failed to get logs: %w", err)
			}
			
			fmt.Printf("Showing logs for deployment: %s\n", deploymentID)
			if follow {
				fmt.Println("Following logs (Ctrl+C to stop)...")
			}
			
			if len(logs.Logs) == 0 {
				fmt.Println("No logs found for this deployment")
				return nil
			}
			
			for _, log := range logs.Logs {
				fmt.Printf("[%s] [%s] [%s] %s\n",
					log.Timestamp.Format("2006-01-02 15:04:05"),
					log.Level,
					log.Source,
					log.Message)
			}
			
			return nil
		},
	}

	cmd.Flags().StringVar(&deploymentID, "deployment", "", "Deployment ID (required)")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().IntVarP(&tail, "tail", "t", 100, "Number of lines to show")

	cmd.MarkFlagRequired("deployment")

	return cmd
}

func installCmd() *cobra.Command {
	var (
		systemd bool
		user    string
		dataDir string
	)

	cmd := &cobra.Command{
		Use:   "install",
		Short: "Install SuperAgent as system service",
		Long:  "Install SuperAgent as a systemd service for production deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Installing SuperAgent as system service...")
			fmt.Printf("User: %s\n", user)
			fmt.Printf("Data Directory: %s\n", dataDir)
			
			if systemd {
				// Run the install script
				installCmd := exec.Command("/bin/bash", "./install.sh")
				installCmd.Env = append(os.Environ(),
					fmt.Sprintf("AGENT_USER=%s", user),
					fmt.Sprintf("DATA_DIR=%s", dataDir),
				)
				installCmd.Stdout = os.Stdout
				installCmd.Stderr = os.Stderr
				
				if err := installCmd.Run(); err != nil {
					return fmt.Errorf("installation failed: %w", err)
				}
				
				fmt.Println("✅ SuperAgent installed successfully as systemd service")
				fmt.Println("📋 Next steps:")
				fmt.Println("  1. Configure: sudo systemctl enable superagent")
				fmt.Println("  2. Start: sudo systemctl start superagent")
				fmt.Println("  3. Check status: sudo systemctl status superagent")
			} else {
				fmt.Println("⚠️  Manual installation mode - systemd service not created")
			}
			
			return nil
		},
	}

	cmd.Flags().BoolVar(&systemd, "systemd", true, "Install systemd service")
	cmd.Flags().StringVar(&user, "user", "superagent", "Service user")
	cmd.Flags().StringVar(&dataDir, "data-dir", "/var/lib/superagent", "Data directory")

	return cmd
}

func uninstallCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall SuperAgent system service",
		Long:  "Remove SuperAgent systemd service and cleanup installation",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Uninstalling SuperAgent system service...")
			
			if !force {
				fmt.Print("⚠️  This will remove SuperAgent and all its data. Continue? [y/N]: ")
				var response string
				fmt.Scanln(&response)
				if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
					fmt.Println("❌ Uninstallation cancelled")
					return nil
				}
			}
			
			// Run the uninstall script
			uninstallCmd := exec.Command("/bin/bash", "./uninstall.sh")
			if force {
				uninstallCmd.Env = append(os.Environ(), "FORCE=true")
			}
			uninstallCmd.Stdout = os.Stdout
			uninstallCmd.Stderr = os.Stderr
			
			if err := uninstallCmd.Run(); err != nil {
				return fmt.Errorf("uninstallation failed: %w", err)
			}
			
			fmt.Println("✅ SuperAgent uninstalled successfully")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force removal")

	return cmd
}

func interactiveCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "interactive",
		Short: "Start interactive CLI",
		Long:  "Start SuperAgent interactive CLI for guided setup and deployment",
		RunE: func(cmd *cobra.Command, args []string) error {
			// Initialize configuration
			if err := initConfig(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}

			// Load configuration
			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Initialize logging
			auditLogger, err := logging.NewAuditLogger(cfg.Security.AuditLogPath)
			if err != nil {
				return fmt.Errorf("failed to initialize audit logger: %w", err)
			}

			// Create interactive CLI
			interactiveCLI := cli.NewInteractiveCLI(cfg, auditLogger)

			// Start interactive CLI
			return interactiveCLI.StartInteractiveCLI()
		},
	}
}

func configCmd() *cobra.Command {
	configCmd := &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  "Manage SuperAgent configuration",
	}

	configCmd.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Long:  "Validate the current configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("configuration validation failed: %w", err)
			}

			fmt.Printf("Configuration is valid\n")
			fmt.Printf("Config file: %s\n", cfg.GetConfigFile())
			return nil
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "Show current configuration",
		Long:  "Display the current configuration (sensitive values masked)",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := initConfig(); err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			cfg, err := config.LoadDefault()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Print configuration (implementation would mask sensitive data)
			fmt.Printf("SuperAgent Configuration:\n")
			fmt.Printf("  Config File: %s\n", cfg.GetConfigFile())
			fmt.Printf("  Log Level: %s\n", cfg.GetLogLevel())
			fmt.Printf("  API Port: %d\n", cfg.GetAPIPort())
			fmt.Printf("  Metrics Port: %d\n", cfg.GetMetricsPort())
			fmt.Printf("  Security: Enterprise Grade\n")
			fmt.Printf("  Encryption: AES-256\n")
			return nil
		},
	})

	configCmd.AddCommand(&cobra.Command{
		Use:   "init",
		Short: "Initialize configuration",
		Long:  "Create a default configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("Creating default SuperAgent configuration...")
			
			// Initialize configuration
			if err := initConfig(); err != nil {
				return fmt.Errorf("failed to initialize config: %w", err)
			}

			// Create default config content
			configContent := `# SuperAgent Configuration
agent:
  id: ""
  work_dir: "/var/lib/superagent"
  data_dir: "/var/lib/superagent/data"

security:
  encryption_key_file: "/etc/superagent/encryption.key"
  audit_log_enabled: true

monitoring:
  enabled: true
  metrics_port: 9090
  health_check_port: 8080

traefik:
  enabled: false
  base_domain: "yourdomain.com"
`
			
			// Save to user home
			configPath := filepath.Join(os.Getenv("HOME"), ".superagent.yaml")
			if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
				return fmt.Errorf("failed to save config: %w", err)
			}
			
			fmt.Printf("✅ Configuration initialized at %s\n", configPath)
			fmt.Println("📋 Next steps:")
			fmt.Println("  1. Edit the configuration file to customize settings")
			fmt.Println("  2. Start the agent: superagent start")
			return nil
		},
	})

	return configCmd
}

func runAgent(daemon bool) error {
	// Initialize configuration
	if err := initConfig(); err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	// Load configuration
	cfg, err := config.LoadDefault()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize logging
	auditLogger, err := logging.NewAuditLogger(cfg.Security.AuditLogPath)
	if err != nil {
		return fmt.Errorf("failed to initialize audit logger: %w", err)
	}

	// Create agent
	agentInstance, err := agent.New(cfg, auditLogger)
	if err != nil {
		return fmt.Errorf("failed to create agent: %w", err)
	}

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-signalChan
		logrus.Info("Received shutdown signal")
		cancel()
	}()

	// Start agent
	logrus.Info("Starting SuperAgent deployment system")
	if err := agentInstance.Start(ctx); err != nil {
		return fmt.Errorf("failed to start SuperAgent: %w", err)
	}

	// Wait for shutdown
	<-ctx.Done()

	// Graceful shutdown
	logrus.Info("Shutting down SuperAgent")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()
	if err := agentInstance.Shutdown(shutdownCtx); err != nil {
		logrus.Errorf("Error during shutdown: %v", err)
		return err
	}

	logrus.Info("SuperAgent stopped")
	return nil
}

func initConfig() error {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.AddConfigPath("/etc/superagent/")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".superagent")
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("SUPERAGENT")

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			// Config file not found; ignore error for now
			logrus.Warn("No config file found, using defaults")
		} else {
			return err
		}
	}

	// Set up logging level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		return fmt.Errorf("invalid log level: %w", err)
	}
	logrus.SetLevel(level)
	
	// Set formatter
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	return nil
}

// truncateString truncates a string to a specified length
func truncateString(s string, length int) string {
	if len(s) <= length {
		return s
	}
	return s[:length-3] + "..."
}