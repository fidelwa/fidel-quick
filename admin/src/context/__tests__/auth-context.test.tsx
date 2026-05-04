import { describe, it, expect, beforeEach } from "vitest"
import { renderHook, act } from "@testing-library/react"
import { AuthProvider, useAuth } from "../auth-context"
import type { ReactNode } from "react"

function wrapper({ children }: { children: ReactNode }) {
  return <AuthProvider>{children}</AuthProvider>
}

beforeEach(() => {
  localStorage.clear()
})

describe("useAuth", () => {
  it("throws when used outside AuthProvider", () => {
    expect(() => {
      renderHook(() => useAuth())
    }).toThrow("useAuth must be used within AuthProvider")
  })

  it("starts unauthenticated when no stored data", () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.token).toBe("")
    expect(result.current.customerId).toBe("")
    expect(result.current.email).toBe("")
  })

  it("login sets auth state", () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    act(() => result.current.login("tok123", "cust1", "user@test.com"))
    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.token).toBe("tok123")
    expect(result.current.customerId).toBe("cust1")
    expect(result.current.email).toBe("user@test.com")
  })

  it("logout clears auth state", () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    act(() => result.current.login("tok123", "cust1", "user@test.com"))
    act(() => result.current.logout())
    expect(result.current.isAuthenticated).toBe(false)
    expect(result.current.token).toBe("")
    expect(result.current.customerId).toBe("")
  })

  it("persists to localStorage on login", () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    act(() => result.current.login("tok", "c1", "e@e.com"))
    const stored = JSON.parse(localStorage.getItem("fidel_auth")!)
    expect(stored.token).toBe("tok")
    expect(stored.customerId).toBe("c1")
    expect(stored.email).toBe("e@e.com")
  })

  it("clears localStorage on logout", () => {
    const { result } = renderHook(() => useAuth(), { wrapper })
    act(() => result.current.login("tok", "c1", "e@e.com"))
    act(() => result.current.logout())
    expect(localStorage.getItem("fidel_auth")).toBeNull()
  })

  it("restores state from localStorage", () => {
    localStorage.setItem(
      "fidel_auth",
      JSON.stringify({ token: "saved-tok", customerId: "saved-c", email: "saved@e.com" })
    )
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated).toBe(true)
    expect(result.current.token).toBe("saved-tok")
    expect(result.current.customerId).toBe("saved-c")
    expect(result.current.email).toBe("saved@e.com")
  })

  it("isAuthenticated requires both token and customerId", () => {
    // Only token, no customerId
    localStorage.setItem(
      "fidel_auth",
      JSON.stringify({ token: "tok", customerId: "", email: "" })
    )
    const { result } = renderHook(() => useAuth(), { wrapper })
    expect(result.current.isAuthenticated).toBe(false)
  })
})
