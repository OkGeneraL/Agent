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
  Edit, 
  Trash2,
  Download,
  Package,
  GitBranch,
  Container,
  CheckCircle,
  XCircle,
  Clock,
  Star,
  TrendingUp
} from "lucide-react"

// Mock data for applications
const applications = [
  {
    id: "1",
    name: "React Starter",
    slug: "react-starter",
    description: "Production-ready React application with TypeScript",
    publisher: "SuperAgent Team",
    category: "Web Applications",
    version: "2.1.0",
    status: "published",
    source_type: "git",
    source_url: "https://github.com/superagent/react-starter",
    port: 3000,
    pricing_type: "free",
    download_count: 1247,
    rating: 4.8,
    rating_count: 156,
    is_featured: true,
    created_at: "2024-01-15",
    updated_at: "2024-03-20",
    deployments: 89,
  },
  {
    id: "2",
    name: "Node.js API Server",
    slug: "nodejs-api",
    description: "Express.js API server with authentication and database",
    publisher: "Community",
    category: "APIs & Microservices",
    version: "1.5.2",
    status: "published",
    source_type: "git",
    source_url: "https://github.com/community/nodejs-api",
    port: 8080,
    pricing_type: "free",
    download_count: 892,
    rating: 4.6,
    rating_count: 98,
    is_featured: false,
    created_at: "2024-02-10",
    updated_at: "2024-03-18",
    deployments: 56,
  },
  {
    id: "3",
    name: "WordPress CMS",
    slug: "wordpress",
    description: "Full WordPress installation with MySQL database",
    publisher: "SuperAgent Team",
    category: "Content Management",
    version: "6.4.0",
    status: "pending",
    source_type: "docker",
    source_url: "wordpress:6.4.0",
    port: 80,
    pricing_type: "subscription",
    price: 29,
    download_count: 423,
    rating: 4.3,
    rating_count: 67,
    is_featured: false,
    created_at: "2024-03-01",
    updated_at: "2024-03-01",
    deployments: 23,
  },
  {
    id: "4",
    name: "Redis Cache",
    slug: "redis-cache",
    description: "High-performance Redis caching server",
    publisher: "Community",
    category: "Databases",
    version: "7.0.5",
    status: "approved",
    source_type: "docker",
    source_url: "redis:7.0.5",
    port: 6379,
    pricing_type: "free",
    download_count: 756,
    rating: 4.9,
    rating_count: 234,
    is_featured: true,
    created_at: "2024-01-30",
    updated_at: "2024-03-15",
    deployments: 145,
  },
  {
    id: "5",
    name: "Analytics Dashboard",
    slug: "analytics-dashboard",
    description: "Real-time analytics dashboard with data visualization",
    publisher: "DevCorp",
    category: "Analytics",
    version: "1.0.0",
    status: "rejected",
    source_type: "git",
    source_url: "https://github.com/devcorp/analytics",
    port: 3000,
    pricing_type: "one_time",
    price: 99,
    download_count: 12,
    rating: 0,
    rating_count: 0,
    is_featured: false,
    created_at: "2024-03-10",
    updated_at: "2024-03-12",
    deployments: 2,
  },
]

// Categories for filtering (can be used for future filter implementation)
// const categories = [
//   "All Categories",
//   "Web Applications", 
//   "APIs & Microservices",
//   "Databases",
//   "Developer Tools",
//   "E-commerce",
//   "Content Management",
//   "Analytics",
//   "Communication"
// ]

function getStatusBadge(status: string) {
  switch (status) {
    case "published":
      return <Badge className="bg-green-500"><CheckCircle className="w-3 h-3 mr-1" />Published</Badge>
    case "approved":
      return <Badge className="bg-blue-500"><CheckCircle className="w-3 h-3 mr-1" />Approved</Badge>
    case "pending":
      return <Badge variant="secondary"><Clock className="w-3 h-3 mr-1" />Pending Review</Badge>
    case "rejected":
      return <Badge variant="destructive"><XCircle className="w-3 h-3 mr-1" />Rejected</Badge>
    default:
      return <Badge variant="outline">{status}</Badge>
  }
}

function getSourceIcon(sourceType: string) {
  switch (sourceType) {
    case "git":
      return <GitBranch className="w-4 h-4" />
    case "docker":
      return <Container className="w-4 h-4" />
    default:
      return <Package className="w-4 h-4" />
  }
}

function getPricingBadge(pricingType: string, price?: number) {
  switch (pricingType) {
    case "free":
      return <Badge variant="outline" className="text-green-600">Free</Badge>
    case "one_time":
      return <Badge variant="secondary">${price} One-time</Badge>
    case "subscription":
      return <Badge className="bg-purple-500">${price}/mo</Badge>
    default:
      return <Badge variant="outline">{pricingType}</Badge>
  }
}

export default function ApplicationsPage() {
  const publishedApps = applications.filter(app => app.status === "published")
  const pendingApps = applications.filter(app => app.status === "pending")
  const totalDownloads = applications.reduce((sum, app) => sum + app.download_count, 0)
  const totalDeployments = applications.reduce((sum, app) => sum + app.deployments, 0)

  return (
    <div className="flex-1 space-y-4 p-4 md:p-8 pt-6">
      <div className="flex items-center justify-between space-y-2">
        <div>
          <h2 className="text-3xl font-bold tracking-tight">Applications</h2>
          <p className="text-muted-foreground">
            Manage your application catalog and deployment marketplace
          </p>
        </div>
        <div className="flex items-center space-x-2">
          <Button variant="outline">
            <Download className="mr-2 h-4 w-4" />
            Export Catalog
          </Button>
          <Button>
            <Plus className="mr-2 h-4 w-4" />
            Add Application
          </Button>
        </div>
      </div>

      {/* Summary Cards */}
      <div className="grid gap-4 md:grid-cols-4">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Applications</CardTitle>
            <Package className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{applications.length}</div>
            <p className="text-xs text-muted-foreground">
              {publishedApps.length} published
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Pending Review</CardTitle>
            <Clock className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{pendingApps.length}</div>
            <p className="text-xs text-muted-foreground">
              Awaiting approval
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Total Downloads</CardTitle>
            <Download className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalDownloads.toLocaleString()}</div>
            <p className="text-xs text-muted-foreground">
              +12% from last month
            </p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
            <CardTitle className="text-sm font-medium">Active Deployments</CardTitle>
            <TrendingUp className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalDeployments}</div>
            <p className="text-xs text-muted-foreground">
              Across all customers
            </p>
          </CardContent>
        </Card>
      </div>

      {/* Applications Management */}
      <Tabs defaultValue="all" className="w-full">
        <TabsList>
          <TabsTrigger value="all">All Applications</TabsTrigger>
          <TabsTrigger value="published">Published</TabsTrigger>
          <TabsTrigger value="pending">Pending Review</TabsTrigger>
          <TabsTrigger value="featured">Featured</TabsTrigger>
        </TabsList>

        <TabsContent value="all" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Application Catalog</CardTitle>
              <CardDescription>
                Complete list of applications available for deployment
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center space-x-2 mb-4">
                <div className="relative flex-1">
                  <Search className="absolute left-2.5 top-2.5 h-4 w-4 text-muted-foreground" />
                  <Input
                    placeholder="Search applications..."
                    className="pl-8"
                  />
                </div>
                <Button variant="outline">
                  <Filter className="mr-2 h-4 w-4" />
                  Filter
                </Button>
              </div>

              <div className="rounded-md border">
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Application</TableHead>
                      <TableHead>Publisher</TableHead>
                      <TableHead>Category</TableHead>
                      <TableHead>Version</TableHead>
                      <TableHead>Source</TableHead>
                      <TableHead>Status</TableHead>
                      <TableHead>Pricing</TableHead>
                      <TableHead>Downloads</TableHead>
                      <TableHead>Rating</TableHead>
                      <TableHead>Deployments</TableHead>
                      <TableHead className="text-right">Actions</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {applications.map((app) => (
                      <TableRow key={app.id}>
                        <TableCell>
                          <div className="flex items-center space-x-2">
                            {app.is_featured && (
                              <Star className="h-4 w-4 text-yellow-500 fill-current" />
                            )}
                            <div>
                              <div className="font-medium">{app.name}</div>
                              <div className="text-sm text-muted-foreground">
                                {app.description.substring(0, 50)}...
                              </div>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>{app.publisher}</TableCell>
                        <TableCell>{app.category}</TableCell>
                        <TableCell>
                          <Badge variant="outline">{app.version}</Badge>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center space-x-1">
                            {getSourceIcon(app.source_type)}
                            <span className="text-sm capitalize">{app.source_type}</span>
                          </div>
                        </TableCell>
                        <TableCell>{getStatusBadge(app.status)}</TableCell>
                        <TableCell>{getPricingBadge(app.pricing_type, app.price)}</TableCell>
                        <TableCell>{app.download_count.toLocaleString()}</TableCell>
                        <TableCell>
                          <div className="flex items-center space-x-1">
                            <Star className="h-3 w-3 text-yellow-500 fill-current" />
                            <span className="text-sm">{app.rating}</span>
                            <span className="text-xs text-muted-foreground">
                              ({app.rating_count})
                            </span>
                          </div>
                        </TableCell>
                        <TableCell>{app.deployments}</TableCell>
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
                                Edit Application
                              </DropdownMenuItem>
                              {app.status === "pending" && (
                                <>
                                  <DropdownMenuItem className="text-green-600">
                                    <CheckCircle className="mr-2 h-4 w-4" />
                                    Approve
                                  </DropdownMenuItem>
                                  <DropdownMenuItem className="text-red-600">
                                    <XCircle className="mr-2 h-4 w-4" />
                                    Reject
                                  </DropdownMenuItem>
                                </>
                              )}
                              <DropdownMenuItem>
                                <Package className="mr-2 h-4 w-4" />
                                View Deployments
                              </DropdownMenuItem>
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
                  Showing 1-{applications.length} of {applications.length} applications
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

        <TabsContent value="published" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Published Applications</CardTitle>
              <CardDescription>
                Applications that are live and available for deployment
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                {publishedApps.length} published applications available for deployment
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="pending" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Pending Review</CardTitle>
              <CardDescription>
                Applications waiting for admin approval
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                {pendingApps.length} applications awaiting review
              </div>
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="featured" className="space-y-4">
          <Card>
            <CardHeader>
              <CardTitle>Featured Applications</CardTitle>
              <CardDescription>
                Highlighted applications promoted on the marketplace
              </CardDescription>
            </CardHeader>
            <CardContent>
              <div className="text-sm text-muted-foreground">
                {applications.filter(app => app.is_featured).length} featured applications
              </div>
            </CardContent>
          </Card>
        </TabsContent>
      </Tabs>
    </div>
  )
}