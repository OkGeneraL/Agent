"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Alert, AlertDescription } from "@/components/ui/alert"
import { Loader2, Server, CheckCircle, XCircle } from "lucide-react"
import { agentManager } from "@/lib/agents"

export default function AddServerPage() {
  const router = useRouter()
  const [isLoading, setIsLoading] = useState(false)
  const [isTestingConnection, setIsTestingConnection] = useState(false)
  const [connectionStatus, setConnectionStatus] = useState<'untested' | 'success' | 'error'>('untested')
  const [error, setError] = useState("")

  const [formData, setFormData] = useState({
    name: "",
    hostname: "",
    ip_address: "",
    location: "",
    provider: "aws",
    api_endpoint: "",
    api_token_hash: ""
  })

  const handleInputChange = (field: string, value: string) => {
    setFormData(prev => ({ ...prev, [field]: value }))
    setConnectionStatus('untested') // Reset connection status when any field changes
  }

  const testConnection = async () => {
    if (!formData.api_endpoint) {
      setError("API endpoint is required to test connection")
      return
    }

    setIsTestingConnection(true)
    setError("")

    try {
      const isHealthy = await agentManager.testAgentConnection(
        formData.api_endpoint,
        formData.api_token_hash || undefined
      )

      setConnectionStatus(isHealthy ? 'success' : 'error')
      if (!isHealthy) {
        setError("Failed to connect to SuperAgent server. Check the endpoint and token.")
      }
    } catch (err) {
      setConnectionStatus('error')
      setError("Connection test failed: " + (err instanceof Error ? err.message : "Unknown error"))
    } finally {
      setIsTestingConnection(false)
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    
    if (connectionStatus !== 'success') {
      setError("Please test the connection successfully before adding the server")
      return
    }

    setIsLoading(true)
    setError("")

    try {
      const result = await agentManager.addAgentServer({
        ...formData,
        status: 'offline' as const // Will be updated after connection test
      })
      
      if (result) {
        router.push('/dashboard/servers')
      } else {
        setError("Failed to add server. Please check your input and try again.")
      }
    } catch (err) {
      setError("Error adding server: " + (err instanceof Error ? err.message : "Unknown error"))
    } finally {
      setIsLoading(false)
    }
  }

  const getConnectionStatusIcon = () => {
    switch (connectionStatus) {
      case 'success':
        return <CheckCircle className="h-4 w-4 text-green-600" />
      case 'error':
        return <XCircle className="h-4 w-4 text-red-600" />
      default:
        return null
    }
  }

  return (
    <div className="flex-1 space-y-4 p-4 md:p-8 pt-6">
      <div className="flex items-center justify-between">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Add SuperAgent Server</h2>
          <p className="text-muted-foreground">
            Register a new SuperAgent server to manage deployments
          </p>
        </div>
      </div>

      <div className="max-w-2xl">
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center">
              <Server className="mr-2 h-5 w-5" />
              Server Configuration
            </CardTitle>
            <CardDescription>
              Configure the connection details for your SuperAgent server instance
            </CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={handleSubmit} className="space-y-4">
              {error && (
                <Alert variant="destructive">
                  <AlertDescription>{error}</AlertDescription>
                </Alert>
              )}

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="name">Server Name</Label>
                  <Input
                    id="name"
                    placeholder="us-east-1-prod"
                    value={formData.name}
                    onChange={(e) => handleInputChange('name', e.target.value)}
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="location">Location</Label>
                  <Input
                    id="location"
                    placeholder="US East (Virginia)"
                    value={formData.location}
                    onChange={(e) => handleInputChange('location', e.target.value)}
                    required
                  />
                </div>
              </div>

              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="hostname">Hostname</Label>
                  <Input
                    id="hostname"
                    placeholder="prod-agent-01.superagent.dev"
                    value={formData.hostname}
                    onChange={(e) => handleInputChange('hostname', e.target.value)}
                    required
                  />
                </div>

                <div className="space-y-2">
                  <Label htmlFor="ip_address">IP Address</Label>
                  <Input
                    id="ip_address"
                    placeholder="10.0.1.100"
                    value={formData.ip_address}
                    onChange={(e) => handleInputChange('ip_address', e.target.value)}
                    required
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label htmlFor="provider">Provider</Label>
                <Select value={formData.provider} onValueChange={(value) => handleInputChange('provider', value)}>
                  <SelectTrigger>
                    <SelectValue placeholder="Select cloud provider" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="aws">Amazon Web Services (AWS)</SelectItem>
                    <SelectItem value="gcp">Google Cloud Platform (GCP)</SelectItem>
                    <SelectItem value="azure">Microsoft Azure</SelectItem>
                    <SelectItem value="digitalocean">DigitalOcean</SelectItem>
                    <SelectItem value="linode">Linode</SelectItem>
                    <SelectItem value="vultr">Vultr</SelectItem>
                    <SelectItem value="dedicated">Dedicated Server</SelectItem>
                    <SelectItem value="other">Other</SelectItem>
                  </SelectContent>
                </Select>
              </div>

              <div className="space-y-2">
                <Label htmlFor="api_endpoint">API Endpoint</Label>
                <Input
                  id="api_endpoint"
                  placeholder="https://prod-agent-01.superagent.dev:8080/api/v1"
                  value={formData.api_endpoint}
                  onChange={(e) => handleInputChange('api_endpoint', e.target.value)}
                  required
                />
                <p className="text-sm text-muted-foreground">
                  Full URL to the SuperAgent API endpoint (including port if needed)
                </p>
              </div>

              <div className="space-y-2">
                <Label htmlFor="api_token_hash">API Token (Optional)</Label>
                <Input
                  id="api_token_hash"
                  type="password"
                  placeholder="Enter authentication token if required"
                  value={formData.api_token_hash}
                  onChange={(e) => handleInputChange('api_token_hash', e.target.value)}
                />
                <p className="text-sm text-muted-foreground">
                  ⚠️ Note: Current SuperAgent version has no authentication. This will be used when authentication is implemented.
                </p>
              </div>

              <div className="flex items-center space-x-2 pt-4">
                <Button
                  type="button"
                  variant="outline"
                  onClick={testConnection}
                  disabled={isTestingConnection || !formData.api_endpoint}
                >
                  {isTestingConnection ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : (
                    getConnectionStatusIcon() && <span className="mr-2">{getConnectionStatusIcon()}</span>
                  )}
                  Test Connection
                </Button>

                {connectionStatus === 'success' && (
                  <span className="text-sm text-green-600">✓ Connection successful</span>
                )}
                {connectionStatus === 'error' && (
                  <span className="text-sm text-red-600">✗ Connection failed</span>
                )}
              </div>

              <div className="flex justify-end space-x-2 pt-6">
                <Button 
                  type="button" 
                  variant="outline" 
                  onClick={() => router.back()}
                >
                  Cancel
                </Button>
                <Button 
                  type="submit" 
                  disabled={isLoading || connectionStatus !== 'success'}
                >
                  {isLoading ? (
                    <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  ) : null}
                  Add Server
                </Button>
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="mt-6">
          <CardHeader>
            <CardTitle>⚠️ Security Notice</CardTitle>
          </CardHeader>
          <CardContent>
            <p className="text-sm text-muted-foreground">
              <strong>Current SuperAgent Implementation:</strong> The SuperAgent API currently has no authentication middleware. 
              This means:
            </p>
            <ul className="list-disc list-inside text-sm text-muted-foreground mt-2 space-y-1">
              <li>Any system that can reach the API can control deployments</li>
              <li>Use network-level security (VPN, firewall rules, private networks)</li>
              <li>The API token field is prepared for future authentication implementation</li>
              <li>Consider implementing authentication before production deployment</li>
            </ul>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}