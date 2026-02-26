import { useState } from "react"
import { Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useAuth } from "@/context/auth-context"
import { usePrograms, useCreateProgram } from "@/hooks/use-programs"
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
import { Plus, ChevronRight } from "lucide-react"
import { formatPoints } from "@/lib/utils"

const createSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  points_ratio: z.number().int().min(1, "Debe ser al menos 1"),
})

type CreateFormValues = z.infer<typeof createSchema>

export function ProgramsListPage() {
  const { customerId } = useAuth()
  const { data: programs, isLoading } = usePrograms(customerId)
  const createProgram = useCreateProgram()
  const [open, setOpen] = useState(false)

  const form = useForm<CreateFormValues>({
    resolver: zodResolver(createSchema),
    defaultValues: { name: "", points_ratio: 1 },
  })

  const onSubmit = (values: CreateFormValues) => {
    createProgram.mutate(
      { ...values, customer_id: customerId, type: "earn_burn" },
      {
        onSuccess: () => {
          toast.success("Programa creado")
          form.reset()
          setOpen(false)
        },
        onError: (err) => toast.error(err.message),
      }
    )
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-2xl font-bold">Programas Earn-Burn</h1>
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
                      <FormLabel>Ratio de puntos (por $1)</FormLabel>
                      <FormControl>
                        <Input type="number" min={1} {...field} onChange={(e) => field.onChange(Number(e.target.value))} />
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
      ) : !programs?.length ? (
        <div className="rounded-lg border border-dashed p-8 text-center">
          <p className="text-muted-foreground">No hay programas aun. Crea el primero.</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Nombre</TableHead>
              <TableHead>Ratio de puntos</TableHead>
              <TableHead>Estado</TableHead>
              <TableHead className="w-10" />
            </TableRow>
          </TableHeader>
          <TableBody>
            {programs.map((p) => (
              <TableRow key={p.id}>
                <TableCell className="font-medium">{p.name}</TableCell>
                <TableCell>{formatPoints(p.points_ratio)} pts/$1</TableCell>
                <TableCell>
                  <Badge variant={p.active ? "default" : "secondary"}>
                    {p.active ? "Activo" : "Inactivo"}
                  </Badge>
                </TableCell>
                <TableCell>
                  <Button variant="ghost" size="icon" asChild>
                    <Link to={`/programas/${p.id}`}>
                      <ChevronRight className="h-4 w-4" />
                    </Link>
                  </Button>
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
