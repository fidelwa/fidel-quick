import { useState } from "react"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { Star, Wallet, Stamp } from "lucide-react"
import { cn } from "@/lib/utils"
import type {
  EarnBurnDraft,
  CashbackDraft,
  PushcardDraft,
} from "@/hooks/use-onboarding"

interface StepProgramProps {
  earnBurnDraft: EarnBurnDraft | null
  cashbackDraft: CashbackDraft | null
  pushcardDraft: PushcardDraft | null
  onEarnBurnChange: (draft: EarnBurnDraft | null) => void
  onCashbackChange: (draft: CashbackDraft | null) => void
  onPushcardChange: (draft: PushcardDraft | null) => void
  onNext: () => void
}

export function StepProgram({
  earnBurnDraft,
  cashbackDraft,
  pushcardDraft,
  onEarnBurnChange,
  onCashbackChange,
  onPushcardChange,
  onNext,
}: StepProgramProps) {
  const [earnSelected, setEarnSelected] = useState(!!earnBurnDraft)
  const [cashbackSelected, setCashbackSelected] = useState(!!cashbackDraft)
  const [pushcardSelected, setPushcardSelected] = useState(!!pushcardDraft)

  const [earnName, setEarnName] = useState(earnBurnDraft?.name ?? "")
  const [earnRatio, setEarnRatio] = useState(
    earnBurnDraft ? String(earnBurnDraft.points_ratio) : "15"
  )

  const [cashbackName, setCashbackName] = useState(cashbackDraft?.name ?? "")
  const [cashbackRate, setCashbackRate] = useState(
    cashbackDraft ? String(cashbackDraft.cashback_rate) : "5"
  )

  const [pushcardName, setPushcardName] = useState(pushcardDraft?.name ?? "")
  const [pushcardSlots, setPushcardSlots] = useState(
    pushcardDraft ? String(pushcardDraft.card_slots) : "10"
  )
  const [pushcardReward, setPushcardReward] = useState(
    pushcardDraft?.reward_on_complete ?? ""
  )

  const handleNext = () => {
    if (!earnSelected && !cashbackSelected && !pushcardSelected) {
      toast.error("Selecciona al menos un tipo de programa")
      return
    }

    if (earnSelected) {
      if (!earnName.trim()) {
        toast.error("Ingresa el nombre del programa de puntos")
        return
      }
      const ratio = Number(earnRatio)
      if (!ratio || ratio < 1) {
        toast.error("El ratio de puntos debe ser mayor a 0")
        return
      }
      onEarnBurnChange({ name: earnName.trim(), points_ratio: ratio })
    } else {
      onEarnBurnChange(null)
    }

    if (cashbackSelected) {
      if (!cashbackName.trim()) {
        toast.error("Ingresa el nombre del programa de cashback")
        return
      }
      const rate = Number(cashbackRate)
      if (!rate || rate < 1 || rate > 100) {
        toast.error("El % de cashback debe estar entre 1 y 100")
        return
      }
      onCashbackChange({ name: cashbackName.trim(), cashback_rate: rate })
    } else {
      onCashbackChange(null)
    }

    if (pushcardSelected) {
      if (!pushcardName.trim()) {
        toast.error("Ingresa el nombre del programa de tarjeta de sellos")
        return
      }
      const slots = Number(pushcardSlots)
      if (!slots || slots < 2 || slots > 50) {
        toast.error("La tarjeta debe tener entre 2 y 50 sellos")
        return
      }
      onPushcardChange({
        name: pushcardName.trim(),
        card_slots: slots,
        reward_on_complete: pushcardReward.trim() || undefined,
      })
    } else {
      onPushcardChange(null)
    }

    onNext()
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Elige tu programa de fidelidad</h2>
        <p className="text-sm text-muted-foreground">
          Selecciona uno o varios. Pod&eacute;s combinarlos.
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        {/* Earn-Burn Card */}
        <button
          type="button"
          onClick={() => setEarnSelected(!earnSelected)}
          className={cn(
            "rounded-lg border-2 p-4 text-left transition-all duration-200",
            earnSelected
              ? "border-primary bg-primary/5 shadow-sm"
              : "border-border hover:border-primary/30"
          )}
        >
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-amber-100 text-amber-600">
              <Star className="h-5 w-5" />
            </div>
            <div>
              <p className="font-medium">Puntos</p>
              <p className="text-xs text-muted-foreground">Acumula y canjea puntos</p>
            </div>
          </div>
        </button>

        {/* Cashback Card */}
        <button
          type="button"
          onClick={() => setCashbackSelected(!cashbackSelected)}
          className={cn(
            "rounded-lg border-2 p-4 text-left transition-all duration-200",
            cashbackSelected
              ? "border-primary bg-primary/5 shadow-sm"
              : "border-border hover:border-primary/30"
          )}
        >
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-green-100 text-green-600">
              <Wallet className="h-5 w-5" />
            </div>
            <div>
              <p className="font-medium">Cashback</p>
              <p className="text-xs text-muted-foreground">Porcentaje de devolucion</p>
            </div>
          </div>
        </button>

        {/* Pushcard Card */}
        <button
          type="button"
          onClick={() => setPushcardSelected(!pushcardSelected)}
          className={cn(
            "rounded-lg border-2 p-4 text-left transition-all duration-200",
            pushcardSelected
              ? "border-primary bg-primary/5 shadow-sm"
              : "border-border hover:border-primary/30"
          )}
        >
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-rose-100 text-rose-600">
              <Stamp className="h-5 w-5" />
            </div>
            <div>
              <p className="font-medium">Tarjeta de sellos</p>
              <p className="text-xs text-muted-foreground">Acumula sellos y completa</p>
            </div>
          </div>
        </button>
      </div>

      {/* Earn-Burn Config */}
      <div
        className={cn(
          "grid transition-all duration-200",
          earnSelected ? "grid-rows-[1fr]" : "grid-rows-[0fr]"
        )}
      >
        <div className="overflow-hidden">
          <div className="space-y-3 pt-2">
            <h3 className="text-sm font-medium">Configurar programa de puntos</h3>
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="earn-name">Nombre del programa</Label>
                <Input
                  id="earn-name"
                  placeholder="Programa de puntos"
                  value={earnName}
                  onChange={(e) => setEarnName(e.target.value)}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="earn-ratio">1 punto por cada $</Label>
                <Input
                  id="earn-ratio"
                  type="number"
                  min={1}
                  value={earnRatio}
                  onChange={(e) => setEarnRatio(e.target.value)}
                />
              </div>
            </div>
            {Number(earnRatio) > 0 && (
              <p className="text-xs text-muted-foreground">
                Ejemplo: compra de $150 = <span className="font-semibold text-foreground">{Math.floor(150 / Number(earnRatio)).toLocaleString()} puntos</span>
              </p>
            )}
          </div>
        </div>
      </div>

      {/* Cashback Config */}
      <div
        className={cn(
          "grid transition-all duration-200",
          cashbackSelected ? "grid-rows-[1fr]" : "grid-rows-[0fr]"
        )}
      >
        <div className="overflow-hidden">
          <div className="space-y-3 pt-2">
            <h3 className="text-sm font-medium">Configurar programa de cashback</h3>
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="cashback-name">Nombre del programa</Label>
                <Input
                  id="cashback-name"
                  placeholder="Programa de cashback"
                  value={cashbackName}
                  onChange={(e) => setCashbackName(e.target.value)}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="cashback-rate">% Cashback</Label>
                <Input
                  id="cashback-rate"
                  type="number"
                  min={1}
                  max={100}
                  value={cashbackRate}
                  onChange={(e) => setCashbackRate(e.target.value)}
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Pushcard Config */}
      <div
        className={cn(
          "grid transition-all duration-200",
          pushcardSelected ? "grid-rows-[1fr]" : "grid-rows-[0fr]"
        )}
      >
        <div className="overflow-hidden">
          <div className="space-y-3 pt-2">
            <h3 className="text-sm font-medium">Configurar tarjeta de sellos</h3>
            <div className="grid gap-3 sm:grid-cols-2">
              <div className="space-y-1.5">
                <Label htmlFor="pushcard-name">Nombre del programa</Label>
                <Input
                  id="pushcard-name"
                  placeholder="Tarjeta de sellos"
                  value={pushcardName}
                  onChange={(e) => setPushcardName(e.target.value)}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="pushcard-slots">Sellos para completar</Label>
                <Input
                  id="pushcard-slots"
                  type="number"
                  min={2}
                  max={50}
                  value={pushcardSlots}
                  onChange={(e) => setPushcardSlots(e.target.value)}
                />
              </div>
              <div className="space-y-1.5 sm:col-span-2">
                <Label htmlFor="pushcard-reward">Recompensa al completar</Label>
                <Input
                  id="pushcard-reward"
                  placeholder="Ej. Café gratis"
                  value={pushcardReward}
                  onChange={(e) => setPushcardReward(e.target.value)}
                />
              </div>
            </div>
            <p className="text-xs text-muted-foreground">
              Lo que el cliente recibe al llenar la tarjeta. Podés dejarlo vacío y
              configurarlo luego desde el panel de pushcard.
            </p>
          </div>
        </div>
      </div>

      <div className="flex justify-end">
        <Button onClick={handleNext}>Siguiente</Button>
      </div>
    </div>
  )
}
