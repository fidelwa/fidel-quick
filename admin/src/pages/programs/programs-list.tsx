import { useState } from "react"
import { Link, useNavigate } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { useMutation, useQueryClient } from "@tanstack/react-query"
import { toast } from "sonner"
import { useAuth } from "@/context/auth-context"
import { usePrograms, useCreateProgram } from "@/hooks/use-programs"
import { useCashbackPrograms, useCreateCashbackProgram } from "@/hooks/use-cashback-programs"
import { usePushcardConfig } from "@/hooks/use-pushcard"
import { createCustomerSisfi, upsertPushcardConfig } from "@/lib/api-client"
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Plus, ChevronRight, Star, Wallet, Stamp } from "lucide-react"
import type { LucideIcon } from "lucide-react"
import { formatPoints } from "@/lib/utils"

const earnBurnSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  points_ratio: z.number().int().min(1, "Debe ser al menos 1"),
})

const cashbackSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  cashback_rate: z.number().min(0.1, "Debe ser mayor a 0").max(100, "Maximo 100%"),
})

const pushcardSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  card_slots: z.number().int().min(1, "Minimo 1 sello").max(50, "Maximo 50"),
})

type EarnBurnFormValues = z.infer<typeof earnBurnSchema>
type CashbackFormValues = z.infer<typeof cashbackSchema>
type PushcardFormValues = z.infer<typeof pushcardSchema>

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
  const navigate = useNavigate()
  const queryClient = useQueryClient()
  const { data: earnBurn, isLoading: loadingEB } = usePrograms(customerId)
  const { data: cashback, isLoading: loadingCB } = useCashbackPrograms(customerId)
  const { data: pushcard, isLoading: loadingPC } = usePushcardConfig(customerId)
  const createProgram = useCreateProgram()
  const createCashback = useCreateCashbackProgram()
  const createPushcard = useMutation({
    mutationFn: async (values: PushcardFormValues) => {
      const cs = await createCustomerSisfi({
        customer_id: customerId,
        sisfi_id: "pushcard",
        name: values.name,
      })
      await upsertPushcardConfig(cs.id, { card_slots: values.card_slots })
      return cs
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["pushcard-config"] })
    },
  })
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

  const earnForm = useForm<EarnBurnFormValues>({
    resolver: zodResolver(earnBurnSchema),
    defaultValues: { name: "", points_ratio: 15 },
  })
  const cashbackForm = useForm<CashbackFormValues>({
    resolver: zodResolver(cashbackSchema),
    defaultValues: { name: "", cashback_rate: 5 },
  })
  const pushcardForm = useForm<PushcardFormValues>({
    resolver: zodResolver(pushcardSchema),
    defaultValues: { name: "Tarjeta de sellos", card_slots: 10 },
  })

  const onCreateEarnBurn = (values: EarnBurnFormValues) => {
    createProgram.mutate(
      { ...values, customer_id: customerId },
      {
        onSuccess: () => {
          toast.success("Programa de puntos creado")
          earnForm.reset()
          setOpen(false)
        },
        onError: (err) => toast.error(err.message),
      },
    )
  }

  const onCreateCashback = (values: CashbackFormValues) => {
    createCashback.mutate(
      { ...values, customer_id: customerId },
      {
        onSuccess: () => {
          toast.success("Programa de cashback creado")
          cashbackForm.reset()
          setOpen(false)
        },
        onError: (err) => toast.error(err.message),
      },
    )
  }

  const onCreatePushcard = (values: PushcardFormValues) => {
    createPushcard.mutate(values, {
      onSuccess: () => {
        toast.success("Tarjeta de sellos creada")
        pushcardForm.reset({ name: "Tarjeta de sellos", card_slots: 10 })
        setOpen(false)
        navigate("/pushcard")
      },
      onError: (err) => toast.error(err.message),
    })
  }

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
              <DialogTitle>Crear programa</DialogTitle>
            </DialogHeader>
            <Tabs defaultValue="earn_burn" className="w-full">
              <TabsList className="grid w-full grid-cols-3">
                <TabsTrigger value="earn_burn">
                  <Star className="mr-2 h-4 w-4" /> Puntos
                </TabsTrigger>
                <TabsTrigger value="cashback">
                  <Wallet className="mr-2 h-4 w-4" /> Cashback
                </TabsTrigger>
                <TabsTrigger value="pushcard" disabled={!!pushcard}>
                  <Stamp className="mr-2 h-4 w-4" /> Sellos
                </TabsTrigger>
              </TabsList>
              <TabsContent value="earn_burn" className="mt-4">
                <Form {...earnForm}>
                  <form onSubmit={earnForm.handleSubmit(onCreateEarnBurn)} className="space-y-4">
                    <FormField
                      control={earnForm.control}
                      name="name"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Nombre</FormLabel>
                          <FormControl>
                            <Input placeholder="Programa de puntos" {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={earnForm.control}
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
                      {createProgram.isPending ? "Creando..." : "Crear programa de puntos"}
                    </Button>
                  </form>
                </Form>
              </TabsContent>
              <TabsContent value="cashback" className="mt-4">
                <Form {...cashbackForm}>
                  <form onSubmit={cashbackForm.handleSubmit(onCreateCashback)} className="space-y-4">
                    <FormField
                      control={cashbackForm.control}
                      name="name"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Nombre</FormLabel>
                          <FormControl>
                            <Input placeholder="Programa de cashback" {...field} />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={cashbackForm.control}
                      name="cashback_rate"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>% de cashback</FormLabel>
                          <FormControl>
                            <Input
                              type="number"
                              min={0.1}
                              max={100}
                              step={0.1}
                              {...field}
                              onChange={(e) => field.onChange(Number(e.target.value))}
                            />
                          </FormControl>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <Button type="submit" disabled={createCashback.isPending}>
                      {createCashback.isPending ? "Creando..." : "Crear programa de cashback"}
                    </Button>
                  </form>
                </Form>
              </TabsContent>
              <TabsContent value="pushcard" className="mt-4">
                {pushcard ? (
                  <p className="text-sm text-muted-foreground">
                    Ya tienes una tarjeta de sellos configurada. Solo se permite una por negocio.
                  </p>
                ) : (
                  <Form {...pushcardForm}>
                    <form onSubmit={pushcardForm.handleSubmit(onCreatePushcard)} className="space-y-4">
                      <FormField
                        control={pushcardForm.control}
                        name="name"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Nombre</FormLabel>
                            <FormControl>
                              <Input placeholder="Tarjeta de sellos" {...field} />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <FormField
                        control={pushcardForm.control}
                        name="card_slots"
                        render={({ field }) => (
                          <FormItem>
                            <FormLabel>Sellos para completar</FormLabel>
                            <FormControl>
                              <Input
                                type="number"
                                min={1}
                                max={50}
                                {...field}
                                onChange={(e) => field.onChange(Number(e.target.value))}
                              />
                            </FormControl>
                            <FormMessage />
                          </FormItem>
                        )}
                      />
                      <Button type="submit" disabled={createPushcard.isPending}>
                        {createPushcard.isPending ? "Creando..." : "Crear tarjeta de sellos"}
                      </Button>
                    </form>
                  </Form>
                )}
              </TabsContent>
            </Tabs>
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
