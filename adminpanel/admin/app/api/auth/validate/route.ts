import { NextRequest, NextResponse } from 'next/server'
import { authTokenManager } from '@/lib/auth-tokens'

/**
 * Token validation endpoint for SuperAgent authentication
 * POST /api/auth/validate
 */
export async function POST(request: NextRequest) {
  try {
    const { token } = await request.json()

    if (!token) {
      return NextResponse.json(
        { 
          valid: false, 
          error: 'Token is required' 
        },
        { status: 400 }
      )
    }

    // Validate the token
    const result = await authTokenManager.validateToken(token)

    if (result.valid) {
      return NextResponse.json({
        valid: true,
        server_id: result.server_id,
        expires_at: result.expires_at
      })
    } else {
      return NextResponse.json(
        {
          valid: false,
          error: result.error
        },
        { status: 401 }
      )
    }
  } catch (error) {
    console.error('Token validation error:', error)
    return NextResponse.json(
      {
        valid: false,
        error: 'Internal server error'
      },
      { status: 500 }
    )
  }
}

/**
 * Health check for the validation endpoint
 * GET /api/auth/validate
 */
export async function GET() {
  return NextResponse.json({
    status: 'healthy',
    service: 'token-validation',
    timestamp: new Date().toISOString()
  })
}