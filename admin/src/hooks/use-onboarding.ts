import { useState, useCallback, useEffect } from "react"
import {
  loadWizardDraft,
  saveWizardDraft,
  newTempId,
  emptyPendingReward,
  emptyPendingCollaborator,
  emptyPendingProgramForm,
  type DraftSisfi,
  type DraftReward,
  type DraftCollaborator,
  type PendingRewardInput,
  type PendingCollaboratorInput,
  type PendingProgramForm,
  type WizardDraft,
} from "@/lib/wizard-draft"

export interface OnboardingState {
  currentStep: number
  direction: "forward" | "backward"
  sisfi: DraftSisfi | null
  rewards: DraftReward[]
  collaborators: DraftCollaborator[]
  pendingProgramForm: PendingProgramForm
  pendingReward: PendingRewardInput
  pendingCollaborator: PendingCollaboratorInput
}

const initialState: OnboardingState = {
  currentStep: 1,
  direction: "forward",
  sisfi: null,
  rewards: [],
  collaborators: [],
  pendingProgramForm: emptyPendingProgramForm,
  pendingReward: emptyPendingReward,
  pendingCollaborator: emptyPendingCollaborator,
}

export function useOnboarding(customerId: string, initialStep?: number) {
  const [state, setState] = useState<OnboardingState>(() => {
    const draft = customerId ? loadWizardDraft(customerId) : null
    if (draft) {
      return {
        currentStep: draft.currentStep,
        direction: "forward",
        sisfi: draft.sisfi,
        rewards: draft.rewards,
        collaborators: draft.collaborators,
        pendingProgramForm: draft.pendingProgramForm ?? emptyPendingProgramForm,
        pendingReward: draft.pendingReward ?? emptyPendingReward,
        pendingCollaborator:
          draft.pendingCollaborator ?? emptyPendingCollaborator,
      }
    }
    return { ...initialState, currentStep: initialStep ?? 1 }
  })

  // Persistir cualquier cambio al draft (excepto direction, que es transitorio).
  useEffect(() => {
    if (!customerId) return
    const draft: Omit<WizardDraft, "expiresAt"> = {
      customerId,
      currentStep: state.currentStep,
      sisfi: state.sisfi,
      rewards: state.rewards,
      collaborators: state.collaborators,
      pendingProgramForm: state.pendingProgramForm,
      pendingReward: state.pendingReward,
      pendingCollaborator: state.pendingCollaborator,
    }
    saveWizardDraft(draft)
  }, [
    customerId,
    state.currentStep,
    state.sisfi,
    state.rewards,
    state.collaborators,
    state.pendingProgramForm,
    state.pendingReward,
    state.pendingCollaborator,
  ])

  const nextStep = useCallback(() => {
    setState((s) => ({
      ...s,
      direction: "forward",
      currentStep: Math.min(s.currentStep + 1, 4),
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
      currentStep: Math.max(1, Math.min(step, 4)),
    }))
  }, [])

  const setSisfi = useCallback((sisfi: DraftSisfi | null) => {
    setState((s) => {
      if (sisfi && s.sisfi && sisfi.type !== s.sisfi.type) {
        return { ...s, sisfi, rewards: [] }
      }
      return { ...s, sisfi }
    })
  }, [])

  const addReward = useCallback((reward: Omit<DraftReward, "tempId">) => {
    setState((s) => ({
      ...s,
      rewards: [...s.rewards, { ...reward, tempId: newTempId() }],
      pendingReward: emptyPendingReward,
    }))
  }, [])

  const removeReward = useCallback((tempId: string) => {
    setState((s) => ({
      ...s,
      rewards: s.rewards.filter((r) => r.tempId !== tempId),
    }))
  }, [])

  const setRewards = useCallback((rewards: DraftReward[]) => {
    setState((s) => ({ ...s, rewards }))
  }, [])

  const setPendingReward = useCallback((pending: PendingRewardInput) => {
    setState((s) => ({ ...s, pendingReward: pending }))
  }, [])

  const addCollaborator = useCallback(
    (collaborator: Omit<DraftCollaborator, "tempId">) => {
      setState((s) => ({
        ...s,
        collaborators: [
          ...s.collaborators,
          { ...collaborator, tempId: newTempId() },
        ],
        pendingCollaborator: emptyPendingCollaborator,
      }))
    },
    [],
  )

  const removeCollaborator = useCallback((tempId: string) => {
    setState((s) => ({
      ...s,
      collaborators: s.collaborators.filter((c) => c.tempId !== tempId),
    }))
  }, [])

  const setCollaborators = useCallback((collaborators: DraftCollaborator[]) => {
    setState((s) => ({ ...s, collaborators }))
  }, [])

  const setPendingCollaborator = useCallback(
    (pending: PendingCollaboratorInput) => {
      setState((s) => ({ ...s, pendingCollaborator: pending }))
    },
    [],
  )

  const setPendingProgramForm = useCallback((form: PendingProgramForm) => {
    setState((s) => ({ ...s, pendingProgramForm: form }))
  }, [])

  return {
    ...state,
    nextStep,
    prevStep,
    goToStep,
    setSisfi,
    addReward,
    removeReward,
    setRewards,
    setPendingReward,
    addCollaborator,
    removeCollaborator,
    setCollaborators,
    setPendingCollaborator,
    setPendingProgramForm,
  }
}
