import { useState } from "react"
import { GoogleLogin, type CredentialResponse } from "@react-oauth/google"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"
import {
  Eye,
  EyeOff,
  Loader2,
  Mail,
  OctagonX,
  Phone,
  Star,
  Wallet,
  Stamp,
  Gift,
  Users,
  Copy,
  ClipboardPaste,
  CheckCircle2,
  AlertCircle,
} from "lucide-react"
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

// Crea todos los recursos del wizard en secuencia con el JWT recién
// obtenido. Si falla a mitad de camino, el customer + admin ya están
// creados (auth OK), y el caller decide cómo recuperar.
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
    await upsertPushcardConfig(cs.id, {
      card_slots: pushcardDraft.card_slots,
      reward_on_complete: pushcardDraft.reward_on_complete,
    })
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
  const [showConfirmPwd, setShowConfirmPwd] = useState(false)
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState("")

  const fullPhone = businessInfo.country_code + businessInfo.phone
  const strength = getPasswordStrength(password)
  const confirmStrength = getPasswordStrength(confirm)
  const totalRewards = rewardDrafts.length + cashbackRewardDrafts.length

  // Match state entre password y confirm:
  //   idle: confirm vacío — no opinamos
  //   match: ambas iguales y no vacías
  //   mismatch: confirm tiene contenido pero no iguala a password
  const matchState: "idle" | "match" | "mismatch" =
    !confirm
      ? "idle"
      : password === confirm
        ? "match"
        : "mismatch"

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
      setConfirm(text)
      setError("")
    } catch {
      toast.error("No se pudo pegar")
    }
  }

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
      toast.error(
        err instanceof Error
          ? `Cuenta creada, pero hubo un error: ${err.message}. Puedes terminar de configurar desde el panel.`
          : "Cuenta creada con errores. Termina de configurar desde el panel.",
        { duration: 6000 }
      )
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
      setError("La password debe tener minimo 8 caracteres")
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

  return (
    <div className="mx-auto w-full max-w-md space-y-6">
      <div className="text-center">
        <h2 className="text-xl font-semibold">Casi listo</h2>
        <p className="mt-1 text-sm text-muted-foreground">
          Crea tu cuenta de administrador para finalizar
        </p>
      </div>

      {/* Summary card — contenido centrado */}
      <div className="rounded-xl border bg-card p-5 shadow-sm">
        <div className="flex flex-col items-center space-y-1 text-center">
          <div className="text-base font-semibold">{businessInfo.name}</div>
          <div className="flex items-center justify-center gap-1.5 text-sm text-muted-foreground">
            <Phone className="h-3.5 w-3.5" />
            <span>{fullPhone}</span>
          </div>
        </div>

        {(earnBurnDraft || cashbackDraft || pushcardDraft) && (
          <div className="mt-3 flex flex-wrap justify-center gap-1.5">
            {earnBurnDraft && (
              <Badge variant="secondary" className="gap-1">
                <Star className="h-3 w-3" />
                {earnBurnDraft.name}
              </Badge>
            )}
            {cashbackDraft && (
              <Badge variant="secondary" className="gap-1">
                <Wallet className="h-3 w-3" />
                {cashbackDraft.cashback_rate}% cashback
              </Badge>
            )}
            {pushcardDraft && (
              <Badge variant="secondary" className="gap-1">
                <Stamp className="h-3 w-3" />
                {pushcardDraft.card_slots} sellos
              </Badge>
            )}
          </div>
        )}

        <div className="mt-3 grid grid-cols-2 gap-2">
          <div className="flex items-center justify-center gap-2 rounded-md bg-muted/50 px-3 py-2 text-xs">
            <Gift className="h-3.5 w-3.5 text-muted-foreground" />
            <span>
              <span className="font-semibold text-foreground">{totalRewards}</span>{" "}
              recompensa{totalRewards === 1 ? "" : "s"}
            </span>
          </div>
          <div className="flex items-center justify-center gap-2 rounded-md bg-muted/50 px-3 py-2 text-xs">
            <Users className="h-3.5 w-3.5 text-muted-foreground" />
            <span>
              <span className="font-semibold text-foreground">{collaboratorDrafts.length}</span>{" "}
              colaborador{collaboratorDrafts.length === 1 ? "" : "es"}
            </span>
          </div>
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

          {/* Botón Anterior — solo en modo choose */}
          <div className="flex justify-center pt-2">
            <Button variant="ghost" size="sm" onClick={onPrev} disabled={loading}>
              Anterior
            </Button>
          </div>

          {loading && (
            <p className="text-center text-xs text-muted-foreground">
              Creando cuenta con Google...
            </p>
          )}
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
                placeholder="Minimo 8 caracteres"
                value={password}
                onChange={(e) => { setPassword(e.target.value); setError("") }}
                className="pr-16"
              />
              <div className="absolute inset-y-0 right-1 flex items-center gap-0.5">
                <button
                  type="button"
                  onClick={() => copyToClipboard(password)}
                  disabled={!password}
                  tabIndex={-1}
                  title="Copiar password"
                  className="rounded p-1 text-muted-foreground transition-colors hover:text-foreground disabled:opacity-40"
                >
                  <Copy className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setShowPwd((v) => !v)}
                  tabIndex={-1}
                  title={showPwd ? "Ocultar password" : "Mostrar password"}
                  className="rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
                >
                  <span className="relative block h-4 w-4">
                    <Eye
                      className={cn(
                        "absolute inset-0 h-4 w-4 transition-all duration-200",
                        showPwd ? "scale-100 opacity-100" : "scale-50 opacity-0"
                      )}
                    />
                    <EyeOff
                      className={cn(
                        "absolute inset-0 h-4 w-4 transition-all duration-200",
                        showPwd ? "scale-50 opacity-0" : "scale-100 opacity-100"
                      )}
                    />
                  </span>
                </button>
              </div>
            </div>
            {password && (
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
          </div>

          <div className="space-y-2">
            <Label htmlFor="admin-confirm">Confirmar password</Label>
            <div className="relative">
              <Input
                id="admin-confirm"
                type={showConfirmPwd ? "text" : "password"}
                placeholder="Repite tu password"
                value={confirm}
                onChange={(e) => { setConfirm(e.target.value); setError("") }}
                className={cn(
                  "pr-24 transition-colors duration-200",
                  matchState === "match" &&
                    "border-green-500 focus-visible:border-green-500 focus-visible:ring-green-500/30",
                  matchState === "mismatch" &&
                    "border-amber-500 focus-visible:border-amber-500 focus-visible:ring-amber-500/30"
                )}
                aria-invalid={matchState === "mismatch"}
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
                  onClick={() => copyToClipboard(confirm)}
                  disabled={!confirm}
                  tabIndex={-1}
                  title="Copiar password"
                  className="rounded p-1 text-muted-foreground transition-colors hover:text-foreground disabled:opacity-40"
                >
                  <Copy className="h-4 w-4" />
                </button>
                <button
                  type="button"
                  onClick={() => setShowConfirmPwd((v) => !v)}
                  tabIndex={-1}
                  title={showConfirmPwd ? "Ocultar password" : "Mostrar password"}
                  className="rounded p-1 text-muted-foreground transition-colors hover:text-foreground"
                >
                  <span className="relative block h-4 w-4">
                    <Eye
                      className={cn(
                        "absolute inset-0 h-4 w-4 transition-all duration-200",
                        showConfirmPwd ? "scale-100 opacity-100" : "scale-50 opacity-0"
                      )}
                    />
                    <EyeOff
                      className={cn(
                        "absolute inset-0 h-4 w-4 transition-all duration-200",
                        showConfirmPwd ? "scale-50 opacity-0" : "scale-100 opacity-100"
                      )}
                    />
                  </span>
                </button>
              </div>
            </div>

            {/* Match indicator */}
            {matchState === "match" && (
              <p className="flex items-center gap-1.5 text-xs text-green-600">
                <CheckCircle2 className="h-3.5 w-3.5" />
                Las passwords coinciden
              </p>
            )}
            {matchState === "mismatch" && (
              <p className="flex items-center gap-1.5 text-xs text-amber-600">
                <AlertCircle className="h-3.5 w-3.5" />
                Las passwords no coinciden todavia
              </p>
            )}

            {/* Strength bar de la confirm (mismo computo que el primer campo) */}
            {confirm && (
              <div className="space-y-1">
                <div className="flex gap-1">
                  {[1, 2, 3, 4, 5].map((i) => (
                    <div
                      key={i}
                      className="h-1.5 flex-1 rounded-full bg-muted transition-colors"
                      style={i <= confirmStrength.score ? { backgroundColor: confirmStrength.color } : undefined}
                    />
                  ))}
                </div>
                <p className="text-xs" style={{ color: confirmStrength.color }}>
                  {confirmStrength.label}
                </p>
              </div>
            )}
          </div>

          <div className="flex gap-2 pt-1">
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
    </div>
  )
}
