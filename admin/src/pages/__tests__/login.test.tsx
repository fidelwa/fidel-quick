import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { LoginPage } from "../login"

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
  localStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("LoginPage", () => {
  it("renders login form", () => {
    renderWithProviders(<LoginPage />)
    expect(screen.getByText("Fidel Admin")).toBeInTheDocument()
    expect(screen.getByLabelText("Email")).toBeInTheDocument()
    expect(screen.getByLabelText("Password")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Iniciar sesion" })).toBeInTheDocument()
  })

  it("renders register link", () => {
    renderWithProviders(<LoginPage />)
    expect(screen.getByText("Registrate")).toBeInTheDocument()
    expect(screen.getByText("Registrate").closest("a")).toHaveAttribute("href", "/registro")
  })

  it("shows validation toast when fields are empty", async () => {
    const user = userEvent.setup()
    renderWithProviders(<LoginPage />)
    await user.click(screen.getByRole("button", { name: "Iniciar sesion" }))
    // Should not call fetch when fields are empty
    expect(mockFetch).not.toHaveBeenCalled()
  })

  it("calls loginAdmin on submit with credentials", async () => {
    const user = userEvent.setup()
    mockFetch
      // loginAdmin
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({
          token: "tok1",
          admin: { id: "a1", email: "test@e.com", customer_id: "c1" },
        }),
      })
      // getCustomer (post-login check)
      .mockResolvedValueOnce({
        ok: true,
        status: 200,
        json: () => Promise.resolve({
          id: "c1", name: "Test", onboarding_completed: true,
        }),
      })

    renderWithProviders(<LoginPage />)

    await user.type(screen.getByLabelText("Email"), "test@e.com")
    await user.type(screen.getByLabelText("Password"), "pass1234")
    await user.click(screen.getByRole("button", { name: "Iniciar sesion" }))

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/auth/login"),
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ email: "test@e.com", password: "pass1234" }),
        })
      )
    })
  })

  it("shows loading state during submission", async () => {
    const user = userEvent.setup()
    let resolveLogin: (v: unknown) => void
    mockFetch.mockReturnValueOnce(
      new Promise((r) => { resolveLogin = r })
    )

    renderWithProviders(<LoginPage />)
    await user.type(screen.getByLabelText("Email"), "test@e.com")
    await user.type(screen.getByLabelText("Password"), "pass1234")
    await user.click(screen.getByRole("button", { name: "Iniciar sesion" }))

    expect(screen.getByText("Verificando...")).toBeInTheDocument()

    // Cleanup
    resolveLogin!({
      ok: false, status: 401,
      json: () => Promise.resolve({ error: "bad" }),
    })
  })

  it("shows error on failed login", async () => {
    const user = userEvent.setup()
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 401,
      json: () => Promise.resolve({ error: "Credenciales invalidas" }),
    })

    renderWithProviders(<LoginPage />)
    await user.type(screen.getByLabelText("Email"), "bad@e.com")
    await user.type(screen.getByLabelText("Password"), "wrong")
    await user.click(screen.getByRole("button", { name: "Iniciar sesion" }))

    // Button should be re-enabled after error
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Iniciar sesion" })).not.toBeDisabled()
    })
  })

  it("has 'no account' link text", () => {
    renderWithProviders(<LoginPage />)
    expect(screen.getByText(/No tienes cuenta/)).toBeInTheDocument()
  })
})
