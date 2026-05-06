import { useRef, useState } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { Gift, Loader2, Plus, Upload, Trash2 } from "lucide-react"
import * as XLSX from "xlsx"
import type {
  DraftReward,
  DraftSisfi,
  PendingRewardInput,
} from "@/lib/wizard-draft"

interface StepRewardsProps {
  sisfi: DraftSisfi | null
  rewards: DraftReward[]
  pendingReward: PendingRewardInput
  onAddReward: (reward: Omit<DraftReward, "tempId">) => void
  onRemoveReward: (tempId: string) => void
  onSetRewards: (rewards: DraftReward[]) => void
  onPendingRewardChange: (pending: PendingRewardInput) => void
  onNext: () => void
  onPrev: () => void
}

interface ExcelRewardRow {
  nombre?: string
  Nombre?: string
  descripcion?: string
  Descripcion?: string
  costo?: number
  Costo?: number
  puntos?: number
  Puntos?: number
}

export function StepRewards({
  sisfi,
  rewards,
  pendingReward,
  onAddReward,
  onRemoveReward,
  onSetRewards,
  onPendingRewardChange,
  onNext,
  onPrev,
}: StepRewardsProps) {
  // Los inputs en proceso se mantienen en el estado del wizard para que
  // sobrevivan a navegación (Anterior/Siguiente) y al refresh de browser.
  const name = pendingReward.name
  const desc = pendingReward.description
  const cost = pendingReward.cost
  const setName = (v: string) =>
    onPendingRewardChange({ ...pendingReward, name: v })
  const setDesc = (v: string) =>
    onPendingRewardChange({ ...pendingReward, description: v })
  const setCost = (v: string) =>
    onPendingRewardChange({ ...pendingReward, cost: v })

  const [importing, setImporting] = useState(false)
  const fileRef = useRef<HTMLInputElement>(null)

  // Si no hay sisfi en el draft, no podemos pedir recompensas — el usuario
  // debió saltearse el paso 1. Lo regresamos.
  if (!sisfi) {
    return (
      <div className="space-y-6">
        <div className="text-center">
          <h2 className="text-xl font-semibold">Crea tus recompensas</h2>
          <p className="text-sm text-muted-foreground">
            Primero elige un programa de fidelidad en el paso anterior.
          </p>
        </div>
        <div className="flex justify-start">
          <Button variant="outline" onClick={onPrev}>
            Anterior
          </Button>
        </div>
      </div>
    )
  }

  const isPushcard = sisfi.type === "pushcard"
  const costLabel =
    sisfi.type === "earn_burn" ? "Puntos" : sisfi.type === "cashback" ? "Costo" : "—"

  const handleAdd = () => {
    if (isPushcard) return // pushcard no usa recompensas múltiples
    if (!name.trim()) {
      toast.error("Ingresa el nombre de la recompensa")
      return
    }
    const c = Number(cost)
    if (!c || c <= 0) {
      toast.error("Ingresa un costo válido")
      return
    }
    onAddReward({
      name: name.trim(),
      description: desc.trim(),
      cost: c,
    })
    // addReward limpia pendingReward en el reducer.
  }

  const handleImport = async (e: React.ChangeEvent<HTMLInputElement>) => {
    if (isPushcard) return
    const file = e.target.files?.[0]
    if (!file) return

    setImporting(true)
    try {
      const buf = await file.arrayBuffer()
      const wb = XLSX.read(buf, { type: "array" })
      const sheet = wb.Sheets[wb.SheetNames[0]]
      const rows = XLSX.utils.sheet_to_json<ExcelRewardRow>(sheet)
      const parsed: Omit<DraftReward, "tempId">[] = []
      for (const r of rows) {
        const n = r.nombre ?? r.Nombre ?? ""
        const d = r.descripcion ?? r.Descripcion ?? ""
        const c = Number(r.costo ?? r.Costo ?? r.puntos ?? r.Puntos ?? 0)
        if (n && c > 0) {
          parsed.push({ name: String(n).trim(), description: String(d).trim(), cost: c })
        }
      }
      if (parsed.length === 0) {
        toast.error("El archivo no tiene recompensas válidas")
        return
      }
      // Concatenar al draft existente.
      const newOnes: DraftReward[] = parsed.map((p) => ({
        ...p,
        tempId: `${Date.now()}-${Math.random().toString(36).slice(2, 7)}`,
      }))
      onSetRewards([...rewards, ...newOnes])
      toast.success(
        `${parsed.length} recompensa${parsed.length > 1 ? "s" : ""} importada${parsed.length > 1 ? "s" : ""}`,
      )
    } catch {
      toast.error("Error al leer el archivo")
    } finally {
      setImporting(false)
      if (fileRef.current) fileRef.current.value = ""
    }
  }

  const handleNext = () => {
    if (!isPushcard && rewards.length === 0) {
      toast.error("Agrega al menos una recompensa")
      return
    }
    onNext()
  }

  const programLabel = sisfi.name

  return (
    <div className="space-y-6">
      <div className="text-center">
        <h2 className="text-xl font-semibold">Crea tus recompensas</h2>
        <p className="text-sm text-muted-foreground">
          Agrega las recompensas que tus clientes podran obtener
        </p>
      </div>

      {isPushcard ? (
        <div className="rounded-lg border bg-muted/30 p-4 text-sm text-muted-foreground">
          Tu tarjeta tiene{" "}
          <span className="font-semibold text-foreground">{sisfi.slots}</span> sellos.
          La recompensa que dará al completarla se asigna después desde la sección{" "}
          <span className="font-semibold text-foreground">Tarjeta de sellos</span> del
          panel — así podés vincularla con un reward específico de tu catálogo.
        </div>
      ) : (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">
              Recompensas — {programLabel}
              {rewards.length > 0 && (
                <span className="ml-2 text-xs text-muted-foreground">
                  ({rewards.length})
                </span>
              )}
            </h3>
            <div>
              <input
                ref={fileRef}
                type="file"
                accept=".xlsx,.xls,.csv"
                className="hidden"
                onChange={handleImport}
              />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => fileRef.current?.click()}
                disabled={importing}
              >
                {importing ? (
                  <Loader2 className="mr-1.5 h-3.5 w-3.5 animate-spin" />
                ) : (
                  <Upload className="mr-1.5 h-3.5 w-3.5" />
                )}
                Excel
              </Button>
            </div>
          </div>

          <div className="rounded-lg border">
            <div className="grid grid-cols-[1fr_1fr_80px_36px] gap-2 border-b bg-muted/50 px-3 py-2 text-xs font-medium text-muted-foreground">
              <span>Nombre</span>
              <span>Descripcion</span>
              <span className="text-right">{costLabel}</span>
              <span />
            </div>

            {rewards.length > 0 && (
              <div className="max-h-[180px] overflow-y-auto">
                {rewards.map((r) => (
                  <div
                    key={r.tempId}
                    className="grid grid-cols-[1fr_1fr_80px_36px] items-center gap-2 border-b px-3 py-2 text-sm last:border-b-0"
                  >
                    <span className="truncate font-medium">{r.name}</span>
                    <span className="truncate text-muted-foreground">
                      {r.description || "—"}
                    </span>
                    <span className="text-right tabular-nums">{r.cost}</span>
                    <Button
                      size="icon"
                      variant="ghost"
                      className="h-7 w-7"
                      onClick={() => onRemoveReward(r.tempId)}
                    >
                      <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
                    </Button>
                  </div>
                ))}
              </div>
            )}

            {rewards.length === 0 && (
              <div className="flex items-center justify-center gap-2 px-3 py-6 text-sm text-muted-foreground">
                <Gift className="h-4 w-4" />
                <span>Agrega tu primera recompensa</span>
              </div>
            )}

            <div className="grid grid-cols-[1fr_1fr_80px_36px] items-center gap-2 border-t bg-muted/30 px-3 py-2">
              <Input
                placeholder="Cafe gratis"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="h-8"
                onKeyDown={(e) => e.key === "Enter" && handleAdd()}
              />
              <Input
                placeholder="Descripcion"
                value={desc}
                onChange={(e) => setDesc(e.target.value)}
                className="h-8"
                onKeyDown={(e) => e.key === "Enter" && handleAdd()}
              />
              <Input
                placeholder={sisfi.type === "earn_burn" ? "100" : "50"}
                type="number"
                min={1}
                value={cost}
                onChange={(e) => setCost(e.target.value)}
                className="h-8 text-right"
                onKeyDown={(e) => e.key === "Enter" && handleAdd()}
              />
              <Button
                size="icon"
                variant="ghost"
                className="h-8 w-8"
                onClick={handleAdd}
              >
                <Plus className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      )}

      <div className="flex justify-between">
        <Button variant="outline" onClick={onPrev}>
          Anterior
        </Button>
        <Button onClick={handleNext}>Siguiente</Button>
      </div>
    </div>
  )
}
