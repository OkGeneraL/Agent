package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"superagent/internal/agent"
	"superagent/internal/config"
	"superagent/internal/logging"
	"superagent/internal/monitoring"
	"superagent/internal/paas"
	"superagent/internal/storage"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile     string
	verbose     bool
	configDir   string
	logLevel    string
	interactive bool
)

// PaaSCLI represents the main CLI application
type PaaSCLI struct {
	config      *config.Config
	agent       *agent.Agent
	userManager *paas.UserManager
	appCatalog  *paas.AppCatalog
	domainMgr   *paas.DomainManager
	store       *storage.SecureStore
	auditLogger *logging.AuditLogger
	monitor     *monitoring.Monitor
}

func main() {
	rootCmd := &cobra.Command{
		Use:   "superagent-cli",
		Short: "SuperAgent PaaS Management CLI",
		Long: `SuperAgent CLI is a comprehensive management interface for the SuperAgent PaaS platform.
		
This CLI allows you to:
- Manage customers and their resource quotas
- Add and configure applications in the catalog
- Manage licenses and permissions
- Deploy and monitor applications
- Configure domains and SSL certificates
- Monitor system resources and health
- Test all platform features before deploying admin panels`,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			initConfig()
		},
	}

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.superagent/config.yaml)")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().StringVar(&configDir, "config-dir", "", "configuration directory")
	rootCmd.PersistentFlags().StringVar(&logLevel, "log-level", "info", "log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolVarP(&interactive, "interactive", "i", true, "interactive mode")

	// Add command groups
	rootCmd.AddCommand(
		newCustomerCmd(),
		newAppCmd(),
		// TODO: Implement these commands
		// newLicenseCmd(),
		// newDeployCmd(),
		// newDomainCmd(),
		// newMonitorCmd(),
		// newServerCmd(),
		// newSetupCmd(),
	)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigFile(cfgFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		configPath := filepath.Join(home, ".superagent")
		viper.AddConfigPath(configPath)
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")

		// Create config directory if it doesn't exist
		if err := os.MkdirAll(configPath, 0755); err != nil {
			fmt.Printf("Error creating config directory: %v\n", err)
			os.Exit(1)
		}
	}

	viper.AutomaticEnv()
	viper.SetEnvPrefix("SUPERAGENT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	if err := viper.ReadInConfig(); err == nil {
		fmt.Printf("Using config file: %s\n", viper.ConfigFileUsed())
	}

	// Set log level
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
}

func initializePaaS() (*PaaSCLI, error) {
	// Load configuration
	cfg, err := config.Load("")
	if err != nil {
		return nil, fmt.Errorf("failed to load config: %w", err)
	}

	// Initialize audit logger first (needed for storage)
	auditLogger, err := logging.NewAuditLogger(cfg.Logging.AuditFile)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize audit logger: %w", err)
	}

	// Initialize storage
	store, err := storage.NewSecureStore(cfg.Storage.DataDir, cfg.Storage.EncryptionKey, auditLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize storage: %w", err)
	}

	// Initialize monitor
	monitor := monitoring.NewMonitor(auditLogger, cfg.Monitoring.MetricsPort)

	// Initialize PaaS components
	userManager := paas.NewUserManager(store, auditLogger)
	appCatalog := paas.NewAppCatalog(store, auditLogger)
	domainManager := paas.NewDomainManager(store, auditLogger, cfg.Domain.BaseDomain, cfg.Domain.DNSProvider, cfg.Domain.ACMEEmail)

	// Initialize agent
	agentInstance, err := agent.New(cfg, auditLogger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent: %w", err)
	}

	return &PaaSCLI{
		config:      cfg,
		agent:       agentInstance,
		userManager: userManager,
		appCatalog:  appCatalog,
		domainMgr:   domainManager,
		store:       store,
		auditLogger: auditLogger,
		monitor:     monitor,
	}, nil
}

// Common helper functions
func confirmAction(message string) bool {
	if !interactive {
		return true
	}

	fmt.Printf("%s (y/N): ", message)
	var response string
	fmt.Scanln(&response)
	return strings.ToLower(response) == "y" || strings.ToLower(response) == "yes"
}

func promptString(message string, defaultValue string) string {
	if defaultValue != "" {
		fmt.Printf("%s [%s]: ", message, defaultValue)
	} else {
		fmt.Printf("%s: ", message)
	}

	var input string
	fmt.Scanln(&input)

	if input == "" && defaultValue != "" {
		return defaultValue
	}

	return input
}

func printSuccess(message string) {
	fmt.Printf("✅ %s\n", message)
}

func printError(message string) {
	fmt.Printf("❌ %s\n", message)
}

func printInfo(message string) {
	fmt.Printf("ℹ️  %s\n", message)
}

func printWarning(message string) {
	fmt.Printf("⚠️  %s\n", message)
}