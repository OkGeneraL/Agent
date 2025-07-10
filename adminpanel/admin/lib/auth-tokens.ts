import { createClient } from '@supabase/supabase-js'
import crypto from 'crypto'

const supabase = createClient(
  process.env.NEXT_PUBLIC_SUPABASE_URL!,
  process.env.SUPABASE_SERVICE_ROLE_KEY!
)

export interface ServerToken {
  id: string
  server_id: string
  token_hash: string
  token_prefix: string // First 8 chars for identification
  expires_at: string
  created_at: string
  is_active: boolean
  last_used_at?: string
}

export interface TokenValidationResult {
  valid: boolean
  server_id?: string
  expires_at?: string
  error?: string
}

export class AuthTokenManager {
  
  /**
   * Generate a cryptographically secure API token
   */
  generateSecureToken(): string {
    // Generate 32 bytes of random data and encode as base64
    const randomBytes = crypto.randomBytes(32)
    const token = 'sa_' + randomBytes.toString('base64url') // sa_ prefix for SuperAgent tokens
    return token
  }

  /**
   * Hash token for secure storage (using SHA-256)
   */
  private hashToken(token: string): string {
    return crypto.createHash('sha256').update(token).digest('hex')
  }

  /**
   * Get token prefix for identification (first 8 chars after prefix)
   */
  private getTokenPrefix(token: string): string {
    return token.substring(0, 12) // sa_ + first 8 chars
  }

  /**
   * Create a new token for a server
   */
  async createServerToken(serverId: string, expiresInDays: number = 365): Promise<{ token: string; tokenData: ServerToken }> {
    const token = this.generateSecureToken()
    const tokenHash = this.hashToken(token)
    const tokenPrefix = this.getTokenPrefix(token)
    const expiresAt = new Date()
    expiresAt.setDate(expiresAt.getDate() + expiresInDays)

    const { data, error } = await supabase
      .from('server_tokens')
      .insert([{
        server_id: serverId,
        token_hash: tokenHash,
        token_prefix: tokenPrefix,
        expires_at: expiresAt.toISOString(),
        is_active: true
      }])
      .select()
      .single()

    if (error) {
      console.error('Failed to create server token:', error)
      throw new Error('Failed to create authentication token')
    }

    return {
      token, // Return plain token only once during creation
      tokenData: data
    }
  }

  /**
   * Validate a token from SuperAgent
   */
  async validateToken(token: string): Promise<TokenValidationResult> {
    if (!token || !token.startsWith('sa_')) {
      return { valid: false, error: 'Invalid token format' }
    }

    const tokenHash = this.hashToken(token)

    const { data, error } = await supabase
      .from('server_tokens')
      .select(`
        *,
        servers (
          id,
          name,
          status
        )
      `)
      .eq('token_hash', tokenHash)
      .eq('is_active', true)
      .single()

    if (error || !data) {
      return { valid: false, error: 'Token not found or inactive' }
    }

    // Check if token is expired
    const now = new Date()
    const expiresAt = new Date(data.expires_at)
    if (now > expiresAt) {
      return { valid: false, error: 'Token expired' }
    }

    // Update last used timestamp
    await supabase
      .from('server_tokens')
      .update({ last_used_at: new Date().toISOString() })
      .eq('id', data.id)

    return {
      valid: true,
      server_id: data.server_id,
      expires_at: data.expires_at
    }
  }

  /**
   * Rotate token for a server (create new, deactivate old)
   */
  async rotateServerToken(serverId: string): Promise<{ token: string; tokenData: ServerToken }> {
    // Deactivate existing tokens
    await supabase
      .from('server_tokens')
      .update({ is_active: false })
      .eq('server_id', serverId)

    // Create new token
    return this.createServerToken(serverId)
  }

  /**
   * Get active token info for a server (without exposing the actual token)
   */
  async getServerTokenInfo(serverId: string): Promise<ServerToken | null> {
    const { data, error } = await supabase
      .from('server_tokens')
      .select('*')
      .eq('server_id', serverId)
      .eq('is_active', true)
      .single()

    if (error || !data) {
      return null
    }

    return data
  }

  /**
   * Revoke a token
   */
  async revokeToken(serverId: string): Promise<boolean> {
    const { error } = await supabase
      .from('server_tokens')
      .update({ is_active: false })
      .eq('server_id', serverId)

    return !error
  }

  /**
   * Test agent connection with token
   */
  async testAgentConnection(endpoint: string, token: string): Promise<{ connected: boolean; agentInfo?: any; error?: string }> {
    try {
      const controller = new AbortController()
      const timeoutId = setTimeout(() => controller.abort(), 10000) // 10 second timeout

      const response = await fetch(`${endpoint}/api/v1/status`, {
        method: 'GET',
        headers: {
          'Content-Type': 'application/json',
          'Authorization': `Bearer ${token}`
        },
        signal: controller.signal
      })

      clearTimeout(timeoutId)

      if (!response.ok) {
        return {
          connected: false,
          error: `HTTP ${response.status}: ${response.statusText}`
        }
      }

      const agentInfo = await response.json()
      return {
        connected: true,
        agentInfo
      }
    } catch (error) {
      return {
        connected: false,
        error: error instanceof Error ? error.message : 'Connection failed'
      }
    }
  }

  /**
   * Get all tokens for admin management (without exposing actual tokens)
   */
  async getAllTokens(): Promise<ServerToken[]> {
    const { data, error } = await supabase
      .from('server_tokens')
      .select(`
        *,
        servers (
          name,
          hostname,
          location
        )
      `)
      .order('created_at', { ascending: false })

    if (error) {
      console.error('Failed to fetch tokens:', error)
      return []
    }

    return data || []
  }

  /**
   * Clean up expired tokens
   */
  async cleanupExpiredTokens(): Promise<number> {
    const { count, error } = await supabase
      .from('server_tokens')
      .update({ is_active: false }, { count: 'exact' })
      .lt('expires_at', new Date().toISOString())
      .eq('is_active', true)

    if (error) {
      console.error('Failed to cleanup expired tokens:', error)
      return 0
    }

    return count || 0
  }
}

// Singleton instance
export const authTokenManager = new AuthTokenManager()

export default authTokenManager