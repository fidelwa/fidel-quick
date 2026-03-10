import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { renderHook, waitFor } from "@testing-library/react"
import { useSlugCheck } from "../use-slug-check"
import { createTestRenderHookWrapper } from "@/test/test-utils"

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
})

afterEach(() => {
  vi.restoreAllMocks()
})

function mockSlugResponse(available: boolean) {
  mockFetch.mockResolvedValue({
    ok: true,
    status: 200,
    json: () => Promise.resolve({ available }),
  })
}

describe("useSlugCheck", () => {
  it("returns null availability for slug shorter than 3 chars", () => {
    const { result } = renderHook(() => useSlugCheck("ab"), {
      wrapper: createTestRenderHookWrapper(),
    })
    expect(result.current.isAvailable).toBeNull()
  })

  it("returns null for empty string", () => {
    const { result } = renderHook(() => useSlugCheck(""), {
      wrapper: createTestRenderHookWrapper(),
    })
    expect(result.current.isAvailable).toBeNull()
    expect(result.current.isChecking).toBe(false)
  })

  it("shows isChecking while debounce is pending for valid slug", () => {
    const { result } = renderHook(() => useSlugCheck("test-slug"), {
      wrapper: createTestRenderHookWrapper(),
    })
    expect(result.current.isChecking).toBe(true)
  })

  it("calls API after debounce and returns available", async () => {
    mockSlugResponse(true)
    const { result } = renderHook(() => useSlugCheck("valid-slug"), {
      wrapper: createTestRenderHookWrapper(),
    })

    await waitFor(
      () => {
        expect(result.current.isAvailable).toBe(true)
      },
      { timeout: 2000 }
    )
  })

  it("reports unavailable slug", async () => {
    mockSlugResponse(false)
    const { result } = renderHook(() => useSlugCheck("taken"), {
      wrapper: createTestRenderHookWrapper(),
    })

    await waitFor(
      () => {
        expect(result.current.isAvailable).toBe(false)
      },
      { timeout: 2000 }
    )
  })

  it("does not call API for slugs with less than 3 chars", async () => {
    mockFetch.mockClear()
    const { result } = renderHook(() => useSlugCheck("ab"), {
      wrapper: createTestRenderHookWrapper(),
    })

    // Wait a bit to ensure no API call happens
    await new Promise((r) => setTimeout(r, 500))
    expect(result.current.isAvailable).toBeNull()
    expect(mockFetch).not.toHaveBeenCalled()
  })
})
