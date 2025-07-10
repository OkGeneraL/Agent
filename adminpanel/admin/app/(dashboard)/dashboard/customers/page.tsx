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
  Plus, 
  Search, 
  Filter, 
  MoreHorizontal, 
  Eye, 
  Edit, 
  Trash2,
  Download,
  Users,
  Building2,
  DollarSign
} from "lucide-react"

// Mock data for demonstration
const customers = [
  {
    id: "1",
    name: "TechCorp Inc.",
    email: "admin@techcorp.com",
    company: "TechCorp Inc.",
    plan: "Enterprise",
    status: "active",
    created_at: "2024-01-15",
    deployments: 24,
    revenue: 2400,
  },
  {
    id: "2",
    name: "StartupXYZ",
    email: "founder@startupxyz.com",
    company: "StartupXYZ",
    plan: "Pro",
    status: "active",
    created_at: "2024-02-20",
    deployments: 8,
    revenue: 199,
  },
  {
    id: "3",
    name: "DevTeam Solutions",
    email: "team@devteam.io",
    company: "DevTeam Solutions",
    plan: "Starter",
    status: "suspended",
    created_at: "2024-01-08",
    deployments: 3,
    revenue: 49,
  },
  {
    id: "4",
    name: "Enterprise Corp",
    email: "it@enterprise.com",
    company: "Enterprise Corp",
    plan: "Enterprise",
    status: "active",
    created_at: "2023-12-10",
    deployments: 156,
    revenue: 4800,
  },
  {
    id: "5",
    name: "Indie Developer",
    email: "john@indie.dev",
    company: "Freelancer",
    plan: "Free",
    status: "active",
    created_at: "2024-03-01",
    deployments: 2,
    revenue: 0,
  },
]

function getStatusBadge(status: string) {
  switch (status) {
    case "active":
      return <Badge className="bg-green-500">Active</Badge>
    case "suspended":
      return <Badge variant="destructive">Suspended</Badge>
    case "cancelled":
      return <Badge variant="outline">Cancelled</Badge>
    default:
      return <Badge variant="secondary">{status}</Badge>
  }
}

function getPlanBadge(plan: string) {
  switch (plan) {
    case "Enterprise":
      return <Badge className="bg-purple-500">Enterprise</Badge>
    case "Pro":
      return <Badge className="bg-blue-500">Pro</Badge>
    case "Starter":
      return <Badge className="bg-orange-500">Starter</Badge>
    case "Free":
      return <Badge variant="outline">Free</Badge>
    default:
      return <Badge variant="secondary">{plan}</Badge>
  }
}

export default function CustomersPage() {
  return (
    <div className="flex-1 space-y-4 p-4 md:p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Customers</h2>
          <p className="text-muted-foreground">
            Manage your platform customers and their subscriptions
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline">
            <Download className="mr-2 h-4 w-4" />
            Export
          </Button>
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Add Customer
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Customers</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{customers.length}</div>
            <p className="text-xs text-muted-foreground">
              +12% from last month
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Enterprise Customers</CardTitle>
            <Building2 className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              {customers.filter(c => c.plan === "Enterprise").length}
            </div>
            <p className="text-xs text-muted-foreground">
              {Math.round((customers.filter(c => c.plan === "Enterprise").length / customers.length) * 100)}% of total
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Monthly Revenue</CardTitle>
            <DollarSign className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">
              ${customers.reduce((sum, c) => sum + c.revenue, 0).toLocaleString()}
            </div>
            <p className="text-xs text-muted-foreground">
              +8% from last month
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Filters and Search */}
      <Card>
        <CardHeader>
          <CardTitle>Customer Directory</CardTitle>
          <CardDescription>
            A list of all customers with their subscription details and status
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex items-center space-x-2 mb-4">
            <div className="relative flex-1">
              <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="Search customers..."
                className="pl-8"
              />
            </div>
            <Button variant="outline">
              <Filter className="mr-2 h-4 w-4" />
              Filter
            </Button>
          </div>

          {/* Customers Table */}
          <div className="rounded-md border">
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Customer</TableHead>
                  <TableHead>Plan</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead>Deployments</TableHead>
                  <TableHead>Revenue</TableHead>
                  <TableHead>Joined</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {customers.map((customer) => (
                  <TableRow key={customer.id}>
                    <TableCell>
                      <div className="flex flex-col">
                        <div className="font-medium">{customer.name}</div>
                        <div className="text-sm text-muted-foreground">{customer.email}</div>
                        {customer.company && customer.company !== customer.name && (
                          <div className="text-xs text-muted-foreground">{customer.company}</div>
                        )}
                      </div>
                    </TableCell>
                    <TableCell>
                      {getPlanBadge(customer.plan)}
                    </TableCell>
                    <TableCell>
                      {getStatusBadge(customer.status)}
                    </TableCell>
                    <TableCell>
                      <div className="font-medium">{customer.deployments}</div>
                    </TableCell>
                    <TableCell>
                      <div className="font-medium">${customer.revenue}/mo</div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">
                        {new Date(customer.created_at).toLocaleDateString()}
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
                            <Edit className="mr-2 h-4 w-4" />
                            Edit Customer
                          </DropdownMenuItem>
                          <DropdownMenuItem>
                            <DollarSign className="mr-2 h-4 w-4" />
                            Billing
                          </DropdownMenuItem>
                          <DropdownMenuItem className="text-red-600">
                            <Trash2 className="mr-2 h-4 w-4" />
                            Delete Customer
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </div>

          {/* Pagination */}
          <div className="flex items-center justify-between space-x-2 py-4">
            <div className="text-sm text-muted-foreground">
              Showing 1-{customers.length} of {customers.length} customers
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
    </div>
  )
}