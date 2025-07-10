-- =====================================================
-- SuperAgent PaaS Admin Panel - Complete Database Schema
-- Compatible with Supabase PostgreSQL
-- =====================================================

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =====================================================
-- AUTHENTICATION & USERS
-- =====================================================

-- Admin users table (separate from auth.users for additional info)
CREATE TABLE admin_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth_user_id UUID REFERENCES auth.users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'admin' CHECK (role IN ('super_admin', 'admin', 'support', 'viewer')),
    avatar_url TEXT,
    is_active BOOLEAN DEFAULT TRUE,
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- CUSTOMERS & SUBSCRIPTIONS
-- =====================================================

-- Customer plans
CREATE TABLE customer_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    price_monthly DECIMAL(10,2) NOT NULL DEFAULT 0,
    price_yearly DECIMAL(10,2) NOT NULL DEFAULT 0,
    max_applications INTEGER NOT NULL DEFAULT 1,
    max_deployments INTEGER NOT NULL DEFAULT 5,
    max_domains INTEGER NOT NULL DEFAULT 1,
    max_storage_gb INTEGER NOT NULL DEFAULT 1,
    max_bandwidth_gb INTEGER NOT NULL DEFAULT 10,
    max_cpu_cores INTEGER NOT NULL DEFAULT 1,
    max_memory_gb INTEGER NOT NULL DEFAULT 1,
    features JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default plans
INSERT INTO customer_plans (name, slug, description, price_monthly, price_yearly, max_applications, max_deployments, max_domains, max_storage_gb, max_bandwidth_gb, max_cpu_cores, max_memory_gb, features) VALUES
('Free', 'free', 'Free tier for testing', 0, 0, 1, 2, 1, 1, 10, 1, 1, '["Basic Support"]'),
('Starter', 'starter', 'Perfect for small projects', 19, 190, 5, 10, 3, 5, 50, 2, 2, '["Email Support", "SSL Certificates", "Custom Domains"]'),
('Pro', 'pro', 'For growing businesses', 99, 990, 25, 50, 10, 25, 250, 4, 8, '["Priority Support", "Advanced Analytics", "Team Collaboration", "API Access"]'),
('Enterprise', 'enterprise', 'For large organizations', 499, 4990, 100, 200, 50, 100, 1000, 8, 32, '["24/7 Phone Support", "SLA Guarantee", "Custom Integrations", "Dedicated Account Manager", "Advanced Security"]');

-- Customers table
CREATE TABLE customers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL UNIQUE,
    company VARCHAR(255),
    phone VARCHAR(50),
    address TEXT,
    plan_id UUID REFERENCES customer_plans(id) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'cancelled', 'pending')),
    billing_email VARCHAR(255),
    stripe_customer_id VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Customer usage tracking
CREATE TABLE customer_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    month DATE NOT NULL, -- First day of the month
    applications_count INTEGER DEFAULT 0,
    deployments_count INTEGER DEFAULT 0,
    domains_count INTEGER DEFAULT 0,
    storage_gb_used DECIMAL(10,2) DEFAULT 0,
    bandwidth_gb_used DECIMAL(10,2) DEFAULT 0,
    cpu_hours_used DECIMAL(10,2) DEFAULT 0,
    memory_hours_used DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(customer_id, month)
);

-- =====================================================
-- APPLICATIONS CATALOG
-- =====================================================

-- Application publishers
CREATE TABLE publishers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    company VARCHAR(255),
    website_url TEXT,
    support_url TEXT,
    is_verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Application categories
CREATE TABLE application_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    slug VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    icon_url TEXT,
    sort_order INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default categories
INSERT INTO application_categories (name, slug, description, sort_order) VALUES
('Web Applications', 'web-apps', 'Full-stack web applications and websites', 1),
('APIs & Microservices', 'apis', 'REST APIs, GraphQL, and microservices', 2),
('Databases', 'databases', 'Database systems and data storage solutions', 3),
('Developer Tools', 'dev-tools', 'Development and productivity tools', 4),
('E-commerce', 'ecommerce', 'Online stores and commerce platforms', 5),
('Content Management', 'cms', 'Content management systems and blogs', 6),
('Analytics', 'analytics', 'Data analytics and monitoring tools', 7),
('Communication', 'communication', 'Chat, email, and collaboration tools', 8);

-- Applications catalog
CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    long_description TEXT,
    publisher_id UUID REFERENCES publishers(id),
    category_id UUID REFERENCES application_categories(id),
    version VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'published', 'archived')),
    source_type VARCHAR(20) NOT NULL CHECK (source_type IN ('git', 'docker', 'zip')),
    source_url TEXT NOT NULL,
    dockerfile_path VARCHAR(255) DEFAULT 'Dockerfile',
    build_command TEXT,
    start_command TEXT,
    port INTEGER DEFAULT 3000,
    environment_variables JSONB DEFAULT '{}',
    pricing_type VARCHAR(20) NOT NULL DEFAULT 'free' CHECK (pricing_type IN ('free', 'one_time', 'subscription')),
    price DECIMAL(10,2) DEFAULT 0,
    currency VARCHAR(3) DEFAULT 'USD',
    billing_period VARCHAR(20) CHECK (billing_period IN ('monthly', 'yearly')),
    logo_url TEXT,
    screenshots JSONB DEFAULT '[]',
    tags JSONB DEFAULT '[]',
    requirements JSONB DEFAULT '{}',
    approval_notes TEXT,
    approved_by UUID REFERENCES admin_users(id),
    approved_at TIMESTAMPTZ,
    download_count INTEGER DEFAULT 0,
    rating DECIMAL(2,1) DEFAULT 0,
    rating_count INTEGER DEFAULT 0,
    is_featured BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Application versions
CREATE TABLE application_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id UUID REFERENCES applications(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    changelog TEXT,
    source_url TEXT NOT NULL,
    dockerfile_path VARCHAR(255) DEFAULT 'Dockerfile',
    build_command TEXT,
    start_command TEXT,
    environment_variables JSONB DEFAULT '{}',
    is_stable BOOLEAN DEFAULT TRUE,
    download_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(application_id, version)
);

-- =====================================================
-- SERVERS & AGENTS
-- =====================================================

-- Server providers
CREATE TYPE server_provider AS ENUM ('aws', 'gcp', 'azure', 'digitalocean', 'linode', 'vultr', 'dedicated', 'other');

-- Servers/Agents
CREATE TABLE servers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    hostname VARCHAR(255) NOT NULL,
    ip_address INET NOT NULL,
    location VARCHAR(100),
    provider server_provider NOT NULL DEFAULT 'other',
    status VARCHAR(20) NOT NULL DEFAULT 'offline' CHECK (status IN ('online', 'offline', 'maintenance', 'error')),
    agent_version VARCHAR(50),
    cpu_cores INTEGER NOT NULL,
    memory_gb INTEGER NOT NULL,
    disk_gb INTEGER NOT NULL,
    bandwidth_gbps DECIMAL(10,2),
    architecture VARCHAR(20) DEFAULT 'x86_64',
    os VARCHAR(100),
    os_version VARCHAR(50),
    max_deployments INTEGER DEFAULT 50,
    current_deployments INTEGER DEFAULT 0,
    api_endpoint TEXT,
    last_heartbeat TIMESTAMPTZ,
    health_score DECIMAL(5,2) DEFAULT 100.0,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Server Authentication Tokens
CREATE TABLE server_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE, -- SHA-256 hash of the token
    token_prefix VARCHAR(12) NOT NULL, -- First 12 chars for identification (sa_xxxxxxxx)
    expires_at TIMESTAMPTZ NOT NULL,
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    created_by UUID REFERENCES admin_users(id),
    revoked_at TIMESTAMPTZ,
    revoked_by UUID REFERENCES admin_users(id),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Server metrics (time series data)
CREATE TABLE server_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    server_id UUID REFERENCES servers(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cpu_usage_percent DECIMAL(5,2),
    memory_usage_percent DECIMAL(5,2),
    disk_usage_percent DECIMAL(5,2),
    load_average DECIMAL[],
    network_in_bytes BIGINT,
    network_out_bytes BIGINT,
    uptime_seconds BIGINT,
    active_deployments INTEGER,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- DEPLOYMENTS
-- =====================================================

-- Deployment environments
CREATE TYPE deployment_environment AS ENUM ('production', 'staging', 'development', 'preview');

-- Deployment status
CREATE TYPE deployment_status AS ENUM ('pending', 'building', 'deploying', 'running', 'stopped', 'failed', 'terminated');

-- Deployments
CREATE TABLE deployments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    application_id UUID REFERENCES applications(id),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    server_id UUID REFERENCES servers(id),
    name VARCHAR(255) NOT NULL,
    subdomain VARCHAR(100) NOT NULL,
    domain VARCHAR(255),
    status deployment_status NOT NULL DEFAULT 'pending',
    environment deployment_environment NOT NULL DEFAULT 'production',
    version VARCHAR(50),
    container_id VARCHAR(100),
    image_name VARCHAR(255),
    port INTEGER DEFAULT 3000,
    cpu_limit DECIMAL(10,2) DEFAULT 1.0,
    memory_limit INTEGER DEFAULT 512, -- MB
    disk_limit INTEGER DEFAULT 1024, -- MB
    auto_scale BOOLEAN DEFAULT FALSE,
    min_instances INTEGER DEFAULT 1,
    max_instances INTEGER DEFAULT 1,
    environment_variables JSONB DEFAULT '{}',
    custom_domains JSONB DEFAULT '[]',
    ssl_enabled BOOLEAN DEFAULT TRUE,
    health_check_url VARCHAR(255),
    health_check_interval INTEGER DEFAULT 30, -- seconds
    build_logs JSONB DEFAULT '[]',
    deployment_logs JSONB DEFAULT '[]',
    error_message TEXT,
    last_deployed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(subdomain)
);

-- Deployment metrics
CREATE TABLE deployment_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    deployment_id UUID REFERENCES deployments(id) ON DELETE CASCADE,
    timestamp TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    cpu_usage_percent DECIMAL(5,2),
    memory_usage_mb INTEGER,
    disk_usage_mb INTEGER,
    network_in_bytes BIGINT,
    network_out_bytes BIGINT,
    requests_per_minute INTEGER,
    response_time_avg_ms DECIMAL(10,2),
    error_rate_percent DECIMAL(5,2),
    status VARCHAR(20),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- DOMAINS & SSL
-- =====================================================

-- Domain status
CREATE TYPE domain_status AS ENUM ('pending', 'verified', 'failed', 'expired');
CREATE TYPE ssl_status AS ENUM ('none', 'pending', 'active', 'expired', 'failed');

-- Domains
CREATE TABLE domains (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    domain VARCHAR(255) NOT NULL UNIQUE,
    status domain_status NOT NULL DEFAULT 'pending',
    ssl_status ssl_status NOT NULL DEFAULT 'none',
    ssl_expires_at TIMESTAMPTZ,
    dns_configured BOOLEAN DEFAULT FALSE,
    auto_ssl BOOLEAN DEFAULT TRUE,
    verification_token VARCHAR(100),
    verification_method VARCHAR(20) DEFAULT 'dns',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- SSL Certificates
CREATE TABLE ssl_certificates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    domain_id UUID REFERENCES domains(id) ON DELETE CASCADE,
    certificate TEXT NOT NULL,
    private_key TEXT NOT NULL,
    chain TEXT,
    issued_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    auto_renew BOOLEAN DEFAULT TRUE,
    provider VARCHAR(50) DEFAULT 'letsencrypt',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- BILLING & PAYMENTS
-- =====================================================

-- Invoice status
CREATE TYPE invoice_status AS ENUM ('draft', 'sent', 'paid', 'overdue', 'cancelled', 'refunded');

-- Invoices
CREATE TABLE invoices (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    invoice_number VARCHAR(50) NOT NULL UNIQUE,
    status invoice_status NOT NULL DEFAULT 'draft',
    subtotal DECIMAL(10,2) NOT NULL DEFAULT 0,
    tax_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    discount_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    total_amount DECIMAL(10,2) NOT NULL DEFAULT 0,
    currency VARCHAR(3) NOT NULL DEFAULT 'USD',
    due_date DATE NOT NULL,
    paid_at TIMESTAMPTZ,
    stripe_invoice_id VARCHAR(100),
    notes TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Invoice line items
CREATE TABLE invoice_line_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    invoice_id UUID REFERENCES invoices(id) ON DELETE CASCADE,
    description TEXT NOT NULL,
    quantity DECIMAL(10,2) NOT NULL DEFAULT 1,
    unit_price DECIMAL(10,2) NOT NULL,
    total_price DECIMAL(10,2) NOT NULL,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Payment methods
CREATE TABLE payment_methods (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    stripe_payment_method_id VARCHAR(100) NOT NULL,
    type VARCHAR(20) NOT NULL, -- card, bank_account, etc.
    is_default BOOLEAN DEFAULT FALSE,
    card_brand VARCHAR(20),
    card_last4 VARCHAR(4),
    card_exp_month INTEGER,
    card_exp_year INTEGER,
    bank_name VARCHAR(100),
    bank_last4 VARCHAR(4),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- ANALYTICS & REPORTING
-- =====================================================

-- Analytics events
CREATE TABLE analytics_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID REFERENCES customers(id),
    deployment_id UUID REFERENCES deployments(id),
    event_type VARCHAR(50) NOT NULL,
    event_name VARCHAR(100) NOT NULL,
    properties JSONB DEFAULT '{}',
    user_agent TEXT,
    ip_address INET,
    country VARCHAR(2),
    city VARCHAR(100),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Daily analytics aggregations
CREATE TABLE analytics_daily (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    date DATE NOT NULL,
    customer_id UUID REFERENCES customers(id),
    deployment_id UUID REFERENCES deployments(id),
    page_views INTEGER DEFAULT 0,
    unique_visitors INTEGER DEFAULT 0,
    requests INTEGER DEFAULT 0,
    bandwidth_bytes BIGINT DEFAULT 0,
    cpu_hours DECIMAL(10,2) DEFAULT 0,
    memory_hours DECIMAL(10,2) DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(date, customer_id, deployment_id)
);

-- =====================================================
-- SECURITY & AUDIT
-- =====================================================

-- Audit log
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES admin_users(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50),
    resource_id VARCHAR(100),
    old_values JSONB,
    new_values JSONB,
    ip_address INET,
    user_agent TEXT,
    success BOOLEAN DEFAULT TRUE,
    error_message TEXT,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Security incidents
CREATE TABLE security_incidents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    severity VARCHAR(20) NOT NULL CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    status VARCHAR(20) NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'investigating', 'resolved', 'closed')),
    assigned_to UUID REFERENCES admin_users(id),
    customer_id UUID REFERENCES customers(id),
    deployment_id UUID REFERENCES deployments(id),
    server_id UUID REFERENCES servers(id),
    source_ip INET,
    metadata JSONB DEFAULT '{}',
    resolved_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- API keys for customers
CREATE TABLE api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    key_hash VARCHAR(255) NOT NULL UNIQUE,
    scopes JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT TRUE,
    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- SETTINGS & CONFIGURATION
-- =====================================================

-- Platform settings
CREATE TABLE platform_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    key VARCHAR(100) NOT NULL UNIQUE,
    value JSONB NOT NULL,
    description TEXT,
    is_public BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Insert default settings
INSERT INTO platform_settings (key, value, description, is_public) VALUES
('platform_name', '"SuperAgent PaaS"', 'Platform display name', true),
('platform_description', '"Enterprise deployment platform"', 'Platform description', true),
('max_deployment_size_mb', '512', 'Maximum deployment size in MB', false),
('default_deployment_timeout_minutes', '30', 'Default deployment timeout', false),
('enable_auto_ssl', 'true', 'Enable automatic SSL certificates', false),
('enable_metrics_collection', 'true', 'Enable metrics collection', false),
('maintenance_mode', 'false', 'Platform maintenance mode', true),
('signup_enabled', 'true', 'Allow new customer signups', true);

-- Integration configurations
CREATE TABLE integrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    config JSONB NOT NULL DEFAULT '{}',
    is_enabled BOOLEAN DEFAULT FALSE,
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- NOTIFICATIONS & WEBHOOKS
-- =====================================================

-- Notification channels
CREATE TABLE notification_channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID REFERENCES customers(id) ON DELETE CASCADE,
    type VARCHAR(20) NOT NULL CHECK (type IN ('email', 'webhook', 'slack', 'discord')),
    name VARCHAR(100) NOT NULL,
    config JSONB NOT NULL,
    is_enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification templates
CREATE TABLE notification_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    type VARCHAR(50) NOT NULL,
    subject VARCHAR(255),
    content TEXT NOT NULL,
    variables JSONB DEFAULT '[]',
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Notification queue
CREATE TABLE notification_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    customer_id UUID REFERENCES customers(id),
    channel_id UUID REFERENCES notification_channels(id),
    template_id UUID REFERENCES notification_templates(id),
    recipient VARCHAR(255) NOT NULL,
    subject VARCHAR(255),
    content TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'sent', 'failed', 'cancelled')),
    attempts INTEGER DEFAULT 0,
    max_attempts INTEGER DEFAULT 3,
    error_message TEXT,
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- =====================================================
-- INDEXES FOR PERFORMANCE
-- =====================================================

-- Customer indexes
CREATE INDEX idx_customers_email ON customers(email);
CREATE INDEX idx_customers_plan_id ON customers(plan_id);
CREATE INDEX idx_customers_status ON customers(status);
CREATE INDEX idx_customers_created_at ON customers(created_at);

-- Application indexes
CREATE INDEX idx_applications_slug ON applications(slug);
CREATE INDEX idx_applications_publisher_id ON applications(publisher_id);
CREATE INDEX idx_applications_category_id ON applications(category_id);
CREATE INDEX idx_applications_status ON applications(status);
CREATE INDEX idx_applications_created_at ON applications(created_at);

-- Deployment indexes
CREATE INDEX idx_deployments_customer_id ON deployments(customer_id);
CREATE INDEX idx_deployments_application_id ON deployments(application_id);
CREATE INDEX idx_deployments_server_id ON deployments(server_id);
CREATE INDEX idx_deployments_status ON deployments(status);
CREATE INDEX idx_deployments_subdomain ON deployments(subdomain);
CREATE INDEX idx_deployments_created_at ON deployments(created_at);

-- Server indexes
CREATE INDEX idx_servers_status ON servers(status);
CREATE INDEX idx_servers_provider ON servers(provider);
CREATE INDEX idx_servers_last_heartbeat ON servers(last_heartbeat);

-- Server tokens indexes
CREATE INDEX idx_server_tokens_server_id ON server_tokens(server_id);
CREATE INDEX idx_server_tokens_hash ON server_tokens(token_hash);
CREATE INDEX idx_server_tokens_active ON server_tokens(is_active);
CREATE INDEX idx_server_tokens_expires ON server_tokens(expires_at);
CREATE INDEX idx_server_tokens_last_used ON server_tokens(last_used_at DESC);

-- Domain indexes
CREATE INDEX idx_domains_customer_id ON domains(customer_id);
CREATE INDEX idx_domains_domain ON domains(domain);
CREATE INDEX idx_domains_status ON domains(status);

-- Metrics indexes (for time series queries)
CREATE INDEX idx_server_metrics_server_timestamp ON server_metrics(server_id, timestamp DESC);
CREATE INDEX idx_deployment_metrics_deployment_timestamp ON deployment_metrics(deployment_id, timestamp DESC);
CREATE INDEX idx_analytics_events_customer_created ON analytics_events(customer_id, created_at DESC);
CREATE INDEX idx_analytics_daily_date_customer ON analytics_daily(date DESC, customer_id);

-- Audit log indexes
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);

-- =====================================================
-- ROW LEVEL SECURITY (RLS) POLICIES
-- =====================================================

-- Enable RLS on all tables
ALTER TABLE admin_users ENABLE ROW LEVEL SECURITY;
ALTER TABLE customers ENABLE ROW LEVEL SECURITY;
ALTER TABLE customer_usage ENABLE ROW LEVEL SECURITY;
ALTER TABLE applications ENABLE ROW LEVEL SECURITY;
ALTER TABLE deployments ENABLE ROW LEVEL SECURITY;
ALTER TABLE servers ENABLE ROW LEVEL SECURITY;
ALTER TABLE domains ENABLE ROW LEVEL SECURITY;
ALTER TABLE invoices ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY;

-- Admin users can see everything
CREATE POLICY "Admin users can view all data" ON customers FOR ALL 
TO authenticated USING (
    EXISTS (
        SELECT 1 FROM admin_users 
        WHERE auth_user_id = auth.uid() 
        AND is_active = true
    )
);

-- Similar policies for other tables...
CREATE POLICY "Admin users can manage deployments" ON deployments FOR ALL 
TO authenticated USING (
    EXISTS (
        SELECT 1 FROM admin_users 
        WHERE auth_user_id = auth.uid() 
        AND is_active = true
    )
);

-- =====================================================
-- FUNCTIONS & TRIGGERS
-- =====================================================

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Add updated_at triggers to relevant tables
CREATE TRIGGER update_customers_updated_at BEFORE UPDATE ON customers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_applications_updated_at BEFORE UPDATE ON applications FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_deployments_updated_at BEFORE UPDATE ON deployments FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_servers_updated_at BEFORE UPDATE ON servers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_server_tokens_updated_at BEFORE UPDATE ON server_tokens FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_domains_updated_at BEFORE UPDATE ON domains FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to calculate invoice totals
CREATE OR REPLACE FUNCTION calculate_invoice_total()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE invoices 
    SET total_amount = subtotal + tax_amount - discount_amount
    WHERE id = COALESCE(NEW.invoice_id, OLD.invoice_id);
    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Trigger to auto-calculate invoice totals
CREATE TRIGGER update_invoice_totals 
AFTER INSERT OR UPDATE OR DELETE ON invoice_line_items 
FOR EACH ROW EXECUTE FUNCTION calculate_invoice_total();

-- =====================================================
-- INITIAL DATA SETUP
-- =====================================================

-- Create default super admin (you'll need to update this with real user ID)
-- INSERT INTO admin_users (auth_user_id, email, name, role) VALUES 
-- ('YOUR_AUTH_USER_ID', 'admin@superagent.dev', 'Super Admin', 'super_admin');

-- Create sample data for development
INSERT INTO publishers (name, email, company, is_verified) VALUES
('SuperAgent Team', 'publisher@superagent.dev', 'SuperAgent Inc.', true),
('Community', 'community@superagent.dev', 'Community Contributors', true);

-- Sample applications
INSERT INTO applications (name, slug, description, publisher_id, category_id, version, status, source_type, source_url, port) VALUES
('Hello World App', 'hello-world', 'Simple Node.js hello world application', 
 (SELECT id FROM publishers WHERE email = 'publisher@superagent.dev'), 
 (SELECT id FROM application_categories WHERE slug = 'web-apps'), 
 '1.0.0', 'published', 'git', 'https://github.com/superagent/hello-world', 3000),
('React Starter', 'react-starter', 'Production-ready React application template', 
 (SELECT id FROM publishers WHERE email = 'publisher@superagent.dev'), 
 (SELECT id FROM application_categories WHERE slug = 'web-apps'), 
 '2.1.0', 'published', 'git', 'https://github.com/superagent/react-starter', 3000);

-- =====================================================
-- VIEWS FOR ANALYTICS
-- =====================================================

-- Customer overview view
CREATE VIEW customer_overview AS
SELECT 
    c.id,
    c.name,
    c.email,
    c.company,
    cp.name as plan_name,
    c.status,
    c.created_at,
    COUNT(d.id) as total_deployments,
    COUNT(CASE WHEN d.status = 'running' THEN 1 END) as active_deployments,
    COUNT(dom.id) as total_domains,
    COALESCE(SUM(i.total_amount), 0) as total_revenue
FROM customers c
LEFT JOIN customer_plans cp ON c.plan_id = cp.id
LEFT JOIN deployments d ON c.id = d.customer_id
LEFT JOIN domains dom ON c.id = dom.customer_id
LEFT JOIN invoices i ON c.id = i.customer_id AND i.status = 'paid'
GROUP BY c.id, c.name, c.email, c.company, cp.name, c.status, c.created_at;

-- Server utilization view
CREATE VIEW server_utilization AS
SELECT 
    s.id,
    s.name,
    s.hostname,
    s.status,
    s.current_deployments,
    s.max_deployments,
    ROUND((s.current_deployments::decimal / s.max_deployments * 100), 2) as utilization_percent,
    s.health_score,
    s.last_heartbeat
FROM servers s;

-- Revenue analytics view
CREATE VIEW revenue_analytics AS
SELECT 
    DATE_TRUNC('month', i.created_at) as month,
    COUNT(i.id) as invoice_count,
    SUM(i.total_amount) as total_revenue,
    AVG(i.total_amount) as avg_invoice_amount,
    COUNT(DISTINCT i.customer_id) as unique_customers
FROM invoices i
WHERE i.status = 'paid'
GROUP BY DATE_TRUNC('month', i.created_at)
ORDER BY month DESC;

-- =====================================================
-- COMMENTS FOR DOCUMENTATION
-- =====================================================

COMMENT ON TABLE customers IS 'Main customer/user accounts table';
COMMENT ON TABLE customer_plans IS 'Subscription plans and pricing tiers';
COMMENT ON TABLE applications IS 'Application catalog - all deployable apps';
COMMENT ON TABLE deployments IS 'Active and historical deployments';
COMMENT ON TABLE servers IS 'Server/agent cluster management';
COMMENT ON TABLE domains IS 'Custom domain management';
COMMENT ON TABLE invoices IS 'Billing and invoice management';
COMMENT ON TABLE audit_logs IS 'Complete audit trail for compliance';
COMMENT ON TABLE analytics_events IS 'Raw analytics event data';
COMMENT ON TABLE platform_settings IS 'Global platform configuration';

-- =====================================================
-- END OF SCHEMA
-- =====================================================