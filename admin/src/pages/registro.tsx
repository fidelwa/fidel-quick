import { useState } from "react"
import { useNavigate, Navigate, Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useAuth } from "@/context/auth-context"
import { useOnboarding } from "@/hooks/use-onboarding"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Loader2 } from "lucide-react"
import { COUNTRY_CODES } from "@/lib/country-codes"

const registroSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  country_code: z.string().min(1, "Selecciona un pais"),
  phone: z.string().min(7, "Minimo 7 digitos").regex(/^\d+$/, "Solo numeros"),
  description: z.string().optional(),
})

type RegistroForm = z.infer<typeof registroSchema>

export function RegistroPage() {
  const { isAuthenticated } = useAuth()
  const navigate = useNavigate()
  const onboarding = useOnboarding()
  const [loading, setLoading] = useState(false)

  const form = useForm<RegistroForm>({
    resolver: zodResolver(registroSchema),
    mode: "onBlur",
    defaultValues: {
      name: onboarding.businessInfo?.name ?? "",
      country_code: onboarding.businessInfo?.country_code ?? "+52",
      phone: onboarding.businessInfo?.phone ?? "",
      description: onboarding.businessInfo?.description ?? "",
    },
  })

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  const handleSubmit = async () => {
    const ok = await form.trigger()
    if (!ok) return

    setLoading(true)
    const data = form.getValues()
    onboarding.setBusinessInfo({
      name: data.name.trim(),
      country_code: data.country_code,
      phone: data.phone.trim(),
      description: (data.description ?? "").trim(),
    })
    onboarding.goToStep(1)
    navigate("/onboarding")
  }

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
            Empezamos con los datos de tu negocio. La cuenta de
            administrador se crea al final.
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

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Continuando...
                </>
              ) : (
                "Continuar"
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
