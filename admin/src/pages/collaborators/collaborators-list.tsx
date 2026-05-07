import { useState } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { useAuth } from "@/context/auth-context"
import { useCollaborators, useCreateCollaborator } from "@/hooks/use-collaborators"
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
import { GlassCard, GlassCardContent } from "@/components/ui/glass-card"
import { Plus } from "lucide-react"

const createSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  phone: z.string().min(1, "El telefono es requerido"),
})

type CreateFormValues = z.infer<typeof createSchema>

export function CollaboratorsListPage() {
  const { customerId } = useAuth()
  const { data: collaborators, isLoading } = useCollaborators(customerId)
  const createCollaborator = useCreateCollaborator(customerId)
  const [open, setOpen] = useState(false)

  const form = useForm<CreateFormValues>({
    resolver: zodResolver(createSchema),
    defaultValues: { name: "", phone: "" },
  })

  const onSubmit = (values: CreateFormValues) => {
    createCollaborator.mutate(values, {
      onSuccess: () => {
        toast.success("Colaborador creado")
        form.reset()
        setOpen(false)
      },
      onError: (err) => toast.error(err.message),
    })
  }

  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <h1 className="text-3xl font-bold tracking-tight">Colaboradores</h1>
        <Dialog open={open} onOpenChange={setOpen}>
          <DialogTrigger asChild>
            <Button>
              <Plus className="mr-2 h-4 w-4" />
              Nuevo colaborador
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Crear colaborador</DialogTitle>
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
                        <Input placeholder="Juan Perez" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="phone"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Telefono (con codigo de pais)</FormLabel>
                      <FormControl>
                        <Input placeholder="521234567890" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <Button type="submit" disabled={createCollaborator.isPending}>
                  {createCollaborator.isPending ? "Creando..." : "Crear"}
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
      ) : !collaborators?.length ? (
        <GlassCard>
          <GlassCardContent>
            <p className="py-6 text-center text-sm text-muted-foreground">
              No hay colaboradores. Crea el primero.
            </p>
          </GlassCardContent>
        </GlassCard>
      ) : (
        <GlassCard>
          <GlassCardContent>
            <Table>
              <TableHeader>
                <TableRow className="border-b border-white/40 hover:bg-transparent">
                  <TableHead className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Nombre
                  </TableHead>
                  <TableHead className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Telefono
                  </TableHead>
                  <TableHead className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Hash ID
                  </TableHead>
                  <TableHead className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Estado
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {collaborators.map((c) => (
                  <TableRow key={c.id} className="border-0 hover:bg-white/40">
                    <TableCell className="rounded-l-2xl py-4 font-medium">{c.name}</TableCell>
                    <TableCell className="py-4">{c.phone}</TableCell>
                    <TableCell className="py-4 font-mono text-xs">{c.hash_id}</TableCell>
                    <TableCell className="rounded-r-2xl py-4">
                      <Badge variant={c.active ? "default" : "secondary"}>
                        {c.active ? "Activo" : "Inactivo"}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </GlassCardContent>
        </GlassCard>
      )}
    </div>
  )
}
