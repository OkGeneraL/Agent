"use client"

import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { 
  ResponsiveContainer, 
  AreaChart, 
  Area, 
  XAxis, 
  YAxis, 
  CartesianGrid, 
  Tooltip, 
  Legend 
} from "recharts"

interface RevenueData {
  month: string
  revenue: number
  previousYear: number
}

interface RevenueChartProps {
  data?: RevenueData[]
}

export function RevenueChart({ data }: RevenueChartProps) {
  // Default data for demo purposes
  const defaultData: RevenueData[] = [
    { month: "Jan", revenue: 65400, previousYear: 54300 },
    { month: "Feb", revenue: 71200, previousYear: 58900 },
    { month: "Mar", revenue: 68900, previousYear: 61200 },
    { month: "Apr", revenue: 75600, previousYear: 64800 },
    { month: "May", revenue: 82300, previousYear: 68700 },
    { month: "Jun", revenue: 78900, previousYear: 72100 },
    { month: "Jul", revenue: 85400, previousYear: 75300 },
    { month: "Aug", revenue: 91200, previousYear: 78900 },
    { month: "Sep", revenue: 87600, previousYear: 81400 },
    { month: "Oct", revenue: 94800, previousYear: 84200 },
    { month: "Nov", revenue: 89200, previousYear: 87600 },
    { month: "Dec", revenue: 96500, previousYear: 89300 },
  ]

  const chartData = data || defaultData

  const formatCurrency = (value: number) => `$${(value / 1000).toFixed(0)}k`

  return (
    <Card className="col-span-4">
      <CardHeader>
        <CardTitle>Revenue Overview</CardTitle>
        <CardDescription>
          Monthly recurring revenue compared to previous year
        </CardDescription>
      </CardHeader>
      <CardContent>
        <ResponsiveContainer width="100%" height={350}>
          <AreaChart data={chartData}>
            <defs>
              <linearGradient id="colorRevenue" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="hsl(var(--primary))" stopOpacity={0.8}/>
                <stop offset="95%" stopColor="hsl(var(--primary))" stopOpacity={0}/>
              </linearGradient>
              <linearGradient id="colorPrevious" x1="0" y1="0" x2="0" y2="1">
                <stop offset="5%" stopColor="hsl(var(--muted-foreground))" stopOpacity={0.8}/>
                <stop offset="95%" stopColor="hsl(var(--muted-foreground))" stopOpacity={0}/>
              </linearGradient>
            </defs>
            <CartesianGrid strokeDasharray="3 3" className="stroke-muted" />
            <XAxis 
              dataKey="month" 
              axisLine={false}
              tickLine={false}
              className="text-xs fill-muted-foreground"
            />
            <YAxis 
              axisLine={false}
              tickLine={false}
              className="text-xs fill-muted-foreground"
              tickFormatter={formatCurrency}
            />
            <Tooltip 
              content={({ active, payload, label }) => {
                if (active && payload && payload.length) {
                  return (
                    <div className="rounded-lg border bg-background p-2 shadow-sm">
                      <div className="grid grid-cols-2 gap-2">
                        <div className="flex flex-col">
                          <span className="text-[0.70rem] uppercase text-muted-foreground">
                            {label}
                          </span>
                          <span className="font-bold text-muted-foreground">
                            Current Year
                          </span>
                          <span className="font-bold">
                            {formatCurrency(payload[0].value as number)}
                          </span>
                        </div>
                        <div className="flex flex-col">
                          <span className="text-[0.70rem] uppercase text-muted-foreground">
                            Previous Year
                          </span>
                          <span className="font-bold">
                            {formatCurrency(payload[1].value as number)}
                          </span>
                        </div>
                      </div>
                    </div>
                  )
                }
                return null
              }}
            />
            <Legend />
            <Area
              type="monotone"
              dataKey="revenue"
              stroke="hsl(var(--primary))"
              fillOpacity={1}
              fill="url(#colorRevenue)"
              name="Current Year"
            />
            <Area
              type="monotone"
              dataKey="previousYear"
              stroke="hsl(var(--muted-foreground))"
              fillOpacity={1}
              fill="url(#colorPrevious)"
              name="Previous Year"
            />
          </AreaChart>
        </ResponsiveContainer>
      </CardContent>
    </Card>
  )
}