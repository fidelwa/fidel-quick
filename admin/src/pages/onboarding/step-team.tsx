import { Input } from "@/components/ui/input"
import { Button } from "@/components/ui/button"
import { toast } from "sonner"
import { Users, Plus, Trash2, MessageCircle } from "lucide-react"
import { COUNTRY_CODES } from "@/lib/country-codes"
import type {
  DraftCollaborator,
  PendingCollaboratorInput,
} from "@/lib/wizard-draft"

interface StepTeamProps {
  collaborators: DraftCollaborator[]
  pendingCollaborator: PendingCollaboratorInput
  onAddCollaborator: (collaborator: Omit<DraftCollaborator, "tempId">) => void
  onRemoveCollaborator: (tempId: string) => void
  onPendingCollaboratorChange: (pending: PendingCollaboratorInput) => void
  onNext: () => void
  onPrev: () => void
}

export function StepTeam({
  collaborators,
  pendingCollaborator,
  onAddCollaborator,
  onRemoveCollaborator,
  onPendingCollaboratorChange,
  onNext,
  onPrev,
}: StepTeamProps) {
  // Inputs en proceso persisten en el draft del wizard.
  const name = pendingCollaborator.name
  const phone = pendingCollaborator.phone
  const countryCode = pendingCollaborator.countryCode
  const setName = (v: string) =>
    onPendingCollaboratorChange({ ...pendingCollaborator, name: v })
  const setPhone = (v: string) =>
    onPendingCollaboratorChange({ ...pendingCollaborator, phone: v })
  const setCountryCode = (v: string) =>
    onPendingCollaboratorChange({ ...pendingCollaborator, countryCode: v })

  const handleAdd = () => {
    if (!name.trim()) {
      toast.error("Ingresa el nombre del colaborador")
      return
    }
    if (!phone.trim()) {
      toast.error("Ingresa el telefono del colaborador")
      return
    }
    if (!/^\d{7,15}$/.test(phone.trim())) {
      toast.error("Ingresa un numero valido (solo digitos)")
      return
    }
    const fullPhone = countryCode + phone.trim()
    if (collaborators.some((c) => c.phone === fullPhone)) {
      toast.error("Ya agregaste este número")
      return
    }
    onAddCollaborator({ name: name.trim(), phone: fullPhone })
    // addCollaborator limpia pendingCollaborator en el reducer.
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
      <div className="text-center">
        <h2 className="text-xl font-semibold">Registra a tu equipo</h2>
        <p className="text-sm text-muted-foreground">
          Agrega a las personas que operaran el programa de fidelidad
        </p>
      </div>

      <div className="flex items-start gap-3 rounded-lg border border-green-200 bg-green-50 p-3 dark:border-green-900 dark:bg-green-950">
        <MessageCircle className="mt-0.5 h-4 w-4 shrink-0 text-green-600" />
        <p className="text-sm text-green-800 dark:text-green-200">
          Los numeros que registres deben tener{" "}
          <span className="font-semibold">WhatsApp activo</span>. Tus colaboradores
          recibirán instrucciones y operarán el programa desde WhatsApp.
        </p>
      </div>

      <div className="rounded-lg border">
        <div className="grid grid-cols-[1fr_1fr_36px] gap-2 border-b bg-muted/50 px-3 py-2 text-xs font-medium text-muted-foreground">
          <span>Nombre</span>
          <span>Telefono</span>
          <span />
        </div>

        {collaborators.length > 0 && (
          <div className="max-h-[200px] overflow-y-auto">
            {collaborators.map((c) => (
              <div
                key={c.tempId}
                className="grid grid-cols-[1fr_1fr_36px] items-center gap-2 border-b px-3 py-2 text-sm last:border-b-0"
              >
                <span className="truncate font-medium">{c.name}</span>
                <span className="truncate text-muted-foreground">{c.phone}</span>
                <Button
                  size="icon"
                  variant="ghost"
                  className="h-7 w-7"
                  onClick={() => onRemoveCollaborator(c.tempId)}
                >
                  <Trash2 className="h-3.5 w-3.5 text-muted-foreground" />
                </Button>
              </div>
            ))}
          </div>
        )}

        {collaborators.length === 0 && (
          <div className="flex items-center justify-center gap-2 px-3 py-8 text-sm text-muted-foreground">
            <Users className="h-4 w-4" />
            <span>Agrega a tu primer colaborador</span>
          </div>
        )}

        <div className="border-t bg-muted/30 px-3 py-2">
          <div className="flex items-center gap-2">
            <Input
              placeholder="Nombre"
              value={name}
              onChange={(e) => setName(e.target.value)}
              className="h-8 min-w-0 flex-[2] text-sm"
              onKeyDown={(e) => e.key === "Enter" && handleAdd()}
            />
            <select
              className="flex h-8 w-[100px] shrink-0 rounded-md border border-input bg-transparent px-2 text-xs shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring"
              value={countryCode}
              onChange={(e) => setCountryCode(e.target.value)}
            >
              {COUNTRY_CODES.map((c) => (
                <option key={c.country} value={c.code}>
                  {c.label}
                </option>
              ))}
            </select>
            <Input
              placeholder="5512345678"
              value={phone}
              onChange={(e) => setPhone(e.target.value.replace(/\D/g, ""))}
              className="h-8 min-w-0 flex-[2] text-sm"
              onKeyDown={(e) => e.key === "Enter" && handleAdd()}
            />
            <Button
              size="icon"
              variant="ghost"
              className="h-8 w-8 shrink-0"
              onClick={handleAdd}
            >
              <Plus className="h-4 w-4" />
            </Button>
          </div>
        </div>
      </div>

      {collaborators.length > 0 && (
        <p className="text-xs text-muted-foreground">
          {collaborators.length} colaborador{collaborators.length > 1 ? "es" : ""}{" "}
          registrado{collaborators.length > 1 ? "s" : ""}
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
