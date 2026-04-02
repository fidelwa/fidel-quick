import { useState, useEffect } from "react"
import { useNavigate, Navigate, Link } from "react-router-dom"
import { GoogleLogin, type CredentialResponse } from "@react-oauth/google"
import { useAuth } from "@/context/auth-context"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { toast } from "sonner"
import { OctagonX } from "lucide-react"
import { loginAdmin, loginGoogle, setToken, getOnboarding } from "@/lib/api-client"

export function LoginPage() {
  const { isAuthenticated, login } = useAuth()
  const navigate = useNavigate()
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [shake, setShake] = useState(false)

  useEffect(() => {
    if (shake) {
      const t = setTimeout(() => setShake(false), 500)
      return () => clearTimeout(t)
    }
  }, [shake])

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  const showError = (msg: string) => {
    setError(msg)
    setShake(true)
  }

  const navigateAfterLogin = async (token: string, customerId: string, adminEmail: string) => {
    setToken(token)
    login(token, customerId, adminEmail)
    try {
      const onboardingStatus = await getOnboarding()
      navigate(onboardingStatus.completed ? "/" : "/onboarding")
    } catch {
      navigate("/onboarding")
    }
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    if (!email.trim() || !password.trim()) {
      showError("Completa todos los campos")
      return
    }

    setLoading(true)
    try {
      const res = await loginAdmin(email.trim(), password)
      await navigateAfterLogin(res.token, res.admin.customer_id, res.admin.email)
    } catch {
      setToken("")
      showError("Credenciales invalidas. Revisa tu email y password.")
    } finally {
      setLoading(false)
    }
  }

  const handleGoogleLogin = async (response: CredentialResponse) => {
    if (!response.credential) return
    setError("")
    setLoading(true)
    try {
      const res = await loginGoogle(response.credential)
      await navigateAfterLogin(res.token, res.admin.customer_id, res.admin.email)
    } catch {
      showError("No se encontro una cuenta con ese email de Google")
    } finally {
      setLoading(false)
    }
  }

  const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className={`w-full max-w-md transition-transform ${shake ? "animate-[shake_0.4s_ease-in-out]" : ""}`}>
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Fidel Admin</CardTitle>
          <CardDescription>
            Ingresa tu email y password para acceder al panel
          </CardDescription>
        </CardHeader>
        <CardContent>
          {/* Inline error banner */}
          <div
            className={`grid transition-all duration-300 ease-in-out ${error ? "grid-rows-[1fr] opacity-100 mb-4" : "grid-rows-[0fr] opacity-0 mb-0"}`}
          >
            <div className="overflow-hidden">
              <div className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2.5">
                <OctagonX className="h-4 w-4 shrink-0 text-destructive" />
                <p className="text-sm text-destructive">{error}</p>
              </div>
            </div>
          </div>

          {googleClientId && (
            <>
              <div className="flex justify-center">
                <GoogleLogin
                  onSuccess={handleGoogleLogin}
                  onError={() => toast.error("Error al conectar con Google")}
                  text="signin_with"
                  shape="rectangular"
                  size="large"
                  width={380}
                />
              </div>
              <div className="relative my-4">
                <div className="absolute inset-0 flex items-center">
                  <span className="w-full border-t" />
                </div>
                <div className="relative flex justify-center text-xs uppercase">
                  <span className="bg-card px-2 text-muted-foreground">o</span>
                </div>
              </div>
            </>
          )}

          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="email">Email</Label>
              <Input
                id="email"
                type="email"
                placeholder="tu@email.com"
                value={email}
                onChange={(e) => { setEmail(e.target.value); setError("") }}
                className={error ? "border-destructive/50 focus-visible:ring-destructive/30" : ""}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="password">Password</Label>
              <Input
                id="password"
                type="password"
                placeholder="Tu password"
                value={password}
                onChange={(e) => { setPassword(e.target.value); setError("") }}
                className={error ? "border-destructive/50 focus-visible:ring-destructive/30" : ""}
              />
            </div>
            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? "Verificando..." : "Iniciar sesion"}
            </Button>
          </form>

          <p className="mt-4 text-center text-sm text-muted-foreground">
            ¿No tienes cuenta?{" "}
            <Link to="/registro" className="text-primary underline-offset-4 hover:underline">
              Registrate
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
