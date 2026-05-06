import { useState, useRef } from "react"
import { useCreateReward } from "@/hooks/use-rewards"
import { useCreateCashbackReward } from "@/hooks/use-cashback-rewards"
import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { Gift, Loader2, Plus, Upload, Check, Trash2 } from "lucide-react"
import * as XLSX from "xlsx"
import type {
  Program,
  CashbackProgram,
  PushcardConfig,
  Reward,
  CashbackReward,
} from "@/types"

interface StepRewardsProps {
  earnBurnProgram: Program | null
  cashbackProgram: CashbackProgram | null
  pushcardConfig: PushcardConfig | null
  rewards: Reward[]
  cashbackRewards: CashbackReward[]
  onRewardsChange: (rewards: Reward[]) => void
  onCashbackRewardsChange: (rewards: CashbackReward[]) => void
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
  earnBurnProgram,
  cashbackProgram,
  pushcardConfig,
  rewards,
  cashbackRewards,
  onRewardsChange,
  onCashbackRewardsChange,
  onNext,
  onPrev,
}: StepRewardsProps) {
  const createReward = useCreateReward(earnBurnProgram?.id ?? "")
  const createCashbackReward = useCreateCashbackReward(cashbackProgram?.id ?? "")

  const [rewardName, setRewardName] = useState("")
  const [rewardDesc, setRewardDesc] = useState("")
  const [rewardCost, setRewardCost] = useState("")
  const [addingReward, setAddingReward] = useState(false)

  const [cbRewardName, setCbRewardName] = useState("")
  const [cbRewardDesc, setCbRewardDesc] = useState("")
  const [cbRewardCost, setCbRewardCost] = useState("")
  const [addingCbReward, setAddingCbReward] = useState(false)

  const [importing, setImporting] = useState(false)

  const earnFileRef = useRef<HTMLInputElement>(null)
  const cbFileRef = useRef<HTMLInputElement>(null)

  const totalRewards = rewards.length + cashbackRewards.length

  const handleAddReward = async () => {
    if (!rewardName.trim()) {
      toast.error("Ingresa el nombre de la recompensa")
      return
    }
    const cost = Number(rewardCost) || 100
    setAddingReward(true)
    try {
      const reward = await createReward.mutateAsync({
        name: rewardName.trim(),
        description: rewardDesc.trim(),
        points_cost: cost,
      })
      onRewardsChange([...rewards, reward])
      setRewardName("")
      setRewardDesc("")
      setRewardCost("")
      toast.success("Recompensa creada")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al crear recompensa")
    } finally {
      setAddingReward(false)
    }
  }

  const handleAddCbReward = async () => {
    if (!cbRewardName.trim()) {
      toast.error("Ingresa el nombre del beneficio")
      return
    }
    const cost = Number(cbRewardCost) || 50
    setAddingCbReward(true)
    try {
      const reward = await createCashbackReward.mutateAsync({
        name: cbRewardName.trim(),
        description: cbRewardDesc.trim(),
        cost,
      })
      onCashbackRewardsChange([...cashbackRewards, reward])
      setCbRewardName("")
      setCbRewardDesc("")
      setCbRewardCost("")
      toast.success("Beneficio creado")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al crear beneficio")
    } finally {
      setAddingCbReward(false)
    }
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

      const created: Reward[] = []
      for (const row of rows) {
        const name = (row.nombre ?? row.Nombre ?? "").toString().trim()
        const description = (row.descripcion ?? row.Descripcion ?? "").toString().trim()
        const pointsCost = Number(row.puntos ?? row.Puntos ?? row.costo ?? row.Costo ?? 0)

        if (!name || pointsCost <= 0) continue

        const reward = await createReward.mutateAsync({
          name,
          description,
          points_cost: pointsCost,
        })
        created.push(reward)
      }

      if (created.length > 0) {
        onRewardsChange([...rewards, ...created])
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

      const created: CashbackReward[] = []
      for (const row of rows) {
        const name = (row.nombre ?? row.Nombre ?? "").toString().trim()
        const description = (row.descripcion ?? row.Descripcion ?? "").toString().trim()
        const cost = Number(row.costo ?? row.Costo ?? row.puntos ?? row.Puntos ?? 0)

        if (!name || cost <= 0) continue

        const reward = await createCashbackReward.mutateAsync({
          name,
          description,
          cost,
        })
        created.push(reward)
      }

      if (created.length > 0) {
        onCashbackRewardsChange([...cashbackRewards, ...created])
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

  const handleRemoveReward = (id: string) => {
    onRewardsChange(rewards.filter((r) => r.id !== id))
  }

  const handleRemoveCbReward = (id: string) => {
    onCashbackRewardsChange(cashbackRewards.filter((r) => r.id !== id))
  }

  // Earn-burn / cashback necesitan al menos una recompensa cargada para avanzar.
  // Pushcard configura su `reward_on_complete` desde la pagina dedicada (/pushcard),
  // no desde el wizard, asi que no bloquea el avance.
  const requiresExplicitRewards = !!earnBurnProgram || !!cashbackProgram

  const handleNext = () => {
    if (requiresExplicitRewards && totalRewards === 0) {
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

      {/* Earn-Burn Rewards */}
      {earnBurnProgram && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">
              Recompensas de Puntos — {earnBurnProgram.name}
              {rewards.length > 0 && (
                <span className="ml-2 text-xs text-muted-foreground">
                  ({rewards.length})
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
            {/* Table header */}
            <div className="grid grid-cols-[1fr_1fr_80px_36px] gap-2 border-b bg-muted/50 px-3 py-2 text-xs font-medium text-muted-foreground">
              <span>Nombre</span>
              <span>Descripcion</span>
              <span className="text-right">Puntos</span>
              <span />
            </div>

            {/* Scrollable reward rows */}
            {rewards.length > 0 && (
              <div className="max-h-[180px] overflow-y-auto">
                {rewards.map((r) => (
                  <div
                    key={r.id}
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

            {/* Empty state (inside table) */}
            {rewards.length === 0 && (
              <div className="flex items-center justify-center gap-2 px-3 py-6 text-sm text-muted-foreground">
                <Gift className="h-4 w-4" />
                <span>Agrega tu primera recompensa</span>
              </div>
            )}

            {/* Inline input row */}
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
                disabled={addingReward}
              >
                {addingReward ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Plus className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Cashback Rewards */}
      {cashbackProgram && (
        <div className="space-y-3">
          <div className="flex items-center justify-between">
            <h3 className="text-sm font-medium">
              Beneficios de Cashback — {cashbackProgram.name}
              {cashbackRewards.length > 0 && (
                <span className="ml-2 text-xs text-muted-foreground">
                  ({cashbackRewards.length})
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
            {/* Table header */}
            <div className="grid grid-cols-[1fr_1fr_80px_36px] gap-2 border-b bg-muted/50 px-3 py-2 text-xs font-medium text-muted-foreground">
              <span>Nombre</span>
              <span>Descripcion</span>
              <span className="text-right">Costo</span>
              <span />
            </div>

            {/* Scrollable reward rows */}
            {cashbackRewards.length > 0 && (
              <div className="max-h-[180px] overflow-y-auto">
                {cashbackRewards.map((r) => (
                  <div
                    key={r.id}
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

            {/* Empty state */}
            {cashbackRewards.length === 0 && (
              <div className="flex items-center justify-center gap-2 px-3 py-6 text-sm text-muted-foreground">
                <Gift className="h-4 w-4" />
                <span>Agrega tu primer beneficio</span>
              </div>
            )}

            {/* Inline input row */}
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
                disabled={addingCbReward}
              >
                {addingCbReward ? (
                  <Loader2 className="h-4 w-4 animate-spin" />
                ) : (
                  <Plus className="h-4 w-4" />
                )}
              </Button>
            </div>
          </div>
        </div>
      )}

      {/* Pushcard nota informativa */}
      {pushcardConfig && !earnBurnProgram && !cashbackProgram && (
        <div className="rounded-lg border bg-muted/30 p-4 space-y-1">
          <div className="flex items-center gap-2 text-sm font-medium">
            <Gift className="h-4 w-4" />
            Tarjeta de sellos: {pushcardConfig.name}
          </div>
          <p className="text-sm text-muted-foreground">
            La recompensa al completar los {pushcardConfig.card_slots} sellos se configura
            desde la pagina <span className="font-medium">Tarjeta de sellos</span>. Continua
            para terminar el onboarding.
          </p>
        </div>
      )}

      {pushcardConfig && (earnBurnProgram || cashbackProgram) && (
        <p className="rounded-md border bg-muted/20 px-3 py-2 text-xs text-muted-foreground">
          Recordatorio: la recompensa de la tarjeta de sellos ({pushcardConfig.name}) se
          configura aparte desde la pagina Tarjeta de sellos.
        </p>
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
