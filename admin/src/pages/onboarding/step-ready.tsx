import { useRef } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "@/context/auth-context"
import { useCustomer } from "@/hooks/use-customer"
import { QRCodeCanvas } from "qrcode.react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { toast } from "sonner"
import { Download, Copy, Star, Wallet, Stamp, CheckCircle2 } from "lucide-react"
import { clearOnboardingDraft } from "@/hooks/use-onboarding"
import type {
  EarnBurnDraft,
  CashbackDraft,
  PushcardDraft,
  RewardDraft,
  CashbackRewardDraft,
  CollaboratorDraft,
} from "@/hooks/use-onboarding"

interface StepReadyProps {
  earnBurnDraft: EarnBurnDraft | null
  cashbackDraft: CashbackDraft | null
  pushcardDraft: PushcardDraft | null
  rewardDrafts: RewardDraft[]
  cashbackRewardDrafts: CashbackRewardDraft[]
  collaboratorDrafts: CollaboratorDraft[]
}

export function StepReady({
  earnBurnDraft,
  cashbackDraft,
  pushcardDraft,
  rewardDrafts,
  cashbackRewardDrafts,
  collaboratorDrafts,
}: StepReadyProps) {
  const { customerId } = useAuth()
  const { data: customer } = useCustomer(customerId)
  const navigate = useNavigate()
  const qrRef = useRef<HTMLDivElement>(null)

  // En prod: origin = https://fidel-quick-...run.app (o el dominio custom).
  // En dev: vite corre en :5173 pero /unirse/* solo lo sirve el backend Go
  // en :8080, así que apuntamos ahí explícitamente.
  const joinOrigin =
    typeof window !== "undefined" && window.location.hostname === "localhost"
      ? "http://localhost:8080"
      : typeof window !== "undefined"
        ? window.location.origin
        : ""
  const joinUrl = `${joinOrigin}/unirse/${customer?.slug ?? ""}`

  const handleDownloadQR = () => {
    const canvas = qrRef.current?.querySelector("canvas")
    if (!canvas) return
    const url = canvas.toDataURL("image/png")
    const link = document.createElement("a")
    link.download = `qr-${customer?.slug ?? "fidel"}.png`
    link.href = url
    link.click()
  }

  const handleCopyUrl = async () => {
    try {
      await navigator.clipboard.writeText(joinUrl)
      toast.success("URL copiada")
    } catch {
      toast.error("No se pudo copiar la URL")
    }
  }

  const handleFinish = () => {
    // Limpiar drafts del localStorage — onboarding completo, ya no se
    // necesitan. Si el usuario vuelve a /registro, arranca desde cero.
    clearOnboardingDraft()
    navigate("/")
  }

  const totalRewards = rewardDrafts.length + cashbackRewardDrafts.length

  return (
    <div className="space-y-6">
      <div className="text-center">
        <div className="mb-2 flex justify-center">
          <CheckCircle2 className="h-12 w-12 text-green-500" />
        </div>
        <h2 className="text-xl font-semibold">¡Todo listo!</h2>
        <p className="text-sm text-muted-foreground">
          Tu programa de fidelidad esta configurado
        </p>
      </div>

      {/* Summary Card */}
      <Card>
        <CardContent className="space-y-4 p-6">
          <div>
            <p className="text-lg font-semibold">{customer?.name}</p>
            <p className="text-sm text-muted-foreground">{customer?.slug}</p>
          </div>

          <div className="flex flex-wrap gap-2">
            {earnBurnDraft && (
              <Badge variant="secondary" className="gap-1">
                <Star className="h-3 w-3" />
                {earnBurnDraft.name}
              </Badge>
            )}
            {cashbackDraft && (
              <Badge variant="secondary" className="gap-1">
                <Wallet className="h-3 w-3" />
                {cashbackDraft.name}
              </Badge>
            )}
            {pushcardDraft && (
              <Badge variant="secondary" className="gap-1">
                <Stamp className="h-3 w-3" />
                {pushcardDraft.name}
              </Badge>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4 text-center">
            <div className="rounded-lg bg-muted p-3">
              <p className="text-2xl font-bold">{totalRewards}</p>
              <p className="text-xs text-muted-foreground">Recompensas</p>
            </div>
            <div className="rounded-lg bg-muted p-3">
              <p className="text-2xl font-bold">{collaboratorDrafts.length}</p>
              <p className="text-xs text-muted-foreground">Colaboradores</p>
            </div>
          </div>
        </CardContent>
      </Card>

      {/* QR Code */}
      <div className="flex flex-col items-center space-y-4">
        <div ref={qrRef} className="rounded-lg border bg-white p-4">
          <QRCodeCanvas value={joinUrl} size={256} />
        </div>

        <p className="break-all text-center text-sm text-muted-foreground">
          {joinUrl}
        </p>

        <div className="flex gap-2">
          <Button variant="outline" size="sm" onClick={handleDownloadQR}>
            <Download className="mr-1.5 h-4 w-4" />
            Descargar QR
          </Button>
          <Button variant="outline" size="sm" onClick={handleCopyUrl}>
            <Copy className="mr-1.5 h-4 w-4" />
            Copiar URL
          </Button>
        </div>
      </div>

      <div className="flex justify-end">
        <Button onClick={handleFinish}>
          Ir al Dashboard
        </Button>
      </div>
    </div>
  )
}
