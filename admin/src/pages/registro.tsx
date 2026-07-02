import { useState, useEffect, useRef } from "react"
import { useNavigate, Navigate, Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useAuth } from "@/context/auth-context"
import { useOnboarding } from "@/hooks/use-onboarding"
import { checkPhoneExists } from "@/lib/api-client"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { CheckCircle2, AlertCircle, Loader2 } from "lucide-react"
import { COUNTRY_CODES } from "@/lib/country-codes"
import { cn } from "@/lib/utils"

const registroSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  country_code: z.string().min(1, "Selecciona un pais"),
  phone: z.string().min(7, "Minimo 7 digitos").regex(/^\d+$/, "Solo numeros"),
  description: z.string().optional(),
})

type RegistroForm = z.infer<typeof registroSchema>

// Estados del check de teléfono:
//   idle: no hay un número válido todavía
//   checking: API call en vuelo
//   available: el número está libre (verde)
//   exists: el número ya está en uso (amber, requiere confirmación)
//   error: el check falló — dejamos pasar para no bloquear por flaky network
type PhoneCheckState = "idle" | "checking" | "available" | "exists" | "error"

const DEBOUNCE_MS = 500

export function RegistroPage() {
  const { isAuthenticated } = useAuth()
  const navigate = useNavigate()
  const onboarding = useOnboarding()
  const [loading, setLoading] = useState(false)
  const [phoneState, setPhoneState] = useState<PhoneCheckState>("idle")
  const [overrideExisting, setOverrideExisting] = useState(false)
  // Token para descartar respuestas viejas de checks fuera de orden.
  const checkSeqRef = useRef(0)

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

  const phone = form.watch("phone")
  const countryCode = form.watch("country_code")

  // Debounced check: dispara cuando el teléfono cambia y es válido.
  useEffect(() => {
    setOverrideExisting(false)
    const cleaned = phone?.trim() ?? ""
    if (!cleaned || !/^\d{7,15}$/.test(cleaned)) {
      setPhoneState("idle")
      return
    }
    setPhoneState("checking")
    const fullPhone = countryCode + cleaned
    const seq = ++checkSeqRef.current
    const timer = setTimeout(() => {
      checkPhoneExists(fullPhone)
        .then((res) => {
          // Ignorar respuesta si llegó otra newer request mientras tanto.
          if (seq !== checkSeqRef.current) return
          setPhoneState(res.exists ? "exists" : "available")
        })
        .catch(() => {
          if (seq !== checkSeqRef.current) return
          setPhoneState("error")
        })
    }, DEBOUNCE_MS)
    return () => clearTimeout(timer)
  }, [phone, countryCode])

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  const canContinue =
    phoneState === "available" ||
    phoneState === "error" ||
    (phoneState === "exists" && overrideExisting)

  const handleSubmit = async () => {
    const ok = await form.trigger()
    if (!ok) return
    if (!canContinue) return

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
                <div className="relative flex-1">
                  <Input
                    id="phone"
                    placeholder="5512345678"
                    className={cn(
                      "pr-10 transition-colors duration-200",
                      phoneState === "available" &&
                        "border-green-500 focus-visible:border-green-500 focus-visible:ring-green-500/30",
                      phoneState === "exists" &&
                        "border-amber-500 focus-visible:border-amber-500 focus-visible:ring-amber-500/30"
                    )}
                    {...form.register("phone", {
                      onChange: (e) => {
                        e.target.value = e.target.value.replace(/\D/g, "")
                      },
                    })}
                  />
                  <div className="pointer-events-none absolute inset-y-0 right-3 flex items-center">
                    {phoneState === "checking" && (
                      <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                    )}
                    {phoneState === "available" && (
                      <CheckCircle2 className="h-4 w-4 text-green-500 animate-[check-pop_0.4s_ease-out]" />
                    )}
                    {phoneState === "exists" && (
                      <AlertCircle className="h-4 w-4 text-amber-500" />
                    )}
                  </div>
                </div>
              </div>
              {form.formState.errors.phone && (
                <p className="text-sm text-destructive">{form.formState.errors.phone.message}</p>
              )}

              {/* Banner: número ya existe — pedir confirmación */}
              {phoneState === "exists" && (
                <div className="space-y-2 rounded-lg border border-amber-300 bg-amber-50 p-3 dark:border-amber-900 dark:bg-amber-950/40">
                  <div className="flex items-start gap-2">
                    <AlertCircle className="mt-0.5 h-4 w-4 shrink-0 text-amber-600" />
                    <div className="space-y-1 text-sm">
                      <p className="font-medium text-amber-900 dark:text-amber-100">
                        Este telefono ya esta registrado.
                      </p>
                      <p className="text-amber-800 dark:text-amber-200">
                        Si esta es otra empresa diferente, marca la casilla para continuar.
                      </p>
                    </div>
                  </div>
                  <label className="flex items-center gap-2 pl-6 text-sm">
                    <input
                      type="checkbox"
                      className="h-4 w-4 rounded border-amber-300 text-amber-600 focus:ring-amber-500"
                      checked={overrideExisting}
                      onChange={(e) => setOverrideExisting(e.target.checked)}
                    />
                    <span className="text-amber-900 dark:text-amber-100">
                      Quiero registrar otra empresa con este mismo numero
                    </span>
                  </label>
                </div>
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

            <Button
              type="submit"
              className="w-full"
              disabled={loading || phoneState === "checking" || !canContinue}
            >
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Continuando...
                </>
              ) : phoneState === "checking" ? (
                "Verificando..."
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
