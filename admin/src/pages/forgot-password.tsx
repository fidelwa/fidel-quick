import { useState } from "react"
import { Navigate, Link } from "react-router-dom"
import { useAuth } from "@/context/auth-context"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { OctagonX, MailCheck, Loader2 } from "lucide-react"
import { forgotPassword } from "@/lib/api-client"

export function ForgotPasswordPage() {
  const { isAuthenticated } = useAuth()
  const [email, setEmail] = useState("")
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [sent, setSent] = useState(false)

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")
    if (!email.trim()) {
      setError("Ingresa tu email")
      return
    }
    setLoading(true)
    try {
      await forgotPassword(email.trim())
      // Mensaje neutro: no revelamos si el email existe.
      setSent(true)
    } catch {
      // El backend responde 200 siempre; un fallo aquí es de red/servidor.
      setError("No pudimos procesar la solicitud. Intenta de nuevo más tarde.")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden p-4">
      {/* Glass background: gradient + soft color blobs */}
      <div className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-br from-violet-100 via-sky-50 to-pink-100 dark:from-violet-950 dark:via-slate-950 dark:to-pink-950" />
      <div className="pointer-events-none absolute -left-32 -top-32 -z-10 h-96 w-96 rounded-full bg-purple-400/40 blur-3xl dark:bg-purple-700/30" />
      <div className="pointer-events-none absolute -bottom-32 -right-32 -z-10 h-96 w-96 rounded-full bg-sky-400/40 blur-3xl dark:bg-sky-700/30" />
      <div className="pointer-events-none absolute left-1/3 top-1/2 -z-10 h-72 w-72 rounded-full bg-pink-400/30 blur-3xl dark:bg-pink-700/20" />

      <Card className="relative w-full max-w-md border-white/30 bg-white/40 shadow-2xl shadow-purple-500/10 backdrop-blur-2xl dark:border-white/10 dark:bg-white/5">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Recuperar contraseña</CardTitle>
          <CardDescription>
            {sent
              ? "Revisa tu correo"
              : "Ingresa tu email y te enviaremos un enlace para restablecerla"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          {sent ? (
            <div className="space-y-4">
              <div className="flex items-start gap-3 rounded-lg border border-green-300 bg-green-50 px-4 py-3 dark:border-green-900 dark:bg-green-950/40">
                <MailCheck className="mt-0.5 h-5 w-5 shrink-0 text-green-600" />
                <p className="text-sm text-green-900 dark:text-green-100">
                  Si el email está registrado, recibirás un enlace para
                  restablecer tu contraseña. El enlace vence en 1 hora.
                </p>
              </div>
              <Button asChild className="w-full">
                <Link to="/login">Volver a iniciar sesión</Link>
              </Button>
            </div>
          ) : (
            <>
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

              <form onSubmit={handleSubmit} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    placeholder="tu@email.com"
                    value={email}
                    onChange={(e) => { setEmail(e.target.value); setError("") }}
                  />
                </div>
                <Button type="submit" className="w-full" disabled={loading}>
                  {loading ? (
                    <>
                      <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                      Enviando...
                    </>
                  ) : (
                    "Enviar enlace"
                  )}
                </Button>
              </form>

              <p className="mt-4 text-center text-sm text-muted-foreground">
                <Link to="/login" className="text-primary underline-offset-4 hover:underline">
                  Volver a iniciar sesión
                </Link>
              </p>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
