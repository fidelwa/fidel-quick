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
import { Loader2, Check, X, Eye, EyeOff, Copy, ClipboardPaste, CheckCircle2, AlertCircle, Mail } from "lucide-react"
import { COUNTRY_CODES } from "@/lib/country-codes"
import { cn } from "@/lib/utils"

// ReDoS-safe email regex (WHATWG HTML5 spec).
// Uses single-pass character classes and bounded quantifiers to keep linear time.
const EMAIL_REGEX =
  /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/

const registroSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  country_code: z.string().min(1, "Selecciona un pais"),
  phone: z.string().min(7, "Minimo 7 digitos").regex(/^\d+$/, "Solo numeros"),
  description: z.string().optional(),
  admin_email: z
    .string()
    .max(254, "Email demasiado largo")
    .refine((v) => v === "" || EMAIL_REGEX.test(v), "Email invalido")
    .optional(),
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
  const [showPassword, setShowPassword] = useState(false)
  const [showConfirmPassword, setShowConfirmPassword] = useState(false)

  const form = useForm<RegistroForm>({
    resolver: zodResolver(registroSchema),
    mode: "onBlur",
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
  const watchedConfirm = form.watch("confirm_password") || ""
  const watchedEmail = form.watch("admin_email") || ""
  const strength = getPasswordStrength(watchedPassword)

  const emailState: "idle" | "valid" | "invalid" =
    watchedEmail === ""
      ? "idle"
      : EMAIL_REGEX.test(watchedEmail) && watchedEmail.length <= 254
        ? "valid"
        : "invalid"

  const copyToClipboard = async (value: string) => {
    if (!value) return
    try {
      await navigator.clipboard.writeText(value)
      toast.success("Copiado al portapapeles")
    } catch {
      toast.error("No se pudo copiar")
    }
  }

  const pasteIntoConfirm = async () => {
    try {
      const text = await navigator.clipboard.readText()
      form.setValue("confirm_password", text, { shouldValidate: true, shouldDirty: true })
    } catch {
      toast.error("No se pudo pegar")
    }
  }

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
      if (!data.admin_email) {
        form.setError("admin_email", { message: "Email requerido" })
        return
      }
      if (!EMAIL_REGEX.test(data.admin_email)) {
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
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden p-4">
      {/* Glass background: gradient + soft color blobs */}
      <div className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-br from-violet-100 via-sky-50 to-pink-100 dark:from-violet-950 dark:via-slate-950 dark:to-pink-950" />
      <div className="pointer-events-none absolute -left-32 -top-32 -z-10 h-96 w-96 rounded-full bg-purple-400/40 blur-3xl dark:bg-purple-700/30" />
      <div className="pointer-events-none absolute -bottom-32 -right-32 -z-10 h-96 w-96 rounded-full bg-sky-400/40 blur-3xl dark:bg-sky-700/30" />
      <div className="pointer-events-none absolute left-1/3 top-1/2 -z-10 h-72 w-72 rounded-full bg-pink-400/30 blur-3xl dark:bg-pink-700/20" />

      <Card className="relative w-full max-w-lg border-white/30 bg-white/40 shadow-2xl shadow-purple-500/10 backdrop-blur-2xl dark:border-white/10 dark:bg-white/5">
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
                placeholder="Mi Negocio"
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
              <div className="relative flex justify-center text-xs">
                <span className="bg-white/40 px-2 text-muted-foreground backdrop-blur-sm dark:bg-white/5">Cuenta de administrador</span>
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
                        <span className="bg-white/40 px-2 text-muted-foreground backdrop-blur-sm dark:bg-white/5">o</span>
                      </div>
                    </div>
                  </>
                )}

                <div className="space-y-2">
                  <Label htmlFor="admin_email">Email del administrador</Label>
                  <div className="relative">
                    <div className="pointer-events-none absolute inset-y-0 left-3 flex items-center">
                      <Mail
                        className={cn(
                          "h-4 w-4 transition-colors duration-200",
                          emailState === "valid" && "text-green-500/80",
                          emailState === "invalid" && "text-destructive/80",
                          emailState === "idle" && "text-muted-foreground"
                        )}
                      />
                    </div>
                    <Input
                      id="admin_email"
                      type="email"
                      autoComplete="email"
                      placeholder="admin@minegocio.com"
                      className={cn(
                        "pl-9 pr-10 transition-all duration-200",
                        emailState === "valid" &&
                          "border-green-500 focus-visible:border-green-500 focus-visible:ring-green-500/30",
                        emailState === "invalid" &&
                          "border-destructive focus-visible:border-destructive focus-visible:ring-destructive/30",
                        form.formState.errors.admin_email && "animate-[shake_0.4s_ease-out]"
                      )}
                      aria-invalid={emailState === "invalid" || !!form.formState.errors.admin_email}
                      {...form.register("admin_email")}
                    />
                    <div className="pointer-events-none absolute inset-y-0 right-3 flex items-center">
                      {emailState === "valid" && (
                        <CheckCircle2
                          key="valid"
                          className="h-4 w-4 text-green-500 animate-[check-pop_0.4s_ease-out]"
                        />
                      )}
                      {emailState === "invalid" && (
                        <AlertCircle
                          key="invalid"
                          className="h-4 w-4 text-destructive animate-[shake_0.4s_ease-out]"
                        />
                      )}
                    </div>
                  </div>
                  {form.formState.errors.admin_email && (
                    <p
                      role="alert"
                      className="flex items-center gap-1.5 text-sm text-destructive animate-[fade-slide-in-left_0.25s_ease-out]"
                    >
                      <AlertCircle className="h-3.5 w-3.5 shrink-0" />
                      <span>{form.formState.errors.admin_email.message}</span>
                    </p>
                  )}
                </div>

                <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
                  <div className="space-y-2">
                    <Label htmlFor="admin_password">Password</Label>
                    <div className="relative">
                      <Input
                        id="admin_password"
                        type={showPassword ? "text" : "password"}
                        placeholder="Minimo 8 caracteres"
                        className="pr-16"
                        {...form.register("admin_password")}
                      />
                      <div className="absolute inset-y-0 right-1 flex items-center gap-0.5">
                        <button
                          type="button"
                          onClick={() => copyToClipboard(watchedPassword)}
                          disabled={!watchedPassword}
                          tabIndex={-1}
                          title="Copiar password"
                          className="rounded p-1 text-muted-foreground transition-colors hover:text-foreground disabled:opacity-40"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                        <button
                          type="button"
                          onClick={() => setShowPassword((v) => !v)}
                          tabIndex={-1}
                          title={showPassword ? "Ocultar password" : "Mostrar password"}
                          className="rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
                        >
                          <span className="relative block h-4 w-4">
                            <Eye
                              className={`absolute inset-0 h-4 w-4 transition-all duration-200 ${
                                showPassword ? "scale-100 opacity-100" : "scale-50 opacity-0"
                              }`}
                            />
                            <EyeOff
                              className={`absolute inset-0 h-4 w-4 transition-all duration-200 ${
                                showPassword ? "scale-50 opacity-0" : "scale-100 opacity-100"
                              }`}
                            />
                          </span>
                        </button>
                      </div>
                    </div>
                    {form.formState.errors.admin_password && (
                      <p className="text-sm text-destructive">{form.formState.errors.admin_password.message}</p>
                    )}
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="confirm_password">Confirmar password</Label>
                    <div className="relative">
                      <Input
                        id="confirm_password"
                        type={showConfirmPassword ? "text" : "password"}
                        placeholder="Repite tu password"
                        className="pr-24"
                        {...form.register("confirm_password")}
                      />
                      <div className="absolute inset-y-0 right-1 flex items-center gap-0.5">
                        <button
                          type="button"
                          onClick={pasteIntoConfirm}
                          tabIndex={-1}
                          title="Pegar password"
                          className="rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
                        >
                          <ClipboardPaste className="h-4 w-4" />
                        </button>
                        <button
                          type="button"
                          onClick={() => copyToClipboard(watchedConfirm)}
                          disabled={!watchedConfirm}
                          tabIndex={-1}
                          title="Copiar password"
                          className="rounded p-1 text-muted-foreground transition-colors hover:text-foreground disabled:opacity-40"
                        >
                          <Copy className="h-4 w-4" />
                        </button>
                        <button
                          type="button"
                          onClick={() => setShowConfirmPassword((v) => !v)}
                          tabIndex={-1}
                          title={showConfirmPassword ? "Ocultar password" : "Mostrar password"}
                          className="rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
                        >
                          <span className="relative block h-4 w-4">
                            <Eye
                              className={`absolute inset-0 h-4 w-4 transition-all duration-200 ${
                                showConfirmPassword ? "scale-100 opacity-100" : "scale-50 opacity-0"
                              }`}
                            />
                            <EyeOff
                              className={`absolute inset-0 h-4 w-4 transition-all duration-200 ${
                                showConfirmPassword ? "scale-50 opacity-0" : "scale-100 opacity-100"
                              }`}
                            />
                          </span>
                        </button>
                      </div>
                    </div>
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
