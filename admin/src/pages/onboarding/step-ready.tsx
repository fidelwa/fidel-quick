import { useRef, useState } from "react"
import { useNavigate } from "react-router-dom"
import { useQueryClient } from "@tanstack/react-query"
import { useAuth } from "@/context/auth-context"
import { useCustomer } from "@/hooks/use-customer"
import { useCompleteOnboarding } from "@/hooks/use-onboarding-status"
import { QRCodeCanvas } from "qrcode.react"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { Card, CardContent } from "@/components/ui/card"
import { toast } from "sonner"
import { Download, Copy, Loader2, Star, Wallet, Stamp } from "lucide-react"
import {
  createProgram,
  createCashbackProgram,
  createPushcardProgram,
  createReward,
  createCashbackReward,
  createCollaborator,
  getCustomerSisfi,
  updateCustomerSisfi,
} from "@/lib/api-client"
import { clearWizardDraft, type DraftSisfi, type DraftReward, type DraftCollaborator } from "@/lib/wizard-draft"

interface StepReadyProps {
  sisfi: DraftSisfi | null
  rewards: DraftReward[]
  collaborators: DraftCollaborator[]
  onPrev: () => void
}

export function StepReady({ sisfi, rewards, collaborators, onPrev }: StepReadyProps) {
  const { customerId } = useAuth()
  const queryClient = useQueryClient()
  const { data: customer } = useCustomer(customerId)
  const completeOnboarding = useCompleteOnboarding()
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
    if (!sisfi) {
      toast.error("Falta elegir un programa de fidelidad")
      return
    }
    if (sisfi.type !== "pushcard" && rewards.length === 0) {
      toast.error("Falta agregar al menos una recompensa")
      return
    }
    if (collaborators.length === 0) {
      toast.error("Falta registrar al menos un colaborador")
      return
    }

    setFinishing(true)
    try {
      // 1. Limpiar sisfis previos del negocio (desactivar) — sólo permitimos
      //    1 sisfi activo a la vez. Esto cubre el caso de re-onboarding
      //    después de haber dejado un sisfi creado en una corrida anterior.
      const existing = await getCustomerSisfi(customerId)
      const sisfiTypeMap: Record<string, string> = {
        earn_burn: "earn_burn",
        cashback: "cashback",
        pushcard: "pushcard",
      }
      const desiredSisfiId = sisfiTypeMap[sisfi.type]
      for (const cs of existing) {
        if (!cs.active) continue
        if (cs.sisfi_id !== desiredSisfiId) {
          await updateCustomerSisfi(cs.id, { active: false })
        }
      }

      // 2. Crear el sisfi seleccionado.
      let customerSisfiId: string
      if (sisfi.type === "earn_burn") {
        const program = await createProgram({
          customer_id: customerId,
          name: sisfi.name,
          points_ratio: sisfi.ratio ?? 15,
        })
        customerSisfiId = program.id
        for (const r of rewards) {
          await createReward(customerSisfiId, {
            name: r.name,
            description: r.description,
            points_cost: r.cost,
          })
        }
      } else if (sisfi.type === "cashback") {
        const program = await createCashbackProgram({
          customer_id: customerId,
          name: sisfi.name,
          cashback_rate: sisfi.rate ?? 5,
        })
        customerSisfiId = program.id
        for (const r of rewards) {
          await createCashbackReward(customerSisfiId, {
            name: r.name,
            description: r.description,
            cost: r.cost,
          })
        }
      } else {
        // pushcard
        const cfg = await createPushcardProgram({
          customer_id: customerId,
          name: sisfi.name,
          card_slots: sisfi.slots ?? 10,
        })
        customerSisfiId = cfg.customer_sisfi_id
        // pushcard no recibe rewards en el wizard — se asigna después.
      }

      // 3. Crear colaboradores.
      for (const col of collaborators) {
        await createCollaborator(customerId, {
          name: col.name,
          phone: col.phone,
        })
      }

      // 4. Marcar onboarding completo.
      await completeOnboarding.mutateAsync()

      // 5. Limpiar draft local.
      clearWizardDraft()

      // 6. Refrescar caches.
      queryClient.invalidateQueries({ queryKey: ["programs"] })
      queryClient.invalidateQueries({ queryKey: ["cashback-programs"] })
      queryClient.invalidateQueries({ queryKey: ["pushcard-config"] })
      queryClient.invalidateQueries({ queryKey: ["collaborators"] })
      queryClient.invalidateQueries({ queryKey: ["customer-sisfi"] })

      toast.success("¡Programa creado!")
      navigate("/")
    } catch (err) {
      toast.error(
        err instanceof Error ? err.message : "Error al finalizar el onboarding",
      )
    } finally {
      setFinishing(false)
    }
  }

  const sisfiBadge = (() => {
    if (!sisfi) return null
    if (sisfi.type === "earn_burn") {
      return (
        <Badge variant="secondary" className="gap-1">
          <Star className="h-3 w-3" />
          {sisfi.name}
        </Badge>
      )
    }
    if (sisfi.type === "cashback") {
      return (
        <Badge variant="secondary" className="gap-1">
          <Wallet className="h-3 w-3" />
          {sisfi.name}
        </Badge>
      )
    }
    return (
      <Badge variant="secondary" className="gap-1">
        <Stamp className="h-3 w-3" />
        {sisfi.name}
      </Badge>
    )
  })()

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-xl font-semibold">¡Todo listo!</h2>
        <p className="text-sm text-muted-foreground">
          Revisá la configuración y presioná "Crear programa" para guardar todo.
        </p>
      </div>

      <Card>
        <CardContent className="space-y-4 p-6">
          <div>
            <p className="text-lg font-semibold">{customer?.name}</p>
            <p className="text-sm text-muted-foreground">{customer?.slug}</p>
          </div>

          <div className="flex flex-wrap gap-2">{sisfiBadge}</div>

          <div className="grid grid-cols-2 gap-4 text-center">
            <div className="rounded-lg bg-muted p-3">
              <p className="text-2xl font-bold">{rewards.length}</p>
              <p className="text-xs text-muted-foreground">Recompensas</p>
            </div>
            <div className="rounded-lg bg-muted p-3">
              <p className="text-2xl font-bold">{collaborators.length}</p>
              <p className="text-xs text-muted-foreground">Colaboradores</p>
            </div>
          </div>
        </CardContent>
      </Card>

      <div className="flex flex-col items-center space-y-4">
        <div ref={qrRef} className="rounded-lg border bg-white p-4">
          <QRCodeCanvas value={joinUrl} size={256} />
        </div>

        <p className="text-sm text-muted-foreground">{joinUrl}</p>

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
        <Button variant="outline" onClick={onPrev} disabled={finishing}>
          Anterior
        </Button>
        <Button onClick={handleFinish} disabled={finishing}>
          {finishing ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Creando programa…
            </>
          ) : (
            "Crear programa"
          )}
        </Button>
      </div>
    </div>
  )
}
