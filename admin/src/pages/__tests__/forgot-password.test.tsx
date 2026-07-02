import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { ForgotPasswordPage } from "../forgot-password"

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
  localStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("ForgotPasswordPage", () => {
  it("renders the email form", () => {
    renderWithProviders(<ForgotPasswordPage />)
    expect(screen.getByText("Recuperar contraseña")).toBeInTheDocument()
    expect(screen.getByLabelText("Email")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Enviar enlace" })).toBeInTheDocument()
  })

  it("has a link back to login", () => {
    renderWithProviders(<ForgotPasswordPage />)
    expect(screen.getByText("Volver a iniciar sesión").closest("a")).toHaveAttribute("href", "/login")
  })

  it("does not call the API when email is empty", async () => {
    const user = userEvent.setup()
    renderWithProviders(<ForgotPasswordPage />)
    await user.click(screen.getByRole("button", { name: "Enviar enlace" }))
    expect(mockFetch).not.toHaveBeenCalled()
    expect(screen.getByText("Ingresa tu email")).toBeInTheDocument()
  })

  it("posts to /auth/forgot-password and shows a neutral success message", async () => {
    const user = userEvent.setup()
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ message: "ok" }),
    })

    renderWithProviders(<ForgotPasswordPage />)
    await user.type(screen.getByLabelText("Email"), "user@test.com")
    await user.click(screen.getByRole("button", { name: "Enviar enlace" }))

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/auth/forgot-password"),
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ email: "user@test.com" }),
        })
      )
    })
    // Neutral confirmation (does not reveal whether the email exists).
    expect(await screen.findByText(/Si el email está registrado/)).toBeInTheDocument()
  })

  it("shows an error banner when the request fails", async () => {
    const user = userEvent.setup()
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 500,
      json: () => Promise.resolve({ error: "boom" }),
    })

    renderWithProviders(<ForgotPasswordPage />)
    await user.type(screen.getByLabelText("Email"), "user@test.com")
    await user.click(screen.getByRole("button", { name: "Enviar enlace" }))

    expect(await screen.findByText(/No pudimos procesar la solicitud/)).toBeInTheDocument()
  })
})
