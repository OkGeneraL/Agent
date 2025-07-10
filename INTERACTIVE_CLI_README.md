# SuperAgent Interactive CLI

## 🚀 Overview

SuperAgent now includes a powerful interactive CLI that provides a guided, user-friendly experience for setting up, configuring, and deploying applications. This interactive mode eliminates the need for complex command-line arguments and provides step-by-step guidance for all operations.

## ✨ Features

### 🎯 Interactive Setup Wizard
- **First-time setup**: Guided configuration of base domain, Traefik, and admin panel connection
- **Base domain configuration**: Set up your domain for automatic subdomain generation
- **Traefik integration**: Automatic installation and configuration of Traefik for routing
- **Admin panel connection**: Optional connection to the web-based admin panel

### 🚀 Smart Deployment
- **Repository detection**: Automatically detects public or private GitHub repositories
- **Environment file handling**: Detects `.env`, `.env.local`, `.env.example` files and prompts for values
- **Framework detection**: Automatically detects Node.js, Next.js, React, Python, Go, and Docker applications
- **Auto-build**: Automatically builds JavaScript applications without requiring a Dockerfile
- **Deployment confirmation**: Shows deployment summary before proceeding

### 🌐 Domain & Routing
- **Automatic subdomain generation**: Creates subdomains based on app names
- **Traefik integration**: Automatic route configuration for deployed applications
- **SSL support**: Automatic Let's Encrypt SSL certificate generation
- **DNS instructions**: Provides clear DNS configuration instructions

### 📊 Management & Monitoring
- **Deployment listing**: View all active deployments with status
- **Log viewing**: Interactive log viewing for any deployment
- **System status**: Real-time system health and status information
- **Configuration management**: View and modify agent configuration

## 🛠️ Installation

### Prerequisites
- SuperAgent installed and running
- Git installed
- Docker installed
- Root/sudo access (for Traefik installation)

### Starting Interactive CLI

```bash
# Start the interactive CLI
superagent interactive
```

## 📖 Usage Guide

### 1. First-Time Setup

When you run `superagent interactive` for the first time, you'll be guided through the setup process:

```bash
🚀 Welcome to SuperAgent Interactive CLI!
==========================================

🔍 Checking admin panel connection...
❌ Admin panel not connected
💡 You can still use the CLI for local management

📋 Main Menu:
1. 🚀 Deploy Application
2. 📊 View Deployments
3. ⚙️  Agent Configuration
4. 🌐 Domain & Traefik Setup
5. 📝 View Logs
6. 🔧 System Status
0. 🚪 Exit
```

### 2. Configuration Setup

Choose option 3 (Agent Configuration) to set up your environment:

```
⚙️  Agent Configuration
======================

Configuration Options:
1. 🔧 Setup Wizard
2. 🌐 Base Domain Configuration
3. 🔐 Admin Panel Connection
4. 📊 View Current Config
0. ↩️  Back to Main Menu
```

#### Base Domain Configuration
```
🌐 Base Domain Configuration
============================
Enter your base domain (default: example.com): mydomain.com
✅ Base domain set to: mydomain.com
```

#### Traefik Configuration
```
🔄 Traefik Configuration
========================
Enable Traefik for automatic routing? [yes/no]: yes
✅ Traefik enabled
⚠️  Traefik not found. Installing...
Installing Traefik...
✅ Traefik installed

⚙️  Traefik Settings
===================
Enable Traefik dashboard? [yes/no]: yes
✅ Traefik dashboard enabled at http://localhost:8080
Enable automatic SSL with Let's Encrypt? [yes/no]: yes
Enter email for Let's Encrypt: admin@mydomain.com
✅ SSL configured with email: admin@mydomain.com
```

### 3. Deploying Applications

Choose option 1 (Deploy Application) to deploy your first app:

```
🚀 Deploy Application
====================
Repository type [public/private]: public
Enter GitHub repository URL: https://github.com/username/myapp
Enter application ID (e.g., myapp): myapp
Enter version (e.g., v1.0.0) (default: latest): v1.0.0
Enter branch (default: main) (default: main): main

📥 Cloning repository to check configuration...
✅ JavaScript application detected

📄 Found environment file: .env.example
Enter value for DATABASE_URL (default: postgresql://localhost/mydb): postgresql://prod-server/mydb
Enter value for API_KEY (default: your-api-key): my-secret-api-key

📋 Deployment Summary:
  App ID: myapp
  Version: v1.0.0
  Repository: https://github.com/username/myapp
  Branch: main
  Environment Variables: 2
  Type: Next.js

Proceed with deployment? [yes/no]: yes

🚀 Creating deployment...
🎉 Deployment Successful!
=========================
Deployment ID: myapp-v1.0.0-1234567890
Status: running

🌐 Access URLs:
  Subdomain: https://myapp.mydomain.com
  IP Address: 203.0.113.1 (for A record)
  CNAME Record: myapp.mydomain.com
✅ Traefik route added for myapp

📝 DNS Configuration:
For custom domain, add these DNS records:
  A Record: @ → 203.0.113.1
  CNAME Record: www → myapp.mydomain.com

📋 Next Steps:
1. Wait for deployment to be ready (check status with 'superagent status')
2. Configure custom domain if needed
3. Set up SSL certificate
4. Monitor logs with 'superagent logs --deployment myapp-v1.0.0-1234567890'
5. View Traefik dashboard: http://localhost:8080
```

### 4. Managing Deployments

#### View All Deployments
```
📊 View Deployments
===================
ID                   APP             VERSION     STATUS       CREATED
--------------------------------------------------------------------------------
myapp-v1.0.0-123...  myapp           v1.0.0      running      2024-01-15 10:30:00
```

#### View Logs
```
📝 View Logs
============
Available deployments:
1. myapp (v1.0.0) - running
Enter deployment number or ID: 1

Logs for deployment: myapp-v1.0.0-1234567890
--------------------------------------------------
[2024-01-15 10:30:15] [info] Container started successfully
[2024-01-15 10:30:20] [info] Application listening on port 3000
[2024-01-15 10:30:25] [info] Health check passed
```

### 5. Domain & Traefik Management

#### DNS Instructions
```
📝 DNS Configuration Instructions
=================================
For domain: mydomain.com
Server IP: 203.0.113.1

Add these DNS records:
  A Record: @ → 203.0.113.1
  CNAME Record: www → mydomain.com

For subdomains (auto-generated):
  CNAME Record: [app-name] → mydomain.com
```

#### Test Traefik Configuration
```
🔧 Testing Traefik Configuration
================================
Testing Traefik API...
✅ Traefik configuration is valid
```

## 🔧 Advanced Features

### Environment File Detection

The interactive CLI automatically detects and handles environment files:

- `.env` - Production environment variables
- `.env.local` - Local environment variables
- `.env.example` - Example environment variables
- `.env.production` - Production-specific variables

For each variable found, you'll be prompted to enter a value or accept the default.

### Framework Detection

The CLI automatically detects application frameworks:

- **Node.js/Next.js**: Detects `package.json` and Next.js dependencies
- **React**: Detects React dependencies
- **Python**: Detects `requirements.txt`
- **Go**: Detects `go.mod`
- **Docker**: Detects `Dockerfile`

### Auto-Build for JavaScript Apps

For JavaScript applications without a Dockerfile, the CLI:

1. Detects `package.json`
2. Runs `npm install` (or `pnpm install` if npm fails)
3. Runs `npm run build` if available
4. Generates a minimal Dockerfile
5. Builds and deploys the container

### Traefik Integration

When Traefik is enabled:

- Automatically installs Traefik if not present
- Configures SSL with Let's Encrypt
- Creates dynamic routes for each deployment
- Provides dashboard access at `http://localhost:8080`

## 🚨 Troubleshooting

### Common Issues

#### Agent Not Running
```
⚠️  SuperAgent is not running. Starting agent...
```
The CLI will automatically start the agent if it's not running.

#### Traefik Installation Fails
```
❌ Failed to install Traefik: permission denied
```
Ensure you have sudo/root access for Traefik installation.

#### Repository Clone Fails
```
❌ Error: failed to clone repository: authentication required
```
For private repositories, ensure SSH keys or tokens are configured.

#### Environment Variables
If environment variables are not being detected:
- Check that the `.env` file is in the repository root
- Ensure the file follows the standard format: `KEY=value`
- Verify the file is not in `.gitignore`

### Getting Help

- Use `superagent --help` for command-line options
- Use `superagent interactive` for guided setup
- Check logs with `superagent logs --deployment <id>`
- View system status with `superagent status`

## 🔄 Migration from Command Line

If you're currently using command-line arguments, you can migrate to interactive mode:

### Before (Command Line)
```bash
superagent deploy \
  --app myapp \
  --version v1.0.0 \
  --source-type git \
  --source https://github.com/username/myapp \
  --branch main
```

### After (Interactive)
```bash
superagent interactive
# Then follow the guided prompts
```

## 📝 Configuration Files

The interactive CLI saves configuration to:
- `~/.superagent-interactive.yaml` - Interactive CLI settings
- `/etc/traefik/traefik.yml` - Traefik configuration
- `/etc/traefik/dynamic/` - Dynamic route configurations

## 🔐 Security Features

- **Environment variable masking**: Sensitive values are not displayed in logs
- **Secure token handling**: API tokens and keys are handled securely
- **Audit logging**: All interactive actions are logged for audit purposes
- **SSL/TLS**: Automatic SSL certificate generation with Let's Encrypt

## 🎯 Best Practices

1. **Use descriptive app IDs**: Use meaningful names like `myapp-frontend` instead of `app1`
2. **Configure base domain early**: Set up your domain before deploying applications
3. **Review environment variables**: Always review and customize environment variables
4. **Monitor deployments**: Use the logs feature to monitor deployment progress
5. **Backup configurations**: Keep backups of your configuration files

## 🚀 Next Steps

After setting up the interactive CLI:

1. **Deploy your first application** using the guided deployment
2. **Configure custom domains** if needed
3. **Set up monitoring** and alerts
4. **Explore the admin panel** if connected
5. **Scale your deployments** as needed

The interactive CLI makes SuperAgent as easy to use as Vercel or Heroku, with the power and flexibility of a self-hosted solution!