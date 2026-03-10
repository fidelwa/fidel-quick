import { useState } from "react"
import { useAuth } from "@/context/auth-context"
import { useCreateCollaborator } from "@/hooks/use-collaborators"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { toast } from "sonner"
import { Users, Loader2, Plus } from "lucide-react"
import type { Collaborator } from "@/types"

interface StepTeamProps {
  collaborators: Collaborator[]
  onCollaboratorsChange: (collaborators: Collaborator[]) => void
  onNext: () => void
  onPrev: () => void
}

export function StepTeam({
  collaborators,
  onCollaboratorsChange,
  onNext,
  onPrev,
}: StepTeamProps) {
  const { customerId } = useAuth()
  const createCollaborator = useCreateCollaborator(customerId)

  const [name, setName] = useState("")
  const [phone, setPhone] = useState("")
  const [adding, setAdding] = useState(false)

  const handleAdd = async () => {
    if (!name.trim()) {
      toast.error("Ingresa el nombre del colaborador")
      return
    }
    if (!phone.trim()) {
      toast.error("Ingresa el telefono del colaborador")
      return
    }
    setAdding(true)
    try {
      const collab = await createCollaborator.mutateAsync({
        name: name.trim(),
        phone: phone.trim(),
      })
      onCollaboratorsChange([...collaborators, collab])
      setName("")
      setPhone("")
      toast.success("Colaborador registrado")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al registrar colaborador")
    } finally {
      setAdding(false)
    }
  }

  const handleNext = () => {
    if (collaborators.length === 0) {
      toast.error("Registra al menos un colaborador")
      return
    }
    onNext()
  }

  return (
    <div className="space-y-6">
      <div>
        <h2 className="text-xl font-semibold">Registra a tu equipo</h2>
        <p className="text-sm text-muted-foreground">
          Agrega a las personas que operaran el programa de fidelidad
        </p>
      </div>

      {collaborators.length > 0 ? (
        <div className="space-y-2">
          {collaborators.map((c) => (
            <div key={c.id} className="flex items-center justify-between rounded-lg border p-3">
              <div>
                <p className="text-sm font-medium">{c.name}</p>
                <p className="text-xs text-muted-foreground">{c.phone}</p>
              </div>
              <Badge variant="secondary">Registrado</Badge>
            </div>
          ))}
        </div>
      ) : (
        <div className="flex flex-col items-center justify-center rounded-lg border border-dashed p-8 text-center">
          <Users className="mb-2 h-10 w-10 text-muted-foreground/50" />
          <p className="text-sm text-muted-foreground">
            Agrega a tu primer colaborador para comenzar
          </p>
        </div>
      )}

      <div className="grid gap-3 sm:grid-cols-2">
        <div className="space-y-1.5">
          <Label>Nombre</Label>
          <Input
            placeholder="Juan Perez"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
        </div>
        <div className="space-y-1.5">
          <Label>Telefono</Label>
          <div className="flex gap-2">
            <Input
              placeholder="+525512345678"
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
            />
            <Button size="sm" onClick={handleAdd} disabled={adding}>
              {adding ? (
                <Loader2 className="h-4 w-4 animate-spin" />
              ) : (
                <Plus className="h-4 w-4" />
              )}
            </Button>
          </div>
        </div>
      </div>

      <div className="flex justify-between">
        <Button variant="outline" onClick={onPrev}>
          Anterior
        </Button>
        <Button onClick={handleNext}>Siguiente</Button>
      </div>
    </div>
  )
}
