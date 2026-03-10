import { useState } from "react"
import { Outlet, Navigate } from "react-router-dom"
import { useAuth } from "@/context/auth-context"
import { useCustomer } from "@/hooks/use-customer"
import { Sidebar } from "./sidebar"
import { Header } from "./header"
import { Sheet, SheetContent } from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"

export function AppLayout() {
  const { isAuthenticated, customerId } = useAuth()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const { data: customer, isLoading: customerLoading } = useCustomer(customerId)

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  if (customerLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Skeleton className="h-12 w-48" />
      </div>
    )
  }

  if (customer && !customer.onboarding_completed) {
    return <Navigate to="/onboarding" replace />
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
