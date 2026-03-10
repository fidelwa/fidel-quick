import { useState } from "react"
import { useCreateReward } from "@/hooks/use-rewards"
import { useCreateCashbackReward } from "@/hooks/use-cashback-rewards"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"
import { Gift, Loader2, Plus } from "lucide-react"
import type { Program, CashbackProgram, Reward, CashbackReward } from "@/types"

interface StepRewardsProps {
  earnBurnProgram: Program | null
  cashbackProgram: CashbackProgram | null
  rewards: Reward[]
  cashbackRewards: CashbackReward[]
  onRewardsChange: (rewards: Reward[]) => void
  onCashbackRewardsChange: (rewards: CashbackReward[]) => void
  onNext: () => void
  onPrev: () => void
}

export function StepRewards({
  earnBurnProgram,
  cashbackProgram,
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
  const [rewardCost, setRewardCost] = useState(100)
  const [addingReward, setAddingReward] = useState(false)

  const [cbRewardName, setCbRewardName] = useState("")
  const [cbRewardDesc, setCbRewardDesc] = useState("")
  const [cbRewardCost, setCbRewardCost] = useState(50)
  const [addingCbReward, setAddingCbReward] = useState(false)

  const totalRewards = rewards.length + cashbackRewards.length

  const handleAddReward = async () => {
    if (!rewardName.trim()) {
      toast.error("Ingresa el nombre de la recompensa")
      return
    }
    setAddingReward(true)
    try {
      const reward = await createReward.mutateAsync({
        name: rewardName.trim(),
        description: rewardDesc.trim(),
        points_cost: rewardCost,
      })
      onRewardsChange([...rewards, reward])
      setRewardName("")
      setRewardDesc("")
      setRewardCost(100)
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
    setAddingCbReward(true)
    try {
      const reward = await createCashbackReward.mutateAsync({
        name: cbRewardName.trim(),
        description: cbRewardDesc.trim(),
        cost: cbRewardCost,
      })
      onCashbackRewardsChange([...cashbackRewards, reward])
      setCbRewardName("")
      setCbRewardDesc("")
      setCbRewardCost(50)
      toast.success("Beneficio creado")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al crear beneficio")
    } finally {
      setAddingCbReward(false)
    }
  }

  const handleNext = () => {
    if (totalRewards === 0) {
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
        <div className="space-y-4">
          <h3 className="text-sm font-medium">Recompensas de Puntos — {earnBurnProgram.name}</h3>

          {rewards.length > 0 ? (
            <div className="space-y-2">
              {rewards.map((r) => (
                <div key={r.id} className="flex items-center justify-between rounded-lg border p-3">
                  <div>
                    <p className="text-sm font-medium">{r.name}</p>
                    {r.description && (
                      <p className="text-xs text-muted-foreground">{r.description}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">{r.points_cost} pts</span>
                    <Badge variant="secondary">Creado</Badge>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center rounded-lg border border-dashed p-6 text-center">
              <Gift className="mb-2 h-8 w-8 text-muted-foreground/50" />
              <p className="text-sm text-muted-foreground">
                Agrega tu primera recompensa de puntos
              </p>
            </div>
          )}

          <div className="grid gap-3 sm:grid-cols-3">
            <div className="space-y-1.5">
              <Label>Nombre</Label>
              <Input
                placeholder="Cafe gratis"
                value={rewardName}
                onChange={(e) => setRewardName(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Descripcion</Label>
              <Input
                placeholder="Cafe americano 12oz"
                value={rewardDesc}
                onChange={(e) => setRewardDesc(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Costo (puntos)</Label>
              <div className="flex gap-2">
                <Input
                  type="number"
                  min={1}
                  value={rewardCost}
                  onChange={(e) => setRewardCost(Number(e.target.value))}
                />
                <Button size="sm" onClick={handleAddReward} disabled={addingReward}>
                  {addingReward ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                </Button>
              </div>
            </div>
          </div>
        </div>
      )}

      {/* Cashback Rewards */}
      {cashbackProgram && (
        <div className="space-y-4">
          <h3 className="text-sm font-medium">Beneficios de Cashback — {cashbackProgram.name}</h3>

          {cashbackRewards.length > 0 ? (
            <div className="space-y-2">
              {cashbackRewards.map((r) => (
                <div key={r.id} className="flex items-center justify-between rounded-lg border p-3">
                  <div>
                    <p className="text-sm font-medium">{r.name}</p>
                    {r.description && (
                      <p className="text-xs text-muted-foreground">{r.description}</p>
                    )}
                  </div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm text-muted-foreground">${r.cost}</span>
                    <Badge variant="secondary">Creado</Badge>
                  </div>
                </div>
              ))}
            </div>
          ) : (
            <div className="flex flex-col items-center justify-center rounded-lg border border-dashed p-6 text-center">
              <Gift className="mb-2 h-8 w-8 text-muted-foreground/50" />
              <p className="text-sm text-muted-foreground">
                Agrega tu primer beneficio de cashback
              </p>
            </div>
          )}

          <div className="grid gap-3 sm:grid-cols-3">
            <div className="space-y-1.5">
              <Label>Nombre</Label>
              <Input
                placeholder="Descuento especial"
                value={cbRewardName}
                onChange={(e) => setCbRewardName(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Descripcion</Label>
              <Input
                placeholder="Descuento en tu proxima compra"
                value={cbRewardDesc}
                onChange={(e) => setCbRewardDesc(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Costo ($)</Label>
              <div className="flex gap-2">
                <Input
                  type="number"
                  min={1}
                  value={cbRewardCost}
                  onChange={(e) => setCbRewardCost(Number(e.target.value))}
                />
                <Button size="sm" onClick={handleAddCbReward} disabled={addingCbReward}>
                  {addingCbReward ? (
                    <Loader2 className="h-4 w-4 animate-spin" />
                  ) : (
                    <Plus className="h-4 w-4" />
                  )}
                </Button>
              </div>
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
