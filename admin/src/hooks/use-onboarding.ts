import { useState, useCallback } from "react"
import type { Program, CashbackProgram, Reward, CashbackReward, Collaborator } from "@/types"

export interface OnboardingState {
  currentStep: number
  direction: "forward" | "backward"
  earnBurnProgram: Program | null
  cashbackProgram: CashbackProgram | null
  rewards: Reward[]
  cashbackRewards: CashbackReward[]
  collaborators: Collaborator[]
}

const initialState: OnboardingState = {
  currentStep: 1,
  direction: "forward",
  earnBurnProgram: null,
  cashbackProgram: null,
  rewards: [],
  cashbackRewards: [],
  collaborators: [],
}

export function useOnboarding(initialStep?: number) {
  const [state, setState] = useState<OnboardingState>(() => ({
    ...initialState,
    currentStep: initialStep ?? 1,
  }))

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

  const setEarnBurnProgram = useCallback((program: Program | null) => {
    setState((s) => ({ ...s, earnBurnProgram: program }))
  }, [])

  const setCashbackProgram = useCallback((program: CashbackProgram | null) => {
    setState((s) => ({ ...s, cashbackProgram: program }))
  }, [])

  const setRewards = useCallback((rewards: Reward[]) => {
    setState((s) => ({ ...s, rewards }))
  }, [])

  const setCashbackRewards = useCallback((cashbackRewards: CashbackReward[]) => {
    setState((s) => ({ ...s, cashbackRewards }))
  }, [])

  const setCollaborators = useCallback((collaborators: Collaborator[]) => {
    setState((s) => ({ ...s, collaborators }))
  }, [])

  return {
    ...state,
    nextStep,
    prevStep,
    goToStep,
    setEarnBurnProgram,
    setCashbackProgram,
    setRewards,
    setCashbackRewards,
    setCollaborators,
  }
}
