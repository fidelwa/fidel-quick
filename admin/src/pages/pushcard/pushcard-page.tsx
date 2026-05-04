import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useAuth } from "@/context/auth-context"
import {
  usePushcardConfig,
  useUpsertPushcardConfig,
  usePushcardCards,
} from "@/hooks/use-pushcard"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"
import type { PushcardCard } from "@/types"

const configSchema = z.object({
  card_slots: z.number().int().min(1, "Debe ser mayor a 0").max(50),
  reward_on_complete: z.string().optional(),
})

type ConfigFormValues = z.infer<typeof configSchema>

function statusVariant(status: PushcardCard["status"]) {
  switch (status) {
    case "open":
      return "default"
    case "completed":
      return "secondary"
    case "redeemed":
      return "outline"
    default:
      return "outline"
  }
}

function statusLabel(status: PushcardCard["status"]) {
  switch (status) {
    case "open":
      return "Abierta"
    case "completed":
      return "Completada"
    case "redeemed":
      return "Canjeada"
    case "cancelled":
      return "Cancelada"
  }
}

function visual(count: number, slots: number) {
  const filled = Math.min(count, slots)
  return "●".repeat(filled) + "○".repeat(Math.max(0, slots - filled))
}

export function PushcardPage() {
  const { customerId } = useAuth()
  const { data: config, isLoading, error } = usePushcardConfig(customerId)
  const upsert = useUpsertPushcardConfig(config?.customer_sisfi_id ?? "")
  const { data: cards, isLoading: cardsLoading } = usePushcardCards(
    config?.customer_sisfi_id ?? ""
  )

  const form = useForm<ConfigFormValues>({
    resolver: zodResolver(configSchema),
    defaultValues: { card_slots: 10, reward_on_complete: "" },
  })

  useEffect(() => {
    if (config) {
      form.reset({
        card_slots: config.card_slots,
        reward_on_complete: config.reward_on_complete ?? "",
      })
    }
  }, [config, form])

  const onSubmit = (values: ConfigFormValues) => {
    upsert.mutate(values, {
      onSuccess: () => toast.success("Configuración guardada"),
      onError: (err) => toast.error(err.message),
    })
  }

  if (isLoading) {
    return (
      <div className="space-y-3">
        <Skeleton className="h-8 w-1/3" />
        <Skeleton className="h-32 w-full" />
      </div>
    )
  }

  // No config yet — show banner pointing the user to activate the sisfi.
  if (error || !config) {
    return (
      <div className="space-y-6">
        <h1 className="text-2xl font-bold">Tarjeta de sellos</h1>
        <div className="rounded-lg border border-dashed p-8 text-center">
          <p className="text-muted-foreground">
            Aún no activaste el sistema de tarjeta de sellos para este negocio.
          </p>
          <p className="text-muted-foreground mt-2 text-sm">
            Activalo desde la sección de Programas seleccionando "pushcard" en el catálogo.
          </p>
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Tarjeta de sellos</h1>
        <Badge variant={config.active ? "default" : "secondary"}>
          {config.active ? "Activo" : "Inactivo"}
        </Badge>
      </div>

      <div className="rounded-lg border p-6">
        <h2 className="mb-4 text-lg font-semibold">Configuración</h2>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4 max-w-md">
            <FormField
              control={form.control}
              name="card_slots"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Sellos por tarjeta</FormLabel>
                  <FormControl>
                    <Input
                      type="number"
                      min="1"
                      max="50"
                      step="1"
                      {...field}
                      onChange={(e) => field.onChange(Number(e.target.value))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="reward_on_complete"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Reward al completar (UUID)</FormLabel>
                  <FormControl>
                    <Input placeholder="opcional" {...field} />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button type="submit" disabled={upsert.isPending}>
              {upsert.isPending ? "Guardando..." : "Guardar"}
            </Button>
          </form>
        </Form>
      </div>

      <div className="rounded-lg border p-6">
        <h2 className="mb-4 text-lg font-semibold">Tarjetas recientes</h2>
        {cardsLoading ? (
          <Skeleton className="h-24 w-full" />
        ) : !cards?.length ? (
          <p className="text-muted-foreground text-sm">Aún no hay tarjetas activas.</p>
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>Cliente</TableHead>
                <TableHead>Progreso</TableHead>
                <TableHead>Estado</TableHead>
                <TableHead>Actualizada</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {cards.map((c) => (
                <TableRow key={c.id}>
                  <TableCell className="font-mono text-xs">{c.client_id.slice(0, 8)}…</TableCell>
                  <TableCell className="font-mono">
                    {visual(c.stamps_count, config.card_slots)}{" "}
                    <span className="text-muted-foreground">
                      {c.stamps_count}/{config.card_slots}
                    </span>
                  </TableCell>
                  <TableCell>
                    <Badge variant={statusVariant(c.status)}>{statusLabel(c.status)}</Badge>
                  </TableCell>
                  <TableCell className="text-muted-foreground text-sm">
                    {new Date(c.updated_at).toLocaleDateString()}
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
