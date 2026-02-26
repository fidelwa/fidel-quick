import { useState } from "react"
import { Outlet, Navigate } from "react-router-dom"
import { useAuth } from "@/context/auth-context"
import { Sidebar } from "./sidebar"
import { Header } from "./header"
import { Sheet, SheetContent } from "@/components/ui/sheet"

export function AppLayout() {
  const { isAuthenticated } = useAuth()
  const [sidebarOpen, setSidebarOpen] = useState(false)

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  return (
    <div className="flex h-screen">
      {/* Desktop sidebar */}
      <aside className="hidden w-64 border-r lg:block">
        <Sidebar />
      </aside>

      {/* Mobile sidebar */}
      <Sheet open={sidebarOpen} onOpenChange={setSidebarOpen}>
        <SheetContent side="left" className="w-64 p-0">
          <Sidebar onNavigate={() => setSidebarOpen(false)} />
        </SheetContent>
      </Sheet>

      {/* Main content */}
      <div className="flex flex-1 flex-col overflow-hidden">
        <Header onMenuClick={() => setSidebarOpen(true)} />
        <main className="flex-1 overflow-y-auto p-4 lg:p-6">
          <Outlet />
        </main>
      </div>
    </div>
  )
}
