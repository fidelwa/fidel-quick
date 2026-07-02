import { useState, useEffect, useRef } from "react"
import { useParams, Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useAuth } from "@/context/auth-context"
import { usePrograms, useUpdateProgram } from "@/hooks/use-programs"
import { useRewards, useCreateReward, useUpdateReward } from "@/hooks/use-rewards"
import type { Reward } from "@/types"
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
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import { ArrowLeft, Plus, Upload, Loader2 } from "lucide-react"
import * as XLSX from "xlsx"
import { formatPoints } from "@/lib/utils"

// Los límites de config (FID-34/36) son opcionales: cadena vacía = sin límite (null).
const optionalNumber = z
  .string()
  .refine((v) => v === "" || (!isNaN(Number(v)) && Number(v) >= 0), "Debe ser un número >= 0")

const programSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  points_ratio: z.number().int().min(1),
  expiry_days: optionalNumber,
  min_ticket_amount: optionalNumber,
})

const rewardSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  description: z.string(),
  points_cost: z.number().int().min(1, "Debe ser al menos 1"),
  // FID-38: stock opcional. Cadena vacía = ilimitado. Entero >= 0 si se define.
  stock_total: z
    .string()
    .refine((v) => v === "" || (/^\d+$/.test(v) && Number(v) >= 0), "Debe ser un entero mayor o igual a 0"),
})

type ProgramFormValues = z.infer<typeof programSchema>
type RewardFormValues = z.infer<typeof rewardSchema>

export function ProgramDetailPage() {
  const { id } = useParams<{ id: string }>()
  const { customerId } = useAuth()
  const { data: programs, isLoading: loadingPrograms } = usePrograms(customerId)
  const program = programs?.find((p) => p.id === id)
  const updateProgram = useUpdateProgram(id!)
  const { data: rewards, isLoading: loadingRewards } = useRewards(id!)
  const createReward = useCreateReward(id!)
  const updateReward = useUpdateReward(id!)
  const [rewardDialogOpen, setRewardDialogOpen] = useState(false)
  const [importingExcel, setImportingExcel] = useState(false)
  const excelFileRef = useRef<HTMLInputElement>(null)

  const programForm = useForm<ProgramFormValues>({
    resolver: zodResolver(programSchema),
    defaultValues: { name: "", points_ratio: 1, expiry_days: "", min_ticket_amount: "" },
  })

  const rewardForm = useForm<RewardFormValues>({
    resolver: zodResolver(rewardSchema),
    defaultValues: { name: "", description: "", points_cost: 1, stock_total: "" },
  })

  useEffect(() => {
    if (program) {
      programForm.reset({
        name: program.name,
        points_ratio: program.points_ratio,
        expiry_days: program.expiry_days != null ? String(program.expiry_days) : "",
        min_ticket_amount: program.min_ticket_amount != null ? String(program.min_ticket_amount) : "",
      })
    }
  }, [program, programForm])

  const onUpdateProgram = (values: ProgramFormValues) => {
    // Cadena vacía => null (sin límite).
    const payload = {
      name: values.name,
      points_ratio: values.points_ratio,
      expiry_days: values.expiry_days === "" ? null : Number(values.expiry_days),
      min_ticket_amount: values.min_ticket_amount === "" ? null : Number(values.min_ticket_amount),
    }
    updateProgram.mutate(payload, {
      onSuccess: () => toast.success("Programa actualizado"),
      onError: (err) => toast.error(err.message),
    })
  }

  const onToggleActive = (active: boolean) => {
    // LG-1: el PUT es full-replace en config (campos ausentes/null se limpian).
    // Al alternar "activo" debemos re-enviar TODA la config actual del programa,
    // o de lo contrario expiry_days/min_ticket_amount quedarían en NULL.
    if (!program) return
    updateProgram.mutate(
      {
        active,
        name: program.name,
        points_ratio: program.points_ratio,
        expiry_days: program.expiry_days ?? null,
        min_ticket_amount: program.min_ticket_amount ?? null,
      },
      {
        onSuccess: () => toast.success(active ? "Programa activado" : "Programa desactivado"),
        onError: (err) => toast.error(err.message),
      },
    )
  }

  const onCreateReward = (values: RewardFormValues) => {
    // FID-38: cadena vacía => null (stock ilimitado).
    const payload = {
      name: values.name,
      description: values.description,
      points_cost: values.points_cost,
      stock_total: values.stock_total === "" ? null : Number(values.stock_total),
    }
    createReward.mutate(payload, {
      onSuccess: () => {
        toast.success("Recompensa creada")
        rewardForm.reset()
        setRewardDialogOpen(false)
      },
      onError: (err) => toast.error(err.message),
    })
  }

  const onToggleReward = (reward: Reward, active: boolean) => {
    // FID-38 (full-replace en stock, como el programa en LG-1): el PUT limpia
    // stock_total si no se envía, así que al alternar "activo" re-enviamos el
    // stock actual del premio para no borrarlo.
    updateReward.mutate(
      { rewardId: reward.id, active, stock_total: reward.stock_total ?? null },
      {
        onSuccess: () => toast.success(active ? "Recompensa activada" : "Recompensa desactivada"),
        onError: (err) => toast.error(err.message),
      },
    )
  }

  const onImportExcel = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return
    e.target.value = ""

    setImportingExcel(true)
    try {
      const data = new Uint8Array(await file.arrayBuffer())
      const workbook = XLSX.read(data, { type: "array" })
      const sheet = workbook.Sheets[workbook.SheetNames[0]]
      const rows = XLSX.utils.sheet_to_json<Record<string, unknown>>(sheet)

      let created = 0
      for (const row of rows) {
        const name = String(row.nombre ?? row.Nombre ?? "").trim()
        const description = String(row.descripcion ?? row.Descripcion ?? "").trim()
        const pointsCost = Number(row.puntos ?? row.Puntos ?? row.costo ?? row.Costo ?? 0)

        if (!name || pointsCost <= 0) continue

        await createReward.mutateAsync({ name, description, points_cost: pointsCost })
        created++
      }

      if (created > 0) {
        toast.success(`${created} recompensa${created > 1 ? "s" : ""} importada${created > 1 ? "s" : ""}`)
      } else {
        toast.error("No se encontraron filas validas. Columnas: Nombre, Descripcion, Puntos")
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al importar")
    } finally {
      setImportingExcel(false)
    }
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
          <Link to="/programas">
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
                  name="points_ratio"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>1 punto por cada $</FormLabel>
                      <FormControl>
                        <Input type="number" min={1} {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={programForm.control}
                  name="expiry_days"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Vencimiento de puntos (días)</FormLabel>
                      <FormControl>
                        <Input type="number" min={1} placeholder="Sin vencimiento" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={programForm.control}
                  name="min_ticket_amount"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Ticket mínimo ($)</FormLabel>
                      <FormControl>
                        <Input type="number" min={0} step="0.01" placeholder="Sin mínimo" {...field} />
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
          <div className="flex gap-2">
            <input
              ref={excelFileRef}
              type="file"
              accept=".xlsx,.xls,.csv"
              className="hidden"
              onChange={onImportExcel}
            />
            <Button
              variant="outline"
              onClick={() => excelFileRef.current?.click()}
              disabled={importingExcel}
            >
              {importingExcel ? (
                <Loader2 className="mr-2 h-4 w-4 animate-spin" />
              ) : (
                <Upload className="mr-2 h-4 w-4" />
              )}
              Importar Excel
            </Button>
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
                          <Input placeholder="Cafe gratis" {...field} />
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
                    name="points_cost"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Costo en puntos</FormLabel>
                        <FormControl>
                          <Input type="number" min={1} {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={rewardForm.control}
                    name="stock_total"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Stock (opcional)</FormLabel>
                        <FormControl>
                          <Input type="number" min={0} placeholder="Vacío = ilimitado" {...field} />
                        </FormControl>
                        <FormDescription>Deja vacío para stock ilimitado.</FormDescription>
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
                <TableHead>Stock</TableHead>
                <TableHead>Estado</TableHead>
                <TableHead className="w-20">Activo</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {rewards.map((r) => (
                <TableRow key={r.id}>
                  <TableCell className="font-medium">{r.name}</TableCell>
                  <TableCell className="text-muted-foreground">{r.description || "—"}</TableCell>
                  <TableCell>{formatPoints(r.points_cost)} pts</TableCell>
                  <TableCell>
                    {r.stock_total == null
                      ? "Ilimitado"
                      : `${Math.max(0, r.stock_total - r.redeemed_count)} / ${r.stock_total}`}
                  </TableCell>
                  <TableCell>
                    <Badge variant={r.active ? "default" : "secondary"}>
                      {r.active ? "Activo" : "Inactivo"}
                    </Badge>
                  </TableCell>
                  <TableCell>
                    <Switch
                      checked={r.active}
                      onCheckedChange={(checked) => onToggleReward(r, checked)}
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
