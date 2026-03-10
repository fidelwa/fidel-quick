import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { RegistroPage } from "../registro"

const mockFetch = vi.fn()

beforeEach(() => {
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
    expect(screen.getByLabelText("Slug (URL)")).toBeInTheDocument()
    expect(screen.getByLabelText("Telefono del negocio")).toBeInTheDocument()
    expect(screen.getByLabelText("Email del administrador")).toBeInTheDocument()
    expect(screen.getByLabelText("Password")).toBeInTheDocument()
    expect(screen.getByLabelText("Confirmar password")).toBeInTheDocument()
  })

  it("renders description field as optional", () => {
    renderWithProviders(<RegistroPage />)
    expect(screen.getByLabelText("Descripcion (opcional)")).toBeInTheDocument()
  })

  it("auto-generates slug from name", async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegistroPage />)
    await user.type(screen.getByLabelText("Nombre del negocio"), "Mi Restaurante")

    await waitFor(() => {
      expect(screen.getByLabelText("Slug (URL)")).toHaveValue("mi-restaurante")
    })
  })

  it("auto-generates slug stripping accents and special chars", async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegistroPage />)
    await user.type(screen.getByLabelText("Nombre del negocio"), "Café Señor!")

    await waitFor(() => {
      const slugValue = (screen.getByLabelText("Slug (URL)") as HTMLInputElement).value
      expect(slugValue).toBe("cafe-senor")
    })
  })

  it("shows URL preview when slug has 3+ chars", async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegistroPage />)
    await user.type(screen.getByLabelText("Nombre del negocio"), "Test Place")

    await waitFor(() => {
      expect(screen.getByText(/fidel\.app\/unirse\//)).toBeInTheDocument()
    })
  })

  it("renders login link", () => {
    renderWithProviders(<RegistroPage />)
    expect(screen.getByText("Inicia sesion")).toBeInTheDocument()
    expect(screen.getByText("Inicia sesion").closest("a")).toHaveAttribute("href", "/login")
  })

  it("has submit button", () => {
    renderWithProviders(<RegistroPage />)
    expect(screen.getByRole("button", { name: "Crear cuenta" })).toBeInTheDocument()
  })

  it("shows validation errors on empty submit", async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegistroPage />)
    await user.click(screen.getByRole("button", { name: "Crear cuenta" }))

    await waitFor(() => {
      expect(screen.getByText("El nombre es requerido")).toBeInTheDocument()
    })
  })

  it("shows password mismatch error", async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegistroPage />)

    await user.type(screen.getByLabelText("Nombre del negocio"), "Test")
    await user.type(screen.getByLabelText("Telefono del negocio"), "+5255123456789")
    await user.type(screen.getByLabelText("Email del administrador"), "a@b.com")
    await user.type(screen.getByLabelText("Password"), "password1")
    await user.type(screen.getByLabelText("Confirmar password"), "password2")
    await user.click(screen.getByRole("button", { name: "Crear cuenta" }))

    await waitFor(() => {
      expect(screen.getByText("Las contraseñas no coinciden")).toBeInTheDocument()
    })
  })

  it("shows slug format error for invalid chars", async () => {
    const user = userEvent.setup()
    renderWithProviders(<RegistroPage />)

    // Manually edit slug to invalid value
    const slugInput = screen.getByLabelText("Slug (URL)")
    await user.clear(slugInput)
    await user.type(slugInput, "INVALID SLUG!!")

    await user.type(screen.getByLabelText("Nombre del negocio"), " ") // trigger to avoid required
    await user.type(screen.getByLabelText("Telefono del negocio"), "+5255123456789")
    await user.type(screen.getByLabelText("Email del administrador"), "a@b.com")
    await user.type(screen.getByLabelText("Password"), "password1")
    await user.type(screen.getByLabelText("Confirmar password"), "password1")
    await user.click(screen.getByRole("button", { name: "Crear cuenta" }))

    await waitFor(() => {
      expect(screen.getByText("Solo letras minusculas, numeros y guiones")).toBeInTheDocument()
    })
  })
})
