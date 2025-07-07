package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"superagent/internal/paas"

	"github.com/spf13/cobra"
	"github.com/olekukonko/tablewriter"
)

func newAppCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "app",
		Short: "Application catalog management",
		Long:  "Manage applications in the catalog, versions, and configurations",
	}

	cmd.AddCommand(
		newAppAddCmd(),
		newAppListCmd(),
		newAppShowCmd(),
		newAppVersionCmd(),
		newAppUpdateCmd(),
		newAppDeleteCmd(),
	)

	return cmd
}

func newAppAddCmd() *cobra.Command {
	var (
		name         string
		description  string
		category     string
		publisher    string
		appType      string
		sourceType   string
		gitURL       string
		gitBranch    string
		dockerImage  string
		dockerTag    string
		port         int
		buildCmd     string
		startCmd     string
		features     []string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new application to the catalog",
		Long:  "Create a new application entry in the catalog with source and configuration",
		Example: `  superagent-cli app add --name "E-commerce Store" --source-type git --git-url https://github.com/example/ecommerce
  superagent-cli app add --name "Node.js API" --source-type docker --docker-image node:16 --port 3000`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paas, err := initializePaaS()
			if err != nil {
				return err
			}

			// Interactive mode
			if interactive {
				if name == "" {
					name = promptString("Application name", "")
					if name == "" {
						printError("Application name is required")
						return fmt.Errorf("application name is required")
					}
				}

				if description == "" {
					description = promptString("Description", "")
				}

				if category == "" {
					fmt.Println("\nAvailable categories:")
					fmt.Println("  - ecommerce: E-commerce applications")
					fmt.Println("  - crm: Customer relationship management")
					fmt.Println("  - cms: Content management systems")
					fmt.Println("  - api: API and microservices")
					fmt.Println("  - dashboard: Dashboards and analytics")
					fmt.Println("  - other: Other applications")
					category = promptString("Category", "other")
				}

				if publisher == "" {
					publisher = promptString("Publisher", "SuperAgent")
				}

				if appType == "" {
					fmt.Println("\nApplication types:")
					fmt.Println("  - webapp: Web application")
					fmt.Println("  - api: API service")
					fmt.Println("  - microservice: Microservice")
					fmt.Println("  - worker: Background worker")
					appType = promptString("Application type", "webapp")
				}

				if sourceType == "" {
					fmt.Println("\nSource types:")
					fmt.Println("  - git: Git repository")
					fmt.Println("  - docker: Docker image")
					sourceType = promptString("Source type", "git")
				}

				if sourceType == "git" && gitURL == "" {
					gitURL = promptString("Git repository URL", "")
					if gitBranch == "" {
						gitBranch = promptString("Git branch", "main")
					}
				}

				if sourceType == "docker" && dockerImage == "" {
					dockerImage = promptString("Docker image", "")
					if dockerTag == "" {
						dockerTag = promptString("Docker tag", "latest")
					}
				}

				if port == 0 {
					portStr := promptString("Application port", "3000")
					fmt.Sscanf(portStr, "%d", &port)
				}

				if startCmd == "" {
					startCmd = promptString("Start command", "npm start")
				}
			}

			// Validate required fields
			if name == "" {
				return fmt.Errorf("application name is required")
			}

			if sourceType == "" {
				sourceType = "git"
			}

			// Create application source
			var appSource paas.ApplicationSource
			switch sourceType {
			case "git":
				if gitURL == "" {
					return fmt.Errorf("git URL is required for git source type")
				}
				appSource = paas.ApplicationSource{
					Type: paas.SourceTypeGit,
					Repository: &paas.GitRepository{
						URL:    gitURL,
						Branch: gitBranch,
					},
				}
			case "docker":
				if dockerImage == "" {
					return fmt.Errorf("docker image is required for docker source type")
				}
				appSource = paas.ApplicationSource{
					Type: paas.SourceTypeDocker,
					DockerImage: &paas.DockerImageSource{
						Image: dockerImage,
						Tag:   dockerTag,
					},
				}
			default:
				return fmt.Errorf("invalid source type: %s", sourceType)
			}

			// Create application config
			appConfig := paas.ApplicationConfig{
				Port:         port,
				Environment:  make(map[string]paas.EnvVar),
				StartCommand: startCmd,
				BuildCommand: buildCmd,
				HealthCheck: paas.HealthCheckConfig{
					Enabled:        true,
					Path:           "/health",
					Interval:       30,
					Timeout:        10,
					Retries:        3,
					InitialDelay:   30,
					ExpectedStatus: 200,
				},
				Resources: paas.ResourceConfig{
					CPU: struct {
						Request string `json:"request"`
						Limit   string `json:"limit"`
					}{
						Request: "0.1",
						Limit:   "1.0",
					},
					Memory: struct {
						Request string `json:"request"`
						Limit   string `json:"limit"`
					}{
						Request: "128Mi",
						Limit:   "512Mi",
					},
				},
				Network: paas.NetworkConfig{
					PublicAccess: true,
					SSL:          true,
					Ports:        []int{port},
					Protocols:    []string{"http", "https"},
				},
				Security: paas.SecurityConfig{
					RunAsNonRoot:   true,
					ReadOnlyRootFS: false,
				},
			}

			// Create application request
			req := &paas.CreateApplicationRequest{
				Name:         name,
				Description:  description,
				Category:     category,
				Publisher:    publisher,
				Type:         paas.ApplicationType(appType),
				Source:       appSource,
				DefaultConfig: appConfig,
				Features:     features,
				Requirements: paas.SystemRequirements{
					MinCPU:       0.1,
					MinMemory:    128,
					MinStorage:   1,
					Architecture: []string{"amd64"},
					OS:           []string{"linux"},
				},
				Pricing: paas.PricingInfo{
					Model:    "free",
					Currency: "USD",
					Tiers:    []paas.PricingTier{},
				},
				Metadata: make(map[string]interface{}),
			}

			// Add application
			app, err := paas.appCatalog.AddApplication(context.Background(), req)
			if err != nil {
				printError(fmt.Sprintf("Failed to add application: %v", err))
				return err
			}

			printSuccess(fmt.Sprintf("Application added successfully: %s", app.ID))

			// Display application details
			fmt.Printf("\n📱 Application Details:\n")
			fmt.Printf("   ID: %s\n", app.ID)
			fmt.Printf("   Name: %s\n", app.Name)
			fmt.Printf("   Description: %s\n", app.Description)
			fmt.Printf("   Category: %s\n", app.Category)
			fmt.Printf("   Type: %s\n", app.Type)
			fmt.Printf("   Publisher: %s\n", app.Publisher)
			fmt.Printf("   Status: %s\n", app.Status)
			fmt.Printf("   Latest Version: %s\n", app.LatestVersion)
			fmt.Printf("   Created: %s\n", app.CreatedAt.Format(time.RFC3339))

			printInfo("💡 Application is now available for deployment")
			printInfo("💡 Use 'superagent-cli license' commands to create licenses for customers")

			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "Application name (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Application description")
	cmd.Flags().StringVarP(&category, "category", "c", "other", "Application category")
	cmd.Flags().StringVarP(&publisher, "publisher", "p", "SuperAgent", "Application publisher")
	cmd.Flags().StringVarP(&appType, "type", "t", "webapp", "Application type")
	cmd.Flags().StringVar(&sourceType, "source-type", "git", "Source type (git, docker)")
	cmd.Flags().StringVar(&gitURL, "git-url", "", "Git repository URL")
	cmd.Flags().StringVar(&gitBranch, "git-branch", "main", "Git branch")
	cmd.Flags().StringVar(&dockerImage, "docker-image", "", "Docker image")
	cmd.Flags().StringVar(&dockerTag, "docker-tag", "latest", "Docker tag")
	cmd.Flags().IntVar(&port, "port", 3000, "Application port")
	cmd.Flags().StringVar(&buildCmd, "build-cmd", "", "Build command")
	cmd.Flags().StringVar(&startCmd, "start-cmd", "npm start", "Start command")
	cmd.Flags().StringSliceVar(&features, "features", []string{}, "Application features")

	return cmd
}

func newAppListCmd() *cobra.Command {
	var (
		category string
		status   string
		appType  string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all applications in the catalog",
		Long:  "Display a table of all applications with their basic information",
		Example: `  superagent-cli app list
  superagent-cli app list --category ecommerce
  superagent-cli app list --status active --type webapp`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paas, err := initializePaaS()
			if err != nil {
				return err
			}

			apps := paas.appCatalog.ListApplications()

			if len(apps) == 0 {
				printInfo("No applications found in catalog")
				return nil
			}

			// Filter applications
			var filteredApps []*paas.Application
			for _, app := range apps {
				if category != "" && app.Category != category {
					continue
				}
				if status != "" && string(app.Status) != status {
					continue
				}
				if appType != "" && string(app.Type) != appType {
					continue
				}
				filteredApps = append(filteredApps, app)
			}

			// Create table
			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"ID", "Name", "Category", "Type", "Publisher", "Status", "Version", "Downloads", "Created"})

			for _, app := range filteredApps {
				table.Append([]string{
					app.ID,
					app.Name,
					app.Category,
					string(app.Type),
					app.Publisher,
					string(app.Status),
					app.LatestVersion,
					fmt.Sprintf("%d", app.Downloads),
					app.CreatedAt.Format("2006-01-02"),
				})
			}

			fmt.Printf("📱 Applications (%d total, %d shown):\n\n", len(apps), len(filteredApps))
			table.Render()

			return nil
		},
	}

	cmd.Flags().StringVar(&category, "category", "", "Filter by category")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")
	cmd.Flags().StringVar(&appType, "type", "", "Filter by application type")

	return cmd
}

func newAppShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <app-id-or-name>",
		Short: "Show detailed application information",
		Long:  "Display comprehensive information about a specific application",
		Args:  cobra.ExactArgs(1),
		Example: `  superagent-cli app show app_12345
  superagent-cli app show "E-commerce Store"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paas, err := initializePaaS()
			if err != nil {
				return err
			}

			appIDOrName := args[0]

			// Try to get by ID first, then by name
			var app *paas.Application
			app, err = paas.appCatalog.GetApplication(appIDOrName)
			if err != nil {
				app, err = paas.appCatalog.GetApplicationByName(appIDOrName)
				if err != nil {
					printError(fmt.Sprintf("Application not found: %s", appIDOrName))
					return err
				}
			}

			// Display application information
			fmt.Printf("📱 Application Details:\n\n")
			fmt.Printf("🆔 Basic Information:\n")
			fmt.Printf("   ID: %s\n", app.ID)
			fmt.Printf("   Name: %s\n", app.Name)
			fmt.Printf("   Description: %s\n", app.Description)
			fmt.Printf("   Category: %s\n", app.Category)
			fmt.Printf("   Type: %s\n", app.Type)
			fmt.Printf("   Publisher: %s\n", app.Publisher)
			fmt.Printf("   Status: %s\n", app.Status)
			fmt.Printf("   License: %s\n", app.License)
			fmt.Printf("   Homepage: %s\n", app.Homepage)
			fmt.Printf("   Support Email: %s\n", app.SupportEmail)
			fmt.Printf("   Created: %s\n", app.CreatedAt.Format(time.RFC3339))
			fmt.Printf("   Updated: %s\n", app.UpdatedAt.Format(time.RFC3339))

			fmt.Printf("\n📊 Statistics:\n")
			fmt.Printf("   Downloads: %d\n", app.Downloads)
			fmt.Printf("   Rating: %.1f/5.0\n", app.Rating)
			fmt.Printf("   Reviews: %d\n", app.Reviews)

			fmt.Printf("\n📋 Source Configuration:\n")
			fmt.Printf("   Type: %s\n", app.Source.Type)
			if app.Source.Repository != nil {
				fmt.Printf("   Git URL: %s\n", app.Source.Repository.URL)
				fmt.Printf("   Branch: %s\n", app.Source.Repository.Branch)
				fmt.Printf("   Private: %t\n", app.Source.Repository.Private)
			}
			if app.Source.DockerImage != nil {
				fmt.Printf("   Docker Image: %s\n", app.Source.DockerImage.Image)
				fmt.Printf("   Docker Tag: %s\n", app.Source.DockerImage.Tag)
				fmt.Printf("   Private: %t\n", app.Source.DockerImage.Private)
			}

			fmt.Printf("\n⚙️  Default Configuration:\n")
			fmt.Printf("   Port: %d\n", app.DefaultConfig.Port)
			fmt.Printf("   Start Command: %s\n", app.DefaultConfig.StartCommand)
			if app.DefaultConfig.BuildCommand != "" {
				fmt.Printf("   Build Command: %s\n", app.DefaultConfig.BuildCommand)
			}
			fmt.Printf("   Health Check Enabled: %t\n", app.DefaultConfig.HealthCheck.Enabled)
			fmt.Printf("   Health Check Path: %s\n", app.DefaultConfig.HealthCheck.Path)
			fmt.Printf("   Public Access: %t\n", app.DefaultConfig.Network.PublicAccess)
			fmt.Printf("   SSL Enabled: %t\n", app.DefaultConfig.Network.SSL)

			fmt.Printf("\n🔧 Resource Requirements:\n")
			fmt.Printf("   CPU Request: %s\n", app.DefaultConfig.Resources.CPU.Request)
			fmt.Printf("   CPU Limit: %s\n", app.DefaultConfig.Resources.CPU.Limit)
			fmt.Printf("   Memory Request: %s\n", app.DefaultConfig.Resources.Memory.Request)
			fmt.Printf("   Memory Limit: %s\n", app.DefaultConfig.Resources.Memory.Limit)

			fmt.Printf("\n🎯 Features (%d):\n", len(app.Features))
			if len(app.Features) == 0 {
				fmt.Printf("   None listed\n")
			} else {
				for _, feature := range app.Features {
					fmt.Printf("   - %s\n", feature)
				}
			}

			fmt.Printf("\n🏷️  Tags (%d):\n", len(app.Tags))
			if len(app.Tags) == 0 {
				fmt.Printf("   None\n")
			} else {
				fmt.Printf("   %s\n", strings.Join(app.Tags, ", "))
			}

			fmt.Printf("\n📦 Versions (%d):\n", len(app.Versions))
			for _, version := range app.Versions {
				status := ""
				if version.Version == app.LatestVersion {
					status = " (latest)"
				}
				fmt.Printf("   - %s%s - %s (%s)\n", version.Version, status, version.Description, version.Status)
			}

			fmt.Printf("\n💰 Pricing:\n")
			fmt.Printf("   Model: %s\n", app.Pricing.Model)
			fmt.Printf("   Currency: %s\n", app.Pricing.Currency)
			if len(app.Pricing.Tiers) > 0 {
				fmt.Printf("   Tiers: %d\n", len(app.Pricing.Tiers))
			}

			return nil
		},
	}

	return cmd
}

func newAppVersionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Manage application versions",
		Long:  "Add new versions and manage existing versions of applications",
	}

	cmd.AddCommand(
		newAppVersionAddCmd(),
		newAppVersionListCmd(),
	)

	return cmd
}

func newAppVersionAddCmd() *cobra.Command {
	var (
		version     string
		description string
		changelog   string
		gitTag      string
		dockerTag   string
		breaking    bool
		security    bool
	)

	cmd := &cobra.Command{
		Use:   "add <app-id-or-name>",
		Short: "Add a new version to an application",
		Args:  cobra.ExactArgs(1),
		Example: `  superagent-cli app version add app_12345 --version 1.1.0 --description "Bug fixes"
  superagent-cli app version add "My App" --version 2.0.0 --breaking --changelog "Major update"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paas, err := initializePaaS()
			if err != nil {
				return err
			}

			appIDOrName := args[0]

			// Get application
			var app *paas.Application
			app, err = paas.appCatalog.GetApplication(appIDOrName)
			if err != nil {
				app, err = paas.appCatalog.GetApplicationByName(appIDOrName)
				if err != nil {
					printError(fmt.Sprintf("Application not found: %s", appIDOrName))
					return err
				}
			}

			// Interactive mode
			if interactive {
				if version == "" {
					version = promptString("Version", "")
					if version == "" {
						printError("Version is required")
						return fmt.Errorf("version is required")
					}
				}

				if description == "" {
					description = promptString("Description", "")
				}

				if changelog == "" {
					changelog = promptString("Changelog", "")
				}
			}

			if version == "" {
				return fmt.Errorf("version is required")
			}

			// Create version source based on app source type
			versionSource := app.Source
			if app.Source.Type == paas.SourceTypeGit && gitTag != "" {
				versionSource.Repository.Tag = gitTag
			}
			if app.Source.Type == paas.SourceTypeDocker && dockerTag != "" {
				versionSource.DockerImage.Tag = dockerTag
			}

			// Create new version
			newVersion := &paas.AppVersion{
				Version:     version,
				Description: description,
				ReleaseDate: time.Now(),
				Source:      versionSource,
				Config:      app.DefaultConfig,
				Status:      paas.VersionStatusStable,
				Changelog:   changelog,
				Breaking:    breaking,
				Security:    security,
				Downloads:   0,
				Metadata:    make(map[string]interface{}),
			}

			// Add version
			err = paas.appCatalog.AddVersion(app.ID, newVersion)
			if err != nil {
				printError(fmt.Sprintf("Failed to add version: %v", err))
				return err
			}

			printSuccess(fmt.Sprintf("Version %s added to application %s", version, app.Name))

			return nil
		},
	}

	cmd.Flags().StringVarP(&version, "version", "v", "", "Version number (required)")
	cmd.Flags().StringVarP(&description, "description", "d", "", "Version description")
	cmd.Flags().StringVar(&changelog, "changelog", "", "Changelog for this version")
	cmd.Flags().StringVar(&gitTag, "git-tag", "", "Git tag for this version")
	cmd.Flags().StringVar(&dockerTag, "docker-tag", "", "Docker tag for this version")
	cmd.Flags().BoolVar(&breaking, "breaking", false, "Mark as breaking change")
	cmd.Flags().BoolVar(&security, "security", false, "Mark as security update")

	return cmd
}

func newAppVersionListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <app-id-or-name>",
		Short: "List all versions of an application",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			paas, err := initializePaaS()
			if err != nil {
				return err
			}

			appIDOrName := args[0]

			// Get application
			var app *paas.Application
			app, err = paas.appCatalog.GetApplication(appIDOrName)
			if err != nil {
				app, err = paas.appCatalog.GetApplicationByName(appIDOrName)
				if err != nil {
					printError(fmt.Sprintf("Application not found: %s", appIDOrName))
					return err
				}
			}

			fmt.Printf("📦 Versions for %s:\n\n", app.Name)

			if len(app.Versions) == 0 {
				printInfo("No versions available")
				return nil
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.SetHeader([]string{"Version", "Status", "Release Date", "Downloads", "Breaking", "Security", "Description"})

			for _, version := range app.Versions {
				latest := ""
				if version.Version == app.LatestVersion {
					latest = " (latest)"
				}

				breakingIcon := ""
				if version.Breaking {
					breakingIcon = "⚠️"
				}

				securityIcon := ""
				if version.Security {
					securityIcon = "🔒"
				}

				table.Append([]string{
					version.Version + latest,
					string(version.Status),
					version.ReleaseDate.Format("2006-01-02"),
					fmt.Sprintf("%d", version.Downloads),
					breakingIcon,
					securityIcon,
					version.Description,
				})
			}

			table.Render()

			return nil
		},
	}

	return cmd
}

func newAppUpdateCmd() *cobra.Command {
	var (
		name        string
		description string
		status      string
		category    string
	)

	cmd := &cobra.Command{
		Use:   "update <app-id-or-name>",
		Short: "Update application information",
		Args:  cobra.ExactArgs(1),
		Example: `  superagent-cli app update app_12345 --status deprecated
  superagent-cli app update "My App" --description "Updated description"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			printInfo("Application update functionality would be implemented here")
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Update application name")
	cmd.Flags().StringVar(&description, "description", "", "Update description")
	cmd.Flags().StringVar(&status, "status", "", "Update status")
	cmd.Flags().StringVar(&category, "category", "", "Update category")

	return cmd
}

func newAppDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <app-id-or-name>",
		Short: "Delete an application from the catalog",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			printInfo("Application deletion functionality would be implemented here")
			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force deletion without confirmation")

	return cmd
}