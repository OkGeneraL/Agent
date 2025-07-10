"use client"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { 
  Users, 
  Rocket, 
  Server, 
  DollarSign, 
  TrendingUp, 
  TrendingDown
} from "lucide-react"

interface StatCardProps {
  title: string
  value: string | number
  description: string
  icon: React.ComponentType<{ className?: string }>
  trend?: {
    value: number
    label: string
    direction: "up" | "down"
  }
  status?: "good" | "warning" | "error"
}

function StatCard({ title, value, description, icon: Icon, trend, status }: StatCardProps) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
        <CardTitle className="text-sm font-medium">{title}</CardTitle>
        <div className="flex items-center gap-2">
          {status && (
            <div 
              className={`w-2 h-2 rounded-full ${
                status === "good" 
                  ? "bg-green-500" 
                  : status === "warning" 
                  ? "bg-yellow-500" 
                  : "bg-red-500"
              }`}
            />
          )}
          <Icon className="h-4 w-4 text-muted-foreground" />
        </div>
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        <p className="text-xs text-muted-foreground">{description}</p>
        {trend && (
          <div className="flex items-center mt-2">
            {trend.direction === "up" ? (
              <TrendingUp className="h-3 w-3 text-green-500 mr-1" />
            ) : (
              <TrendingDown className="h-3 w-3 text-red-500 mr-1" />
            )}
            <span 
              className={`text-xs ${
                trend.direction === "up" ? "text-green-500" : "text-red-500"
              }`}
            >
              {trend.value}% {trend.label}
            </span>
          </div>
        )}
      </CardContent>
    </Card>
  )
}

interface StatsCardsProps {
  data?: {
    totalCustomers: number
    totalDeployments: number
    totalServers: number
    monthlyRevenue: number
    activeDeployments: number
    serverHealth: number
    customersGrowth: number
    revenueGrowth: number
  }
}

export function StatsCards({ data }: StatsCardsProps) {
  // Default data for demo purposes
  const stats = data || {
    totalCustomers: 1247,
    totalDeployments: 3891,
    totalServers: 12,
    monthlyRevenue: 89240,
    activeDeployments: 2847,
    serverHealth: 98.5,
    customersGrowth: 12.5,
    revenueGrowth: 8.3,
  }

  return (
    <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
      <StatCard
        title="Total Customers"
        value={stats.totalCustomers.toLocaleString()}
        description="Active platform users"
        icon={Users}
        trend={{
          value: stats.customersGrowth,
          label: "from last month",
          direction: "up"
        }}
        status="good"
      />
      
      <StatCard
        title="Active Deployments"
        value={stats.activeDeployments.toLocaleString()}
        description={`${stats.totalDeployments.toLocaleString()} total deployments`}
        icon={Rocket}
        trend={{
          value: 5.2,
          label: "from last week",
          direction: "up"
        }}
        status="good"
      />
      
      <StatCard
        title="Server Cluster"
        value={`${stats.totalServers} servers`}
        description={`${stats.serverHealth}% health score`}
        icon={Server}
        status={stats.serverHealth > 95 ? "good" : stats.serverHealth > 85 ? "warning" : "error"}
      />
      
      <StatCard
        title="Monthly Revenue"
        value={`$${(stats.monthlyRevenue / 1000).toFixed(1)}k`}
        description="Recurring revenue"
        icon={DollarSign}
        trend={{
          value: stats.revenueGrowth,
          label: "from last month",
          direction: "up"
        }}
        status="good"
      />
    </div>
  )
}