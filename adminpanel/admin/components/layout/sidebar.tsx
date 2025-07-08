"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import { cn } from "@/lib/utils"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import {
  LayoutDashboard,
  Users,
  Package,
  Rocket,
  Server,
  Globe,
  CreditCard,
  BarChart3,
  Shield,
  Settings,
  HelpCircle,
  Building2,
  Database,
  Activity,
  Bell,
  Lock,
  UserCheck,
  FileText,
  Zap,
  Cloud
} from "lucide-react"

interface NavItem {
  title: string
  href: string
  icon: React.ComponentType<{ className?: string }>
  badge?: string
  children?: NavItem[]
}

const navItems: NavItem[] = [
  {
    title: "Dashboard",
    href: "/dashboard",
    icon: LayoutDashboard,
  },
  {
    title: "Customers",
    href: "/dashboard/customers",
    icon: Users,
    children: [
      { title: "All Customers", href: "/dashboard/customers", icon: Users },
      { title: "Add Customer", href: "/dashboard/customers/new", icon: Users },
    ]
  },
  {
    title: "Applications",
    href: "/dashboard/applications",
    icon: Package,
    children: [
      { title: "App Catalog", href: "/dashboard/applications", icon: Package },
      { title: "Add Application", href: "/dashboard/applications/new", icon: Package },
      { title: "Publishers", href: "/dashboard/applications/publishers", icon: Building2 },
    ]
  },
  {
    title: "Deployments",
    href: "/dashboard/deployments",
    icon: Rocket,
    badge: "Live",
    children: [
      { title: "Active Deployments", href: "/dashboard/deployments", icon: Rocket },
      { title: "Queue", href: "/dashboard/deployments/queue", icon: Activity },
    ]
  },
  {
    title: "Servers",
    href: "/dashboard/servers",
    icon: Server,
    children: [
      { title: "Server Cluster", href: "/dashboard/servers", icon: Server },
      { title: "Add Server", href: "/dashboard/servers/add", icon: Server },
      { title: "Load Balancer", href: "/dashboard/servers/load-balancer", icon: Zap },
    ]
  },
  {
    title: "Domains",
    href: "/dashboard/domains",
    icon: Globe,
    children: [
      { title: "Domain Management", href: "/dashboard/domains", icon: Globe },
      { title: "Subdomains", href: "/dashboard/domains/subdomains", icon: Globe },
      { title: "SSL Certificates", href: "/dashboard/domains/certificates", icon: Lock },
    ]
  },
  {
    title: "Billing",
    href: "/dashboard/billing",
    icon: CreditCard,
    children: [
      { title: "Overview", href: "/dashboard/billing", icon: CreditCard },
      { title: "Invoices", href: "/dashboard/billing/invoices", icon: FileText },
      { title: "Payments", href: "/dashboard/billing/payments", icon: CreditCard },
      { title: "Plans", href: "/dashboard/billing/plans", icon: Package },
    ]
  },
  {
    title: "Analytics",
    href: "/dashboard/analytics",
    icon: BarChart3,
    children: [
      { title: "Overview", href: "/dashboard/analytics", icon: BarChart3 },
      { title: "Customers", href: "/dashboard/analytics/customers", icon: Users },
      { title: "Revenue", href: "/dashboard/analytics/revenue", icon: CreditCard },
      { title: "Usage", href: "/dashboard/analytics/usage", icon: Activity },
      { title: "Reports", href: "/dashboard/analytics/reports", icon: FileText },
    ]
  },
  {
    title: "Security",
    href: "/dashboard/security",
    icon: Shield,
    children: [
      { title: "Overview", href: "/dashboard/security", icon: Shield },
      { title: "Admin Users", href: "/dashboard/security/users", icon: UserCheck },
      { title: "Audit Logs", href: "/dashboard/security/audit", icon: FileText },
      { title: "Compliance", href: "/dashboard/security/compliance", icon: Shield },
      { title: "Incidents", href: "/dashboard/security/incidents", icon: Bell },
    ]
  },
  {
    title: "Settings",
    href: "/dashboard/settings",
    icon: Settings,
    children: [
      { title: "General", href: "/dashboard/settings/general", icon: Settings },
      { title: "Integrations", href: "/dashboard/settings/integrations", icon: Cloud },
      { title: "Notifications", href: "/dashboard/settings/notifications", icon: Bell },
      { title: "API", href: "/dashboard/settings/api", icon: Database },
      { title: "Backup", href: "/dashboard/settings/backup", icon: Database },
    ]
  },
  {
    title: "Support",
    href: "/dashboard/support",
    icon: HelpCircle,
    children: [
      { title: "Overview", href: "/dashboard/support", icon: HelpCircle },
      { title: "Tickets", href: "/dashboard/support/tickets", icon: FileText },
      { title: "Documentation", href: "/dashboard/support/documentation", icon: FileText },
      { title: "System Status", href: "/dashboard/support/system-status", icon: Activity },
    ]
  },
]

interface SidebarProps {
  className?: string
}

export function Sidebar({ className }: SidebarProps) {
  const pathname = usePathname()

  return (
    <div className={cn("pb-12 w-64", className)}>
      <div className="space-y-4 py-4">
        <div className="px-3 py-2">
          <div className="mb-2 px-4 text-lg font-semibold tracking-tight">
            SuperAgent PaaS
          </div>
          <div className="space-y-1">
            {navItems.map((item) => (
              <div key={item.href}>
                <Button
                  variant={pathname === item.href ? "secondary" : "ghost"}
                  className={cn(
                    "w-full justify-start",
                    pathname === item.href && "bg-secondary"
                  )}
                  asChild
                >
                  <Link href={item.href}>
                    <item.icon className="mr-2 h-4 w-4" />
                    {item.title}
                    {item.badge && (
                      <Badge variant="secondary" className="ml-auto">
                        {item.badge}
                      </Badge>
                    )}
                  </Link>
                </Button>
                
                {/* Render children if they exist and parent is active */}
                {item.children && pathname.startsWith(item.href) && (
                  <div className="ml-4 mt-1 space-y-1">
                    {item.children.map((child) => (
                      <Button
                        key={child.href}
                        variant={pathname === child.href ? "secondary" : "ghost"}
                        size="sm"
                        className={cn(
                          "w-full justify-start",
                          pathname === child.href && "bg-secondary"
                        )}
                        asChild
                      >
                        <Link href={child.href}>
                          <child.icon className="mr-2 h-3 w-3" />
                          {child.title}
                        </Link>
                      </Button>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  )
}