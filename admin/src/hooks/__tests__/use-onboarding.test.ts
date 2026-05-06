import { describe, it, expect, beforeEach, afterEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { useOnboarding } from "../use-onboarding"
import type { DraftSisfi, DraftReward, DraftCollaborator } from "@/lib/wizard-draft"

const earnSisfi: DraftSisfi = { type: "earn_burn", name: "Puntos", ratio: 15 }
const cashbackSisfi: DraftSisfi = { type: "cashback", name: "Cash", rate: 5 }

const reward1 = { name: "Cafe", description: "Un cafe", cost: 100 }
const collab1 = { name: "Juan", phone: "+525512345678" }

beforeEach(() => {
  localStorage.clear()
})

afterEach(() => {
  localStorage.clear()
})

describe("useOnboarding (draft mode)", () => {
  describe("initial state", () => {
    it("starts at step 1 with forward direction", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      expect(result.current.currentStep).toBe(1)
      expect(result.current.direction).toBe("forward")
    })

    it("starts with no sisfi and empty arrays", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      expect(result.current.sisfi).toBeNull()
      expect(result.current.rewards).toEqual([])
      expect(result.current.collaborators).toEqual([])
    })

    it("accepts an initial step", () => {
      const { result } = renderHook(() => useOnboarding("c1", 3))
      expect(result.current.currentStep).toBe(3)
    })
  })

  describe("step navigation", () => {
    it("nextStep increments and clamps at 4", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.nextStep())
      expect(result.current.currentStep).toBe(2)
    })

    it("prevStep decrements and clamps at 1", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.prevStep())
      expect(result.current.currentStep).toBe(1)
    })

    it("goToStep clamps to [1, 4]", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.goToStep(10))
      expect(result.current.currentStep).toBe(4)
      act(() => result.current.goToStep(0))
      expect(result.current.currentStep).toBe(1)
    })
  })

  describe("draft setters", () => {
    it("setSisfi sets the sisfi draft", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.setSisfi(earnSisfi))
      expect(result.current.sisfi).toEqual(earnSisfi)
    })

    it("changing sisfi type clears rewards (different units)", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.setSisfi(earnSisfi))
      act(() => result.current.addReward(reward1))
      expect(result.current.rewards).toHaveLength(1)
      act(() => result.current.setSisfi(cashbackSisfi))
      expect(result.current.rewards).toHaveLength(0)
    })

    it("addReward appends with tempId", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.addReward(reward1))
      expect(result.current.rewards).toHaveLength(1)
      expect(result.current.rewards[0].name).toBe("Cafe")
      expect(result.current.rewards[0].tempId).toBeTruthy()
    })

    it("removeReward removes by tempId", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.addReward(reward1))
      const id = result.current.rewards[0].tempId
      act(() => result.current.removeReward(id))
      expect(result.current.rewards).toHaveLength(0)
    })

    it("addCollaborator appends with tempId", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.addCollaborator(collab1))
      expect(result.current.collaborators).toHaveLength(1)
      expect(result.current.collaborators[0].tempId).toBeTruthy()
    })

    it("removeCollaborator removes by tempId", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.addCollaborator(collab1))
      const id = result.current.collaborators[0].tempId
      act(() => result.current.removeCollaborator(id))
      expect(result.current.collaborators).toHaveLength(0)
    })
  })

  describe("localStorage persistence", () => {
    it("persists draft and restores on remount", () => {
      const { result, unmount } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.setSisfi(earnSisfi))
      act(() => result.current.addReward(reward1))
      act(() => result.current.addCollaborator(collab1))
      act(() => result.current.nextStep())
      unmount()

      const { result: result2 } = renderHook(() => useOnboarding("c1"))
      expect(result2.current.sisfi).toEqual(earnSisfi)
      expect(result2.current.rewards).toHaveLength(1)
      expect(result2.current.collaborators).toHaveLength(1)
      expect(result2.current.currentStep).toBe(2)
    })

    it("draft for a different customerId is ignored", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      act(() => result.current.setSisfi(earnSisfi))

      const { result: r2 } = renderHook(() => useOnboarding("c2"))
      expect(r2.current.sisfi).toBeNull()
    })
  })

  describe("legacy: typed draft items receive tempIds", () => {
    it("setRewards replaces array as-is", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      const rewards: DraftReward[] = [
        { tempId: "x1", name: "A", description: "", cost: 1 },
        { tempId: "x2", name: "B", description: "", cost: 2 },
      ]
      act(() => result.current.setRewards(rewards))
      expect(result.current.rewards).toEqual(rewards)
    })

    it("setCollaborators replaces array as-is", () => {
      const { result } = renderHook(() => useOnboarding("c1"))
      const list: DraftCollaborator[] = [
        { tempId: "y1", name: "A", phone: "+1" },
      ]
      act(() => result.current.setCollaborators(list))
      expect(result.current.collaborators).toEqual(list)
    })
  })
})
