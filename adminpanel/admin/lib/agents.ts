import { createClient } from '@supabase/supabase-js'

const supabase = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL!,
  process.env.SUPABASE_SERVICE_ROLE_KEY!
)

export interface AgentServer {
  id: string
  name: string
  hostname: string
  ip_address: string
  api_endpoint: string
  api_token_hash: string
  status: 'online' | 'offline' | 'maintenance' | 'error'
  location: string
  provider: string
  created_at: string
  updated_at: string
}

export interface AgentCredentials {
  endpoint: string
  token: string
}

export class AgentManager {
  private agentCredentials: Map<string, AgentCredentials> = new Map()

  constructor() {
    this.loadAgentCredentials()
  }

  /**
   * Load all agent credentials from database
   */
  private async loadAgentCredentials() {
    try {
      const { data: servers, error } = await supabase
        .from('servers')
        .select('id, api_endpoint, api_token_hash')
        .eq('status', 'online')

      if (error) throw error

      for (const server of servers || []) {
        this.agentCredentials.set(server.id, {
          endpoint: server.api_endpoint,
          token: server.api_token_hash // In production, decrypt this
        })
      }
    } catch (error) {
      console.error('Failed to load agent credentials:', error)
    }
  }

  /**
   * Get all registered SuperAgent servers
   */
  async getAgentServers(): Promise<AgentServer[]> {
    const { data: servers, error } = await supabase
      .from('servers')
      .select('*')
      .order('created_at', { ascending: false })

    if (error) {
      console.error('Failed to fetch agent servers:', error)
      return []
    }

    return servers || []
  }

  /**
   * Add a new SuperAgent server
   */
  async addAgentServer(server: Omit<AgentServer, 'id' | 'created_at' | 'updated_at'>): Promise<AgentServer | null> {
    // Test connectivity first
    const isHealthy = await this.testAgentConnection(server.api_endpoint, server.api_token_hash)
    
    const { data, error } = await supabase
      .from('servers')
      .insert([{
        ...server,
        status: isHealthy ? 'online' : 'error'
      }])
      .select()
      .single()

    if (error) {
      console.error('Failed to add agent server:', error)
      return null
    }

    // Update in-memory credentials
    this.agentCredentials.set(data.id, {
      endpoint: server.api_endpoint,
      token: server.api_token_hash
    })

    return data
  }

  /**
   * Update agent server credentials
   */
  async updateAgentCredentials(serverId: string, endpoint: string, token: string): Promise<boolean> {
    // Test new credentials first
    const isHealthy = await this.testAgentConnection(endpoint, token)
    
    const { error } = await supabase
      .from('servers')
      .update({
        api_endpoint: endpoint,
        api_token_hash: token,
        status: isHealthy ? 'online' : 'error',
        updated_at: new Date().toISOString()
      })
      .eq('id', serverId)

    if (error) {
      console.error('Failed to update agent credentials:', error)
      return false
    }

    // Update in-memory credentials
    this.agentCredentials.set(serverId, { endpoint, token })
    return true
  }

  /**
   * Remove an agent server
   */
  async removeAgentServer(serverId: string): Promise<boolean> {
    const { error } = await supabase
      .from('servers')
      .delete()
      .eq('id', serverId)

    if (error) {
      console.error('Failed to remove agent server:', error)
      return false
    }

    this.agentCredentials.delete(serverId)
    return true
  }

  /**
   * Test agent connection and health
   */
  async testAgentConnection(endpoint: string, token?: string): Promise<boolean> {
    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 5000)

      const response = await fetch(`${endpoint}/health`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          ...(token && { 'Authorization': `Bearer ${token}` })
        },
        signal: controller.signal
      })

      clearTimeout(timeoutId)
      return response.ok
    } catch (error) {
      console.error('Agent connection test failed:', error)
      return false
    }
  }

  /**
   * Make authenticated request to specific agent
   */
  async makeAgentRequest(serverId: string, path: string, options: RequestInit = {}): Promise<Response> {
    const credentials = this.agentCredentials.get(serverId)
    if (!credentials) {
      throw new Error(`No credentials found for server ${serverId}`)
    }

    const url = `${credentials.endpoint}${path}`
    const headers = {
      'Content-Type': 'application/json',
      ...(credentials.token && { 'Authorization': `Bearer ${credentials.token}` }),
      ...options.headers
    }

    const response = await fetch(url, {
      ...options,
      headers
    })

    if (!response.ok) {
      throw new Error(`Agent request failed: ${response.status} ${response.statusText}`)
    }

    return response
  }

  /**
   * Get deployments from specific agent
   */
  async getAgentDeployments(serverId: string) {
    const response = await this.makeAgentRequest(serverId, '/api/v1/deployments')
    return response.json()
  }

  /**
   * Get agent status and metrics
   */
  async getAgentStatus(serverId: string) {
    const response = await this.makeAgentRequest(serverId, '/api/v1/status')
    return response.json()
  }

  /**
   * Create deployment on specific agent
   */
  async createDeployment(serverId: string, deploymentData: Record<string, unknown>) {
    const response = await this.makeAgentRequest(serverId, '/api/v1/deployments', {
      method: 'POST',
      body: JSON.stringify(deploymentData)
    })
    return response.json()
  }

  /**
   * Control deployment on specific agent
   */
  async controlDeployment(serverId: string, deploymentId: string, action: 'start' | 'stop' | 'restart' | 'rollback') {
    const response = await this.makeAgentRequest(serverId, `/api/v1/deployments/${deploymentId}/${action}`, {
      method: 'POST'
    })
    return response.json()
  }

  /**
   * Get deployment logs from specific agent
   */
  async getDeploymentLogs(serverId: string, deploymentId: string, tail = 100) {
    const response = await this.makeAgentRequest(serverId, `/api/v1/deployments/${deploymentId}/logs?tail=${tail}`)
    return response.json()
  }

  /**
   * Health check all agents and update status
   */
  async healthCheckAllAgents(): Promise<void> {
    const servers = await this.getAgentServers()
    
    const healthPromises = servers.map(async (server) => {
      const isHealthy = await this.testAgentConnection(server.api_endpoint, server.api_token_hash)
      
      // Update status in database
      await supabase
        .from('servers')
        .update({
          status: isHealthy ? 'online' : 'offline',
          last_heartbeat: new Date().toISOString()
        })
        .eq('id', server.id)

      return { serverId: server.id, healthy: isHealthy }
    })

    const results = await Promise.allSettled(healthPromises)
    console.log('Health check results:', results)
  }

  /**
   * Get aggregated data from all agents
   */
  async getAggregatedData() {
    const servers = await this.getAgentServers()
    const onlineServers = servers.filter(s => s.status === 'online')

    type AgentData = {
      serverId: string
      serverName: string
      location: string
      status: Record<string, unknown>
      deployments: Record<string, unknown>[]
    }

    const dataPromises = onlineServers.map(async (server): Promise<AgentData | null> => {
      try {
        const [status, deployments] = await Promise.all([
          this.getAgentStatus(server.id),
          this.getAgentDeployments(server.id)
        ])

        return {
          serverId: server.id,
          serverName: server.name,
          location: server.location,
          status,
          deployments
        }
      } catch (error) {
        console.error(`Failed to get data from agent ${server.id}:`, error)
        return null
      }
    })

    const results = await Promise.allSettled(dataPromises)
    return results
      .filter((result): result is PromiseFulfilledResult<AgentData> => 
        result.status === 'fulfilled' && result.value !== null)
      .map(result => result.value)
  }
}

// Singleton instance
export const agentManager = new AgentManager()

export default agentManager