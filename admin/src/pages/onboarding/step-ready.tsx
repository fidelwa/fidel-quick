import { useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useAuth } from "@/context/auth-context"
import { useCustomer, useUpdateCustomer } from "@/hooks/use-customer"
import { QRCodeCanvas } from "qrcode.react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { toast } from "sonner"
import { Download, Copy, Loader2, Star, Wallet } from "lucide-react"
import type { Program, CashbackProgram, Reward, CashbackReward, Collaborator } from "@/types"

interface StepReadyProps {
  earnBurnProgram: Program | null
  cashbackProgram: CashbackProgram | null
  rewards: Reward[]
  cashbackRewards: CashbackReward[]
  collaborators: Collaborator[]
  onPrev: () => void
}

export function StepReady({
  earnBurnProgram,
  cashbackProgram,
  rewards,
  cashbackRewards,
  collaborators,
  onPrev,
}: StepReadyProps) {
  const { customerId } = useAuth()
  const { data: customer } = useCustomer(customerId)
  const updateCustomer = useUpdateCustomer(customerId)
  const navigate = useNavigate()
  const qrRef = useRef<HTMLDivElement>(null)
  const [finishing, setFinishing] = useState(false)

  const joinUrl = `https://fidel.app/unirse/${customer?.slug ?? ""}`

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

  const handleFinish = async () => {
    setFinishing(true)
    try {
      await updateCustomer.mutateAsync({ onboarding_completed: true } as Partial<import("@/types").Customer>)
      navigate("/")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al finalizar")
    } finally {
      setFinishing(false)
    }
  }

  return (
    <div className="space-y-6">
      <div className="text-center">
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
            {earnBurnProgram && (
              <Badge variant="secondary" className="gap-1">
                <Star className="h-3 w-3" />
                {earnBurnProgram.name}
              </Badge>
            )}
            {cashbackProgram && (
              <Badge variant="secondary" className="gap-1">
                <Wallet className="h-3 w-3" />
                {cashbackProgram.name}
              </Badge>
            )}
          </div>

          <div className="grid grid-cols-2 gap-4 text-center">
            <div className="rounded-lg bg-muted p-3">
              <p className="text-2xl font-bold">{rewards.length + cashbackRewards.length}</p>
              <p className="text-xs text-muted-foreground">Recompensas</p>
            </div>
            <div className="rounded-lg bg-muted p-3">
              <p className="text-2xl font-bold">{collaborators.length}</p>
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

        <p className="text-sm text-muted-foreground">
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

      <div className="flex justify-between">
        <Button variant="outline" onClick={onPrev}>
          Anterior
        </Button>
        <Button onClick={handleFinish} disabled={finishing}>
          {finishing ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Finalizando...
            </>
          ) : (
            "Ir al Dashboard"
          )}
        </Button>
      </div>
    </div>
  )
}
