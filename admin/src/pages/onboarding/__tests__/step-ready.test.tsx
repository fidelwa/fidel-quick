import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/test-utils"
import { StepReady } from "../step-ready"
import type { Program, CashbackProgram, Reward, CashbackReward, Collaborator } from "@/types"

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
  localStorage.setItem(
    "fidel_auth",
    JSON.stringify({ token: "tok", customerId: "c1", email: "a@b.com" })
  )
  // Mock getCustomer response
  mockFetch.mockResolvedValue({
    ok: true,
    status: 200,
    json: () => Promise.resolve({
      id: "c1", name: "Mi Negocio", slug: "mi-negocio",
      onboarding_completed: false, active: true,
    }),
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
})

const earnBurnProgram: Program = {
  id: "p1", customer_id: "c1",
  name: "Puntos", points_ratio: 100, active: true,
}

const cashbackProgram: CashbackProgram = {
  id: "cb1", customer_id: "c1",
  name: "Cashback 5%", cashback_rate: 5, active: true,
}

const rewards: Reward[] = [
  { id: "r1", customer_id: "c1", customer_sisfi_id: "p1", name: "Cafe", description: "", points_cost: 100, active: true },
  { id: "r2", customer_id: "c1", customer_sisfi_id: "p1", name: "Postre", description: "", points_cost: 200, active: true },
]

const cashbackRewards: CashbackReward[] = [
  { id: "cr1", customer_id: "c1", customer_sisfi_id: "cb1", name: "Descuento", description: "", cost: 50, active: true },
]

const collaborators: Collaborator[] = [
  { id: "col1", customer_id: "c1", name: "Juan", phone: "+521", hash_id: "a", active: true },
  { id: "col2", customer_id: "c1", name: "Ana", phone: "+522", hash_id: "b", active: true },
]

const defaultProps = {
  earnBurnProgram,
  cashbackProgram,
  rewards,
  cashbackRewards,
  collaborators,
  onPrev: vi.fn(),
}

describe("StepReady", () => {
  it("renders completion title", () => {
    renderWithProviders(<StepReady {...defaultProps} />)
    expect(screen.getByText("¡Todo listo!")).toBeInTheDocument()
  })

  it("shows program badges", () => {
    renderWithProviders(<StepReady {...defaultProps} />)
    expect(screen.getByText("Puntos")).toBeInTheDocument()
    expect(screen.getByText("Cashback 5%")).toBeInTheDocument()
  })

  it("shows correct reward count", () => {
    renderWithProviders(<StepReady {...defaultProps} />)
    // 2 earn-burn + 1 cashback = 3
    expect(screen.getByText("3")).toBeInTheDocument()
    expect(screen.getByText("Recompensas")).toBeInTheDocument()
  })

  it("shows correct collaborator count", () => {
    renderWithProviders(<StepReady {...defaultProps} />)
    expect(screen.getByText("2")).toBeInTheDocument()
    expect(screen.getByText("Colaboradores")).toBeInTheDocument()
  })

  it("renders QR download button", () => {
    renderWithProviders(<StepReady {...defaultProps} />)
    expect(screen.getByRole("button", { name: /Descargar QR/ })).toBeInTheDocument()
  })

  it("renders copy URL button", () => {
    renderWithProviders(<StepReady {...defaultProps} />)
    expect(screen.getByRole("button", { name: /Copiar URL/ })).toBeInTheDocument()
  })

  it("renders finish button", () => {
    renderWithProviders(<StepReady {...defaultProps} />)
    expect(screen.getByRole("button", { name: "Ir al Dashboard" })).toBeInTheDocument()
  })

  it("renders previous button", () => {
    renderWithProviders(<StepReady {...defaultProps} />)
    expect(screen.getByRole("button", { name: "Anterior" })).toBeInTheDocument()
  })

  it("shows only earn-burn badge when no cashback", () => {
    renderWithProviders(
      <StepReady {...defaultProps} cashbackProgram={null} cashbackRewards={[]} />
    )
    expect(screen.getByText("Puntos")).toBeInTheDocument()
    expect(screen.queryByText("Cashback 5%")).not.toBeInTheDocument()
  })

  it("displays join URL", () => {
    renderWithProviders(<StepReady {...defaultProps} />)
    // URL should contain fidel.app/unirse
    expect(screen.getByText(/fidel\.app\/unirse/)).toBeInTheDocument()
  })
})
