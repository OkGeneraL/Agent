// Core User Types
export interface User {
  id: string
  email: string
  name: string
  role: 'admin' | 'super_admin' | 'support'
  avatar_url?: string
  created_at: string
  updated_at: string
  last_login?: string
  is_active: boolean
}

// Customer Types
export interface Customer {
  id: string
  name: string
  email: string
  company?: string
  plan: 'free' | 'starter' | 'pro' | 'enterprise'
  status: 'active' | 'suspended' | 'cancelled'
  created_at: string
  updated_at: string
  billing_email?: string
  phone?: string
  address?: string
  quota: CustomerQuota
  usage: CustomerUsage
}

export interface CustomerQuota {
  max_applications: number
  max_deployments: number
  max_domains: number
  max_storage_gb: number
  max_bandwidth_gb: number
  max_cpu_cores: number
  max_memory_gb: number
}

export interface CustomerUsage {
  applications: number
  deployments: number
  domains: number
  storage_gb: number
  bandwidth_gb: number
  cpu_hours: number
  memory_hours: number
}

// Application Types
export interface Application {
  id: string
  name: string
  description: string
  category: string
  publisher_id: string
  publisher_name: string
  version: string
  status: 'pending' | 'approved' | 'rejected' | 'published'
  source_type: 'git' | 'docker' | 'zip'
  source_url: string
  dockerfile_path?: string
  build_command?: string
  start_command?: string
  port: number
  environment_variables: Record<string, string>
  pricing: ApplicationPricing
  created_at: string
  updated_at: string
  approval_notes?: string
  download_count: number
  rating: number
  screenshots: string[]
  logo_url?: string
}

export interface ApplicationPricing {
  type: 'free' | 'one_time' | 'subscription'
  price?: number
  currency?: string
  billing_period?: 'monthly' | 'yearly'
}

// Deployment Types
export interface Deployment {
  id: string
  application_id: string
  application_name: string
  customer_id: string
  customer_name: string
  server_id: string
  server_name: string
  domain?: string
  subdomain: string
  status: 'pending' | 'building' | 'deploying' | 'running' | 'stopped' | 'failed' | 'terminated'
  environment: 'production' | 'staging' | 'development'
  build_logs: string[]
  runtime_logs: string[]
  created_at: string
  updated_at: string
  last_deployed_at?: string
  health_status: 'healthy' | 'unhealthy' | 'unknown'
  metrics: DeploymentMetrics
  config: DeploymentConfig
}

export interface DeploymentMetrics {
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  network_in: number
  network_out: number
  requests_per_minute: number
  response_time_avg: number
  error_rate: number
}

export interface DeploymentConfig {
  cpu_limit: number
  memory_limit: number
  disk_limit: number
  auto_scale: boolean
  min_instances: number
  max_instances: number
  environment_variables: Record<string, string>
  custom_domains: string[]
  ssl_enabled: boolean
}

// Server Types
export interface Server {
  id: string
  name: string
  hostname: string
  ip_address: string
  location: string
  provider: 'aws' | 'gcp' | 'azure' | 'digitalocean' | 'linode' | 'vultr' | 'dedicated'
  status: 'online' | 'offline' | 'maintenance' | 'error'
  agent_version: string
  specifications: ServerSpecs
  metrics: ServerMetrics
  capacity: ServerCapacity
  created_at: string
  updated_at: string
  last_heartbeat: string
}

export interface ServerSpecs {
  cpu_cores: number
  memory_gb: number
  disk_gb: number
  bandwidth_gbps: number
  architecture: 'x86_64' | 'arm64'
  os: string
  os_version: string
}

export interface ServerMetrics {
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  load_average: number[]
  network_in: number
  network_out: number
  uptime: number
}

export interface ServerCapacity {
  max_deployments: number
  current_deployments: number
  available_cpu: number
  available_memory: number
  available_disk: number
}

// Domain Types
export interface Domain {
  id: string
  domain: string
  customer_id: string
  customer_name: string
  status: 'pending' | 'verified' | 'failed'
  ssl_status: 'none' | 'pending' | 'active' | 'expired' | 'failed'
  ssl_expires_at?: string
  dns_configured: boolean
  auto_ssl: boolean
  created_at: string
  updated_at: string
  verification_token?: string
  ssl_certificate?: SSLCertificate
}

export interface SSLCertificate {
  id: string
  domain: string
  certificate: string
  private_key: string
  chain: string
  issued_at: string
  expires_at: string
  auto_renew: boolean
}

// Analytics Types
export interface AnalyticsData {
  total_customers: number
  total_applications: number
  total_deployments: number
  total_servers: number
  revenue_current_month: number
  revenue_previous_month: number
  active_deployments: number
  server_usage: number
  customer_growth: AnalyticsPoint[]
  revenue_trend: AnalyticsPoint[]
  deployment_stats: AnalyticsPoint[]
  top_applications: ApplicationStat[]
  server_performance: ServerStat[]
}

export interface AnalyticsPoint {
  date: string
  value: number
}

export interface ApplicationStat {
  id: string
  name: string
  deployments: number
  revenue: number
}

export interface ServerStat {
  id: string
  name: string
  cpu_usage: number
  memory_usage: number
  deployments: number
}

// Billing Types
export interface Invoice {
  id: string
  customer_id: string
  customer_name: string
  amount: number
  currency: string
  status: 'draft' | 'sent' | 'paid' | 'overdue' | 'cancelled'
  due_date: string
  paid_at?: string
  created_at: string
  line_items: InvoiceLineItem[]
}

export interface InvoiceLineItem {
  description: string
  quantity: number
  unit_price: number
  total: number
}

// Common Types
export interface PaginationInfo {
  page: number
  per_page: number
  total: number
  total_pages: number
}

export interface ApiResponse<T> {
  data: T
  pagination?: PaginationInfo
  message?: string
  success: boolean
}

export interface SearchFilters {
  query?: string
  status?: string
  category?: string
  date_from?: string
  date_to?: string
  page?: number
  per_page?: number
  sort_by?: string
  sort_order?: 'asc' | 'desc'
}

// Navigation Types
export interface NavItem {
  title: string
  href: string
  icon: string
  description?: string
  children?: NavItem[]
  badge?: string
  disabled?: boolean
}