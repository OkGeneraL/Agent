import { NextRequest, NextResponse } from 'next/server'
import { agentManager } from '@/lib/agents'

// Types for SuperAgent API responses
interface SuperAgentDeployment {
  id: string
  status: string
  message: string
  app_id: string
  version: string
  container_id?: string
  created_at: string
  metadata: Record<string, unknown>
}

interface EnrichedDeployment extends SuperAgentDeployment {
  application_name: string
  customer_name: string
  server_name: string
  subdomain: string
  domain: string
  environment: string
  cpu_usage: number
  memory_usage: number
  disk_usage: number
  requests_per_minute: number
  response_time_avg: number
  error_rate: number
  uptime: string
  health_status: string
}

// No longer needed - using AgentManager for multi-server support

// GET /api/deployments - List all deployments
export async function GET(request: NextRequest) {
  try {
    const { searchParams } = new URL(request.url)
    const status = searchParams.get('status')
    const server_id = searchParams.get('server_id')
    const customer_id = searchParams.get('customer_id')

    // Fetch deployments from all SuperAgent servers
    const agentData = await agentManager.getAggregatedData()
    const deployments: SuperAgentDeployment[] = agentData.flatMap(agent => {
      // Safely convert unknown records to deployment format
      const agentDeployments = agent.deployments as Record<string, unknown>[]
      return agentDeployments.map(dep => ({
        id: String(dep.id || ''),
        status: String(dep.status || 'unknown'),
        message: String(dep.message || ''),
        app_id: String(dep.app_id || ''),
        version: String(dep.version || ''),
        container_id: dep.container_id ? String(dep.container_id) : undefined,
        created_at: String(dep.created_at || new Date().toISOString()),
        metadata: (dep.metadata as Record<string, unknown>) || {}
      }))
    })
    
    // Fetch additional data from database (customers, applications, servers)
    // In a real implementation, you would fetch this from Supabase
    const enrichedDeployments: EnrichedDeployment[] = deployments.map((deployment: SuperAgentDeployment): EnrichedDeployment => ({
      ...deployment,
      // Mock enriched data - replace with actual database queries
      application_name: getApplicationName(deployment.app_id),
      customer_name: getCustomerName(String(deployment.metadata.customer_id || '')),
      server_name: getServerName(String(deployment.metadata.server_id || '')),
      subdomain: String(deployment.metadata.subdomain || deployment.app_id),
      domain: String(deployment.metadata.domain || ''),
      environment: String(deployment.metadata.environment || 'production'),
      cpu_usage: Math.random() * 100,
      memory_usage: Math.random() * 1000,
      disk_usage: Math.random() * 2000,
      requests_per_minute: Math.floor(Math.random() * 1000),
      response_time_avg: Math.floor(Math.random() * 200),
      error_rate: Math.random() * 5,
      uptime: `${(Math.random() * 10 + 90).toFixed(2)}%`,
      health_status: deployment.status === 'running' ? 'healthy' : 'unknown',
    }))

    // Apply filters
    let filteredDeployments: EnrichedDeployment[] = enrichedDeployments
    if (status) {
      filteredDeployments = filteredDeployments.filter((d: EnrichedDeployment) => d.status === status)
    }
    if (server_id) {
      filteredDeployments = filteredDeployments.filter((d: EnrichedDeployment) => String(d.metadata.server_id) === server_id)
    }
    if (customer_id) {
      filteredDeployments = filteredDeployments.filter((d: EnrichedDeployment) => String(d.metadata.customer_id) === customer_id)
    }

    return NextResponse.json({
      deployments: filteredDeployments,
      total: filteredDeployments.length,
      status: 'success'
    })
  } catch (error) {
    console.error('Error fetching deployments:', error)
    return NextResponse.json(
      { error: 'Failed to fetch deployments', details: error instanceof Error ? error.message : 'Unknown error' },
      { status: 500 }
    )
  }
}

// POST /api/deployments - Create new deployment
export async function POST(request: NextRequest) {
  try {
    const body = await request.json()
    
    // Validate required fields
    const { app_id, version, customer_id, server_id, environment = 'production' } = body
    
    if (!app_id || !version || !customer_id) {
      return NextResponse.json(
        { error: 'Missing required fields: app_id, version, customer_id' },
        { status: 400 }
      )
    }

    // Prepare deployment request for SuperAgent
    const deploymentRequest = {
      app_id,
      version,
      source_type: body.source_type || 'git',
      source: body.source_url,
      environment_variables: body.environment_variables || {},
      port: body.port || 3000,
      cpu_limit: body.cpu_limit || 1.0,
      memory_limit: body.memory_limit || 512,
      metadata: {
        customer_id,
        server_id,
        environment,
        subdomain: body.subdomain,
        domain: body.domain,
        created_by: 'admin-panel'
      }
    }

    // Create deployment via specific SuperAgent server
    if (!server_id) {
      return NextResponse.json(
        { error: 'server_id is required to create deployment' },
        { status: 400 }
      )
    }

    const deployment = await agentManager.createDeployment(server_id, deploymentRequest)

    // Save deployment info to database
    // In a real implementation, you would save to Supabase here
    
    return NextResponse.json({
      deployment: {
        ...deployment,
        application_name: getApplicationName(app_id),
        customer_name: getCustomerName(customer_id),
        server_name: getServerName(server_id),
      },
      status: 'success',
      message: 'Deployment created successfully'
    }, { status: 201 })
  } catch (error) {
    console.error('Error creating deployment:', error)
    return NextResponse.json(
      { error: 'Failed to create deployment', details: error instanceof Error ? error.message : 'Unknown error' },
      { status: 500 }
    )
  }
}

// Mock helper functions (replace with actual database queries)
function getApplicationName(appId: string): string {
  const apps: Record<string, string> = {
    'react-starter': 'React Starter',
    'nodejs-api': 'Node.js API Server',
    'wordpress': 'WordPress CMS',
    'redis-cache': 'Redis Cache',
    'analytics-dashboard': 'Analytics Dashboard'
  }
  return apps[appId] || appId
}

function getCustomerName(customerId: string): string {
  const customers: Record<string, string> = {
    'cust-1': 'TechCorp Inc.',
    'cust-2': 'StartupXYZ',
    'cust-3': 'BlogCorp',
    'cust-4': 'Enterprise Corp',
    'cust-5': 'DataCorp'
  }
  return customers[customerId] || 'Unknown Customer'
}

function getServerName(serverId: string): string {
  const servers: Record<string, string> = {
    'srv-1': 'us-east-1-prod',
    'srv-2': 'us-west-2-staging',
    'srv-3': 'eu-central-1-prod',
    'srv-4': 'us-west-2-dev',
    'srv-5': 'ap-southeast-1-prod'
  }
  return servers[serverId] || 'Unknown Server'
}