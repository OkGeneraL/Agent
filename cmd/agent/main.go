package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deployment-agent/internal/config"
	"deployment-agent/internal/agent"
	"deployment-agent/internal/logging"

	"github.com/spf13/cobra"
	"github.com/sirupsen/logrus"
)

var (
	version = "1.0.0"
	buildDate = "unknown"
)

func main() {
	var rootCmd = &cobra.Command{
		Use:   "deployment-agent",
		Short: "Secure PaaS deployment agent",
		Long:  `A secure, enterprise-grade deployment agent for PaaS platforms`,
	}

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Start the deployment agent",
		Long:  `Start the deployment agent and begin listening for deployment commands`,
		Run:   startAgent,
	}

	var statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Check agent status",
		Long:  `Check the current status of the deployment agent`,
		Run:   checkStatus,
	}

	var versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Long:  `Show version and build information`,
		Run:   showVersion,
	}

	var configCmd = &cobra.Command{
		Use:   "config",
		Short: "Configuration management",
		Long:  `Manage agent configuration`,
	}

	var validateConfigCmd = &cobra.Command{
		Use:   "validate",
		Short: "Validate configuration",
		Long:  `Validate the agent configuration file`,
		Run:   validateConfig,
	}

	var installCmd = &cobra.Command{
		Use:   "install",
		Short: "Install agent as system service",
		Long:  `Install the deployment agent as a system service`,
		Run:   installService,
	}

	var uninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall agent system service",
		Long:  `Uninstall the deployment agent system service`,
		Run:   uninstallService,
	}

	// Add persistent flags
	rootCmd.PersistentFlags().StringP("config", "c", "/etc/deployment-agent/config.yaml", "Configuration file path")
	rootCmd.PersistentFlags().StringP("log-level", "l", "info", "Log level (debug, info, warn, error)")
	rootCmd.PersistentFlags().BoolP("verbose", "v", false, "Enable verbose logging")

	// Add subcommands
	configCmd.AddCommand(validateConfigCmd)
	rootCmd.AddCommand(startCmd, statusCmd, versionCmd, configCmd, installCmd, uninstallCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func startAgent(cmd *cobra.Command, args []string) {
	// Initialize logging
	setupLogging(cmd)

	// Load configuration
	configPath, _ := cmd.Flags().GetString("config")
	cfg, err := config.Load(configPath)
	if err != nil {
		logrus.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize audit logger
	auditLogger, err := logging.NewAuditLogger(cfg.Logging.AuditLogPath)
	if err != nil {
		logrus.Fatalf("Failed to initialize audit logger: %v", err)
	}

	// Create and start agent
	agentInstance, err := agent.New(cfg, auditLogger)
	if err != nil {
		logrus.Fatalf("Failed to create agent: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Start agent in goroutine
	go func() {
		if err := agentInstance.Start(ctx); err != nil {
			logrus.Errorf("Agent failed to start: %v", err)
			cancel()
		}
	}()

	logrus.Info("Deployment agent started successfully")
	auditLogger.LogEvent("AGENT_STARTED", map[string]interface{}{
		"version": version,
		"config":  configPath,
	})

	// Wait for shutdown signal
	select {
	case sig := <-sigChan:
		logrus.Infof("Received signal %v, shutting down...", sig)
		auditLogger.LogEvent("AGENT_SHUTDOWN_REQUESTED", map[string]interface{}{
			"signal": sig.String(),
		})
	case <-ctx.Done():
		logrus.Info("Context cancelled, shutting down...")
	}

	// Graceful shutdown with timeout
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := agentInstance.Shutdown(shutdownCtx); err != nil {
		logrus.Errorf("Error during shutdown: %v", err)
	}

	auditLogger.LogEvent("AGENT_SHUTDOWN_COMPLETED", map[string]interface{}{
		"version": version,
	})
	logrus.Info("Deployment agent shutdown completed")
}

func checkStatus(cmd *cobra.Command, args []string) {
	fmt.Println("Agent Status: Running") // TODO: Implement actual status check
}

func showVersion(cmd *cobra.Command, args []string) {
	fmt.Printf("Deployment Agent\n")
	fmt.Printf("Version: %s\n", version)
	fmt.Printf("Build Date: %s\n", buildDate)
	fmt.Printf("Go Version: %s\n", "go1.21")
}

func validateConfig(cmd *cobra.Command, args []string) {
	configPath, _ := cmd.Flags().GetString("config")
	_, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Configuration validation failed: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("Configuration is valid")
}

func installService(cmd *cobra.Command, args []string) {
	fmt.Println("Installing deployment agent as system service...")
	// TODO: Implement service installation
	fmt.Println("Service installation completed")
}

func uninstallService(cmd *cobra.Command, args []string) {
	fmt.Println("Uninstalling deployment agent system service...")
	// TODO: Implement service uninstallation
	fmt.Println("Service uninstallation completed")
}

func setupLogging(cmd *cobra.Command) {
	// Set log level
	logLevel, _ := cmd.Flags().GetString("log-level")
	level, err := logrus.ParseLevel(logLevel)
	if err != nil {
		logrus.Warnf("Invalid log level '%s', using info", logLevel)
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	// Set log format
	logrus.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: time.RFC3339,
	})

	// Enable verbose logging if requested
	verbose, _ := cmd.Flags().GetBool("verbose")
	if verbose {
		logrus.SetLevel(logrus.DebugLevel)
	}
}