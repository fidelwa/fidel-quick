import { useState } from "react"
import { Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useAuth } from "@/context/auth-context"
import { usePrograms, useCreateProgram } from "@/hooks/use-programs"
import { useCashbackPrograms } from "@/hooks/use-cashback-programs"
import { usePushcardConfig } from "@/hooks/use-pushcard"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  GlassCard,
  GlassCardContent,
} from "@/components/ui/glass-card"
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
import { Plus, ChevronRight, Star, Wallet, Stamp } from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { formatPoints } from "@/lib/utils"

const createSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  points_ratio: z.number().int().min(1, "Debe ser al menos 1"),
})

type CreateFormValues = z.infer<typeof createSchema>

type SisfiType = "earn_burn" | "cashback" | "pushcard"

type ProgramRow = {
  id: string
  type: SisfiType
  name: string
  detail: string
  active: boolean
  href: string | null
}

const TYPE_META: Record<SisfiType, { label: string; icon: LucideIcon; chip: string }> = {
  earn_burn: { label: "Puntos", icon: Star, chip: "bg-[#E0B0CC]/40 text-[#8a3d6a]" },
  cashback: { label: "Cashback", icon: Wallet, chip: "bg-[#A8CDE0]/40 text-[#2c5d75]" },
  pushcard: { label: "Tarjeta de sellos", icon: Stamp, chip: "bg-[#B49DD9]/35 text-[#5b3d8a]" },
}

export function ProgramsListPage() {
  const { customerId } = useAuth()
  const { data: earnBurn, isLoading: loadingEB } = usePrograms(customerId)
  const { data: cashback, isLoading: loadingCB } = useCashbackPrograms(customerId)
  const { data: pushcard, isLoading: loadingPC } = usePushcardConfig(customerId)
  const createProgram = useCreateProgram()
  const [open, setOpen] = useState(false)

  const isLoading = loadingEB || loadingCB || loadingPC

  // Combinar todos los sisfis del customer en filas unificadas.
  // El backend ya filtra active=true en list (FID-18); pushcard llega
  // como objeto único — solo lo mostramos si active.
  const rows: ProgramRow[] = [
    ...(earnBurn ?? []).map<ProgramRow>((p) => ({
      id: p.id,
      type: "earn_burn",
      name: p.name,
      detail: `1 pt / $${formatPoints(p.points_ratio)}`,
      active: p.active,
      href: `/programas/${p.id}`,
    })),
    ...(cashback ?? []).map<ProgramRow>((p) => ({
      id: p.id,
      type: "cashback",
      name: p.name,
      // El backend devuelve la rate como fracción (0.05) o porcentaje (5)
      // según cómo se creó; normalizamos a porcentaje para display.
      detail: `${p.cashback_rate > 1 ? p.cashback_rate : p.cashback_rate * 100}% de cashback`,
      active: p.active,
      href: null,
    })),
    ...(pushcard && pushcard.active
      ? [
          {
            id: pushcard.customer_sisfi_id,
            type: "pushcard" as const,
            name: pushcard.name,
            detail: `${pushcard.card_slots} sellos para canjear`,
            active: pushcard.active,
            href: "/pushcard",
          } satisfies ProgramRow,
        ]
      : []),
  ]

  const onSubmit = (values: CreateFormValues) => {
    createProgram.mutate(
      { ...values, customer_id: customerId },
      {
        onSuccess: () => {
          toast.success("Programa creado")
          form.reset()
          setOpen(false)
        },
        onError: (err) => toast.error(err.message),
      },
    )
  }

  const form = useForm<CreateFormValues>({
    resolver: zodResolver(createSchema),
    defaultValues: { name: "", points_ratio: 15 },
  })

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold tracking-tight">Programas</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Nuevo programa
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Crear programa de puntos</DialogTitle>
            </DialogHeader>
            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Nombre</FormLabel>
                      <FormControl>
                        <Input placeholder="Mi programa de puntos" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="points_ratio"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>1 punto por cada $</FormLabel>
                      <FormControl>
                        <Input
                          type="number"
                          min={1}
                          {...field}
                          onChange={(e) => field.onChange(Number(e.target.value))}
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <Button type="submit" disabled={createProgram.isPending}>
                  {createProgram.isPending ? "Creando..." : "Crear"}
                </Button>
              </form>
            </Form>
          </DialogContent>
        </Dialog>
      </div>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : rows.length === 0 ? (
        <GlassCard>
          <GlassCardContent>
            <p className="py-6 text-center text-sm text-muted-foreground">
              No hay programas activos. Activa un sistema de fidelizacion desde el wizard
              de onboarding.
            </p>
          </GlassCardContent>
        </GlassCard>
      ) : (
        <GlassCard>
          <GlassCardContent>
            <Table>
              <TableHeader>
                <TableRow className="border-white/30 hover:bg-transparent">
                  <TableHead>Nombre</TableHead>
                  <TableHead>Tipo</TableHead>
                  <TableHead>Detalle</TableHead>
                  <TableHead>Estado</TableHead>
                  <TableHead className="w-10" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {rows.map((r) => {
                  const meta = TYPE_META[r.type]
                  const Icon = meta.icon
                  return (
                    <TableRow
                      key={`${r.type}:${r.id}`}
                      className="border-white/20 hover:bg-white/30"
                    >
                      <TableCell className="font-medium">{r.name}</TableCell>
                      <TableCell>
                        <span
                          className={`inline-flex items-center gap-1.5 rounded-full px-2.5 py-0.5 text-xs font-medium ${meta.chip}`}
                        >
                          <Icon className="h-3 w-3" />
                          {meta.label}
                        </span>
                      </TableCell>
                      <TableCell className="text-muted-foreground">{r.detail}</TableCell>
                      <TableCell>
                        <Badge variant={r.active ? "default" : "secondary"}>
                          {r.active ? "Activo" : "Inactivo"}
                        </Badge>
                      </TableCell>
                      <TableCell>
                        {r.href ? (
                          <Button variant="ghost" size="icon" asChild>
                            <Link to={r.href}>
                              <ChevronRight className="h-4 w-4" />
                            </Link>
                          </Button>
                        ) : null}
                      </TableCell>
                    </TableRow>
                  )
                })}
              </TableBody>
            </Table>
          </GlassCardContent>
        </GlassCard>
      )}
    </div>
  )
}
