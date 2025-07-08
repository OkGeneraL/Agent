'use client'

import { useState } from 'react'
import { useRouter } from 'next/navigation'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { Badge } from '@/components/ui/badge'
import { AlertCircle, CheckCircle, Copy, Eye, EyeOff, Server, Settings, Shield } from 'lucide-react'
import { Alert, AlertDescription } from '@/components/ui/alert'
import { Textarea } from '@/components/ui/textarea'
import { authTokenManager } from '@/lib/auth-tokens'
import { agentManager } from '@/lib/agents'

interface ServerInfo {
  name: string
  hostname: string
  ip_address: string
  location: string
  provider: string
  cpu_cores: number
  memory_gb: number
  disk_gb: number
  api_endpoint: string
  description?: string
}

interface StepProps {
  serverInfo: ServerInfo
  setServerInfo: (info: ServerInfo) => void
  onNext: () => void
  onPrev?: () => void
  token?: string
  serverId?: string
}

const Step1_ServerDetails = ({ serverInfo, setServerInfo, onNext }: StepProps) => {
  const [errors, setErrors] = useState<Record<string, string>>({})

  const validateAndNext = () => {
    const newErrors: Record<string, string> = {}

    if (!serverInfo.name.trim()) newErrors.name = 'Server name is required'
    if (!serverInfo.hostname.trim()) newErrors.hostname = 'Hostname is required'
    if (!serverInfo.ip_address.trim()) newErrors.ip_address = 'IP address is required'
    if (!serverInfo.location.trim()) newErrors.location = 'Location is required'
    if (!serverInfo.provider) newErrors.provider = 'Provider is required'
    if (!serverInfo.cpu_cores || serverInfo.cpu_cores < 1) newErrors.cpu_cores = 'CPU cores must be at least 1'
    if (!serverInfo.memory_gb || serverInfo.memory_gb < 1) newErrors.memory_gb = 'Memory must be at least 1 GB'
    if (!serverInfo.disk_gb || serverInfo.disk_gb < 10) newErrors.disk_gb = 'Disk space must be at least 10 GB'
    if (!serverInfo.api_endpoint.trim()) newErrors.api_endpoint = 'API endpoint is required'

    // Validate IP address format
    const ipRegex = /^(?:(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)\.){3}(?:25[0-5]|2[0-4][0-9]|[01]?[0-9][0-9]?)$/
    if (serverInfo.ip_address && !ipRegex.test(serverInfo.ip_address)) {
      newErrors.ip_address = 'Invalid IP address format'
    }

    // Validate API endpoint format
    if (serverInfo.api_endpoint && !serverInfo.api_endpoint.startsWith('http')) {
      newErrors.api_endpoint = 'API endpoint must start with http:// or https://'
    }

    setErrors(newErrors)

    if (Object.keys(newErrors).length === 0) {
      onNext()
    }
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Server className="h-5 w-5" />
          Server Information
        </CardTitle>
        <CardDescription>
          Enter the basic information about your SuperAgent server
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-4">
        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="name">Server Name *</Label>
            <Input
              id="name"
              placeholder="e.g., production-east-1"
              value={serverInfo.name}
              onChange={(e) => setServerInfo({ ...serverInfo, name: e.target.value })}
              className={errors.name ? 'border-red-500' : ''}
            />
            {errors.name && <p className="text-sm text-red-500">{errors.name}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="hostname">Hostname *</Label>
            <Input
              id="hostname"
              placeholder="e.g., superagent-prod-01"
              value={serverInfo.hostname}
              onChange={(e) => setServerInfo({ ...serverInfo, hostname: e.target.value })}
              className={errors.hostname ? 'border-red-500' : ''}
            />
            {errors.hostname && <p className="text-sm text-red-500">{errors.hostname}</p>}
          </div>
        </div>

        <div className="grid grid-cols-2 gap-4">
          <div className="space-y-2">
            <Label htmlFor="ip">IP Address *</Label>
            <Input
              id="ip"
              placeholder="e.g., 10.0.1.100"
              value={serverInfo.ip_address}
              onChange={(e) => setServerInfo({ ...serverInfo, ip_address: e.target.value })}
              className={errors.ip_address ? 'border-red-500' : ''}
            />
            {errors.ip_address && <p className="text-sm text-red-500">{errors.ip_address}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="location">Location *</Label>
            <Input
              id="location"
              placeholder="e.g., us-east-1, eu-west-1"
              value={serverInfo.location}
              onChange={(e) => setServerInfo({ ...serverInfo, location: e.target.value })}
              className={errors.location ? 'border-red-500' : ''}
            />
            {errors.location && <p className="text-sm text-red-500">{errors.location}</p>}
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="provider">Cloud Provider *</Label>
          <Select
            value={serverInfo.provider}
            onValueChange={(value) => setServerInfo({ ...serverInfo, provider: value })}
          >
            <SelectTrigger className={errors.provider ? 'border-red-500' : ''}>
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
          {errors.provider && <p className="text-sm text-red-500">{errors.provider}</p>}
        </div>

        <div className="grid grid-cols-3 gap-4">
          <div className="space-y-2">
            <Label htmlFor="cpu">CPU Cores *</Label>
            <Input
              id="cpu"
              type="number"
              min="1"
              value={serverInfo.cpu_cores || ''}
              onChange={(e) => setServerInfo({ ...serverInfo, cpu_cores: parseInt(e.target.value) || 0 })}
              className={errors.cpu_cores ? 'border-red-500' : ''}
            />
            {errors.cpu_cores && <p className="text-sm text-red-500">{errors.cpu_cores}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="memory">Memory (GB) *</Label>
            <Input
              id="memory"
              type="number"
              min="1"
              value={serverInfo.memory_gb || ''}
              onChange={(e) => setServerInfo({ ...serverInfo, memory_gb: parseInt(e.target.value) || 0 })}
              className={errors.memory_gb ? 'border-red-500' : ''}
            />
            {errors.memory_gb && <p className="text-sm text-red-500">{errors.memory_gb}</p>}
          </div>

          <div className="space-y-2">
            <Label htmlFor="disk">Disk Space (GB) *</Label>
            <Input
              id="disk"
              type="number"
              min="10"
              value={serverInfo.disk_gb || ''}
              onChange={(e) => setServerInfo({ ...serverInfo, disk_gb: parseInt(e.target.value) || 0 })}
              className={errors.disk_gb ? 'border-red-500' : ''}
            />
            {errors.disk_gb && <p className="text-sm text-red-500">{errors.disk_gb}</p>}
          </div>
        </div>

        <div className="space-y-2">
          <Label htmlFor="endpoint">API Endpoint *</Label>
          <Input
            id="endpoint"
            placeholder="e.g., http://10.0.1.100:8080"
            value={serverInfo.api_endpoint}
            onChange={(e) => setServerInfo({ ...serverInfo, api_endpoint: e.target.value })}
            className={errors.api_endpoint ? 'border-red-500' : ''}
          />
          {errors.api_endpoint && <p className="text-sm text-red-500">{errors.api_endpoint}</p>}
          <p className="text-sm text-muted-foreground">
            The SuperAgent API endpoint (usually port 8080)
          </p>
        </div>

        <div className="space-y-2">
          <Label htmlFor="description">Description (Optional)</Label>
          <Textarea
            id="description"
            placeholder="Additional notes about this server..."
            value={serverInfo.description || ''}
                         onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => setServerInfo({ ...serverInfo, description: e.target.value })}
          />
        </div>

        <div className="flex justify-end">
          <Button onClick={validateAndNext}>
            Next: Generate Token
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

const Step2_TokenGeneration = ({ serverInfo, token, onNext, onPrev }: StepProps & { token: string }) => {
  const [showToken, setShowToken] = useState(false)
  const [copied, setCopied] = useState(false)

  const copyToken = async () => {
    await navigator.clipboard.writeText(token)
    setCopied(true)
    setTimeout(() => setCopied(false), 2000)
  }

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Shield className="h-5 w-5" />
          Authentication Token
        </CardTitle>
        <CardDescription>
          A secure authentication token has been generated for {serverInfo.name}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <Alert>
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            <strong>Important:</strong> This token will only be displayed once. Copy it now and store it securely.
            You will need this token to configure the SuperAgent on your server.
          </AlertDescription>
        </Alert>

        <div className="space-y-2">
          <Label>Authentication Token</Label>
          <div className="flex items-center gap-2">
            <div className="flex-1 relative">
              <Input
                value={token}
                type={showToken ? 'text' : 'password'}
                readOnly
                className="font-mono text-sm pr-20"
              />
              <div className="absolute right-2 top-1/2 -translate-y-1/2 flex gap-1">
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={() => setShowToken(!showToken)}
                  className="h-6 w-6 p-0"
                >
                  {showToken ? <EyeOff className="h-3 w-3" /> : <Eye className="h-3 w-3" />}
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="sm"
                  onClick={copyToken}
                  className="h-6 w-6 p-0"
                >
                  <Copy className="h-3 w-3" />
                </Button>
              </div>
            </div>
          </div>
          {copied && (
            <p className="text-sm text-green-600">Token copied to clipboard!</p>
          )}
        </div>

        <div className="space-y-3">
          <h4 className="font-medium">Token Information:</h4>
          <div className="grid grid-cols-1 gap-2 text-sm">
            <div className="flex justify-between">
              <span className="text-muted-foreground">Token Prefix:</span>
              <code className="font-mono">{token.substring(0, 12)}...</code>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Expires:</span>
              <span>1 year from now</span>
            </div>
            <div className="flex justify-between">
              <span className="text-muted-foreground">Format:</span>
              <span>Bearer Token</span>
            </div>
          </div>
        </div>

        <div className="flex justify-between">
          <Button variant="outline" onClick={onPrev}>
            Back
          </Button>
          <Button onClick={onNext}>
            Next: Setup Instructions
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

const Step3_SetupInstructions = ({ serverInfo, token, serverId, onNext, onPrev }: StepProps & { token: string; serverId: string }) => {
  const [testResult, setTestResult] = useState<{ testing: boolean; result?: any }>({ testing: false })

  const testConnection = async () => {
    setTestResult({ testing: true })
    try {
      const result = await authTokenManager.testAgentConnection(serverInfo.api_endpoint, token)
      setTestResult({ testing: false, result })
    } catch (error) {
      setTestResult({ testing: false, result: { connected: false, error: 'Connection test failed' } })
    }
  }

  const configYaml = `# SuperAgent Configuration
agent:
  id: "${serverId}"
  location: "${serverInfo.location}"
  work_dir: "/var/lib/superagent"
  data_dir: "/var/lib/superagent/data"

backend:
  base_url: "${process.env.NEXT_PUBLIC_ADMIN_PANEL_URL || 'https://your-admin-panel.com'}"
  api_token: "${token}"
  refresh_interval: "30s"
  timeout: "30s"

monitoring:
  enabled: true
  metrics_port: 9090
  health_check_port: 8080

security:
  audit_log_enabled: true
  run_as_non_root: true`

  return (
    <Card>
      <CardHeader>
        <CardTitle className="flex items-center gap-2">
          <Settings className="h-5 w-5" />
          Setup Instructions
        </CardTitle>
        <CardDescription>
          Follow these steps to configure SuperAgent on {serverInfo.name}
        </CardDescription>
      </CardHeader>
      <CardContent className="space-y-6">
        <div className="space-y-4">
          <h4 className="font-medium">Step 1: Update Configuration File</h4>
          <p className="text-sm text-muted-foreground">
            Edit the SuperAgent configuration file on your server:
          </p>
          <div className="space-y-2">
            <Label>Configuration File: /etc/superagent/config.yaml</Label>
            <Textarea
              value={configYaml}
              readOnly
              className="font-mono text-xs h-40 resize-none"
            />
            <div className="flex gap-2">
              <Button
                variant="outline"
                size="sm"
                onClick={() => navigator.clipboard.writeText(configYaml)}
              >
                <Copy className="h-4 w-4 mr-2" />
                Copy Configuration
              </Button>
            </div>
          </div>
        </div>

        <div className="space-y-4">
          <h4 className="font-medium">Step 2: Restart SuperAgent Service</h4>
          <div className="bg-gray-100 p-3 rounded-md">
            <code className="text-sm">
              sudo systemctl restart superagent<br />
              sudo systemctl status superagent
            </code>
          </div>
        </div>

        <div className="space-y-4">
          <h4 className="font-medium">Step 3: Verify Connection</h4>
          <p className="text-sm text-muted-foreground">
            Test the connection between the admin panel and your SuperAgent server:
          </p>
          <div className="flex items-center gap-4">
            <Button 
              onClick={testConnection} 
              disabled={testResult.testing}
              variant={testResult.result?.connected ? "default" : "outline"}
            >
              {testResult.testing ? 'Testing...' : 'Test Connection'}
            </Button>
            
            {testResult.result && (
              <div className="flex items-center gap-2">
                {testResult.result.connected ? (
                  <>
                    <CheckCircle className="h-4 w-4 text-green-500" />
                    <span className="text-sm text-green-600">Connected successfully!</span>
                  </>
                ) : (
                  <>
                    <AlertCircle className="h-4 w-4 text-red-500" />
                    <span className="text-sm text-red-600">
                      Connection failed: {testResult.result.error}
                    </span>
                  </>
                )}
              </div>
            )}
          </div>

          {testResult.result?.agentInfo && (
            <div className="bg-green-50 p-3 rounded-md">
              <h5 className="font-medium text-green-800 mb-2">Agent Information:</h5>
              <div className="grid grid-cols-2 gap-2 text-sm">
                <div>Status: <Badge variant="secondary">{testResult.result.agentInfo.status}</Badge></div>
                <div>Version: {testResult.result.agentInfo.version}</div>
                <div>Health: <Badge variant="secondary">{testResult.result.agentInfo.health}</Badge></div>
                <div>Uptime: {testResult.result.agentInfo.uptime}</div>
              </div>
            </div>
          )}
        </div>

        <Alert>
          <AlertCircle className="h-4 w-4" />
          <AlertDescription>
            <strong>Security Note:</strong> Make sure port 8080 (API) and 9090 (metrics) are accessible 
            from the admin panel but blocked from public internet access.
          </AlertDescription>
        </Alert>

        <div className="flex justify-between">
          <Button variant="outline" onClick={onPrev}>
            Back
          </Button>
          <Button 
            onClick={onNext}
            disabled={!testResult.result?.connected}
          >
            Complete Setup
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}

export default function AddServerWizard() {
  const router = useRouter()
  const [currentStep, setCurrentStep] = useState(1)
  const [serverInfo, setServerInfo] = useState<ServerInfo>({
    name: '',
    hostname: '',
    ip_address: '',
    location: '',
    provider: '',
    cpu_cores: 0,
    memory_gb: 0,
    disk_gb: 0,
    api_endpoint: '',
    description: ''
  })
  const [token, setToken] = useState('')
  const [serverId, setServerId] = useState('')
  const [loading, setLoading] = useState(false)

  const handleStep1Next = async () => {
    setLoading(true)
    try {
      // Create server record
      const server = await agentManager.addAgentServer({
        ...serverInfo,
        status: 'offline' as const
      })

      if (server) {
        // Generate authentication token
        const { token: newToken } = await authTokenManager.createServerToken(server.id)
        setToken(newToken)
        setServerId(server.id)
        setCurrentStep(2)
      }
    } catch (error) {
      console.error('Failed to create server:', error)
      // Handle error - show notification
    } finally {
      setLoading(false)
    }
  }

  const handleComplete = () => {
    router.push('/dashboard/servers')
  }

  const steps = [
    {
      number: 1,
      title: 'Server Details',
      description: 'Basic server information'
    },
    {
      number: 2,
      title: 'Authentication',
      description: 'Generate secure token'
    },
    {
      number: 3,
      title: 'Setup & Test',
      description: 'Configure and verify'
    }
  ]

  return (
    <div className="max-w-4xl mx-auto space-y-6">
      {/* Progress indicator */}
      <div className="flex items-center justify-between">
        {steps.map((step, index) => (
          <div key={step.number} className="flex items-center">
            <div className={`flex items-center justify-center w-8 h-8 rounded-full border-2 ${
              currentStep >= step.number 
                ? 'bg-blue-500 border-blue-500 text-white' 
                : 'border-gray-300 text-gray-500'
            }`}>
              {step.number}
            </div>
            <div className="ml-3">
              <p className={`text-sm font-medium ${
                currentStep >= step.number ? 'text-blue-600' : 'text-gray-500'
              }`}>
                {step.title}
              </p>
              <p className="text-xs text-gray-500">{step.description}</p>
            </div>
            {index < steps.length - 1 && (
              <div className={`w-16 h-0.5 mx-4 ${
                currentStep > step.number ? 'bg-blue-500' : 'bg-gray-300'
              }`} />
            )}
          </div>
        ))}
      </div>

      {/* Step content */}
      {currentStep === 1 && (
        <Step1_ServerDetails
          serverInfo={serverInfo}
          setServerInfo={setServerInfo}
          onNext={handleStep1Next}
        />
      )}

      {currentStep === 2 && token && (
        <Step2_TokenGeneration
          serverInfo={serverInfo}
          setServerInfo={setServerInfo}
          token={token}
          onNext={() => setCurrentStep(3)}
          onPrev={() => setCurrentStep(1)}
        />
      )}

      {currentStep === 3 && token && serverId && (
        <Step3_SetupInstructions
          serverInfo={serverInfo}
          setServerInfo={setServerInfo}
          token={token}
          serverId={serverId}
          onNext={handleComplete}
          onPrev={() => setCurrentStep(2)}
        />
      )}
    </div>
  )
}