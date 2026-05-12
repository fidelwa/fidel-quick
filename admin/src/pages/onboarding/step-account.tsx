import { useState } from "react"
import { GoogleLogin, type CredentialResponse } from "@react-oauth/google"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { Eye, EyeOff, Loader2, Mail, OctagonX } from "lucide-react"
import { useAuth } from "@/context/auth-context"
import {
  onboardingRegister,
  onboardingGoogle,
  createProgram,
  createCashbackProgram,
  createCustomerSisfi,
  upsertPushcardConfig,
  createReward,
  createCashbackReward,
  createCollaborator,
  completeOnboarding,
  setToken,
} from "@/lib/api-client"
import type {
  BusinessInfo,
  EarnBurnDraft,
  CashbackDraft,
  PushcardDraft,
  RewardDraft,
  CashbackRewardDraft,
  CollaboratorDraft,
} from "@/hooks/use-onboarding"
import { cn } from "@/lib/utils"

const EMAIL_REGEX =
  /^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$/

interface StepAccountProps {
  businessInfo: BusinessInfo
  earnBurnDraft: EarnBurnDraft | null
  cashbackDraft: CashbackDraft | null
  pushcardDraft: PushcardDraft | null
  rewardDrafts: RewardDraft[]
  cashbackRewardDrafts: CashbackRewardDraft[]
  collaboratorDrafts: CollaboratorDraft[]
  onSuccess: () => void
  onPrev: () => void
}

// Crea todos los recursos del wizard en secuencia con el JWT recién
// obtenido. Si falla a mitad de camino, el customer + admin ya están
// creados (auth OK), y se devuelve el customer_id para que el caller
// pueda decidir cómo recuperar (mostrar dashboard, retry, etc.).
async function batchCreateEntities(args: {
  customerId: string
  earnBurnDraft: EarnBurnDraft | null
  cashbackDraft: CashbackDraft | null
  pushcardDraft: PushcardDraft | null
  rewardDrafts: RewardDraft[]
  cashbackRewardDrafts: CashbackRewardDraft[]
  collaboratorDrafts: CollaboratorDraft[]
}) {
  const {
    customerId,
    earnBurnDraft,
    cashbackDraft,
    pushcardDraft,
    rewardDrafts,
    cashbackRewardDrafts,
    collaboratorDrafts,
  } = args

  let earnProgramId: string | null = null
  let cashbackProgramId: string | null = null

  if (earnBurnDraft) {
    const p = await createProgram({
      customer_id: customerId,
      name: earnBurnDraft.name,
      points_ratio: earnBurnDraft.points_ratio,
    })
    earnProgramId = p.id
  }

  if (cashbackDraft) {
    const p = await createCashbackProgram({
      customer_id: customerId,
      name: cashbackDraft.name,
      cashback_rate: cashbackDraft.cashback_rate,
    })
    cashbackProgramId = p.id
  }

  if (pushcardDraft) {
    const cs = await createCustomerSisfi({
      customer_id: customerId,
      sisfi_id: "pushcard",
      name: pushcardDraft.name,
    })
    await upsertPushcardConfig(cs.id, { card_slots: pushcardDraft.card_slots })
  }

  if (earnProgramId) {
    for (const r of rewardDrafts) {
      await createReward(earnProgramId, {
        name: r.name,
        description: r.description,
        points_cost: r.points_cost,
      })
    }
  }

  if (cashbackProgramId) {
    for (const r of cashbackRewardDrafts) {
      await createCashbackReward(cashbackProgramId, {
        name: r.name,
        description: r.description,
        cost: r.cost,
      })
    }
  }

  for (const c of collaboratorDrafts) {
    await createCollaborator(customerId, {
      name: c.name,
      phone: c.phone,
    })
  }

  await completeOnboarding()
}

export function StepAccount({
  businessInfo,
  earnBurnDraft,
  cashbackDraft,
  pushcardDraft,
  rewardDrafts,
  cashbackRewardDrafts,
  collaboratorDrafts,
  onSuccess,
  onPrev,
}: StepAccountProps) {
  const { login } = useAuth()
  const [mode, setMode] = useState<"choose" | "email">("choose")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirm, setConfirm] = useState("")
  const [showPwd, setShowPwd] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")

  const fullPhone = businessInfo.country_code + businessInfo.phone

  const finishAuth = async (token: string, customerId: string, adminEmail: string) => {
    // setToken para que el request() del api-client adjunte el header.
    setToken(token)
    login(token, customerId, adminEmail)
    try {
      await batchCreateEntities({
        customerId,
        earnBurnDraft,
        cashbackDraft,
        pushcardDraft,
        rewardDrafts,
        cashbackRewardDrafts,
        collaboratorDrafts,
      })
      onSuccess()
    } catch (err) {
      // Cuenta creada OK pero algo falló al crear programas/rewards/etc.
      // El usuario ya está logueado — puede recuperar desde el dashboard.
      toast.error(
        err instanceof Error
          ? `Cuenta creada, pero hubo un error: ${err.message}. Puedes terminar de configurar desde el panel.`
          : "Cuenta creada con errores. Termina de configurar desde el panel.",
        { duration: 6000 }
      )
      // Igual avanzamos — el usuario está logueado.
      onSuccess()
    }
  }

  const handleGoogle = async (resp: CredentialResponse) => {
    if (!resp.credential) {
      toast.error("Error al obtener credencial de Google")
      return
    }
    setLoading(true)
    setError("")
    try {
      const res = await onboardingGoogle({
        google_token: resp.credential,
        name: businessInfo.name,
        phone: fullPhone,
        description: businessInfo.description || undefined,
      })
      await finishAuth(res.token, res.admin.customer_id, res.admin.email)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error al registrar con Google")
      setLoading(false)
    }
  }

  const handleEmailSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setError("")

    if (!email.trim() || !EMAIL_REGEX.test(email.trim())) {
      setError("Email invalido")
      return
    }
    if (password.length < 8) {
      setError("La password debe tener mínimo 8 caracteres")
      return
    }
    if (password !== confirm) {
      setError("Las passwords no coinciden")
      return
    }

    setLoading(true)
    try {
      const res = await onboardingRegister({
        name: businessInfo.name,
        phone: fullPhone,
        country_code: businessInfo.country_code,
        description: businessInfo.description || undefined,
        admin_email: email.trim(),
        admin_password: password,
      })
      await finishAuth(res.token, res.admin.customer_id, res.admin.email)
    } catch (err) {
      setError(err instanceof Error ? err.message : "Error al registrar")
      setLoading(false)
    }
  }

  // Resumen de lo que se va a crear
  const summaryItems: string[] = []
  if (earnBurnDraft) summaryItems.push(`Puntos (${earnBurnDraft.name})`)
  if (cashbackDraft) summaryItems.push(`Cashback ${cashbackDraft.cashback_rate}% (${cashbackDraft.name})`)
  if (pushcardDraft) summaryItems.push(`Tarjeta de ${pushcardDraft.card_slots} sellos`)
  const totalRewards = rewardDrafts.length + cashbackRewardDrafts.length

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Casi listo. Crea tu cuenta de administrador</h2>
        <p className="text-sm text-muted-foreground">
          Te logueas con Google o con email/password. La cuenta queda
          asociada al negocio "{businessInfo.name}".
        </p>
      </div>

      {/* Summary */}
      <div className="rounded-lg border bg-muted/30 p-4 text-sm">
        <div className="font-medium">{businessInfo.name}</div>
        <div className="text-muted-foreground">{fullPhone}</div>
        {summaryItems.length > 0 && (
          <ul className="mt-2 list-disc space-y-0.5 pl-5 text-muted-foreground">
            {summaryItems.map((s) => <li key={s}>{s}</li>)}
          </ul>
        )}
        <div className="mt-2 text-xs text-muted-foreground">
          {totalRewards} recompensa{totalRewards === 1 ? "" : "s"} · {collaboratorDrafts.length} colaborador{collaboratorDrafts.length === 1 ? "" : "es"}
        </div>
      </div>

      {/* Inline error */}
      {error && (
        <div className="flex items-center gap-2 rounded-lg border border-destructive/30 bg-destructive/5 px-3 py-2.5">
          <OctagonX className="h-4 w-4 shrink-0 text-destructive" />
          <p className="text-sm text-destructive">{error}</p>
        </div>
      )}

      {/* Auth options */}
      {mode === "choose" && (
        <div className="space-y-3">
          <div className="flex justify-center">
            <GoogleLogin
              onSuccess={handleGoogle}
              onError={() => toast.error("Error al conectar con Google")}
              text="continue_with"
              shape="rectangular"
              size="large"
              width={380}
            />
          </div>
          <div className="relative">
            <div className="absolute inset-0 flex items-center">
              <span className="w-full border-t" />
            </div>
            <div className="relative flex justify-center text-xs">
              <span className="bg-background px-2 text-muted-foreground">o</span>
            </div>
          </div>
          <Button
            type="button"
            variant="outline"
            className="w-full"
            onClick={() => setMode("email")}
            disabled={loading}
          >
            <Mail className="mr-2 h-4 w-4" />
            Crear cuenta con email
          </Button>
        </div>
      )}

      {mode === "email" && (
        <form onSubmit={handleEmailSubmit} className="space-y-4">
          <div className="space-y-2">
            <Label htmlFor="admin-email">Email</Label>
            <Input
              id="admin-email"
              type="email"
              autoComplete="email"
              placeholder="admin@minegocio.com"
              value={email}
              onChange={(e) => { setEmail(e.target.value); setError("") }}
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="admin-password">Password</Label>
            <div className="relative">
              <Input
                id="admin-password"
                type={showPwd ? "text" : "password"}
                placeholder="Mínimo 8 caracteres"
                value={password}
                onChange={(e) => { setPassword(e.target.value); setError("") }}
                className="pr-10"
              />
              <button
                type="button"
                onClick={() => setShowPwd((v) => !v)}
                className={cn(
                  "absolute right-2 top-1/2 -translate-y-1/2 rounded p-1",
                  "text-muted-foreground hover:text-foreground"
                )}
                tabIndex={-1}
                aria-label={showPwd ? "Ocultar password" : "Mostrar password"}
              >
                {showPwd ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
              </button>
            </div>
          </div>
          <div className="space-y-2">
            <Label htmlFor="admin-confirm">Confirmar password</Label>
            <Input
              id="admin-confirm"
              type={showPwd ? "text" : "password"}
              placeholder="Repite tu password"
              value={confirm}
              onChange={(e) => { setConfirm(e.target.value); setError("") }}
            />
          </div>

          <div className="flex gap-2">
            <Button
              type="button"
              variant="outline"
              onClick={() => { setMode("choose"); setError("") }}
              disabled={loading}
            >
              Volver
            </Button>
            <Button type="submit" className="flex-1" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Creando cuenta...
                </>
              ) : (
                "Crear cuenta y finalizar"
              )}
            </Button>
          </div>
        </form>
      )}

      <div className="flex justify-start">
        <Button variant="ghost" onClick={onPrev} disabled={loading}>
          Anterior
        </Button>
      </div>

      {loading && mode === "choose" && (
        <p className="text-center text-xs text-muted-foreground">
          Creando cuenta con Google...
        </p>
      )}
    </div>
  )
}
