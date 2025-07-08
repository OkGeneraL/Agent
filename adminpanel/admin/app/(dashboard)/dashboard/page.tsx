import { StatsCards } from "@/components/dashboard/stats-cards"
import { RevenueChart } from "@/components/dashboard/revenue-chart"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { 
  Activity, 
  AlertTriangle, 
  CheckCircle, 
  Server, 
  Users,
  ExternalLink,
  Plus
} from "lucide-react"

export default function DashboardPage() {
  return (
    <div className="flex-1 space-y-4 p-4 md:p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <h2 className="text-3xl font-bold tracking-tight">Dashboard</h2>
        <div className="flex items-center space-x-2">
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Quick Deploy
          </Button>
        </div>
      </div>

      {/* Stats Cards */}
      <StatsCards />

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-7">
        {/* Revenue Chart */}
        <div className="col-span-4">
          <RevenueChart />
        </div>

        {/* Recent Activity */}
        <Card className="col-span-3">
          <CardHeader>
            <CardTitle>Recent Activity</CardTitle>
            <CardDescription>
              Latest platform events and activities
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="space-y-4">
              <div className="flex items-center space-x-4">
                <div className="flex items-center justify-center w-8 h-8 bg-green-100 dark:bg-green-900 rounded-full">
                  <CheckCircle className="w-4 h-4 text-green-600 dark:text-green-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium">Deployment Successful</p>
                  <p className="text-xs text-muted-foreground">webapp-prod deployed to us-east-1</p>
                  <p className="text-xs text-muted-foreground">2 minutes ago</p>
                </div>
              </div>

              <div className="flex items-center space-x-4">
                <div className="flex items-center justify-center w-8 h-8 bg-blue-100 dark:bg-blue-900 rounded-full">
                  <Users className="w-4 h-4 text-blue-600 dark:text-blue-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium">New Customer</p>
                  <p className="text-xs text-muted-foreground">TechCorp signed up for Enterprise plan</p>
                  <p className="text-xs text-muted-foreground">5 minutes ago</p>
                </div>
              </div>

              <div className="flex items-center space-x-4">
                <div className="flex items-center justify-center w-8 h-8 bg-orange-100 dark:bg-orange-900 rounded-full">
                  <AlertTriangle className="w-4 h-4 text-orange-600 dark:text-orange-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium">High Memory Usage</p>
                  <p className="text-xs text-muted-foreground">Server us-west-2-prod at 85% memory</p>
                  <p className="text-xs text-muted-foreground">8 minutes ago</p>
                </div>
              </div>

              <div className="flex items-center space-x-4">
                <div className="flex items-center justify-center w-8 h-8 bg-green-100 dark:bg-green-900 rounded-full">
                  <Server className="w-4 h-4 text-green-600 dark:text-green-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium">Server Added</p>
                  <p className="text-xs text-muted-foreground">eu-central-1-prod joined cluster</p>
                  <p className="text-xs text-muted-foreground">12 minutes ago</p>
                </div>
              </div>

              <div className="flex items-center space-x-4">
                <div className="flex items-center justify-center w-8 h-8 bg-purple-100 dark:bg-purple-900 rounded-full">
                  <Activity className="w-4 h-4 text-purple-600 dark:text-purple-400" />
                </div>
                <div className="flex-1 min-w-0">
                  <p className="text-sm font-medium">App Approved</p>
                  <p className="text-xs text-muted-foreground">Redis v7.0.5 approved for marketplace</p>
                  <p className="text-xs text-muted-foreground">15 minutes ago</p>
                </div>
              </div>
            </div>
            <Button variant="ghost" className="w-full mt-4">
              View All Activity
              <ExternalLink className="ml-2 h-3 w-3" />
            </Button>
          </CardContent>
        </Card>
      </div>

      <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
        {/* Active Deployments */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Deployments</CardTitle>
            <Activity className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">2,847</div>
            <p className="text-xs text-muted-foreground">+5.2% from last week</p>
            <div className="mt-2 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs">Production</span>
                <Badge variant="secondary">1,892</Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs">Staging</span>
                <Badge variant="outline">743</Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs">Development</span>
                <Badge variant="outline">212</Badge>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Server Health */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Server Health</CardTitle>
            <Server className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">98.5%</div>
            <p className="text-xs text-muted-foreground">Overall cluster health</p>
            <div className="mt-2 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs">Online</span>
                <Badge className="bg-green-500">11</Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs">Maintenance</span>
                <Badge variant="secondary">1</Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs">Offline</span>
                <Badge variant="destructive">0</Badge>
              </div>
            </div>
          </CardContent>
        </Card>

        {/* Quick Actions */}
        <Card>
          <CardHeader>
            <CardTitle className="text-sm font-medium">Quick Actions</CardTitle>
          </CardHeader>
          <CardContent className="space-y-2">
            <Button className="w-full justify-start" variant="ghost" size="sm">
              <Plus className="mr-2 h-3 w-3" />
              Add Customer
            </Button>
            <Button className="w-full justify-start" variant="ghost" size="sm">
              <Server className="mr-2 h-3 w-3" />
              Add Server
            </Button>
            <Button className="w-full justify-start" variant="ghost" size="sm">
              <Activity className="mr-2 h-3 w-3" />
              View Logs
            </Button>
            <Button className="w-full justify-start" variant="ghost" size="sm">
              <AlertTriangle className="mr-2 h-3 w-3" />
              View Alerts
            </Button>
          </CardContent>
        </Card>

        {/* System Status */}
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">System Status</CardTitle>
            <div className="w-2 h-2 bg-green-500 rounded-full"></div>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold text-green-600">Operational</div>
            <p className="text-xs text-muted-foreground">All systems running normally</p>
            <div className="mt-2 space-y-2">
              <div className="flex items-center justify-between">
                <span className="text-xs">API</span>
                <Badge className="bg-green-500">Online</Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs">Database</span>
                <Badge className="bg-green-500">Online</Badge>
              </div>
              <div className="flex items-center justify-between">
                <span className="text-xs">Storage</span>
                <Badge className="bg-green-500">Online</Badge>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}