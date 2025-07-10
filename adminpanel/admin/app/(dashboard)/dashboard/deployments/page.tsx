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
import { 
  Plus, 
  Search, 
  Filter, 
  MoreHorizontal, 
  Eye, 
  Play,
  Square,
  RotateCcw,
  Trash2,
  Download,
  Rocket,
  Activity,
  CheckCircle,
  XCircle,
  Clock,
  AlertTriangle,
  ExternalLink,
  FileText,
  Settings,
  TrendingUp,
  Cpu,
  MemoryStick,
  HardDrive
} from "lucide-react"

// Mock data for deployments (in real app, this would come from SuperAgent API)
const deployments = [
  {
    id: "dep-001",
    name: "react-app-prod",
    application_name: "React Starter",
    customer_name: "TechCorp Inc.",
    server_name: "us-east-1-prod",
    subdomain: "techcorp-app",
    domain: "app.techcorp.com",
    status: "running",
    environment: "production",
    version: "2.1.0",
    container_id: "container_abc123",
    created_at: "2024-03-20T10:30:00Z",
    last_deployed_at: "2024-03-20T10:32:15Z",
    health_status: "healthy",
    cpu_usage: 45.2,
    memory_usage: 312,
    disk_usage: 1024,
    requests_per_minute: 152,
    response_time_avg: 95,
    error_rate: 0.02,
    uptime: "99.98%",
  },
  {
    id: "dep-002",
    name: "api-server-staging",
    application_name: "Node.js API Server",
    customer_name: "StartupXYZ",
    server_name: "us-west-2-staging",
    subdomain: "api-staging",
    domain: null,
    status: "building",
    environment: "staging",
    version: "1.5.3",
    container_id: null,
    created_at: "2024-03-20T11:15:00Z",
    last_deployed_at: null,
    health_status: "unknown",
    cpu_usage: 0,
    memory_usage: 0,
    disk_usage: 0,
    requests_per_minute: 0,
    response_time_avg: 0,
    error_rate: 0,
    uptime: "0%",
  },
  {
    id: "dep-003",
    name: "wordpress-site",
    application_name: "WordPress CMS",
    customer_name: "BlogCorp",
    server_name: "eu-central-1-prod",
    subdomain: "blog",
    domain: "blog.blogcorp.com",
    status: "failed",
    environment: "production",
    version: "6.4.0",
    container_id: null,
    created_at: "2024-03-20T09:45:00Z",
    last_deployed_at: "2024-03-20T09:47:30Z",
    health_status: "unhealthy",
    cpu_usage: 0,
    memory_usage: 0,
    disk_usage: 512,
    requests_per_minute: 0,
    response_time_avg: 0,
    error_rate: 100,
    uptime: "0%",
  },
  {
    id: "dep-004",
    name: "redis-cache-prod",
    application_name: "Redis Cache",
    customer_name: "Enterprise Corp",
    server_name: "us-east-1-prod",
    subdomain: "cache",
    domain: null,
    status: "running",
    environment: "production",
    version: "7.0.5",
    container_id: "container_def456",
    created_at: "2024-03-19T14:20:00Z",
    last_deployed_at: "2024-03-19T14:21:45Z",
    health_status: "healthy",
    cpu_usage: 12.8,
    memory_usage: 1024,
    disk_usage: 256,
    requests_per_minute: 890,
    response_time_avg: 2,
    error_rate: 0,
    uptime: "99.99%",
  },
  {
    id: "dep-005",
    name: "analytics-dev",
    application_name: "Analytics Dashboard",
    customer_name: "DataCorp",
    server_name: "us-west-2-dev",
    subdomain: "analytics-dev",
    domain: null,
    status: "stopped",
    environment: "development",
    version: "1.0.0",
    container_id: "container_ghi789",
    created_at: "2024-03-18T16:00:00Z",
    last_deployed_at: "2024-03-18T16:02:30Z",
    health_status: "unknown",
    cpu_usage: 0,
    memory_usage: 0,
    disk_usage: 768,
    requests_per_minute: 0,
    response_time_avg: 0,
    error_rate: 0,
    uptime: "95.5%",
  },
]

function getStatusBadge(status: string) {
  switch (status) {
    case "running":
      return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" />Running</Badge>
    case "building":
      return <Badge className="bg-blue-500"><Clock className="w-3 h-3 mr-1" />Building</Badge>
    case "deploying":
      return <Badge className="bg-yellow-500"><Rocket className="w-3 h-3 mr-1" />Deploying</Badge>
    case "stopped":
      return <Badge variant="secondary"><Square className="w-3 h-3 mr-1" />Stopped</Badge>
    case "failed":
      return <Badge variant="destructive"><XCircle className="w-3 h-3 mr-1" />Failed</Badge>
    case "pending":
      return <Badge variant="outline"><Clock className="w-3 h-3 mr-1" />Pending</Badge>
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

function getHealthBadge(health: string) {
  switch (health) {
    case "healthy":
      return <Badge variant="outline" className="text-green-600">Healthy</Badge>
    case "unhealthy":
      return <Badge variant="destructive">Unhealthy</Badge>
    case "unknown":
      return <Badge variant="secondary">Unknown</Badge>
    default:
      return <Badge variant="outline">{health}</Badge>
  }
}

function getEnvironmentBadge(environment: string) {
  switch (environment) {
    case "production":
      return <Badge className="bg-red-600">Production</Badge>
    case "staging":
      return <Badge className="bg-yellow-600">Staging</Badge>
    case "development":
      return <Badge className="bg-green-600">Development</Badge>
    case "preview":
      return <Badge className="bg-purple-600">Preview</Badge>
    default:
      return <Badge variant="outline">{environment}</Badge>
  }
}

export default function DeploymentsPage() {
  const runningDeployments = deployments.filter(d => d.status === "running")
  const failedDeployments = deployments.filter(d => d.status === "failed")
  const buildingDeployments = deployments.filter(d => d.status === "building" || d.status === "deploying")
  const totalRequests = deployments.reduce((sum, d) => sum + d.requests_per_minute, 0)

  return (
    <div className="flex-1 space-y-4 p-4 md:p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Deployments</h2>
          <p className="text-muted-foreground">
            Monitor and manage application deployments across your infrastructure
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline">
            <Download className="mr-2 h-4 w-4" />
            Export Logs
          </Button>
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            New Deployment
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Deployments</CardTitle>
            <Rocket className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{deployments.length}</div>
            <p className="text-xs text-muted-foreground">
              {runningDeployments.length} currently running
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Building/Deploying</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{buildingDeployments.length}</div>
            <p className="text-xs text-muted-foreground">
              In progress
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Failed Deployments</CardTitle>
            <AlertTriangle className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-red-600">{failedDeployments.length}</div>
            <p className="text-xs text-muted-foreground">
              Require attention
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Requests/min</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalRequests.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground">
              Across all deployments
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Deployments Management */}
      <Tabs defaultValue="all" className="w-full">
        <TabsList>
          <TabsTrigger value="all">All Deployments</TabsTrigger>
          <TabsTrigger value="running">Running</TabsTrigger>
          <TabsTrigger value="building">Building</TabsTrigger>
          <TabsTrigger value="failed">Failed</TabsTrigger>
        </TabsList>

        <TabsContent value="all" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Live Deployments</CardTitle>
              <CardDescription>
                Real-time status of all deployments managed by SuperAgent
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center space-x-2 mb-4">
                <div className="relative flex-1">
                  <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search deployments..."
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
                      <TableHead>Deployment</TableHead>
                      <TableHead>Application</TableHead>
                      <TableHead>Customer</TableHead>
                      <TableHead>Server</TableHead>
                      <TableHead>Environment</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Health</TableHead>
                      <TableHead>Resources</TableHead>
                      <TableHead>Performance</TableHead>
                      <TableHead>Uptime</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {deployments.map((deployment) => (
                      <TableRow key={deployment.id}>
                        <TableCell>
                          <div>
                            <div className="font-medium">{deployment.name}</div>
                            <div className="text-sm text-muted-foreground">
                              {deployment.domain || `${deployment.subdomain}.superagent.dev`}
                            </div>
                            <div className="text-xs text-muted-foreground">
                              v{deployment.version}
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>{deployment.application_name}</TableCell>
                        <TableCell>{deployment.customer_name}</TableCell>
                        <TableCell>{deployment.server_name}</TableCell>
                        <TableCell>{getEnvironmentBadge(deployment.environment)}</TableCell>
                        <TableCell>{getStatusBadge(deployment.status)}</TableCell>
                        <TableCell>{getHealthBadge(deployment.health_status)}</TableCell>
                        <TableCell>
                          <div className="text-xs space-y-1">
                            <div className="flex items-center">
                              <Cpu className="w-3 h-3 mr-1" />
                              {deployment.cpu_usage}%
                            </div>
                            <div className="flex items-center">
                              <MemoryStick className="w-3 h-3 mr-1" />
                              {deployment.memory_usage}MB
                            </div>
                            <div className="flex items-center">
                              <HardDrive className="w-3 h-3 mr-1" />
                              {deployment.disk_usage}MB
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="text-xs space-y-1">
                            <div>{deployment.requests_per_minute} req/min</div>
                            <div>{deployment.response_time_avg}ms avg</div>
                            <div className={deployment.error_rate > 5 ? "text-red-600" : ""}>
                              {deployment.error_rate}% errors
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className={deployment.uptime.startsWith("99") ? "text-green-600" : "text-yellow-600"}>
                            {deployment.uptime}
                          </span>
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
                                <FileText className="mr-2 h-4 w-4" />
                                View Logs
                              </DropdownMenuItem>
                              <DropdownMenuItem>
                                <Settings className="mr-2 h-4 w-4" />
                                Configuration
                              </DropdownMenuItem>
                              {deployment.domain && (
                                <DropdownMenuItem>
                                  <ExternalLink className="mr-2 h-4 w-4" />
                                  Open Site
                                </DropdownMenuItem>
                              )}
                              {deployment.status === "running" && (
                                <>
                                  <DropdownMenuItem>
                                    <Square className="mr-2 h-4 w-4" />
                                    Stop
                                  </DropdownMenuItem>
                                  <DropdownMenuItem>
                                    <RotateCcw className="mr-2 h-4 w-4" />
                                    Restart
                                  </DropdownMenuItem>
                                </>
                              )}
                              {deployment.status === "stopped" && (
                                <DropdownMenuItem className="text-green-600">
                                  <Play className="mr-2 h-4 w-4" />
                                  Start
                                </DropdownMenuItem>
                              )}
                              {deployment.status === "failed" && (
                                <DropdownMenuItem className="text-blue-600">
                                  <RotateCcw className="mr-2 h-4 w-4" />
                                  Retry Deployment
                                </DropdownMenuItem>
                              )}
                              <DropdownMenuItem className="text-red-600">
                                <Trash2 className="mr-2 h-4 w-4" />
                                Delete
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
                  Showing 1-{deployments.length} of {deployments.length} deployments
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

        <TabsContent value="running" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Running Deployments</CardTitle>
              <CardDescription>
                {runningDeployments.length} deployments currently running
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                All running deployments are healthy and serving traffic
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="building" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Building & Deploying</CardTitle>
              <CardDescription>
                {buildingDeployments.length} deployments in progress
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                Monitor build and deployment progress in real-time
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="failed" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Failed Deployments</CardTitle>
              <CardDescription>
                {failedDeployments.length} deployments that require attention
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                Review error logs and retry failed deployments
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}