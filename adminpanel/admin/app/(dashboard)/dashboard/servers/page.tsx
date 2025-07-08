import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs"
import { Progress } from "@/components/ui/progress"
import { 
  Plus, 
  Search, 
  Filter, 
  MoreHorizontal, 
  Eye, 
  Settings,
  Trash2,
  Download,
  Server,
  Activity,
  CheckCircle,
  XCircle,
  AlertTriangle,
  Wifi,
  WifiOff,
  Cpu,
  MemoryStick,
  HardDrive,
  MapPin,
  Zap,
  Wrench,
  Globe,
  TrendingUp
} from "lucide-react"

// Mock data for servers/agents
const servers = [
  {
    id: "srv-001",
    name: "us-east-1-prod",
    hostname: "prod-agent-01.superagent.dev",
    ip_address: "10.0.1.100",
    location: "US East (Virginia)",
    provider: "aws",
    status: "online",
    agent_version: "1.0.0",
    cpu_cores: 8,
    memory_gb: 32,
    disk_gb: 500,
    bandwidth_gbps: 10.0,
    architecture: "x86_64",
    os: "Ubuntu",
    os_version: "22.04 LTS",
    max_deployments: 50,
    current_deployments: 23,
    api_endpoint: "https://prod-agent-01.superagent.dev:8080",
    last_heartbeat: "2024-03-20T12:45:00Z",
    health_score: 98.5,
    uptime_days: 45,
    cpu_usage: 68.2,
    memory_usage: 71.5,
    disk_usage: 45.8,
    network_in_mbps: 125.3,
    network_out_mbps: 89.7,
    deployments_today: 12,
    error_rate: 0.01,
  },
  {
    id: "srv-002",
    name: "us-west-2-staging",
    hostname: "staging-agent-01.superagent.dev",
    ip_address: "10.0.2.100",
    location: "US West (Oregon)",
    provider: "aws",
    status: "online",
    agent_version: "1.0.0",
    cpu_cores: 4,
    memory_gb: 16,
    disk_gb: 200,
    bandwidth_gbps: 5.0,
    architecture: "x86_64",
    os: "Ubuntu",
    os_version: "22.04 LTS",
    max_deployments: 25,
    current_deployments: 8,
    api_endpoint: "https://staging-agent-01.superagent.dev:8080",
    last_heartbeat: "2024-03-20T12:44:30Z",
    health_score: 95.2,
    uptime_days: 30,
    cpu_usage: 34.7,
    memory_usage: 42.3,
    disk_usage: 28.9,
    network_in_mbps: 45.2,
    network_out_mbps: 32.1,
    deployments_today: 5,
    error_rate: 0.02,
  },
  {
    id: "srv-003",
    name: "eu-central-1-prod",
    hostname: "eu-agent-01.superagent.dev",
    ip_address: "10.0.3.100",
    location: "EU Central (Frankfurt)",
    provider: "aws",
    status: "maintenance",
    agent_version: "0.9.8",
    cpu_cores: 8,
    memory_gb: 32,
    disk_gb: 500,
    bandwidth_gbps: 10.0,
    architecture: "x86_64",
    os: "Ubuntu",
    os_version: "22.04 LTS",
    max_deployments: 50,
    current_deployments: 0,
    api_endpoint: "https://eu-agent-01.superagent.dev:8080",
    last_heartbeat: "2024-03-20T10:30:00Z",
    health_score: 85.0,
    uptime_days: 67,
    cpu_usage: 5.2,
    memory_usage: 15.7,
    disk_usage: 52.3,
    network_in_mbps: 2.1,
    network_out_mbps: 1.8,
    deployments_today: 0,
    error_rate: 0,
  },
  {
    id: "srv-004",
    name: "us-west-2-dev",
    hostname: "dev-agent-01.superagent.dev",
    ip_address: "10.0.4.100",
    location: "US West (Oregon)",
    provider: "aws",
    status: "error",
    agent_version: "1.0.0",
    cpu_cores: 2,
    memory_gb: 8,
    disk_gb: 100,
    bandwidth_gbps: 1.0,
    architecture: "x86_64",
    os: "Ubuntu",
    os_version: "22.04 LTS",
    max_deployments: 10,
    current_deployments: 2,
    api_endpoint: "https://dev-agent-01.superagent.dev:8080",
    last_heartbeat: "2024-03-20T11:15:00Z",
    health_score: 45.0,
    uptime_days: 15,
    cpu_usage: 95.8,
    memory_usage: 89.2,
    disk_usage: 78.5,
    network_in_mbps: 156.7,
    network_out_mbps: 98.4,
    deployments_today: 3,
    error_rate: 5.2,
  },
  {
    id: "srv-005",
    name: "ap-southeast-1-prod",
    hostname: "asia-agent-01.superagent.dev",
    ip_address: "10.0.5.100",
    location: "Asia Pacific (Singapore)",
    provider: "aws",
    status: "offline",
    agent_version: "1.0.0",
    cpu_cores: 4,
    memory_gb: 16,
    disk_gb: 250,
    bandwidth_gbps: 5.0,
    architecture: "x86_64",
    os: "Ubuntu",
    os_version: "22.04 LTS",
    max_deployments: 30,
    current_deployments: 0,
    api_endpoint: "https://asia-agent-01.superagent.dev:8080",
    last_heartbeat: "2024-03-20T08:22:00Z",
    health_score: 0,
    uptime_days: 89,
    cpu_usage: 0,
    memory_usage: 0,
    disk_usage: 65.4,
    network_in_mbps: 0,
    network_out_mbps: 0,
    deployments_today: 0,
    error_rate: 0,
  },
]

function getStatusBadge(status: string) {
  switch (status) {
    case "online":
      return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" />Online</Badge>
    case "offline":
      return <Badge variant="destructive"><XCircle className="w-3 h-3 mr-1" />Offline</Badge>
    case "maintenance":
      return <Badge className="bg-yellow-500"><Wrench className="w-3 h-3 mr-1" />Maintenance</Badge>
    case "error":
      return <Badge variant="destructive"><AlertTriangle className="w-3 h-3 mr-1" />Error</Badge>
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

function getHealthScore(score: number) {
  if (score >= 90) return "text-green-600"
  if (score >= 70) return "text-yellow-600"
  return "text-red-600"
}

function getUtilizationColor(usage: number) {
  if (usage >= 90) return "bg-red-500"
  if (usage >= 75) return "bg-yellow-500"
  return "bg-green-500"
}

function getProviderBadge(provider: string) {
  switch (provider) {
    case "aws":
      return <Badge variant="outline" className="text-orange-600">AWS</Badge>
    case "gcp":
      return <Badge variant="outline" className="text-blue-600">GCP</Badge>
    case "azure":
      return <Badge variant="outline" className="text-blue-500">Azure</Badge>
    case "digitalocean":
      return <Badge variant="outline" className="text-blue-400">DigitalOcean</Badge>
    default:
      return <Badge variant="outline">{provider.toUpperCase()}</Badge>
  }
}

export default function ServersPage() {
  const onlineServers = servers.filter(s => s.status === "online")
  const offlineServers = servers.filter(s => s.status === "offline")
  const totalDeployments = servers.reduce((sum, s) => sum + s.current_deployments, 0)
  const totalCapacity = servers.reduce((sum, s) => sum + s.max_deployments, 0)
  const averageHealth = servers.reduce((sum, s) => sum + s.health_score, 0) / servers.length

  return (
    <div className="flex-1 space-y-4 p-4 md:p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Servers & Agents</h2>
          <p className="text-muted-foreground">
            Monitor and manage your SuperAgent deployment cluster
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline">
            <Download className="mr-2 h-4 w-4" />
            Export Metrics
          </Button>
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Add Server
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Servers</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{servers.length}</div>
            <p className="text-xs text-muted-foreground">
              {onlineServers.length} online, {offlineServers.length} offline
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Cluster Health</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className={`text-2xl font-bold ${getHealthScore(averageHealth)}`}>
              {averageHealth.toFixed(1)}%
            </div>
            <p className="text-xs text-muted-foreground">
              Average health score
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Deployment Capacity</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalDeployments}/{totalCapacity}</div>
            <p className="text-xs text-muted-foreground">
              {Math.round((totalDeployments / totalCapacity) * 100)}% utilized
            </p>
            <Progress value={(totalDeployments / totalCapacity) * 100} className="mt-2" />
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Deployments</CardTitle>
            <Zap className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalDeployments}</div>
            <p className="text-xs text-muted-foreground">
              Across all servers
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Servers Management */}
      <Tabs defaultValue="all" className="w-full">
        <TabsList>
          <TabsTrigger value="all">All Servers</TabsTrigger>
          <TabsTrigger value="online">Online</TabsTrigger>
          <TabsTrigger value="offline">Offline</TabsTrigger>
          <TabsTrigger value="monitoring">Monitoring</TabsTrigger>
        </TabsList>

        <TabsContent value="all" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Server Cluster Status</CardTitle>
              <CardDescription>
                Real-time monitoring of your SuperAgent deployment infrastructure
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center space-x-2 mb-4">
                <div className="relative flex-1">
                  <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search servers..."
                    className="pl-8"
                  />
                </div>
                <Button variant="outline">
                  <Filter className="mr-2 h-4 w-4" />
                  Filter
                </Button>
                <Button variant="outline">
                  <Activity className="mr-2 h-4 w-4" />
                  Refresh
                </Button>
              </div>

              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Server</TableHead>
                      <TableHead>Location</TableHead>
                      <TableHead>Provider</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Health</TableHead>
                      <TableHead>Version</TableHead>
                      <TableHead>Resources</TableHead>
                      <TableHead>Utilization</TableHead>
                      <TableHead>Deployments</TableHead>
                      <TableHead>Last Seen</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {servers.map((server) => (
                      <TableRow key={server.id}>
                        <TableCell>
                          <div>
                            <div className="font-medium flex items-center">
                              {server.status === "online" ? (
                                <Wifi className="w-4 h-4 text-green-500 mr-2" />
                              ) : (
                                <WifiOff className="w-4 h-4 text-red-500 mr-2" />
                              )}
                              {server.name}
                            </div>
                            <div className="text-sm text-muted-foreground">
                              {server.hostname}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {server.ip_address}
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center">
                            <MapPin className="w-3 h-3 mr-1" />
                            {server.location}
                          </div>
                        </TableCell>
                        <TableCell>{getProviderBadge(server.provider)}</TableCell>
                        <TableCell>{getStatusBadge(server.status)}</TableCell>
                        <TableCell>
                          <div className={`font-medium ${getHealthScore(server.health_score)}`}>
                            {server.health_score.toFixed(1)}%
                          </div>
                        </TableCell>
                        <TableCell>
                          <Badge variant="outline">v{server.agent_version}</Badge>
                        </TableCell>
                        <TableCell>
                          <div className="text-xs space-y-1">
                            <div>{server.cpu_cores} cores</div>
                            <div>{server.memory_gb}GB RAM</div>
                            <div>{server.disk_gb}GB disk</div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="space-y-2">
                            <div className="flex items-center text-xs">
                              <Cpu className="w-3 h-3 mr-1" />
                              <span className="w-8">{server.cpu_usage.toFixed(0)}%</span>
                              <Progress 
                                value={server.cpu_usage} 
                                className="flex-1 ml-2 h-1"
                                color={getUtilizationColor(server.cpu_usage)}
                              />
                            </div>
                            <div className="flex items-center text-xs">
                              <MemoryStick className="w-3 h-3 mr-1" />
                              <span className="w-8">{server.memory_usage.toFixed(0)}%</span>
                              <Progress 
                                value={server.memory_usage} 
                                className="flex-1 ml-2 h-1"
                                color={getUtilizationColor(server.memory_usage)}
                              />
                            </div>
                            <div className="flex items-center text-xs">
                              <HardDrive className="w-3 h-3 mr-1" />
                              <span className="w-8">{server.disk_usage.toFixed(0)}%</span>
                              <Progress 
                                value={server.disk_usage} 
                                className="flex-1 ml-2 h-1"
                                color={getUtilizationColor(server.disk_usage)}
                              />
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="text-center">
                            <div className="text-lg font-bold">
                              {server.current_deployments}/{server.max_deployments}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              {server.deployments_today} today
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="text-xs">
                            {new Date(server.last_heartbeat).toLocaleString()}
                          </div>
                          <div className="text-xs text-muted-foreground">
                            {server.uptime_days} days uptime
                          </div>
                        </TableCell>
                        <TableCell className="text-right">
                          <DropdownMenu>
                            <DropdownMenuTrigger asChild>
                              <Button variant="ghost" className="h-8 w-8 p-0">
                                <span className="sr-only">Open menu</span>
                                <MoreHorizontal className="h-4 w-4" />
                              </Button>
                            </DropdownMenuTrigger>
                            <DropdownMenuContent align="end">
                              <DropdownMenuItem>
                                <Eye className="mr-2 h-4 w-4" />
                                View Details
                              </DropdownMenuItem>
                              <DropdownMenuItem>
                                <Activity className="mr-2 h-4 w-4" />
                                Live Monitoring
                              </DropdownMenuItem>
                              <DropdownMenuItem>
                                <Settings className="mr-2 h-4 w-4" />
                                Configuration
                              </DropdownMenuItem>
                              <DropdownMenuItem>
                                <Globe className="mr-2 h-4 w-4" />
                                View Deployments
                              </DropdownMenuItem>
                              {server.status === "maintenance" && (
                                <DropdownMenuItem className="text-green-600">
                                  <CheckCircle className="mr-2 h-4 w-4" />
                                  Exit Maintenance
                                </DropdownMenuItem>
                              )}
                              {server.status === "online" && (
                                <DropdownMenuItem className="text-yellow-600">
                                  <Wrench className="mr-2 h-4 w-4" />
                                  Enter Maintenance
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuItem>
                                <Download className="mr-2 h-4 w-4" />
                                Download Logs
                              </DropdownMenuItem>
                              <DropdownMenuItem className="text-red-600">
                                <Trash2 className="mr-2 h-4 w-4" />
                                Remove Server
                              </DropdownMenuItem>
                            </DropdownMenuContent>
                          </DropdownMenu>
                        </TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              </div>

              <div className="flex items-center justify-between space-x-2 py-4">
                <div className="text-sm text-muted-foreground">
                  Showing 1-{servers.length} of {servers.length} servers
                </div>
                <div className="flex items-center space-x-2">
                  <Button variant="outline" size="sm" disabled>
                    Previous
                  </Button>
                  <Button variant="outline" size="sm" disabled>
                    Next
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="online" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Online Servers</CardTitle>
              <CardDescription>
                {onlineServers.length} servers currently online and accepting deployments
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                All online servers are healthy and processing requests
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="offline" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Offline Servers</CardTitle>
              <CardDescription>
                {offlineServers.length} servers currently offline or unreachable
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                Check server connectivity and agent status
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="monitoring" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Real-time Monitoring</CardTitle>
              <CardDescription>
                Live metrics and performance monitoring for all servers
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                Real-time charts and metrics dashboard coming soon
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}