import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { Star, Wallet, Stamp } from "lucide-react"
import { cn } from "@/lib/utils"
import type {
  DraftSisfi,
  PendingProgramForm,
  SisfiType,
} from "@/lib/wizard-draft"

interface StepProgramProps {
  sisfi: DraftSisfi | null
  pendingProgramForm: PendingProgramForm
  onSisfiChange: (sisfi: DraftSisfi | null) => void
  onPendingProgramFormChange: (form: PendingProgramForm) => void
  onNext: () => void
}

export function StepProgram({
  sisfi,
  pendingProgramForm,
  onSisfiChange,
  onPendingProgramFormChange,
  onNext,
}: StepProgramProps) {
  // Todos los inputs viven en `pendingProgramForm` para persistir en el draft.
  const selected = pendingProgramForm.selected
  const earnName = pendingProgramForm.earnName
  const earnRatio = pendingProgramForm.earnRatio
  const cashbackName = pendingProgramForm.cashbackName
  const cashbackRate = pendingProgramForm.cashbackRate
  const pushcardName = pendingProgramForm.pushcardName
  const pushcardSlots = pendingProgramForm.pushcardSlots

  const update = (patch: Partial<PendingProgramForm>) =>
    onPendingProgramFormChange({ ...pendingProgramForm, ...patch })

  const setSelected = (next: SisfiType | null) => update({ selected: next })
  const setEarnName = (v: string) => update({ earnName: v })
  const setEarnRatio = (v: string) => update({ earnRatio: v })
  const setCashbackName = (v: string) => update({ cashbackName: v })
  const setCashbackRate = (v: string) => update({ cashbackRate: v })
  const setPushcardName = (v: string) => update({ pushcardName: v })
  const setPushcardSlots = (v: string) => update({ pushcardSlots: v })


  const handleNext = () => {
    if (!selected) {
      toast.error("Selecciona un programa de fidelidad")
      return
    }

    if (selected === "earn_burn") {
      if (!earnName.trim()) {
        toast.error("Ingresa el nombre del programa de puntos")
        return
      }
      onSisfiChange({
        type: "earn_burn",
        name: earnName.trim(),
        ratio: Number(earnRatio) || 15,
      })
    } else if (selected === "cashback") {
      if (!cashbackName.trim()) {
        toast.error("Ingresa el nombre del programa de cashback")
        return
      }
      onSisfiChange({
        type: "cashback",
        name: cashbackName.trim(),
        rate: Number(cashbackRate) || 5,
      })
    } else if (selected === "pushcard") {
      const slots = Number(pushcardSlots)
      if (!slots || slots < 1 || slots > 50) {
        toast.error("Sellos por tarjeta debe estar entre 1 y 50")
        return
      }
      onSisfiChange({
        type: "pushcard",
        name: pushcardName.trim() || "Tarjeta de sellos",
        slots,
      })
    }

    onNext()
  }

  const pickSelected = (type: SisfiType) => {
    update({ selected: selected === type ? null : type })
  }

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-xl font-semibold">Elige tu programa de fidelidad</h2>
        <p className="text-sm text-muted-foreground">
          Selecciona el tipo de programa para tu negocio
        </p>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        <button
          type="button"
          onClick={() => pickSelected("earn_burn")}
          className={cn(
            "rounded-lg border-2 p-4 text-left transition-all duration-200",
            selected === "earn_burn"
              ? "border-primary bg-primary/5 shadow-sm"
              : "border-border hover:border-primary/30",
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

        <button
          type="button"
          onClick={() => pickSelected("cashback")}
          className={cn(
            "rounded-lg border-2 p-4 text-left transition-all duration-200",
            selected === "cashback"
              ? "border-primary bg-primary/5 shadow-sm"
              : "border-border hover:border-primary/30",
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

        <button
          type="button"
          onClick={() => pickSelected("pushcard")}
          className={cn(
            "rounded-lg border-2 p-4 text-left transition-all duration-200",
            selected === "pushcard"
              ? "border-primary bg-primary/5 shadow-sm"
              : "border-border hover:border-primary/30",
          )}
        >
          <div className="flex items-center gap-3">
            <div className="flex h-10 w-10 items-center justify-center rounded-full bg-purple-100 text-purple-600">
              <Stamp className="h-5 w-5" />
            </div>
            <div>
              <p className="font-medium">Tarjeta de sellos</p>
              <p className="text-xs text-muted-foreground">Sellos hasta completar</p>
            </div>
          </div>
        </button>
      </div>

      <div
        className={cn(
          "grid transition-all duration-200",
          selected === "earn_burn" ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
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
                Ejemplo: compra de $150 ={" "}
                <span className="font-semibold text-foreground">
                  {Math.floor(150 / Number(earnRatio)).toLocaleString()} puntos
                </span>
              </p>
            )}
          </div>
        </div>
      </div>

      <div
        className={cn(
          "grid transition-all duration-200",
          selected === "cashback" ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
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

      <div
        className={cn(
          "grid transition-all duration-200",
          selected === "pushcard" ? "grid-rows-[1fr]" : "grid-rows-[0fr]",
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
                <Label htmlFor="pushcard-slots">Sellos por tarjeta</Label>
                <Input
                  id="pushcard-slots"
                  type="number"
                  min={1}
                  max={50}
                  step={1}
                  value={pushcardSlots}
                  onChange={(e) => setPushcardSlots(e.target.value)}
                />
              </div>
            </div>
            <p className="text-xs text-muted-foreground">
              La recompensa al completar la tarjeta se asigna en el siguiente paso.
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
