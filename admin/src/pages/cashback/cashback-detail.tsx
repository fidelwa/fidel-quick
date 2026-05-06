import { useState, useEffect } from "react"
import { useParams, Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useAuth } from "@/context/auth-context"
import { useCashbackPrograms, useUpdateCashbackProgram } from "@/hooks/use-cashback-programs"
import { useCashbackRewards, useCreateCashbackReward, useUpdateCashbackReward } from "@/hooks/use-cashback-rewards"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Switch } from "@/components/ui/switch"
import { Skeleton } from "@/components/ui/skeleton"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { ArrowLeft, Plus } from "lucide-react"
import { formatCurrency } from "@/lib/utils"

const programSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  cashback_rate: z.number().min(0.01),
})

const rewardSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  description: z.string(),
  cost: z.number().min(0.01, "Debe ser mayor a 0"),
})

type ProgramFormValues = z.infer<typeof programSchema>
type RewardFormValues = z.infer<typeof rewardSchema>

export function CashbackDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { customerId } = useAuth()
  const { data: programs, isLoading: loadingPrograms } = useCashbackPrograms(customerId)
  const program = programs?.find((p) => p.id === id)
  const updateProgram = useUpdateCashbackProgram(id!)
  const { data: rewards, isLoading: loadingRewards } = useCashbackRewards(id!)
  const createReward = useCreateCashbackReward(id!)
  const updateReward = useUpdateCashbackReward(id!)
  const [rewardDialogOpen, setRewardDialogOpen] = useState(false)

  const programForm = useForm<ProgramFormValues>({
    resolver: zodResolver(programSchema),
    defaultValues: { name: "", cashback_rate: 0.05 },
  })

  const rewardForm = useForm<RewardFormValues>({
    resolver: zodResolver(rewardSchema),
    defaultValues: { name: "", description: "", cost: 1 },
  })

  useEffect(() => {
    if (program) {
      programForm.reset({ name: program.name, cashback_rate: program.cashback_rate })
    }
  }, [program, programForm])

  const onUpdateProgram = (values: ProgramFormValues) => {
    updateProgram.mutate(values, {
      onSuccess: () => toast.success("Programa actualizado"),
      onError: (err) => toast.error(err.message),
    })
  }

  const onToggleActive = (active: boolean) => {
    updateProgram.mutate({ active }, {
      onSuccess: () => toast.success(active ? "Programa activado" : "Programa desactivado"),
      onError: (err) => toast.error(err.message),
    })
  }

  const onCreateReward = (values: RewardFormValues) => {
    createReward.mutate(values, {
      onSuccess: () => {
        toast.success("Recompensa creada")
        rewardForm.reset()
        setRewardDialogOpen(false)
      },
      onError: (err) => toast.error(err.message),
    })
  }

  const onToggleReward = (rewardId: string, active: boolean) => {
    updateReward.mutate({ rewardId, active }, {
      onSuccess: () => toast.success(active ? "Recompensa activada" : "Recompensa desactivada"),
      onError: (err) => toast.error(err.message),
    })
  }

  if (loadingPrograms) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
      </div>
    )
  }

  if (!program) {
    return <p className="text-muted-foreground">Programa no encontrado.</p>
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center gap-3">
        <Button variant="ghost" size="icon" asChild>
          <Link to="/cashback">
            <ArrowLeft className="h-4 w-4" />
          </Link>
        </Button>
        <h1 className="text-2xl font-bold">{program.name}</h1>
        <Badge variant={program.active ? "default" : "secondary"}>
          {program.active ? "Activo" : "Inactivo"}
        </Badge>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Configuracion del programa</CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...programForm}>
            <form onSubmit={programForm.handleSubmit(onUpdateProgram)} className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <FormField
                  control={programForm.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Nombre</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={programForm.control}
                  name="cashback_rate"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Tasa de cashback</FormLabel>
                      <FormControl>
                        <Input type="number" step="0.01" min="0.01" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <div className="flex items-center gap-4">
                <Button type="submit" disabled={updateProgram.isPending}>
                  {updateProgram.isPending ? "Guardando..." : "Guardar"}
                </Button>
                <div className="flex items-center gap-2">
                  <Switch
                    checked={program.active}
                    onCheckedChange={onToggleActive}
                  />
                  <span className="text-sm text-muted-foreground">
                    {program.active ? "Activo" : "Inactivo"}
                  </span>
                </div>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>

      <div className="space-y-4">
        <div className="flex items-center justify-between">
          <h2 className="text-xl font-semibold">Recompensas</h2>
          <Dialog open={rewardDialogOpen} onOpenChange={setRewardDialogOpen}>
            <DialogTrigger asChild>
              <Button variant="outline">
                <Plus className="mr-2 h-4 w-4" />
                Nueva recompensa
              </Button>
            </DialogTrigger>
            <DialogContent>
              <DialogHeader>
                <DialogTitle>Crear recompensa</DialogTitle>
              </DialogHeader>
              <Form {...rewardForm}>
                <form onSubmit={rewardForm.handleSubmit(onCreateReward)} className="space-y-4">
                  <FormField
                    control={rewardForm.control}
                    name="name"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Nombre</FormLabel>
                        <FormControl>
                          <Input placeholder="Descuento 10%" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={rewardForm.control}
                    name="description"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Descripcion</FormLabel>
                        <FormControl>
                          <Input placeholder="Descripcion opcional" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={rewardForm.control}
                    name="cost"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Costo (en cashback)</FormLabel>
                        <FormControl>
                          <Input type="number" step="0.01" min="0.01" {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <Button type="submit" disabled={createReward.isPending}>
                    {createReward.isPending ? "Creando..." : "Crear"}
                  </Button>
                </form>
              </Form>
            </DialogContent>
          </Dialog>
        </div>

        {loadingRewards ? (
          <div className="space-y-2">
            {Array.from({ length: 3 }).map((_, i) => (
              <Skeleton key={i} className="h-12 w-full" />
            ))}
          </div>
        ) : !rewards?.length ? (
          <div className="rounded-lg border border-dashed p-8 text-center">
            <p className="text-muted-foreground">No hay recompensas. Crea la primera.</p>
          </div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Nombre</TableHead>
                <TableHead>Descripcion</TableHead>
                <TableHead>Costo</TableHead>
                <TableHead>Estado</TableHead>
                <TableHead className="w-20">Activo</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rewards.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-medium">{r.name}</TableCell>
                  <TableCell className="text-muted-foreground">{r.description || "—"}</TableCell>
                  <TableCell>{formatCurrency(r.cost)}</TableCell>
                  <TableCell>
                    <Badge variant={r.active ? "default" : "secondary"}>
                      {r.active ? "Activo" : "Inactivo"}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Switch
                      checked={r.active}
                      onCheckedChange={(checked) => onToggleReward(r.id, checked)}
                    />
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
      </div>
    </div>
  )
}
