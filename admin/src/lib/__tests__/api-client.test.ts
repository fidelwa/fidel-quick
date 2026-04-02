import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import {
  setToken,
  getToken,
  getCustomer,
  updateCustomer,
  getPrograms,
  createProgram,
  getRewards,
  createReward,
  getCashbackPrograms,
  createCashbackProgram,
  getCashbackRewards,
  createCashbackReward,
  getCollaborators,
  createCollaborator,
  getClients,
  getFeedback,
  loginAdmin,
  registerAdmin,
  onboardingRegister,
  onboardingGoogle,
  loginGoogle,
} from "../api-client"

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
  setToken("")
})

afterEach(() => {
  vi.restoreAllMocks()
})

function mockResponse(data: unknown, status = 200) {
  mockFetch.mockResolvedValueOnce({
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(data),
  })
}

function mockNoContent() {
  mockFetch.mockResolvedValueOnce({
    ok: true,
    status: 204,
    json: () => Promise.resolve({}),
  })
}

function mockError(message: string, status = 400) {
  mockFetch.mockResolvedValueOnce({
    ok: false,
    status,
    json: () => Promise.resolve({ error: message }),
  })
}

describe("token management", () => {
  it("starts with empty token", () => {
    expect(getToken()).toBe("")
  })

  it("sets and gets token", () => {
    setToken("my-token")
    expect(getToken()).toBe("my-token")
  })
})

describe("request headers", () => {
  it("includes Content-Type header", async () => {
    mockResponse({ id: "1", name: "Test" })
    await getCustomer("1")
    expect(mockFetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          "Content-Type": "application/json",
        }),
      })
    )
  })

  it("includes Authorization header when token is set", async () => {
    setToken("test-token")
    mockResponse({ id: "1" })
    await getCustomer("1")
    expect(mockFetch).toHaveBeenCalledWith(
      expect.any(String),
      expect.objectContaining({
        headers: expect.objectContaining({
          Authorization: "Bearer test-token",
        }),
      })
    )
  })

  it("omits Authorization when no token", async () => {
    mockResponse({ id: "1" })
    await getCustomer("1")
    const headers = mockFetch.mock.calls[0][1].headers
    expect(headers.Authorization).toBeUndefined()
  })
})

describe("error handling", () => {
  it("throws on non-ok response with error message", async () => {
    mockError("Not found")
    await expect(getCustomer("1")).rejects.toThrow("Not found")
  })

  it("throws generic message when no error body", async () => {
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: () => Promise.reject(new Error("parse error")),
    })
    await expect(getCustomer("1")).rejects.toThrow("Request failed: 500")
  })
})

describe("customer endpoints", () => {
  it("getCustomer calls correct URL", async () => {
    mockResponse({ id: "c1", name: "Test Co" })
    const result = await getCustomer("c1")
    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining("/customers/c1"), expect.any(Object))
    expect(result.name).toBe("Test Co")
  })

  it("updateCustomer sends PUT with data", async () => {
    mockResponse({ id: "c1", name: "Updated" })
    await updateCustomer("c1", { name: "Updated" })
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/customers/c1"),
      expect.objectContaining({
        method: "PUT",
        body: JSON.stringify({ name: "Updated" }),
      })
    )
  })
})

describe("program endpoints", () => {
  it("getPrograms calls correct URL", async () => {
    mockResponse([])
    await getPrograms("c1")
    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining("/programs?customer_id=c1"), expect.any(Object))
  })

  it("createProgram sends POST", async () => {
    const data = { customer_id: "c1", name: "Test", points_ratio: 100 }
    mockResponse({ id: "p1", ...data })
    const result = await createProgram(data)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/programs"),
      expect.objectContaining({ method: "POST", body: JSON.stringify(data) })
    )
    expect(result.id).toBe("p1")
  })
})

describe("reward endpoints", () => {
  it("getRewards calls correct URL", async () => {
    mockResponse([])
    await getRewards("p1")
    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining("/programs/p1/rewards"), expect.any(Object))
  })

  it("createReward sends POST with programId", async () => {
    const data = { name: "Free Coffee", description: "A coffee", points_cost: 100 }
    mockResponse({ id: "r1", ...data })
    await createReward("p1", data)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/programs/p1/rewards"),
      expect.objectContaining({ method: "POST" })
    )
  })
})

describe("cashback endpoints", () => {
  it("getCashbackPrograms calls correct URL", async () => {
    mockResponse([])
    await getCashbackPrograms("c1")
    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining("/cashback-programs?customer_id=c1"), expect.any(Object))
  })

  it("createCashbackProgram sends POST", async () => {
    const data = { customer_id: "c1", name: "CB", cashback_rate: 5 }
    mockResponse({ id: "cb1", ...data })
    await createCashbackProgram(data)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/cashback-programs"),
      expect.objectContaining({ method: "POST" })
    )
  })

  it("getCashbackRewards calls correct URL", async () => {
    mockResponse([])
    await getCashbackRewards("cb1")
    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining("/cashback-programs/cb1/rewards"), expect.any(Object))
  })

  it("createCashbackReward sends POST with programId", async () => {
    const data = { name: "Discount", description: "10% off", cost: 50 }
    mockResponse({ id: "cr1", ...data })
    await createCashbackReward("cb1", data)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/cashback-programs/cb1/rewards"),
      expect.objectContaining({ method: "POST" })
    )
  })
})

describe("collaborator endpoints", () => {
  it("getCollaborators calls correct URL", async () => {
    mockResponse([])
    await getCollaborators("c1")
    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining("/customers/c1/collaborators"), expect.any(Object))
  })

  it("createCollaborator sends POST", async () => {
    const data = { name: "Juan", phone: "+525512345678" }
    mockResponse({ id: "col1", ...data })
    await createCollaborator("c1", data)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/customers/c1/collaborators"),
      expect.objectContaining({ method: "POST" })
    )
  })
})

describe("client endpoints", () => {
  it("getClients calls correct URL", async () => {
    mockResponse([])
    await getClients("c1")
    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining("/customers/c1/clients"), expect.any(Object))
  })
})

describe("feedback endpoints", () => {
  it("getFeedback calls correct URL", async () => {
    mockResponse([])
    await getFeedback("c1")
    expect(mockFetch).toHaveBeenCalledWith(expect.stringContaining("/customers/c1/feedback"), expect.any(Object))
  })
})

describe("auth endpoints", () => {
  it("loginAdmin sends POST with credentials", async () => {
    mockResponse({ token: "t1", admin: { id: "a1", email: "a@b.com", customer_id: "c1" } })
    const result = await loginAdmin("a@b.com", "pass123")
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/auth/login"),
      expect.objectContaining({
        method: "POST",
        body: JSON.stringify({ email: "a@b.com", password: "pass123" }),
      })
    )
    expect(result.token).toBe("t1")
  })

  it("registerAdmin sends POST with data", async () => {
    mockResponse({ token: "t1", admin: { id: "a1", email: "a@b.com", customer_id: "c1" } })
    await registerAdmin("a@b.com", "pass123", "c1")
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/auth/register"),
      expect.objectContaining({ method: "POST" })
    )
  })
})

describe("onboarding endpoints", () => {
  it("onboardingRegister sends POST", async () => {
    const data = {
      name: "Test", slug: "test", phone: "+52551234",
      admin_email: "a@b.com", admin_password: "pass1234",
    }
    mockResponse({ token: "t1", admin: { id: "a1", email: "a@b.com", customer_id: "c1" } })
    await onboardingRegister(data)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/onboarding/register"),
      expect.objectContaining({ method: "POST", body: JSON.stringify(data) })
    )
  })

  it("onboardingGoogle sends POST with google token", async () => {
    const data = {
      google_token: "gtoken", name: "Test", phone: "+52551234",
    }
    mockResponse({ token: "t1", admin: { id: "a1", email: "a@b.com", customer_id: "c1" } })
    await onboardingGoogle(data)
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/onboarding/register/google"),
      expect.objectContaining({ method: "POST", body: JSON.stringify(data) })
    )
  })

  it("loginGoogle sends POST with google token", async () => {
    mockResponse({ token: "t1", admin: { id: "a1", email: "a@b.com", customer_id: "c1" } })
    const result = await loginGoogle("gtoken")
    expect(mockFetch).toHaveBeenCalledWith(
      expect.stringContaining("/auth/login/google"),
      expect.objectContaining({ method: "POST" })
    )
    expect(result.token).toBe("t1")
  })
})

describe("204 No Content handling", () => {
  it("returns undefined for 204 responses", async () => {
    mockNoContent()
    const result = await updateCustomer("c1", { name: "x" })
    expect(result).toBeUndefined()
  })
})
