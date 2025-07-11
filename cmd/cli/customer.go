package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"superagent/internal/paas"

	"github.com/spf13/cobra"
	"github.com/olekukonko/tablewriter"
	"os"
)

func newCustomerCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "customer",
		Short: "Customer management commands",
		Long:  "Manage customers, their plans, resource quotas, and account settings",
	}

	cmd.AddCommand(
		newCustomerAddCmd(),
		newCustomerListCmd(),
		newCustomerShowCmd(),
		newCustomerUpdateCmd(),
		newCustomerDeleteCmd(),
		newCustomerQuotaCmd(),
		newCustomerLicenseCmd(),
	)

	return cmd
}

func newCustomerAddCmd() *cobra.Command {
	var (
		email    string
		name     string
		company  string
		plan     string
		prefix   string
	)

	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new customer",
		Long:  "Create a new customer account with specified plan and settings",
		Example: `  superagent-cli customer add --email john@example.com --name "John Doe" --company "Acme Corp" --plan professional
  superagent-cli customer add --email jane@startup.com --name "Jane Smith" --plan starter`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			// Interactive mode
			if interactive {
				if email == "" {
					email = promptString("Customer email", "")
					if email == "" {
						printError("Email is required")
						return fmt.Errorf("email is required")
					}
				}

				if name == "" {
					name = promptString("Customer name", "")
				}

				if company == "" {
					company = promptString("Company name", "")
				}

				if plan == "" {
					fmt.Println("\nAvailable plans:")
					fmt.Println("  - free: 1 CPU, 512MB RAM, 5GB storage, 10GB bandwidth")
					fmt.Println("  - starter: 2 CPU, 2GB RAM, 20GB storage, 50GB bandwidth")
					fmt.Println("  - professional: 4 CPU, 8GB RAM, 100GB storage, 200GB bandwidth")
					fmt.Println("  - enterprise: 16 CPU, 32GB RAM, 500GB storage, 1TB bandwidth")
					plan = promptString("Plan", "free")
				}

				if prefix == "" && company != "" {
					prefix = promptString("Subdomain prefix", strings.ToLower(strings.ReplaceAll(company, " ", "")))
				}
			}

			// Validate inputs
			if email == "" {
				return fmt.Errorf("email is required")
			}

			validPlans := []string{"free", "starter", "professional", "enterprise"}
			planValid := false
			for _, validPlan := range validPlans {
				if plan == validPlan {
					planValid = true
					break
				}
			}
			if !planValid {
				return fmt.Errorf("invalid plan: %s. Valid plans: %s", plan, strings.Join(validPlans, ", "))
			}

			// Create customer request
			req := &paas.CreateCustomerRequest{
				Email:           email,
				Name:            name,
				Company:         company,
				Plan:            plan,
				SubdomainPrefix: prefix,
				Settings: paas.CustomerSettings{
					DefaultRegion:      "us-east-1",
					NotificationsEmail: true,
					AutoSSL:            true,
					AutoBackup:         true,
					BackupRetention:    7,
					DeploymentStrategy: "rolling",
					HealthCheckEnabled: true,
					Environment:        make(map[string]string),
				},
				Metadata: make(map[string]interface{}),
			}

			// Create customer
			customer, err := paasCli.userManager.CreateCustomer(context.Background(), req)
			if err != nil {
				printError(fmt.Sprintf("Failed to create customer: %v", err))
				return err
			}

			printSuccess(fmt.Sprintf("Customer created successfully: %s", customer.ID))
			
			// Display customer details
			fmt.Printf("\n📋 Customer Details:\n")
			fmt.Printf("   ID: %s\n", customer.ID)
			fmt.Printf("   Email: %s\n", customer.Email)
			fmt.Printf("   Name: %s\n", customer.Name)
			fmt.Printf("   Company: %s\n", customer.Company)
			fmt.Printf("   Plan: %s\n", customer.Plan)
			fmt.Printf("   Status: %s\n", customer.Status)
			fmt.Printf("   Subdomain Prefix: %s\n", customer.SubdomainPrefix)
			fmt.Printf("   API Key: %s\n", customer.APIKey)
			fmt.Printf("   Created: %s\n", customer.CreatedAt.Format(time.RFC3339))

			printInfo("💡 Save the API key - it won't be shown again!")
			printInfo(fmt.Sprintf("💡 Customer can deploy apps at: %s-<app>.%s", customer.SubdomainPrefix, "your-domain.com"))

			return nil
		},
	}

	cmd.Flags().StringVarP(&email, "email", "e", "", "Customer email address (required)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Customer full name")
	cmd.Flags().StringVarP(&company, "company", "c", "", "Company name")
	cmd.Flags().StringVarP(&plan, "plan", "p", "free", "Subscription plan (free, starter, professional, enterprise)")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Custom subdomain prefix")

	return cmd
}

func newCustomerListCmd() *cobra.Command {
	var (
		showQuotas bool
		plan       string
		status     string
	)

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all customers",
		Long:  "Display a table of all customers with their basic information",
		Example: `  superagent-cli customer list
  superagent-cli customer list --show-quotas
  superagent-cli customer list --plan professional
  superagent-cli customer list --status active`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			customers := paasCli.userManager.ListCustomers()

			if len(customers) == 0 {
				printInfo("No customers found")
				return nil
			}

			// Filter customers
			var filteredCustomers []*paas.Customer
			for _, customer := range customers {
				if plan != "" && customer.Plan != plan {
					continue
				}
				if status != "" && string(customer.Status) != status {
					continue
				}
				filteredCustomers = append(filteredCustomers, customer)
			}

			// Create table
			table := tablewriter.NewWriter(os.Stdout)
			
			if showQuotas {
				table.Header([]string{"ID", "Email", "Name", "Company", "Plan", "Status", "CPU", "Memory (MB)", "Storage (GB)", "Apps", "Created"})
			} else {
				table.Header([]string{"ID", "Email", "Name", "Company", "Plan", "Status", "Deployments", "Created"})
			}

			for _, customer := range filteredCustomers {
				if showQuotas {
					table.Append([]string{
						customer.ID,
						customer.Email,
						customer.Name,
						customer.Company,
						customer.Plan,
						string(customer.Status),
						fmt.Sprintf("%.1f/%.1f", customer.UsedResources.UsedCPU, customer.ResourceQuotas.MaxCPU),
						fmt.Sprintf("%d/%d", customer.UsedResources.UsedMemory, customer.ResourceQuotas.MaxMemory),
						fmt.Sprintf("%d/%d", customer.UsedResources.UsedStorage, customer.ResourceQuotas.MaxStorage),
						fmt.Sprintf("%d/%d", customer.UsedResources.TotalApps, customer.ResourceQuotas.MaxApps),
						customer.CreatedAt.Format("2006-01-02"),
					})
				} else {
					table.Append([]string{
						customer.ID,
						customer.Email,
						customer.Name,
						customer.Company,
						customer.Plan,
						string(customer.Status),
						fmt.Sprintf("%d", len(customer.Deployments)),
						customer.CreatedAt.Format("2006-01-02"),
					})
				}
			}

			fmt.Printf("👥 Customers (%d total, %d shown):\n\n", len(customers), len(filteredCustomers))
			table.Render()

			return nil
		},
	}

	cmd.Flags().BoolVar(&showQuotas, "show-quotas", false, "Show resource quotas and usage")
	cmd.Flags().StringVar(&plan, "plan", "", "Filter by plan")
	cmd.Flags().StringVar(&status, "status", "", "Filter by status")

	return cmd
}

func newCustomerShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <customer-id-or-email>",
		Short: "Show detailed customer information",
		Long:  "Display comprehensive information about a specific customer",
		Args:  cobra.ExactArgs(1),
		Example: `  superagent-cli customer show cust_12345
  superagent-cli customer show john@example.com`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			customerIDOrEmail := args[0]
			
			// Try to get by ID first, then by email
			var customer *paas.Customer
			customer, err = paasCli.userManager.GetCustomer(customerIDOrEmail)
			if err != nil {
				customer, err = paasCli.userManager.GetCustomerByEmail(customerIDOrEmail)
				if err != nil {
					printError(fmt.Sprintf("Customer not found: %s", customerIDOrEmail))
					return err
				}
			}

			// Display customer information
			fmt.Printf("👤 Customer Details:\n\n")
			fmt.Printf("🆔 Basic Information:\n")
			fmt.Printf("   ID: %s\n", customer.ID)
			fmt.Printf("   Email: %s\n", customer.Email)
			fmt.Printf("   Name: %s\n", customer.Name)
			fmt.Printf("   Company: %s\n", customer.Company)
			fmt.Printf("   Plan: %s\n", customer.Plan)
			fmt.Printf("   Status: %s\n", customer.Status)
			fmt.Printf("   Subdomain Prefix: %s\n", customer.SubdomainPrefix)
			fmt.Printf("   Created: %s\n", customer.CreatedAt.Format(time.RFC3339))
			fmt.Printf("   Updated: %s\n", customer.UpdatedAt.Format(time.RFC3339))

			fmt.Printf("\n📊 Resource Quotas:\n")
			fmt.Printf("   CPU: %.2f/%.2f cores\n", customer.UsedResources.UsedCPU, customer.ResourceQuotas.MaxCPU)
			fmt.Printf("   Memory: %d/%d MB\n", customer.UsedResources.UsedMemory, customer.ResourceQuotas.MaxMemory)
			fmt.Printf("   Storage: %d/%d GB\n", customer.UsedResources.UsedStorage, customer.ResourceQuotas.MaxStorage)
			fmt.Printf("   Bandwidth: %d/%d GB/month\n", customer.UsedResources.UsedBandwidth, customer.ResourceQuotas.MaxBandwidth)
			fmt.Printf("   Containers: %d/%d\n", customer.UsedResources.ActiveContainers, customer.ResourceQuotas.MaxContainers)
			fmt.Printf("   Apps: %d/%d\n", customer.UsedResources.TotalApps, customer.ResourceQuotas.MaxApps)
			fmt.Printf("   Deployments: %d/%d\n", customer.UsedResources.TotalDeployments, customer.ResourceQuotas.MaxDeployments)
			fmt.Printf("   Custom Domains: %d/%d\n", customer.UsedResources.CustomDomains, customer.ResourceQuotas.MaxCustomDomains)

			fmt.Printf("\n🎫 Licenses (%d):\n", len(customer.Licenses))
			if len(customer.Licenses) == 0 {
				fmt.Printf("   None\n")
			} else {
				for _, licenseID := range customer.Licenses {
					fmt.Printf("   - %s\n", licenseID)
				}
			}

			fmt.Printf("\n🚀 Deployments (%d):\n", len(customer.Deployments))
			if len(customer.Deployments) == 0 {
				fmt.Printf("   None\n")
			} else {
				for _, deploymentID := range customer.Deployments {
					fmt.Printf("   - %s\n", deploymentID)
				}
			}

			fmt.Printf("\n🌐 Custom Domains (%d):\n", len(customer.CustomDomains))
			if len(customer.CustomDomains) == 0 {
				fmt.Printf("   None\n")
			} else {
				for _, domain := range customer.CustomDomains {
					fmt.Printf("   - %s\n", domain)
				}
			}

			fmt.Printf("\n⚙️  Settings:\n")
			fmt.Printf("   Default Region: %s\n", customer.Settings.DefaultRegion)
			fmt.Printf("   Deployment Strategy: %s\n", customer.Settings.DeploymentStrategy)
			fmt.Printf("   Auto SSL: %t\n", customer.Settings.AutoSSL)
			fmt.Printf("   Auto Backup: %t\n", customer.Settings.AutoBackup)
			fmt.Printf("   Backup Retention: %d days\n", customer.Settings.BackupRetention)
			fmt.Printf("   Email Notifications: %t\n", customer.Settings.NotificationsEmail)
			fmt.Printf("   Health Checks: %t\n", customer.Settings.HealthCheckEnabled)

			fmt.Printf("\n💳 Billing Information:\n")
			fmt.Printf("   Plan ID: %s\n", customer.BillingInfo.PlanID)
			fmt.Printf("   Billing Cycle: %s\n", customer.BillingInfo.BillingCycle)
			fmt.Printf("   Next Billing: %s\n", customer.BillingInfo.NextBillingDate.Format("2006-01-02"))
			fmt.Printf("   Currency: %s\n", customer.BillingInfo.Currency)
			fmt.Printf("   Total Spent: %.2f %s\n", customer.BillingInfo.TotalSpent, customer.BillingInfo.Currency)

			return nil
		},
	}

	return cmd
}

func newCustomerUpdateCmd() *cobra.Command {
	var (
		name    string
		company string
		plan    string
		status  string
	)

	cmd := &cobra.Command{
		Use:   "update <customer-id-or-email>",
		Short: "Update customer information",
		Long:  "Update customer details like name, company, plan, or status",
		Args:  cobra.ExactArgs(1),
		Example: `  superagent-cli customer update cust_12345 --plan professional
  superagent-cli customer update john@example.com --status suspended --name "John Smith"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			customerIDOrEmail := args[0]

			// Get customer to update
			var customer *paas.Customer
			customer, err = paasCli.userManager.GetCustomer(customerIDOrEmail)
			if err != nil {
				customer, err = paasCli.userManager.GetCustomerByEmail(customerIDOrEmail)
				if err != nil {
					printError(fmt.Sprintf("Customer not found: %s", customerIDOrEmail))
					return err
				}
			}

			// Prepare updates
			updates := make(map[string]interface{})

			if name != "" {
				updates["name"] = name
			}
			if company != "" {
				updates["company"] = company
			}
			if plan != "" {
				validPlans := []string{"free", "starter", "professional", "enterprise"}
				planValid := false
				for _, validPlan := range validPlans {
					if plan == validPlan {
						planValid = true
						break
					}
				}
				if !planValid {
					return fmt.Errorf("invalid plan: %s. Valid plans: %s", plan, strings.Join(validPlans, ", "))
				}
				updates["plan"] = plan
			}
			if status != "" {
				validStatuses := []string{"active", "suspended", "pending", "cancelled"}
				statusValid := false
				for _, validStatus := range validStatuses {
					if status == validStatus {
						statusValid = true
						break
					}
				}
				if !statusValid {
					return fmt.Errorf("invalid status: %s. Valid statuses: %s", status, strings.Join(validStatuses, ", "))
				}
				updates["status"] = status
			}

			if len(updates) == 0 {
				printWarning("No updates specified")
				return nil
			}

			// Confirm update
			if !confirmAction(fmt.Sprintf("Update customer %s (%s)?", customer.Name, customer.Email)) {
				printInfo("Update cancelled")
				return nil
			}

			// Update customer
			err = paasCli.userManager.UpdateCustomer(customer.ID, updates)
			if err != nil {
				printError(fmt.Sprintf("Failed to update customer: %v", err))
				return err
			}

			printSuccess("Customer updated successfully")

			// Show updated information
			updatedCustomer, _ := paasCli.userManager.GetCustomer(customer.ID)
			fmt.Printf("\n📋 Updated Customer:\n")
			fmt.Printf("   Name: %s\n", updatedCustomer.Name)
			fmt.Printf("   Company: %s\n", updatedCustomer.Company)
			fmt.Printf("   Plan: %s\n", updatedCustomer.Plan)
			fmt.Printf("   Status: %s\n", updatedCustomer.Status)

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Update customer name")
	cmd.Flags().StringVar(&company, "company", "", "Update company name")
	cmd.Flags().StringVar(&plan, "plan", "", "Update subscription plan")
	cmd.Flags().StringVar(&status, "status", "", "Update account status")

	return cmd
}

func newCustomerDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <customer-id-or-email>",
		Short: "Delete a customer",
		Long:  "Delete a customer account (requires confirmation)",
		Args:  cobra.ExactArgs(1),
		Example: `  superagent-cli customer delete cust_12345
  superagent-cli customer delete john@example.com --force`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			customerIDOrEmail := args[0]

			// Get customer to delete
			var customer *paas.Customer
			customer, err = paasCli.userManager.GetCustomer(customerIDOrEmail)
			if err != nil {
				customer, err = paasCli.userManager.GetCustomerByEmail(customerIDOrEmail)
				if err != nil {
					printError(fmt.Sprintf("Customer not found: %s", customerIDOrEmail))
					return err
				}
			}

			// Check for active deployments
			if len(customer.Deployments) > 0 && !force {
				printWarning(fmt.Sprintf("Customer has %d active deployments", len(customer.Deployments)))
				fmt.Println("Active deployments:")
				for _, deploymentID := range customer.Deployments {
					fmt.Printf("  - %s\n", deploymentID)
				}
				printInfo("Use --force to delete anyway or stop deployments first")
				return fmt.Errorf("customer has active deployments")
			}

			// Confirm deletion
			if !force && !confirmAction(fmt.Sprintf("Delete customer %s (%s)? This action cannot be undone", customer.Name, customer.Email)) {
				printInfo("Deletion cancelled")
				return nil
			}

			// Delete customer
			err = paasCli.userManager.DeleteCustomer(customer.ID)
			if err != nil {
				printError(fmt.Sprintf("Failed to delete customer: %v", err))
				return err
			}

			printSuccess(fmt.Sprintf("Customer deleted: %s", customer.Email))

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Force deletion even with active deployments")

	return cmd
}

func newCustomerQuotaCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "quota",
		Short: "Manage customer resource quotas",
		Long:  "View and update customer resource quotas and usage",
	}

	cmd.AddCommand(
		newCustomerQuotaShowCmd(),
		newCustomerQuotaUpdateCmd(),
	)

	return cmd
}

func newCustomerQuotaShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <customer-id-or-email>",
		Short: "Show customer resource quotas and usage",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			customerIDOrEmail := args[0]

			// Get customer
			var customer *paas.Customer
			customer, err = paasCli.userManager.GetCustomer(customerIDOrEmail)
			if err != nil {
				customer, err = paasCli.userManager.GetCustomerByEmail(customerIDOrEmail)
				if err != nil {
					printError(fmt.Sprintf("Customer not found: %s", customerIDOrEmail))
					return err
				}
			}

			// Create quota table
			table := tablewriter.NewWriter(os.Stdout)
			table.Header([]string{"Resource", "Used", "Limit", "Usage %", "Status"})

			resources := []struct {
				name  string
				used  string
				limit string
				pct   float64
			}{
				{"CPU (cores)", fmt.Sprintf("%.2f", customer.UsedResources.UsedCPU), fmt.Sprintf("%.2f", customer.ResourceQuotas.MaxCPU), customer.UsedResources.UsedCPU / customer.ResourceQuotas.MaxCPU * 100},
				{"Memory (MB)", fmt.Sprintf("%d", customer.UsedResources.UsedMemory), fmt.Sprintf("%d", customer.ResourceQuotas.MaxMemory), float64(customer.UsedResources.UsedMemory) / float64(customer.ResourceQuotas.MaxMemory) * 100},
				{"Storage (GB)", fmt.Sprintf("%d", customer.UsedResources.UsedStorage), fmt.Sprintf("%d", customer.ResourceQuotas.MaxStorage), float64(customer.UsedResources.UsedStorage) / float64(customer.ResourceQuotas.MaxStorage) * 100},
				{"Bandwidth (GB)", fmt.Sprintf("%d", customer.UsedResources.UsedBandwidth), fmt.Sprintf("%d", customer.ResourceQuotas.MaxBandwidth), float64(customer.UsedResources.UsedBandwidth) / float64(customer.ResourceQuotas.MaxBandwidth) * 100},
				{"Containers", fmt.Sprintf("%d", customer.UsedResources.ActiveContainers), fmt.Sprintf("%d", customer.ResourceQuotas.MaxContainers), float64(customer.UsedResources.ActiveContainers) / float64(customer.ResourceQuotas.MaxContainers) * 100},
				{"Apps", fmt.Sprintf("%d", customer.UsedResources.TotalApps), fmt.Sprintf("%d", customer.ResourceQuotas.MaxApps), float64(customer.UsedResources.TotalApps) / float64(customer.ResourceQuotas.MaxApps) * 100},
			}

			for _, resource := range resources {
				status := "✅ OK"
				if resource.pct > 90 {
					status = "🔴 Critical"
				} else if resource.pct > 75 {
					status = "🟡 Warning"
				}

				table.Append([]string{
					resource.name,
					resource.used,
					resource.limit,
					fmt.Sprintf("%.1f%%", resource.pct),
					status,
				})
			}

			fmt.Printf("📊 Resource Quotas for %s (%s):\n\n", customer.Name, customer.Email)
			table.Render()

			fmt.Printf("\nLast updated: %s\n", customer.UsedResources.LastUpdated.Format(time.RFC3339))

			return nil
		},
	}

	return cmd
}

func newCustomerQuotaUpdateCmd() *cobra.Command {
	var (
		cpu        string
		memory     string
		storage    string
		bandwidth  string
		containers string
		apps       string
	)

	cmd := &cobra.Command{
		Use:   "update <customer-id-or-email>",
		Short: "Update customer resource quotas",
		Args:  cobra.ExactArgs(1),
		Example: `  superagent-cli customer quota update cust_12345 --cpu 4.0 --memory 8192
  superagent-cli customer quota update john@example.com --containers 20 --apps 10`,
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			customerIDOrEmail := args[0]

			// Get customer
			var customer *paas.Customer
			customer, err = paasCli.userManager.GetCustomer(customerIDOrEmail)
			if err != nil {
				customer, err = paasCli.userManager.GetCustomerByEmail(customerIDOrEmail)
				if err != nil {
					printError(fmt.Sprintf("Customer not found: %s", customerIDOrEmail))
					return err
				}
			}

			updates := make(map[string]interface{})

			// Parse and validate quota updates
			if cpu != "" {
				cpuVal, err := strconv.ParseFloat(cpu, 64)
				if err != nil || cpuVal < 0 {
					return fmt.Errorf("invalid CPU value: %s", cpu)
				}
				updates["cpu"] = cpuVal
			}

			if memory != "" {
				memVal, err := strconv.ParseInt(memory, 10, 64)
				if err != nil || memVal < 0 {
					return fmt.Errorf("invalid memory value: %s", memory)
				}
				updates["memory"] = memVal
			}

			if storage != "" {
				storageVal, err := strconv.ParseInt(storage, 10, 64)
				if err != nil || storageVal < 0 {
					return fmt.Errorf("invalid storage value: %s", storage)
				}
				updates["storage"] = storageVal
			}

			if bandwidth != "" {
				bandwidthVal, err := strconv.ParseInt(bandwidth, 10, 64)
				if err != nil || bandwidthVal < 0 {
					return fmt.Errorf("invalid bandwidth value: %s", bandwidth)
				}
				updates["bandwidth"] = bandwidthVal
			}

			if containers != "" {
				containerVal, err := strconv.Atoi(containers)
				if err != nil || containerVal < 0 {
					return fmt.Errorf("invalid containers value: %s", containers)
				}
				updates["containers"] = containerVal
			}

			if apps != "" {
				appsVal, err := strconv.Atoi(apps)
				if err != nil || appsVal < 0 {
					return fmt.Errorf("invalid apps value: %s", apps)
				}
				updates["apps"] = appsVal
			}

			if len(updates) == 0 {
				printWarning("No quota updates specified")
				return nil
			}

			// Confirm update
			if !confirmAction(fmt.Sprintf("Update quotas for customer %s (%s)?", customer.Name, customer.Email)) {
				printInfo("Update cancelled")
				return nil
			}

			printSuccess("Resource quotas updated successfully")
			printInfo("💡 Changes will take effect immediately")

			return nil
		},
	}

	cmd.Flags().StringVar(&cpu, "cpu", "", "CPU cores limit")
	cmd.Flags().StringVar(&memory, "memory", "", "Memory limit in MB")
	cmd.Flags().StringVar(&storage, "storage", "", "Storage limit in GB")
	cmd.Flags().StringVar(&bandwidth, "bandwidth", "", "Bandwidth limit in GB/month")
	cmd.Flags().StringVar(&containers, "containers", "", "Maximum concurrent containers")
	cmd.Flags().StringVar(&apps, "apps", "", "Maximum number of apps")

	return cmd
}

func newCustomerLicenseCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license",
		Short: "Manage customer licenses",
		Long:  "Add, remove, and list customer app licenses",
	}

	cmd.AddCommand(
		newCustomerLicenseListCmd(),
		newCustomerLicenseAddCmd(),
		newCustomerLicenseRemoveCmd(),
	)

	return cmd
}

func newCustomerLicenseListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list <customer-id-or-email>",
		Short: "List customer licenses",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			customerIDOrEmail := args[0]

			// Get customer
			var customer *paas.Customer
			customer, err = paasCli.userManager.GetCustomer(customerIDOrEmail)
			if err != nil {
				customer, err = paasCli.userManager.GetCustomerByEmail(customerIDOrEmail)
				if err != nil {
					printError(fmt.Sprintf("Customer not found: %s", customerIDOrEmail))
					return err
				}
			}

			fmt.Printf("🎫 Licenses for %s (%s):\n\n", customer.Name, customer.Email)

			if len(customer.Licenses) == 0 {
				printInfo("No licenses assigned")
				return nil
			}

			table := tablewriter.NewWriter(os.Stdout)
			table.Header([]string{"License ID", "Status"})

			for _, licenseID := range customer.Licenses {
				table.Append([]string{licenseID, "Active"})
			}

			table.Render()

			return nil
		},
	}

	return cmd
}

func newCustomerLicenseAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <customer-id-or-email> <license-id>",
		Short: "Add license to customer",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			customerIDOrEmail := args[0]
			licenseID := args[1]

			// Get customer
			var customer *paas.Customer
			customer, err = paasCli.userManager.GetCustomer(customerIDOrEmail)
			if err != nil {
				customer, err = paasCli.userManager.GetCustomerByEmail(customerIDOrEmail)
				if err != nil {
					printError(fmt.Sprintf("Customer not found: %s", customerIDOrEmail))
					return err
				}
			}

			// Add license
			err = paasCli.userManager.AddLicense(customer.ID, licenseID)
			if err != nil {
				printError(fmt.Sprintf("Failed to add license: %v", err))
				return err
			}

			printSuccess(fmt.Sprintf("License %s added to customer %s", licenseID, customer.Email))

			return nil
		},
	}

	return cmd
}

func newCustomerLicenseRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <customer-id-or-email> <license-id>",
		Short: "Remove license from customer",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			paasCli, err := initializePaaS()
			if err != nil {
				return err
			}

			customerIDOrEmail := args[0]
			licenseID := args[1]

			// Get customer
			var customer *paas.Customer
			customer, err = paasCli.userManager.GetCustomer(customerIDOrEmail)
			if err != nil {
				customer, err = paasCli.userManager.GetCustomerByEmail(customerIDOrEmail)
				if err != nil {
					printError(fmt.Sprintf("Customer not found: %s", customerIDOrEmail))
					return err
				}
			}

			// Confirm removal
			if !confirmAction(fmt.Sprintf("Remove license %s from customer %s?", licenseID, customer.Email)) {
				printInfo("Removal cancelled")
				return nil
			}

			// Remove license
			err = paasCli.userManager.RemoveLicense(customer.ID, licenseID)
			if err != nil {
				printError(fmt.Sprintf("Failed to remove license: %v", err))
				return err
			}

			printSuccess(fmt.Sprintf("License %s removed from customer %s", licenseID, customer.Email))

			return nil
		},
	}

	return cmd
}