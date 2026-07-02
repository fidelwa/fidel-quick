import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { RegistroPage } from "../registro"

const mockFetch = vi.fn()

// checkPhoneExists() -> request() -> fetch(). Devolvemos un Response-like con
// { exists } para que el check de teléfono resuelva a "available"/"exists".
function phoneCheckResponse(exists: boolean) {
  return {
    ok: true,
    status: 200,
    json: async () => ({ exists }),
  }
}

beforeEach(() => {
  mockFetch.mockReset()
  mockFetch.mockResolvedValue(phoneCheckResponse(false))
  vi.stubGlobal("fetch", mockFetch)
  localStorage.clear()
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe("RegistroPage", () => {
  it("renders registration form", () => {
    renderWithProviders(<RegistroPage />)
    expect(screen.getByText("Crea tu programa de fidelidad")).toBeInTheDocument()
    expect(screen.getByLabelText("Nombre del negocio")).toBeInTheDocument()
    expect(screen.getByText("Telefono del negocio")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("5512345678")).toBeInTheDocument()
  })

  it("renders description field as optional", () => {
    renderWithProviders(<RegistroPage />)
    expect(screen.getByLabelText("Descripcion (opcional)")).toBeInTheDocument()
  })

  it("renders login link", () => {
    renderWithProviders(<RegistroPage />)
    expect(screen.getByText("Inicia sesion")).toBeInTheDocument()
    expect(screen.getByText("Inicia sesion").closest("a")).toHaveAttribute("href", "/login")
  })

  it("has submit button", () => {
    renderWithProviders(<RegistroPage />)
    expect(screen.getByRole("button", { name: "Continuar" })).toBeInTheDocument()
  })

  it("disables submit until a phone number is verified", () => {
    renderWithProviders(<RegistroPage />)
    expect(screen.getByRole("button", { name: "Continuar" })).toBeDisabled()
  })

  it("checks phone availability and enables submit when available", async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegistroPage />)

    await user.type(screen.getByLabelText("Nombre del negocio"), "Test")
    await user.type(screen.getByPlaceholderText("5512345678"), "5512345678")

    await waitFor(() => {
      expect(mockFetch).toHaveBeenCalled()
    })
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Continuar" })).toBeEnabled()
    })
  })

  it("warns and requires confirmation when phone already exists", async () => {
    mockFetch.mockResolvedValue(phoneCheckResponse(true))
    const user = userEvent.setup()
    renderWithProviders(<RegistroPage />)

    await user.type(screen.getByPlaceholderText("5512345678"), "5512345678")

    await waitFor(() => {
      expect(screen.getByText("Este telefono ya esta registrado.")).toBeInTheDocument()
    })
    // Aún bloqueado hasta marcar la casilla de confirmación.
    expect(screen.getByRole("button", { name: "Continuar" })).toBeDisabled()

    await user.click(screen.getByRole("checkbox"))
    await waitFor(() => {
      expect(screen.getByRole("button", { name: "Continuar" })).toBeEnabled()
    })
  })
})
