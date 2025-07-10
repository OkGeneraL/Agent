# SuperAgent & Admin Panel Authentication Setup Guide

## Overview

This guide explains how to set up secure authentication between SuperAgent deployment servers and the Admin Panel. The authentication system uses cryptographically secure Bearer tokens that are managed through the admin panel and configured on each SuperAgent server.

## Authentication Architecture

```
[Admin Panel] ←→ [Token Validation API] ←→ [SuperAgent Server]
     ↓                                            ↑
[Database]                                   [Config File]
- Stores token hashes                        - Contains API token
- Server registration                        - Admin panel URL
- Token metadata                            - Server configuration
```

## Security Features

- **Cryptographically Secure Tokens**: 256-bit random tokens with `sa_` prefix
- **Hash Storage**: Only SHA-256 hashes stored in database, never plain tokens
- **Token Expiration**: Configurable expiration (default 1 year)
- **Token Rotation**: Ability to rotate tokens for security
- **Audit Logging**: Complete audit trail of all authentication events
- **Rate Limiting**: Built-in protection against brute force attacks

## Admin Panel Setup

### 1. Database Schema Update

First, ensure your Supabase database has the updated schema with the `server_tokens` table:

```sql
-- Run this in your Supabase SQL editor
-- (This is already included in the main schema.sql file)

CREATE TABLE server_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    token_prefix VARCHAR(12) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    created_by UUID REFERENCES admin_users(id),
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES admin_users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);
```

### 2. Environment Configuration

Update your admin panel environment variables:

```env
# Required for Supabase connection
NEXT_PUBLIC_SUPABASE_URL=your-supabase-project-url
SUPABASE_SERVICE_ROLE_KEY=your-service-role-key

# Optional: Admin panel URL for SuperAgent configuration generation
NEXT_PUBLIC_ADMIN_PANEL_URL=https://your-admin-panel.com
```

### 3. Deploy Admin Panel

```bash
cd adminpanel/admin
npm install
npm run build
npm start
```

## SuperAgent Server Setup

### Step-by-Step Process

#### 1. Add Server via Admin Panel

1. **Login to Admin Panel**
   - Navigate to your admin panel dashboard
   - Go to "Servers" section
   - Click "Add Server"

2. **Fill Server Details** (Step 1 of 3)
   - **Server Name**: e.g., `production-east-1`
   - **Hostname**: e.g., `superagent-prod-01`
   - **IP Address**: e.g., `10.0.1.100`
   - **Location**: e.g., `us-east-1`
   - **Provider**: Select your cloud provider
   - **Resources**: CPU cores, memory, disk space
   - **API Endpoint**: e.g., `http://10.0.1.100:8080`

3. **Copy Authentication Token** (Step 2 of 3)
   - Secure token is auto-generated (e.g., `sa_AbCdEf123...`)
   - **Important**: Copy and store securely - shown only once
   - Token expires in 1 year by default

4. **Follow Setup Instructions** (Step 3 of 3)
   - Copy the generated configuration
   - Test connection after setup

#### 2. Configure SuperAgent Server

**On your SuperAgent server**, edit the configuration file:

```bash
sudo nano /etc/superagent/config.yaml
```

**Update the configuration** with values from the admin panel:

```yaml
# SuperAgent Configuration
agent:
  id: "generated-server-id-from-admin-panel"
  location: "us-east-1"  # Your server location
  work_dir: "/var/lib/superagent"
  data_dir: "/var/lib/superagent/data"

backend:
  base_url: "https://your-admin-panel.com"
  api_token: "sa_your-secure-token-from-admin-panel"
  refresh_interval: "30s"
  timeout: "30s"

monitoring:
  enabled: true
  metrics_port: 9090
  health_check_port: 8080

security:
  audit_log_enabled: true
  run_as_non_root: true

docker:
  host: "unix:///var/run/docker.sock"
  network_name: "superagent"

logging:
  level: "info"
  format: "json"
  output: "file"
  log_file: "/var/log/superagent/agent.log"
```

#### 3. Restart SuperAgent Service

```bash
# Restart the service
sudo systemctl restart superagent

# Check service status
sudo systemctl status superagent

# View logs if needed
sudo journalctl -u superagent -f
```

#### 4. Verify Connection

Back in the **Admin Panel**, click "Test Connection" to verify:

- ✅ Connection successful → Server will show as "Online"
- ❌ Connection failed → Check configuration and network access

## Network Requirements

### Required Ports

| Port | Service | Access | Description |
|------|---------|--------|-------------|
| 8080 | API Server | Admin Panel → SuperAgent | REST API endpoints |
| 9090 | Metrics | Admin Panel → SuperAgent | Prometheus metrics (optional) |

### Firewall Configuration

```bash
# Allow API access (adjust source IP as needed)
sudo ufw allow from admin-panel-ip to any port 8080

# Allow metrics access (optional)
sudo ufw allow from admin-panel-ip to any port 9090

# Or for internal networks:
sudo ufw allow 8080/tcp
sudo ufw allow 9090/tcp
```

### Security Recommendations

1. **Private Networks**: Use VPC/private networks when possible
2. **IP Whitelisting**: Restrict access to admin panel IPs only
3. **SSL/TLS**: Use HTTPS for admin panel and consider TLS for agent communication
4. **VPN Access**: Route admin panel through VPN for additional security

## Token Management

### Token Rotation

To rotate a token for security:

1. **In Admin Panel**:
   - Go to server details
   - Click "Rotate Token"
   - Copy the new token

2. **On SuperAgent Server**:
   - Update `api_token` in `/etc/superagent/config.yaml`
   - Restart SuperAgent service

### Token Revocation

To revoke a token:

1. **In Admin Panel**: Click "Revoke Token"
2. **On SuperAgent Server**: Service will start failing authentication
3. **Generate New Token**: Follow token rotation process

## Troubleshooting

### Common Issues

#### 1. Connection Failed
```
Error: Connection failed: dial tcp: connect: connection refused
```

**Solution**:
- Check SuperAgent service status: `sudo systemctl status superagent`
- Verify port 8080 is accessible: `telnet server-ip 8080`
- Check firewall rules

#### 2. Authentication Failed
```
Error: HTTP 401: Unauthorized
```

**Solution**:
- Verify token in config file matches admin panel
- Check token hasn't expired
- Ensure `base_url` points to correct admin panel URL

#### 3. Invalid Token Format
```
Error: Invalid token format
```

**Solution**:
- Ensure token starts with `sa_`
- Check for whitespace/newlines in config file
- Re-copy token from admin panel

#### 4. Admin Panel URL Not Accessible
```
Error: validation failed: dial tcp: no such host
```

**Solution**:
- Verify `base_url` in SuperAgent config
- Check DNS resolution from SuperAgent server
- Test admin panel accessibility: `curl https://your-admin-panel.com/api/auth/validate`

### Log Analysis

#### SuperAgent Logs
```bash
# View authentication logs
sudo journalctl -u superagent | grep AUTH

# View all logs
sudo tail -f /var/log/superagent/agent.log
```

#### Admin Panel Logs
Check the admin panel logs for token validation requests and any errors.

## Security Best Practices

### 1. Token Security
- Never commit tokens to version control
- Store tokens securely (environment variables or secure files)
- Rotate tokens regularly (every 3-6 months)
- Use different tokens for different environments

### 2. Network Security
- Use private networks/VPCs
- Implement IP whitelisting
- Enable SSL/TLS encryption
- Monitor for suspicious activity

### 3. Access Control
- Limit admin panel access to authorized personnel
- Use strong authentication for admin panel
- Implement audit logging
- Regular security reviews

### 4. Monitoring
- Monitor authentication failures
- Set up alerts for suspicious activity
- Regular token usage audits
- Track token expiration dates

## API Reference

### Token Validation Endpoint

**URL**: `POST /api/auth/validate`

**Request**:
```json
{
  "token": "sa_your-secure-token"
}
```

**Response (Success)**:
```json
{
  "valid": true,
  "server_id": "uuid-of-server",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

**Response (Failure)**:
```json
{
  "valid": false,
  "error": "Invalid or expired token"
}
```

## Configuration Reference

### SuperAgent Config Schema

```yaml
agent:
  id: string              # Server UUID from admin panel
  location: string        # Geographic location identifier
  work_dir: string        # Working directory path
  data_dir: string        # Data storage directory

backend:
  base_url: string        # Admin panel URL (required)
  api_token: string       # Authentication token (required)
  refresh_interval: duration  # Token refresh check interval
  timeout: duration       # Request timeout

monitoring:
  enabled: boolean        # Enable monitoring
  metrics_port: integer   # Prometheus metrics port
  health_check_port: integer  # Health check port

security:
  audit_log_enabled: boolean   # Enable audit logging
  run_as_non_root: boolean     # Security flag
```

## Support

For issues or questions:

1. Check the troubleshooting section above
2. Review SuperAgent logs: `/var/log/superagent/agent.log`
3. Check admin panel logs for validation errors
4. Verify network connectivity and firewall rules

---

**Security Notice**: This authentication system provides enterprise-grade security when properly configured. Always follow security best practices and regularly audit your token usage.