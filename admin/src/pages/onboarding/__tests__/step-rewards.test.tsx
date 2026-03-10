import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/test-utils"
import { StepRewards } from "../step-rewards"
import type { Program, CashbackProgram, Reward, CashbackReward } from "@/types"

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

const earnBurnProgram: Program = {
  id: "p1", customer_id: "c1", type: "earn-burn",
  name: "Puntos", points_ratio: 100, active: true,
}

const cashbackProgram: CashbackProgram = {
  id: "cb1", customer_id: "c1", type: "cashback",
  name: "Cashback", cashback_rate: 5, active: true,
}

const mockReward: Reward = {
  id: "r1", customer_id: "c1", program_id: "p1",
  name: "Cafe gratis", description: "Un cafe", points_cost: 100, active: true,
}

const mockCbReward: CashbackReward = {
  id: "cr1", customer_id: "c1", program_id: "cb1",
  name: "Descuento", description: "10% off", cost: 50, active: true,
}

const defaultProps = {
  earnBurnProgram,
  cashbackProgram: null as CashbackProgram | null,
  rewards: [] as Reward[],
  cashbackRewards: [] as CashbackReward[],
  onRewardsChange: vi.fn(),
  onCashbackRewardsChange: vi.fn(),
  onNext: vi.fn(),
  onPrev: vi.fn(),
}

describe("StepRewards", () => {
  it("renders title", () => {
    renderWithProviders(<StepRewards {...defaultProps} />)
    expect(screen.getByText("Crea tus recompensas")).toBeInTheDocument()
  })

  it("shows earn-burn section when program exists", () => {
    renderWithProviders(<StepRewards {...defaultProps} />)
    expect(screen.getByText(/Recompensas de Puntos/)).toBeInTheDocument()
  })

  it("shows cashback section when cashback program exists", () => {
    renderWithProviders(
      <StepRewards {...defaultProps} cashbackProgram={cashbackProgram} />
    )
    expect(screen.getByText(/Beneficios de Cashback/)).toBeInTheDocument()
  })

  it("does not show cashback section when no cashback program", () => {
    renderWithProviders(<StepRewards {...defaultProps} />)
    expect(screen.queryByText(/Beneficios de Cashback/)).not.toBeInTheDocument()
  })

  it("shows empty state for earn-burn when no rewards", () => {
    renderWithProviders(<StepRewards {...defaultProps} />)
    expect(screen.getByText("Agrega tu primera recompensa de puntos")).toBeInTheDocument()
  })

  it("shows empty state for cashback when no rewards", () => {
    renderWithProviders(
      <StepRewards {...defaultProps} cashbackProgram={cashbackProgram} />
    )
    expect(screen.getByText("Agrega tu primer beneficio de cashback")).toBeInTheDocument()
  })

  it("displays existing rewards with Creado badge", () => {
    renderWithProviders(
      <StepRewards {...defaultProps} rewards={[mockReward]} />
    )
    expect(screen.getByText("Cafe gratis")).toBeInTheDocument()
    expect(screen.getByText("Un cafe")).toBeInTheDocument()
    expect(screen.getByText("100 pts")).toBeInTheDocument()
    expect(screen.getByText("Creado")).toBeInTheDocument()
  })

  it("displays existing cashback rewards with Creado badge", () => {
    renderWithProviders(
      <StepRewards
        {...defaultProps}
        cashbackProgram={cashbackProgram}
        cashbackRewards={[mockCbReward]}
      />
    )
    expect(screen.getByText("Descuento")).toBeInTheDocument()
    expect(screen.getByText("$50")).toBeInTheDocument()
  })

  it("has navigation buttons", () => {
    renderWithProviders(<StepRewards {...defaultProps} />)
    expect(screen.getByRole("button", { name: "Anterior" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Siguiente" })).toBeInTheDocument()
  })

  it("renders inline form fields for earn-burn rewards", () => {
    renderWithProviders(<StepRewards {...defaultProps} />)
    const nameLabels = screen.getAllByText("Nombre")
    expect(nameLabels.length).toBeGreaterThanOrEqual(1)
  })

  it("shows both sections with both programs", () => {
    renderWithProviders(
      <StepRewards {...defaultProps} cashbackProgram={cashbackProgram} />
    )
    expect(screen.getByText(/Recompensas de Puntos/)).toBeInTheDocument()
    expect(screen.getByText(/Beneficios de Cashback/)).toBeInTheDocument()
  })
})
