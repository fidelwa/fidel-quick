import { BrowserRouter, Routes, Route, Navigate } from "react-router-dom"
import { QueryClient, QueryClientProvider } from "@tanstack/react-query"
import { GoogleOAuthProvider } from "@react-oauth/google"
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
import { PushcardPage } from "@/pages/pushcard/pushcard-page"
import { CollaboratorsListPage } from "@/pages/collaborators/collaborators-list"
import { ClientLookupPage } from "@/pages/clients/client-lookup"
import { FeedbackListPage } from "@/pages/feedback/feedback-list"
import { RegistroPage } from "@/pages/registro"
import { OnboardingLayout } from "@/pages/onboarding/onboarding-layout"

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 30_000,
      retry: 1,
    },
  },
})

const GOOGLE_CLIENT_ID = import.meta.env.VITE_GOOGLE_CLIENT_ID || ""

function GoogleWrapper({ children }: { children: React.ReactNode }) {
  if (!GOOGLE_CLIENT_ID) return <>{children}</>
  return <GoogleOAuthProvider clientId={GOOGLE_CLIENT_ID}>{children}</GoogleOAuthProvider>
}

function App() {
  return (
    <GoogleWrapper>
    <QueryClientProvider client={queryClient}>
      <AuthProvider>
        <BrowserRouter basename={import.meta.env.BASE_URL.replace(/\/$/, "")}>
          <Routes>
            <Route path="/login" element={<LoginPage />} />
            <Route path="/registro" element={<RegistroPage />} />
            <Route path="/onboarding" element={<OnboardingLayout />} />
            <Route element={<AppLayout />}>
              <Route index element={<DashboardPage />} />
              <Route path="perfil" element={<ProfilePage />} />
              <Route path="programas" element={<ProgramsListPage />} />
              <Route path="programas/:id" element={<ProgramDetailPage />} />
              <Route path="cashback" element={<CashbackListPage />} />
              <Route path="cashback/:id" element={<CashbackDetailPage />} />
              <Route path="pushcard" element={<PushcardPage />} />
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
    </GoogleWrapper>
  )
}

export default App
