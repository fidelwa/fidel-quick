import { useState } from "react"
import { Navigate, Link, useNavigate, useSearchParams } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useAuth } from "@/context/auth-context"
import { resetPassword } from "@/lib/api-client"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { OctagonX, Eye, EyeOff, Loader2, AlertTriangle } from "lucide-react"

const resetSchema = z
  .object({
    password: z.string().min(8, "Mínimo 8 caracteres"),
    confirm: z.string().min(1, "Confirma tu contraseña"),
  })
  .refine((d) => d.password === d.confirm, {
    message: "Las contraseñas no coinciden",
    path: ["confirm"],
  })

type ResetForm = z.infer<typeof resetSchema>

export function ResetPasswordPage() {
  const { isAuthenticated } = useAuth()
  const navigate = useNavigate()
  const [params] = useSearchParams()
  const token = params.get("token") ?? ""

  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")
  const [showPassword, setShowPassword] = useState(false)

  const form = useForm<ResetForm>({
    resolver: zodResolver(resetSchema),
    mode: "onBlur",
    defaultValues: { password: "", confirm: "" },
  })

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  // Sin token en la URL el enlace es inválido — mostramos estado de error.
  if (!token) {
    return (
      <ResetShell title="Enlace inválido" description="Este enlace no es válido">
        <div className="space-y-4">
          <div className="flex items-start gap-3 rounded-lg border border-amber-300 bg-amber-50 px-4 py-3 dark:border-amber-900 dark:bg-amber-950/40">
            <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-amber-600" />
            <p className="text-sm text-amber-900 dark:text-amber-100">
              El enlace de restablecimiento es inválido o está incompleto.
              Solicita uno nuevo.
            </p>
          </div>
          <Button asChild className="w-full">
            <Link to="/forgot-password">Solicitar un nuevo enlace</Link>
          </Button>
        </div>
      </ResetShell>
    )
  }

  const onSubmit = async (data: ResetForm) => {
    setError("")
    setLoading(true)
    try {
      await resetPassword(token, data.password)
      toast.success("Contraseña actualizada. Inicia sesión con tu nueva contraseña.")
      navigate("/login")
    } catch (err) {
      // Token inválido/expirado/usado → el backend responde 400.
      const msg =
        err instanceof Error && err.message
          ? err.message
          : "No se pudo restablecer la contraseña."
      setError(
        /inválido|expirado|usado|token/i.test(msg)
          ? "El enlace es inválido o ya expiró. Solicita uno nuevo."
          : msg
      )
    } finally {
      // Re-enable the button in every case, including success, so it isn't
      // left disabled if navigation is delayed.
      setLoading(false)
    }
  }

  return (
    <ResetShell
      title="Nueva contraseña"
      description="Elige una contraseña nueva para tu cuenta"
    >
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

      <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
        <div className="space-y-2">
          <Label htmlFor="password">Nueva contraseña</Label>
          <div className="relative">
            <Input
              id="password"
              type={showPassword ? "text" : "password"}
              placeholder="Mínimo 8 caracteres"
              className="pr-10"
              {...form.register("password")}
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-1 text-muted-foreground hover:text-foreground focus:outline-none focus-visible:ring-2 focus-visible:ring-ring"
              aria-label={showPassword ? "Ocultar contraseña" : "Mostrar contraseña"}
              tabIndex={-1}
            >
              {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
          {form.formState.errors.password && (
            <p className="text-sm text-destructive">{form.formState.errors.password.message}</p>
          )}
        </div>

        <div className="space-y-2">
          <Label htmlFor="confirm">Confirmar contraseña</Label>
          <Input
            id="confirm"
            type={showPassword ? "text" : "password"}
            placeholder="Repite tu contraseña"
            {...form.register("confirm")}
          />
          {form.formState.errors.confirm && (
            <p className="text-sm text-destructive">{form.formState.errors.confirm.message}</p>
          )}
        </div>

        <Button type="submit" className="w-full" disabled={loading}>
          {loading ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Guardando...
            </>
          ) : (
            "Restablecer contraseña"
          )}
        </Button>
      </form>

      <p className="mt-4 text-center text-sm text-muted-foreground">
        <Link to="/login" className="text-primary underline-offset-4 hover:underline">
          Volver a iniciar sesión
        </Link>
      </p>
    </ResetShell>
  )
}

// ResetShell renders the shared glass card layout so the error/loaded states
// look identical.
function ResetShell({
  title,
  description,
  children,
}: {
  title: string
  description: string
  children: React.ReactNode
}) {
  return (
    <div className="relative flex min-h-screen items-center justify-center overflow-hidden p-4">
      <div className="pointer-events-none absolute inset-0 -z-10 bg-gradient-to-br from-violet-100 via-sky-50 to-pink-100 dark:from-violet-950 dark:via-slate-950 dark:to-pink-950" />
      <div className="pointer-events-none absolute -left-32 -top-32 -z-10 h-96 w-96 rounded-full bg-purple-400/40 blur-3xl dark:bg-purple-700/30" />
      <div className="pointer-events-none absolute -bottom-32 -right-32 -z-10 h-96 w-96 rounded-full bg-sky-400/40 blur-3xl dark:bg-sky-700/30" />
      <div className="pointer-events-none absolute left-1/3 top-1/2 -z-10 h-72 w-72 rounded-full bg-pink-400/30 blur-3xl dark:bg-pink-700/20" />

      <Card className="relative w-full max-w-md border-white/30 bg-white/40 shadow-2xl shadow-purple-500/10 backdrop-blur-2xl dark:border-white/10 dark:bg-white/5">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">{title}</CardTitle>
          <CardDescription>{description}</CardDescription>
        </CardHeader>
        <CardContent>{children}</CardContent>
      </Card>
    </div>
  )
}
