import { describe, it, expect } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { useOnboarding } from "../use-onboarding"
import type { Program, CashbackProgram, Reward, CashbackReward, Collaborator } from "@/types"

const mockProgram: Program = {
  id: "p1", customer_id: "c1", type: "earn-burn",
  name: "Points", points_ratio: 100, active: true,
}

const mockCashbackProgram: CashbackProgram = {
  id: "cb1", customer_id: "c1", type: "cashback",
  name: "Cashback", cashback_rate: 5, active: true,
}

const mockReward: Reward = {
  id: "r1", customer_id: "c1", program_id: "p1",
  name: "Free Coffee", description: "A coffee", points_cost: 100, active: true,
}

const mockCashbackReward: CashbackReward = {
  id: "cr1", customer_id: "c1", program_id: "cb1",
  name: "Discount", description: "10% off", cost: 50, active: true,
}

const mockCollaborator: Collaborator = {
  id: "col1", customer_id: "c1", name: "Juan",
  phone: "+525512345678", hash_id: "abc123", active: true,
}

describe("useOnboarding", () => {
  describe("initial state", () => {
    it("starts at step 1 with forward direction", () => {
      const { result } = renderHook(() => useOnboarding())
      expect(result.current.currentStep).toBe(1)
      expect(result.current.direction).toBe("forward")
    })

    it("starts with null programs", () => {
      const { result } = renderHook(() => useOnboarding())
      expect(result.current.earnBurnProgram).toBeNull()
      expect(result.current.cashbackProgram).toBeNull()
    })

    it("starts with empty arrays", () => {
      const { result } = renderHook(() => useOnboarding())
      expect(result.current.rewards).toEqual([])
      expect(result.current.cashbackRewards).toEqual([])
      expect(result.current.collaborators).toEqual([])
    })

    it("accepts an initial step", () => {
      const { result } = renderHook(() => useOnboarding(3))
      expect(result.current.currentStep).toBe(3)
    })
  })

  describe("step navigation", () => {
    it("nextStep increments and sets direction forward", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.nextStep())
      expect(result.current.currentStep).toBe(2)
      expect(result.current.direction).toBe("forward")
    })

    it("nextStep does not exceed step 4", () => {
      const { result } = renderHook(() => useOnboarding(4))
      act(() => result.current.nextStep())
      expect(result.current.currentStep).toBe(4)
    })

    it("prevStep decrements and sets direction backward", () => {
      const { result } = renderHook(() => useOnboarding(3))
      act(() => result.current.prevStep())
      expect(result.current.currentStep).toBe(2)
      expect(result.current.direction).toBe("backward")
    })

    it("prevStep does not go below step 1", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.prevStep())
      expect(result.current.currentStep).toBe(1)
    })

    it("goToStep sets correct direction going forward", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.goToStep(3))
      expect(result.current.currentStep).toBe(3)
      expect(result.current.direction).toBe("forward")
    })

    it("goToStep sets correct direction going backward", () => {
      const { result } = renderHook(() => useOnboarding(4))
      act(() => result.current.goToStep(2))
      expect(result.current.currentStep).toBe(2)
      expect(result.current.direction).toBe("backward")
    })

    it("goToStep clamps to valid range", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.goToStep(10))
      expect(result.current.currentStep).toBe(4)
      act(() => result.current.goToStep(0))
      expect(result.current.currentStep).toBe(1)
    })

    it("navigates through full flow: 1 → 2 → 3 → 4", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.nextStep())
      expect(result.current.currentStep).toBe(2)
      act(() => result.current.nextStep())
      expect(result.current.currentStep).toBe(3)
      act(() => result.current.nextStep())
      expect(result.current.currentStep).toBe(4)
    })

    it("navigates backward: 4 → 3 → 2 → 1", () => {
      const { result } = renderHook(() => useOnboarding(4))
      act(() => result.current.prevStep())
      expect(result.current.currentStep).toBe(3)
      act(() => result.current.prevStep())
      expect(result.current.currentStep).toBe(2)
      act(() => result.current.prevStep())
      expect(result.current.currentStep).toBe(1)
    })
  })

  describe("data setters", () => {
    it("setEarnBurnProgram updates program", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.setEarnBurnProgram(mockProgram))
      expect(result.current.earnBurnProgram).toEqual(mockProgram)
    })

    it("setEarnBurnProgram can be set to null", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.setEarnBurnProgram(mockProgram))
      act(() => result.current.setEarnBurnProgram(null))
      expect(result.current.earnBurnProgram).toBeNull()
    })

    it("setCashbackProgram updates program", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.setCashbackProgram(mockCashbackProgram))
      expect(result.current.cashbackProgram).toEqual(mockCashbackProgram)
    })

    it("setRewards updates rewards array", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.setRewards([mockReward]))
      expect(result.current.rewards).toEqual([mockReward])
    })

    it("setCashbackRewards updates cashback rewards array", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.setCashbackRewards([mockCashbackReward]))
      expect(result.current.cashbackRewards).toEqual([mockCashbackReward])
    })

    it("setCollaborators updates collaborators array", () => {
      const { result } = renderHook(() => useOnboarding())
      act(() => result.current.setCollaborators([mockCollaborator]))
      expect(result.current.collaborators).toEqual([mockCollaborator])
    })

    it("setters do not affect step or direction", () => {
      const { result } = renderHook(() => useOnboarding(2))
      act(() => {
        result.current.setEarnBurnProgram(mockProgram)
        result.current.setRewards([mockReward])
        result.current.setCollaborators([mockCollaborator])
      })
      expect(result.current.currentStep).toBe(2)
      expect(result.current.direction).toBe("forward")
    })
  })

  describe("combined navigation and data", () => {
    it("full wizard flow with data at each step", () => {
      const { result } = renderHook(() => useOnboarding())

      // Step 1: set programs
      act(() => result.current.setEarnBurnProgram(mockProgram))
      act(() => result.current.nextStep())

      // Step 2: set rewards
      act(() => result.current.setRewards([mockReward]))
      act(() => result.current.nextStep())

      // Step 3: set collaborators
      act(() => result.current.setCollaborators([mockCollaborator]))
      act(() => result.current.nextStep())

      // Step 4: verify all data persists
      expect(result.current.currentStep).toBe(4)
      expect(result.current.earnBurnProgram).toEqual(mockProgram)
      expect(result.current.rewards).toEqual([mockReward])
      expect(result.current.collaborators).toEqual([mockCollaborator])
    })

    it("data persists when navigating backward", () => {
      const { result } = renderHook(() => useOnboarding(3))
      act(() => result.current.setEarnBurnProgram(mockProgram))
      act(() => result.current.setRewards([mockReward]))
      act(() => result.current.prevStep())
      expect(result.current.earnBurnProgram).toEqual(mockProgram)
      expect(result.current.rewards).toEqual([mockReward])
    })
  })
})
