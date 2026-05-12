import { useState, useRef } from "react"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { Gift, Loader2, Plus, Upload, Check } from "lucide-react"
import * as XLSX from "xlsx"
import {
  generateLocalId,
  type EarnBurnDraft,
  type CashbackDraft,
  type PushcardDraft,
  type RewardDraft,
  type CashbackRewardDraft,
} from "@/hooks/use-onboarding"

interface StepRewardsProps {
  earnBurnDraft: EarnBurnDraft | null
  cashbackDraft: CashbackDraft | null
  pushcardDraft: PushcardDraft | null
  rewardDrafts: RewardDraft[]
  cashbackRewardDrafts: CashbackRewardDraft[]
  onRewardsChange: (drafts: RewardDraft[]) => void
  onCashbackRewardsChange: (drafts: CashbackRewardDraft[]) => void
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
  earnBurnDraft,
  cashbackDraft,
  pushcardDraft,
  rewardDrafts,
  cashbackRewardDrafts,
  onRewardsChange,
  onCashbackRewardsChange,
  onNext,
  onPrev,
}: StepRewardsProps) {
  const [rewardName, setRewardName] = useState("")
  const [rewardDesc, setRewardDesc] = useState("")
  const [rewardCost, setRewardCost] = useState("")

  const [cbRewardName, setCbRewardName] = useState("")
  const [cbRewardDesc, setCbRewardDesc] = useState("")
  const [cbRewardCost, setCbRewardCost] = useState("")

  const [importing, setImporting] = useState(false)

  const earnFileRef = useRef<HTMLInputElement>(null)
  const cbFileRef = useRef<HTMLInputElement>(null)

  const totalRewards = rewardDrafts.length + cashbackRewardDrafts.length

  const handleAddReward = () => {
    if (!rewardName.trim()) {
      toast.error("Ingresa el nombre de la recompensa")
      return
    }
    const cost = Number(rewardCost) || 100
    const draft: RewardDraft = {
      local_id: generateLocalId(),
      name: rewardName.trim(),
      description: rewardDesc.trim(),
      points_cost: cost,
    }
    onRewardsChange([...rewardDrafts, draft])
    setRewardName("")
    setRewardDesc("")
    setRewardCost("")
  }

  const handleAddCbReward = () => {
    if (!cbRewardName.trim()) {
      toast.error("Ingresa el nombre del beneficio")
      return
    }
    const cost = Number(cbRewardCost) || 50
    const draft: CashbackRewardDraft = {
      local_id: generateLocalId(),
      name: cbRewardName.trim(),
      description: cbRewardDesc.trim(),
      cost,
    }
    onCashbackRewardsChange([...cashbackRewardDrafts, draft])
    setCbRewardName("")
    setCbRewardDesc("")
    setCbRewardCost("")
  }

  const parseExcelFile = (file: File): Promise<ExcelRewardRow[]> => {
    return new Promise((resolve, reject) => {
      const reader = new FileReader()
      reader.onload = (e) => {
        try {
          const data = new Uint8Array(e.target?.result as ArrayBuffer)
          const workbook = XLSX.read(data, { type: "array" })
          const sheet = workbook.Sheets[workbook.SheetNames[0]]
          const rows = XLSX.utils.sheet_to_json<ExcelRewardRow>(sheet)
          resolve(rows)
        } catch {
          reject(new Error("No se pudo leer el archivo Excel"))
        }
      }
      reader.onerror = () => reject(new Error("Error al leer el archivo"))
      reader.readAsArrayBuffer(file)
    })
  }

  const handleImportEarnBurn = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = ""

    setImporting(true)
    try {
      const rows = await parseExcelFile(file)
      if (rows.length === 0) {
        toast.error("El archivo esta vacio")
        return
      }

      const created: RewardDraft[] = []
      for (const row of rows) {
        const name = (row.nombre ?? row.Nombre ?? "").toString().trim()
        const description = (row.descripcion ?? row.Descripcion ?? "").toString().trim()
        const pointsCost = Number(row.puntos ?? row.Puntos ?? row.costo ?? row.Costo ?? 0)

        if (!name || pointsCost <= 0) continue

        created.push({
          local_id: generateLocalId(),
          name,
          description,
          points_cost: pointsCost,
        })
      }

      if (created.length > 0) {
        onRewardsChange([...rewardDrafts, ...created])
        toast.success(`${created.length} recompensa${created.length > 1 ? "s" : ""} importada${created.length > 1 ? "s" : ""}`)
      } else {
        toast.error("No se encontraron filas validas. Columnas esperadas: Nombre, Descripcion, Puntos")
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al importar")
    } finally {
      setImporting(false)
    }
  }

  const handleImportCashback = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = ""

    setImporting(true)
    try {
      const rows = await parseExcelFile(file)
      if (rows.length === 0) {
        toast.error("El archivo esta vacio")
        return
      }

      const created: CashbackRewardDraft[] = []
      for (const row of rows) {
        const name = (row.nombre ?? row.Nombre ?? "").toString().trim()
        const description = (row.descripcion ?? row.Descripcion ?? "").toString().trim()
        const cost = Number(row.costo ?? row.Costo ?? row.puntos ?? row.Puntos ?? 0)

        if (!name || cost <= 0) continue

        created.push({
          local_id: generateLocalId(),
          name,
          description,
          cost,
        })
      }

      if (created.length > 0) {
        onCashbackRewardsChange([...cashbackRewardDrafts, ...created])
        toast.success(`${created.length} beneficio${created.length > 1 ? "s" : ""} importado${created.length > 1 ? "s" : ""}`)
      } else {
        toast.error("No se encontraron filas validas. Columnas esperadas: Nombre, Descripcion, Costo")
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al importar")
    } finally {
      setImporting(false)
    }
  }

  const handleNext = () => {
    // Si solo hay pushcard (sin earnburn ni cashback), no hace falta crear
    // recompensas en este step — la recompensa de pushcard se configura
    // luego en /admin/pushcard.
    const onlyPushcard = !earnBurnDraft && !cashbackDraft && !!pushcardDraft
    if (!onlyPushcard && totalRewards === 0) {
      toast.error("Crea al menos una recompensa")
      return
    }
    onNext()
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Crea tus recompensas</h2>
        <p className="text-sm text-muted-foreground">
          Agrega las recompensas que tus clientes podran obtener
        </p>
      </div>

      {/* Pushcard solo: no hay rewards en este step */}
      {!earnBurnDraft && !cashbackDraft && pushcardDraft && (
        <div className="rounded-lg border border-dashed p-6 text-center text-sm text-muted-foreground">
          Tu programa de tarjeta de sellos no necesita recompensas en este paso.
          Configura la recompensa al completar la tarjeta luego desde
          <span className="mx-1 font-medium text-foreground">/admin/pushcard</span>.
        </div>
      )}

      {/* Earn-Burn Rewards */}
      {earnBurnDraft && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">
              Recompensas de Puntos — {earnBurnDraft.name}
              {rewardDrafts.length > 0 && (
                <span className="ml-2 text-xs text-muted-foreground">
                  ({rewardDrafts.length})
                </span>
              )}
            </h3>
            <div>
              <input
                ref={earnFileRef}
                type="file"
                accept=".xlsx,.xls,.csv"
                className="hidden"
                onChange={handleImportEarnBurn}
              />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => earnFileRef.current?.click()}
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
              <span className="text-right">Puntos</span>
              <span />
            </div>

            {rewardDrafts.length > 0 && (
              <div className="max-h-[180px] overflow-y-auto">
                {rewardDrafts.map((r) => (
                  <div
                    key={r.local_id}
                    className="grid grid-cols-[1fr_1fr_80px_36px] items-center gap-2 border-b px-3 py-2 text-sm last:border-b-0"
                  >
                    <span className="truncate font-medium">{r.name}</span>
                    <span className="truncate text-muted-foreground">{r.description || "—"}</span>
                    <span className="text-right tabular-nums">{r.points_cost}</span>
                    <div className="flex justify-center">
                      <Check className="h-3.5 w-3.5 text-green-600" />
                    </div>
                  </div>
                ))}
              </div>
            )}

            {rewardDrafts.length === 0 && (
              <div className="flex items-center justify-center gap-2 px-3 py-6 text-sm text-muted-foreground">
                <Gift className="h-4 w-4" />
                <span>Agrega tu primera recompensa</span>
              </div>
            )}

            <div className="grid grid-cols-[1fr_1fr_80px_36px] items-center gap-2 border-t bg-muted/30 px-3 py-2">
              <Input
                placeholder="Cafe gratis"
                value={rewardName}
                onChange={(e) => setRewardName(e.target.value)}
                className="h-8 text-sm"
                onKeyDown={(e) => e.key === "Enter" && handleAddReward()}
              />
              <Input
                placeholder="Descripcion"
                value={rewardDesc}
                onChange={(e) => setRewardDesc(e.target.value)}
                className="h-8 text-sm"
                onKeyDown={(e) => e.key === "Enter" && handleAddReward()}
              />
              <Input
                type="number"
                min={1}
                placeholder="100"
                value={rewardCost}
                onChange={(e) => setRewardCost(e.target.value)}
                className="h-8 text-sm text-right"
                onKeyDown={(e) => e.key === "Enter" && handleAddReward()}
              />
              <Button
                size="icon"
                variant="ghost"
                className="h-8 w-8"
                onClick={handleAddReward}
              >
                <Plus className="h-4 w-4" />
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Cashback Rewards */}
      {cashbackDraft && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">
              Beneficios de Cashback — {cashbackDraft.name}
              {cashbackRewardDrafts.length > 0 && (
                <span className="ml-2 text-xs text-muted-foreground">
                  ({cashbackRewardDrafts.length})
                </span>
              )}
            </h3>
            <div>
              <input
                ref={cbFileRef}
                type="file"
                accept=".xlsx,.xls,.csv"
                className="hidden"
                onChange={handleImportCashback}
              />
              <Button
                variant="ghost"
                size="sm"
                onClick={() => cbFileRef.current?.click()}
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
              <span className="text-right">Costo</span>
              <span />
            </div>

            {cashbackRewardDrafts.length > 0 && (
              <div className="max-h-[180px] overflow-y-auto">
                {cashbackRewardDrafts.map((r) => (
                  <div
                    key={r.local_id}
                    className="grid grid-cols-[1fr_1fr_80px_36px] items-center gap-2 border-b px-3 py-2 text-sm last:border-b-0"
                  >
                    <span className="truncate font-medium">{r.name}</span>
                    <span className="truncate text-muted-foreground">{r.description || "—"}</span>
                    <span className="text-right tabular-nums">${r.cost}</span>
                    <div className="flex justify-center">
                      <Check className="h-3.5 w-3.5 text-green-600" />
                    </div>
                  </div>
                ))}
              </div>
            )}

            {cashbackRewardDrafts.length === 0 && (
              <div className="flex items-center justify-center gap-2 px-3 py-6 text-sm text-muted-foreground">
                <Gift className="h-4 w-4" />
                <span>Agrega tu primer beneficio</span>
              </div>
            )}

            <div className="grid grid-cols-[1fr_1fr_80px_36px] items-center gap-2 border-t bg-muted/30 px-3 py-2">
              <Input
                placeholder="Descuento especial"
                value={cbRewardName}
                onChange={(e) => setCbRewardName(e.target.value)}
                className="h-8 text-sm"
                onKeyDown={(e) => e.key === "Enter" && handleAddCbReward()}
              />
              <Input
                placeholder="Descripcion"
                value={cbRewardDesc}
                onChange={(e) => setCbRewardDesc(e.target.value)}
                className="h-8 text-sm"
                onKeyDown={(e) => e.key === "Enter" && handleAddCbReward()}
              />
              <Input
                type="number"
                min={1}
                placeholder="50"
                value={cbRewardCost}
                onChange={(e) => setCbRewardCost(e.target.value)}
                className="h-8 text-sm text-right"
                onKeyDown={(e) => e.key === "Enter" && handleAddCbReward()}
              />
              <Button
                size="icon"
                variant="ghost"
                className="h-8 w-8"
                onClick={handleAddCbReward}
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
