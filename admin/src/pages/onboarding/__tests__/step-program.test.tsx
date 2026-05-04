import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { StepProgram } from "../step-program"

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
  localStorage.setItem(
    "fidel_auth",
    JSON.stringify({ token: "tok", customerId: "c1", email: "a@b.com" })
  )
})

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
})

const defaultProps = {
  earnBurnProgram: null,
  cashbackProgram: null,
  onEarnBurnCreated: vi.fn(),
  onCashbackCreated: vi.fn(),
  onNext: vi.fn(),
}

describe("StepProgram", () => {
  it("renders title and description", () => {
    renderWithProviders(<StepProgram {...defaultProps} />)
    expect(screen.getByText("Elige tu programa de fidelidad")).toBeInTheDocument()
    expect(screen.getByText(/Selecciona el tipo de programa/)).toBeInTheDocument()
  })

  it("renders both program type cards", () => {
    renderWithProviders(<StepProgram {...defaultProps} />)
    expect(screen.getByText("Puntos")).toBeInTheDocument()
    expect(screen.getByText("Cashback")).toBeInTheDocument()
  })

  it("renders descriptions for program types", () => {
    renderWithProviders(<StepProgram {...defaultProps} />)
    expect(screen.getByText("Acumula y canjea puntos")).toBeInTheDocument()
    expect(screen.getByText("Porcentaje de devolucion")).toBeInTheDocument()
  })

  it("shows config fields when Puntos is selected", async () => {
    const user = userEvent.setup()
    renderWithProviders(<StepProgram {...defaultProps} />)
    await user.click(screen.getByText("Puntos"))
    expect(screen.getByText("Configurar programa de puntos")).toBeInTheDocument()
    // Both program configs have "Nombre del programa", use getAllByLabelText
    const nameInputs = screen.getAllByLabelText("Nombre del programa")
    expect(nameInputs.length).toBeGreaterThanOrEqual(1)
    expect(screen.getByLabelText("1 punto por cada $")).toBeInTheDocument()
  })

  it("shows config fields when Cashback is selected", async () => {
    const user = userEvent.setup()
    renderWithProviders(<StepProgram {...defaultProps} />)
    await user.click(screen.getByText("Cashback"))
    expect(screen.getByText("Configurar programa de cashback")).toBeInTheDocument()
  })

  it("has Siguiente button", () => {
    renderWithProviders(<StepProgram {...defaultProps} />)
    expect(screen.getByRole("button", { name: /Siguiente/ })).toBeInTheDocument()
  })

  it("shows 'Creado' badge when earn-burn program exists", () => {
    renderWithProviders(
      <StepProgram
        {...defaultProps}
        earnBurnProgram={{
          id: "p1", customer_id: "c1",
          name: "Programa de puntos", points_ratio: 100, active: true,
        }}
      />
    )
    expect(screen.getByText("Creado")).toBeInTheDocument()
  })

  it("shows 'Creado' badge when cashback program exists", () => {
    renderWithProviders(
      <StepProgram
        {...defaultProps}
        cashbackProgram={{
          id: "cb1", customer_id: "c1",
          name: "Cashback", cashback_rate: 5, active: true,
        }}
      />
    )
    // There should be at least one "Creado" text
    const creados = screen.getAllByText("Creado")
    expect(creados.length).toBeGreaterThanOrEqual(1)
  })

  it("pre-fills config when program already exists", () => {
    renderWithProviders(
      <StepProgram
        {...defaultProps}
        earnBurnProgram={{
          id: "p1", customer_id: "c1",
          name: "Mi Programa", points_ratio: 50, active: true,
        }}
      />
    )
    // Config should be visible and fields disabled
    const nameInputs = screen.getAllByLabelText("Nombre del programa")
    expect(nameInputs[0]).toHaveValue("Mi Programa")
    expect(nameInputs[0]).toBeDisabled()
  })
})
