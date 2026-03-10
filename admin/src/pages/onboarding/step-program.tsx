import { useState } from "react"
import { useAuth } from "@/context/auth-context"
import { useCreateProgram } from "@/hooks/use-programs"
import { useCreateCashbackProgram } from "@/hooks/use-cashback-programs"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { Star, Wallet, Loader2 } from "lucide-react"
import { cn } from "@/lib/utils"
import type { Program, CashbackProgram } from "@/types"

interface StepProgramProps {
  earnBurnProgram: Program | null
  cashbackProgram: CashbackProgram | null
  onEarnBurnCreated: (program: Program | null) => void
  onCashbackCreated: (program: CashbackProgram | null) => void
  onNext: () => void
}

export function StepProgram({
  earnBurnProgram,
  cashbackProgram,
  onEarnBurnCreated,
  onCashbackCreated,
  onNext,
}: StepProgramProps) {
  const { customerId } = useAuth()
  const createProgram = useCreateProgram()
  const createCashbackProgram = useCreateCashbackProgram()

  const [earnSelected, setEarnSelected] = useState(!!earnBurnProgram)
  const [cashbackSelected, setCashbackSelected] = useState(!!cashbackProgram)

  const [earnName, setEarnName] = useState(earnBurnProgram?.name ?? "")
  const [earnRatio, setEarnRatio] = useState(earnBurnProgram?.points_ratio ?? 100)

  const [cashbackName, setCashbackName] = useState(cashbackProgram?.name ?? "")
  const [cashbackRate, setCashbackRate] = useState(cashbackProgram?.cashback_rate ?? 5)

  const [saving, setSaving] = useState(false)

  const handleNext = async () => {
    if (!earnSelected && !cashbackSelected) {
      toast.error("Selecciona al menos un tipo de programa")
      return
    }

    setSaving(true)
    try {
      if (earnSelected && !earnBurnProgram) {
        if (!earnName.trim()) {
          toast.error("Ingresa el nombre del programa de puntos")
          setSaving(false)
          return
        }
        const program = await createProgram.mutateAsync({
          customer_id: customerId,
          type: "earn-burn",
          name: earnName.trim(),
          points_ratio: earnRatio,
        })
        onEarnBurnCreated(program)
      }

      if (cashbackSelected && !cashbackProgram) {
        if (!cashbackName.trim()) {
          toast.error("Ingresa el nombre del programa de cashback")
          setSaving(false)
          return
        }
        const program = await createCashbackProgram.mutateAsync({
          customer_id: customerId,
          name: cashbackName.trim(),
          cashback_rate: cashbackRate,
        })
        onCashbackCreated(program)
      }

      if (!earnSelected && earnBurnProgram) {
        onEarnBurnCreated(null)
      }
      if (!cashbackSelected && cashbackProgram) {
        onCashbackCreated(null)
      }

      onNext()
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al crear programa")
    } finally {
      setSaving(false)
    }
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Elige tu programa de fidelidad</h2>
        <p className="text-sm text-muted-foreground">
          Selecciona uno o ambos tipos de programa para tu negocio
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-2">
        {/* Earn-Burn Card */}
        <button
          type="button"
          onClick={() => !earnBurnProgram && setEarnSelected(!earnSelected)}
          className={cn(
            "rounded-lg border-2 p-4 text-left transition-all duration-200",
            earnSelected || earnBurnProgram
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
          {earnBurnProgram && (
            <p className="mt-2 text-xs text-green-600 font-medium">Creado</p>
          )}
        </button>

        {/* Cashback Card */}
        <button
          type="button"
          onClick={() => !cashbackProgram && setCashbackSelected(!cashbackSelected)}
          className={cn(
            "rounded-lg border-2 p-4 text-left transition-all duration-200",
            cashbackSelected || cashbackProgram
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
          {cashbackProgram && (
            <p className="mt-2 text-xs text-green-600 font-medium">Creado</p>
          )}
        </button>
      </div>

      {/* Earn-Burn Config */}
      <div
        className={cn(
          "grid transition-all duration-200",
          earnSelected || earnBurnProgram ? "grid-rows-[1fr]" : "grid-rows-[0fr]"
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
                  disabled={!!earnBurnProgram}
                />
              </div>
              <div className="space-y-1.5">
                <Label htmlFor="earn-ratio">Puntos por compra</Label>
                <Input
                  id="earn-ratio"
                  type="number"
                  min={1}
                  value={earnRatio}
                  onChange={(e) => setEarnRatio(Number(e.target.value))}
                  disabled={!!earnBurnProgram}
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      {/* Cashback Config */}
      <div
        className={cn(
          "grid transition-all duration-200",
          cashbackSelected || cashbackProgram ? "grid-rows-[1fr]" : "grid-rows-[0fr]"
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
                  disabled={!!cashbackProgram}
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
                  onChange={(e) => setCashbackRate(Number(e.target.value))}
                  disabled={!!cashbackProgram}
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div className="flex justify-end">
        <Button onClick={handleNext} disabled={saving}>
          {saving ? (
            <>
              <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              Guardando...
            </>
          ) : (
            "Siguiente"
          )}
        </Button>
      </div>
    </div>
  )
}
