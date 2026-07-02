import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { ResetPasswordPage } from "../reset-password"

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
  localStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("ResetPasswordPage", () => {
  it("shows an invalid-link state when there is no token", () => {
    renderWithProviders(<ResetPasswordPage />, { initialEntries: ["/reset-password"] })
    expect(screen.getByText("Enlace inválido")).toBeInTheDocument()
    expect(screen.getByText("Solicitar un nuevo enlace").closest("a")).toHaveAttribute(
      "href",
      "/forgot-password"
    )
    expect(mockFetch).not.toHaveBeenCalled()
  })

  it("renders the password form when a token is present", () => {
    renderWithProviders(<ResetPasswordPage />, { initialEntries: ["/reset-password?token=abc"] })
    expect(screen.getByLabelText("Nueva contraseña")).toBeInTheDocument()
    expect(screen.getByLabelText("Confirmar contraseña")).toBeInTheDocument()
    expect(screen.getByText("Elige una contraseña nueva para tu cuenta")).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Restablecer contraseña" })).toBeInTheDocument()
  })

  it("validates min length and password match (no API call)", async () => {
    const user = userEvent.setup()
    renderWithProviders(<ResetPasswordPage />, { initialEntries: ["/reset-password?token=abc"] })

    await user.type(screen.getByLabelText("Nueva contraseña"), "short")
    await user.type(screen.getByLabelText("Confirmar contraseña"), "different")
    await user.click(screen.getByRole("button", { name: "Restablecer contraseña" }))

    expect(await screen.findByText("Mínimo 8 caracteres")).toBeInTheDocument()
    expect(mockFetch).not.toHaveBeenCalled()

    // Fix length but keep a mismatch → should complain about mismatch.
    await user.clear(screen.getByLabelText("Nueva contraseña"))
    await user.type(screen.getByLabelText("Nueva contraseña"), "longenough1")
    await user.click(screen.getByRole("button", { name: "Restablecer contraseña" }))
    expect(await screen.findByText("Las contraseñas no coinciden")).toBeInTheDocument()
    expect(mockFetch).not.toHaveBeenCalled()
  })

  it("posts token + new_password on a valid submit", async () => {
    const user = userEvent.setup()
    mockFetch.mockResolvedValueOnce({
      ok: true,
      status: 200,
      json: () => Promise.resolve({ message: "ok" }),
    })

    renderWithProviders(<ResetPasswordPage />, { initialEntries: ["/reset-password?token=tok123"] })

    await user.type(screen.getByLabelText("Nueva contraseña"), "newpassword123")
    await user.type(screen.getByLabelText("Confirmar contraseña"), "newpassword123")
    await user.click(screen.getByRole("button", { name: "Restablecer contraseña" }))

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalledWith(
        expect.stringContaining("/auth/reset-password"),
        expect.objectContaining({
          method: "POST",
          body: JSON.stringify({ token: "tok123", new_password: "newpassword123" }),
        })
      )
    })
  })

  it("surfaces an expired/invalid token error from the API", async () => {
    const user = userEvent.setup()
    mockFetch.mockResolvedValueOnce({
      ok: false,
      status: 400,
      json: () => Promise.resolve({ error: "token inválido o expirado" }),
    })

    renderWithProviders(<ResetPasswordPage />, { initialEntries: ["/reset-password?token=old"] })

    await user.type(screen.getByLabelText("Nueva contraseña"), "newpassword123")
    await user.type(screen.getByLabelText("Confirmar contraseña"), "newpassword123")
    await user.click(screen.getByRole("button", { name: "Restablecer contraseña" }))

    expect(await screen.findByText(/inválido o ya expiró/)).toBeInTheDocument()
  })
})
