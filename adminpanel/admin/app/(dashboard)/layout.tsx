import { Sidebar } from "@/components/layout/sidebar"
import { Header } from "@/components/layout/header"

export default function DashboardLayout({
  children,
}: {
  children: React.ReactNode
}) {
  return (
    <div className="flex h-screen bg-background">
      {/* Sidebar */}
      <aside className="hidden md:flex md:flex-col md:fixed md:inset-y-0 z-50 w-64 bg-background border-r">
        <Sidebar />
      </aside>

      {/* Main content */}
      <div className="flex flex-col flex-1 md:pl-64">
        <Header />
        <main className="flex-1 overflow-y-auto p-6">
          {children}
        </main>
      </div>
    </div>
  )
}