// Cliente-side draft del wizard de onboarding. Persiste en localStorage
// con TTL de 30 días para que el usuario pueda dejar el proceso a la mitad
// y retomarlo después. Nada se escribe al backend hasta que el usuario
// presiona "Finalizar" en el último paso.

const STORAGE_KEY = "fidel:wizard-draft"
const TTL_MS = 30 * 24 * 60 * 60 * 1000

export type SisfiType = "earn_burn" | "cashback" | "pushcard"

export interface DraftSisfi {
  type: SisfiType
  name: string
  ratio?: number // earn_burn — puntos por cada $
  rate?: number // cashback — % de devolución
  slots?: number // pushcard — sellos por tarjeta
}

export interface DraftReward {
  tempId: string
  name: string
  description: string
  cost: number
}

export interface DraftCollaborator {
  tempId: string
  name: string
  phone: string
}

// Buffers de inputs en proceso (lo que el usuario tipeó pero aún no
// confirmó con +/Siguiente). Permiten que si retrocede o cierra el
// browser, al volver vea exactamente lo que estaba escribiendo.
export interface PendingRewardInput {
  name: string
  description: string
  cost: string
}

export interface PendingCollaboratorInput {
  name: string
  phone: string
  countryCode: string
}

export interface PendingProgramForm {
  selected: SisfiType | null
  earnName: string
  earnRatio: string
  cashbackName: string
  cashbackRate: string
  pushcardName: string
  pushcardSlots: string
}

export interface WizardDraft {
  customerId: string
  expiresAt: number
  currentStep: number
  sisfi: DraftSisfi | null
  rewards: DraftReward[]
  collaborators: DraftCollaborator[]
  pendingProgramForm: PendingProgramForm
  pendingReward: PendingRewardInput
  pendingCollaborator: PendingCollaboratorInput
}

export const emptyPendingReward: PendingRewardInput = {
  name: "",
  description: "",
  cost: "",
}

export const emptyPendingCollaborator: PendingCollaboratorInput = {
  name: "",
  phone: "",
  countryCode: "+52",
}

export const emptyPendingProgramForm: PendingProgramForm = {
  selected: null,
  earnName: "",
  earnRatio: "15",
  cashbackName: "",
  cashbackRate: "5",
  pushcardName: "",
  pushcardSlots: "10",
}

export function loadWizardDraft(customerId: string): WizardDraft | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const draft = JSON.parse(raw) as WizardDraft
    if (!draft || draft.customerId !== customerId) return null
    if (Date.now() > draft.expiresAt) {
      localStorage.removeItem(STORAGE_KEY)
      return null
    }
    return draft
  } catch {
    return null
  }
}

export function saveWizardDraft(draft: Omit<WizardDraft, "expiresAt">): void {
  try {
    const data: WizardDraft = { ...draft, expiresAt: Date.now() + TTL_MS }
    localStorage.setItem(STORAGE_KEY, JSON.stringify(data))
  } catch {
    // localStorage podría estar lleno o deshabilitado (modo privado, cuotas)
    // — el wizard sigue funcionando solo en memoria.
  }
}

export function clearWizardDraft(): void {
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // ignore
  }
}

export function newTempId(): string {
  return `tmp_${Math.random().toString(36).slice(2, 11)}_${Date.now().toString(36)}`
}
