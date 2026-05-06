import { useState } from "react"
import { Outlet, Navigate } from "react-router-dom"
import { useAuth } from "@/context/auth-context"
import { useCustomer } from "@/hooks/use-customer"
import { useOnboardingStatus } from "@/hooks/use-onboarding-status"
import { Sidebar } from "./sidebar"
import { Header } from "./header"
import { Sheet, SheetContent } from "@/components/ui/sheet"
import { Skeleton } from "@/components/ui/skeleton"

export function AppLayout() {
  const { isAuthenticated, customerId, logout } = useAuth()
  const [sidebarOpen, setSidebarOpen] = useState(false)
  const { isLoading: customerLoading, isError } = useCustomer(customerId)
  const { data: onboardingStatus, isLoading: onboardingLoading } = useOnboardingStatus(isAuthenticated)

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  if (isError) {
    logout()
    return <Navigate to="/login" replace />
  }

  if (customerLoading || onboardingLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <Skeleton className="h-12 w-48" />
      </div>
    )
  }

  if (onboardingStatus && !onboardingStatus.completed) {
    return <Navigate to="/onboarding" replace />
  }

  return (
    <div className="bg-aurora flex h-screen">
      {/* Desktop sidebar */}
      <aside className="hidden w-64 border-r border-white/40 lg:block">
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
