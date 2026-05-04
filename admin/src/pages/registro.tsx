import { useState } from "react"
import { useNavigate, Navigate, Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { GoogleLogin, type CredentialResponse } from "@react-oauth/google"
import { useAuth } from "@/context/auth-context"
import { onboardingRegister, onboardingGoogle, setToken } from "@/lib/api-client"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { toast } from "sonner"
import { Loader2, Check, X } from "lucide-react"
import { COUNTRY_CODES } from "@/lib/country-codes"

const registroSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  country_code: z.string().min(1, "Selecciona un pais"),
  phone: z.string().min(7, "Minimo 7 digitos").regex(/^\d+$/, "Solo numeros"),
  description: z.string().optional(),
  admin_email: z.string().optional(),
  admin_password: z.string().optional(),
  confirm_password: z.string().optional(),
})

type RegistroForm = z.infer<typeof registroSchema>

function getPasswordStrength(password: string) {
  let score = 0
  if (password.length >= 8) score++
  if (/[a-z]/.test(password)) score++
  if (/[A-Z]/.test(password)) score++
  if (/[0-9]/.test(password)) score++
  if (/[^a-zA-Z0-9]/.test(password)) score++

  const levels = [
    { label: "", color: "" },
    { label: "Muy debil", color: "#ef4444" },
    { label: "Debil", color: "#f97316" },
    { label: "Aceptable", color: "#eab308" },
    { label: "Fuerte", color: "#22c55e" },
    { label: "Muy fuerte", color: "#16a34a" },
  ]

  return { score, ...levels[score] }
}

export function RegistroPage() {
  const { isAuthenticated, login } = useAuth()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const [googleCredential, setGoogleCredential] = useState<string | null>(null)
  const [googleEmail, setGoogleEmail] = useState<string | null>(null)

  const form = useForm<RegistroForm>({
    resolver: zodResolver(registroSchema),
    defaultValues: {
      name: "",
      country_code: "+52",
      phone: "",
      description: "",
      admin_email: "",
      admin_password: "",
      confirm_password: "",
    },
  })

  const watchedPassword = form.watch("admin_password") || ""
  const strength = getPasswordStrength(watchedPassword)

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  const handleGoogleSuccess = (response: CredentialResponse) => {
    if (!response.credential) return
    try {
      const payload = JSON.parse(atob(response.credential.split(".")[1]))
      setGoogleCredential(response.credential)
      setGoogleEmail(payload.email)
    } catch {
      toast.error("Error al procesar la respuesta de Google")
    }
  }

  const handleSubmit = async () => {
    const businessValid = await form.trigger(["name", "country_code", "phone"])
    if (!businessValid) return

    const data = form.getValues()
    const fullPhone = data.country_code + data.phone

    if (googleCredential) {
      setLoading(true)
      try {
        const res = await onboardingGoogle({
          google_token: googleCredential,
          name: data.name,
          phone: fullPhone,
          description: data.description,
        })
        setToken(res.token)
        login(res.token, res.admin.customer_id, res.admin.email)
        navigate("/onboarding")
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Error al registrar con Google")
      } finally {
        setLoading(false)
      }
    } else {
      if (!data.admin_email || !/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(data.admin_email)) {
        form.setError("admin_email", { message: "Email invalido" })
        return
      }
      if (!data.admin_password || data.admin_password.length < 8) {
        form.setError("admin_password", { message: "Minimo 8 caracteres" })
        return
      }
      if (data.admin_password !== data.confirm_password) {
        form.setError("confirm_password", { message: "Las contraseñas no coinciden" })
        return
      }

      setLoading(true)
      try {
        const res = await onboardingRegister({
          name: data.name,
          phone: fullPhone,
          country_code: data.country_code,
          description: data.description,
          admin_email: data.admin_email,
          admin_password: data.admin_password,
        })
        setToken(res.token)
        login(res.token, res.admin.customer_id, res.admin.email)
        navigate("/onboarding")
      } catch (err) {
        toast.error(err instanceof Error ? err.message : "Error al registrar")
      } finally {
        setLoading(false)
      }
    }
  }

  const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-lg">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Crea tu programa de fidelidad</CardTitle>
          <CardDescription>
            Registra tu negocio y configura todo en minutos
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={(e) => { e.preventDefault(); handleSubmit() }} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Nombre del negocio</Label>
              <Input
                id="name"
                placeholder="Mi Restaurante"
                {...form.register("name")}
              />
              {form.formState.errors.name && (
                <p className="text-sm text-destructive">{form.formState.errors.name.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label>Telefono del negocio</Label>
              <div className="flex gap-2">
                <select
                  className="flex h-9 w-[180px] shrink-0 rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
                  {...form.register("country_code")}
                >
                  {COUNTRY_CODES.map((c) => (
                    <option key={c.country} value={c.code}>
                      {c.label}
                    </option>
                  ))}
                </select>
                <Input
                  id="phone"
                  placeholder="5512345678"
                  className="flex-1"
                  {...form.register("phone", {
                    onChange: (e) => {
                      e.target.value = e.target.value.replace(/\D/g, "")
                    },
                  })}
                />
              </div>
              {form.formState.errors.phone && (
                <p className="text-sm text-destructive">{form.formState.errors.phone.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Descripcion (opcional)</Label>
              <Textarea
                id="description"
                placeholder="Describe brevemente tu negocio..."
                rows={3}
                {...form.register("description")}
              />
            </div>

            {/* Separator */}
            <div className="relative py-2">
              <div className="absolute inset-0 flex items-center">
                <span className="w-full border-t" />
              </div>
              <div className="relative flex justify-center text-xs uppercase">
                <span className="bg-card px-2 text-muted-foreground">Cuenta de administrador</span>
              </div>
            </div>

            {/* Auth section */}
            {googleCredential ? (
              <div className="flex items-center justify-between rounded-md border p-3">
                <div className="flex items-center gap-2">
                  <Check className="h-4 w-4 text-green-500" />
                  <span className="text-sm">{googleEmail}</span>
                </div>
                <button
                  type="button"
                  className="text-muted-foreground hover:text-foreground"
                  onClick={() => { setGoogleCredential(null); setGoogleEmail(null) }}
                >
                  <X className="h-4 w-4" />
                </button>
              </div>
            ) : (
              <>
                {googleClientId && (
                  <>
                    <div className="flex justify-center">
                      <GoogleLogin
                        onSuccess={handleGoogleSuccess}
                        onError={() => toast.error("Error al conectar con Google")}
                        text="signup_with"
                        shape="rectangular"
                        size="large"
                        width={400}
                      />
                    </div>
                    <div className="relative">
                      <div className="absolute inset-0 flex items-center">
                        <span className="w-full border-t" />
                      </div>
                      <div className="relative flex justify-center text-xs uppercase">
                        <span className="bg-card px-2 text-muted-foreground">o</span>
                      </div>
                    </div>
                  </>
                )}

                <div className="space-y-2">
                  <Label htmlFor="admin_email">Email del administrador</Label>
                  <Input
                    id="admin_email"
                    type="email"
                    placeholder="admin@minegocio.com"
                    {...form.register("admin_email")}
                  />
                  {form.formState.errors.admin_email && (
                    <p className="text-sm text-destructive">{form.formState.errors.admin_email.message}</p>
                  )}
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="admin_password">Password</Label>
                    <Input
                      id="admin_password"
                      type="password"
                      placeholder="Minimo 8 caracteres"
                      {...form.register("admin_password")}
                    />
                    {form.formState.errors.admin_password && (
                      <p className="text-sm text-destructive">{form.formState.errors.admin_password.message}</p>
                    )}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="confirm_password">Confirmar password</Label>
                    <Input
                      id="confirm_password"
                      type="password"
                      placeholder="Repite tu password"
                      {...form.register("confirm_password")}
                    />
                    {form.formState.errors.confirm_password && (
                      <p className="text-sm text-destructive">{form.formState.errors.confirm_password.message}</p>
                    )}
                  </div>
                </div>

                {/* Password strength bar */}
                {watchedPassword && (
                  <div className="space-y-1">
                    <div className="flex gap-1">
                      {[1, 2, 3, 4, 5].map((i) => (
                        <div
                          key={i}
                          className="h-1.5 flex-1 rounded-full bg-muted transition-colors"
                          style={i <= strength.score ? { backgroundColor: strength.color } : undefined}
                        />
                      ))}
                    </div>
                    <p className="text-xs" style={{ color: strength.color }}>
                      {strength.label}
                    </p>
                  </div>
                )}
              </>
            )}

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Registrando...
                </>
              ) : (
                "Crear cuenta"
              )}
            </Button>
          </form>

          <p className="mt-4 text-center text-sm text-muted-foreground">
            ¿Ya tienes cuenta?{" "}
            <Link to="/login" className="text-primary underline-offset-4 hover:underline">
              Inicia sesion
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
