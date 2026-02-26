import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { Toaster } from "@/components/ui/sonner"
import { AuthProvider } from "@/context/auth-context"
import { AppLayout } from "@/components/layout/app-layout"
import { LoginPage } from "@/pages/login"
import { DashboardPage } from "@/pages/dashboard"
import { ProfilePage } from "@/pages/profile"
import { ProgramsListPage } from "@/pages/programs/programs-list"
import { ProgramDetailPage } from "@/pages/programs/program-detail"
import { CashbackListPage } from "@/pages/cashback/cashback-list"
import { CashbackDetailPage } from "@/pages/cashback/cashback-detail"
import { CollaboratorsListPage } from "@/pages/collaborators/collaborators-list"
import { ClientLookupPage } from "@/pages/clients/client-lookup"
import { FeedbackListPage } from "@/pages/feedback/feedback-list"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

function App() {
  return (
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route element={<AppLayout />}>
              <Route index element={<DashboardPage />} />
              <Route path="perfil" element={<ProfilePage />} />
              <Route path="programas" element={<ProgramsListPage />} />
              <Route path="programas/:id" element={<ProgramDetailPage />} />
              <Route path="cashback" element={<CashbackListPage />} />
              <Route path="cashback/:id" element={<CashbackDetailPage />} />
              <Route path="colaboradores" element={<CollaboratorsListPage />} />
              <Route path="clientes" element={<ClientLookupPage />} />
              <Route path="feedback" element={<FeedbackListPage />} />
            </Route>
            <Route path="*" element={<Navigate to="/" replace />} />
          </Routes>
        </BrowserRouter>
        <Toaster />
      </AuthProvider>
    </QueryClientProvider>
  )
}

export default App
