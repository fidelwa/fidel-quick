import { useState, useCallback, useEffect } from "react"

// Drafts: la "config pendiente" que el wizard anónimo recolecta antes
// de que el usuario cree la cuenta. Mantienen solo los campos que el
// usuario rellena — no IDs ni timestamps, porque esos los asigna el
// backend cuando el step-account dispara las llamadas POST en batch.
export interface BusinessInfo {
  name: string
  country_code: string
  phone: string
  description: string
}

export interface EarnBurnDraft {
  name: string
  points_ratio: number
}

export interface CashbackDraft {
  name: string
  cashback_rate: number
}

export interface PushcardDraft {
  name: string
  card_slots: number
  // Descripción libre de la recompensa al completar la tarjeta (ej. "Café
  // gratis"). Se asigna como reward_on_complete de la pushcard_config al crear
  // la cuenta. Opcional: si queda vacío, se puede configurar luego en /pushcard.
  reward_on_complete?: string
}

export interface RewardDraft {
  // local UUID solo para React keys; descartado al crear en backend.
  local_id: string
  name: string
  description: string
  points_cost: number
}

export interface CashbackRewardDraft {
  local_id: string
  name: string
  description: string
  cost: number
}

export interface CollaboratorDraft {
  local_id: string
  name: string
  phone: string
}

export interface OnboardingState {
  currentStep: number
  direction: "forward" | "backward"
  businessInfo: BusinessInfo | null
  earnBurnDraft: EarnBurnDraft | null
  cashbackDraft: CashbackDraft | null
  pushcardDraft: PushcardDraft | null
  rewardDrafts: RewardDraft[]
  cashbackRewardDrafts: CashbackRewardDraft[]
  collaboratorDrafts: CollaboratorDraft[]
}

const TOTAL_STEPS = 5

const initialState: OnboardingState = {
  currentStep: 1,
  direction: "forward",
  businessInfo: null,
  earnBurnDraft: null,
  cashbackDraft: null,
  pushcardDraft: null,
  rewardDrafts: [],
  cashbackRewardDrafts: [],
  collaboratorDrafts: [],
}

const STORAGE_KEY = "fidel_onboarding_draft"

// TTL del borrador cacheado. Pasado este lapso desde el último guardado, el
// draft se considera vencido y se descarta al cargar — así un onboarding
// abandonado hace meses no reaparece con datos rancios.
const DRAFT_TTL_MS = 30 * 24 * 60 * 60 * 1000 // 30 días

type PersistedDraft = Omit<OnboardingState, "currentStep" | "direction">

// Envoltura persistida: el draft + el timestamp de guardado para aplicar TTL.
interface StoredDraft {
  savedAt: number
  draft: PersistedDraft
}

function loadDraft(): Partial<OnboardingState> {
  if (typeof window === "undefined") return {}
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return {}
    const parsed = JSON.parse(raw) as Partial<StoredDraft> & Partial<PersistedDraft>
    // Formato nuevo con { savedAt, draft }.
    if (parsed && typeof parsed === "object" && "savedAt" in parsed && "draft" in parsed) {
      const stored = parsed as StoredDraft
      if (Date.now() - stored.savedAt > DRAFT_TTL_MS) {
        // Vencido: limpiamos y arrancamos de cero.
        try {
          localStorage.removeItem(STORAGE_KEY)
        } catch {
          // ignore
        }
        return {}
      }
      return stored.draft
    }
    // Formato viejo (draft plano, sin savedAt): lo aceptamos una vez; el
    // próximo saveDraft lo migra al formato con timestamp.
    return parsed as Partial<PersistedDraft>
  } catch {
    return {}
  }
}

function saveDraft(state: OnboardingState) {
  if (typeof window === "undefined") return
  const stored: StoredDraft = {
    savedAt: Date.now(),
    draft: {
      businessInfo: state.businessInfo,
      earnBurnDraft: state.earnBurnDraft,
      cashbackDraft: state.cashbackDraft,
      pushcardDraft: state.pushcardDraft,
      rewardDrafts: state.rewardDrafts,
      cashbackRewardDrafts: state.cashbackRewardDrafts,
      collaboratorDrafts: state.collaboratorDrafts,
    },
  }
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(stored))
  } catch {
    // localStorage puede fallar en modo privado o si está lleno —
    // silencioso; el wizard sigue funcionando solo perdiendo el resume.
  }
}

export function clearOnboardingDraft() {
  if (typeof window === "undefined") return
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // ignore
  }
}

export function generateLocalId(): string {
  // No necesitamos UUID criptográfico; sirve solo como React key.
  return `local-${Date.now()}-${Math.random().toString(36).slice(2, 9)}`
}

export function useOnboarding(initialStep?: number) {
  const [state, setState] = useState<OnboardingState>(() => {
    const draft = loadDraft()
    return {
      ...initialState,
      ...draft,
      currentStep: initialStep ?? 1,
    }
  })

  useEffect(() => {
    saveDraft(state)
  }, [state])

  const nextStep = useCallback(() => {
    setState((s) => ({
      ...s,
      direction: "forward",
      currentStep: Math.min(s.currentStep + 1, TOTAL_STEPS),
    }))
  }, [])

  const prevStep = useCallback(() => {
    setState((s) => ({
      ...s,
      direction: "backward",
      currentStep: Math.max(s.currentStep - 1, 1),
    }))
  }, [])

  const goToStep = useCallback((step: number) => {
    setState((s) => ({
      ...s,
      direction: step > s.currentStep ? "forward" : "backward",
      currentStep: Math.max(1, Math.min(step, TOTAL_STEPS)),
    }))
  }, [])

  const setBusinessInfo = useCallback((info: BusinessInfo | null) => {
    setState((s) => ({ ...s, businessInfo: info }))
  }, [])

  const setEarnBurnDraft = useCallback((draft: EarnBurnDraft | null) => {
    setState((s) => ({ ...s, earnBurnDraft: draft }))
  }, [])

  const setCashbackDraft = useCallback((draft: CashbackDraft | null) => {
    setState((s) => ({ ...s, cashbackDraft: draft }))
  }, [])

  const setPushcardDraft = useCallback((draft: PushcardDraft | null) => {
    setState((s) => ({ ...s, pushcardDraft: draft }))
  }, [])

  const setRewardDrafts = useCallback((drafts: RewardDraft[]) => {
    setState((s) => ({ ...s, rewardDrafts: drafts }))
  }, [])

  const setCashbackRewardDrafts = useCallback((drafts: CashbackRewardDraft[]) => {
    setState((s) => ({ ...s, cashbackRewardDrafts: drafts }))
  }, [])

  const setCollaboratorDrafts = useCallback((drafts: CollaboratorDraft[]) => {
    setState((s) => ({ ...s, collaboratorDrafts: drafts }))
  }, [])

  const reset = useCallback(() => {
    clearOnboardingDraft()
    setState({ ...initialState, currentStep: 1 })
  }, [])

  return {
    ...state,
    totalSteps: TOTAL_STEPS,
    nextStep,
    prevStep,
    goToStep,
    setBusinessInfo,
    setEarnBurnDraft,
    setCashbackDraft,
    setPushcardDraft,
    setRewardDrafts,
    setCashbackRewardDrafts,
    setCollaboratorDrafts,
    reset,
  }
}
