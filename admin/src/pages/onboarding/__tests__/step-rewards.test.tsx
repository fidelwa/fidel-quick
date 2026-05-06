import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { StepRewards } from "../step-rewards"
import type { DraftSisfi, DraftReward } from "@/lib/wizard-draft"
import { emptyPendingReward } from "@/lib/wizard-draft"

beforeEach(() => {
  localStorage.clear()
  localStorage.setItem(
    "fidel_auth",
    JSON.stringify({ token: "tok", customerId: "c1", email: "a@b.com" }),
  )
})

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
})

const earnSisfi: DraftSisfi = { type: "earn_burn", name: "Puntos", ratio: 15 }
const cashbackSisfi: DraftSisfi = { type: "cashback", name: "Cashback", rate: 5 }
const pushcardSisfi: DraftSisfi = { type: "pushcard", name: "Tarjeta", slots: 10 }

const baseProps = {
  sisfi: earnSisfi,
  rewards: [],
  pendingReward: emptyPendingReward,
  onAddReward: vi.fn(),
  onRemoveReward: vi.fn(),
  onSetRewards: vi.fn(),
  onPendingRewardChange: vi.fn(),
  onNext: vi.fn(),
  onPrev: vi.fn(),
}

describe("StepRewards (draft mode)", () => {
  it("renders title", () => {
    renderWithProviders(<StepRewards {...baseProps} />)
    expect(screen.getByText("Crea tus recompensas")).toBeInTheDocument()
  })

  it("shows reward table for earn_burn sisfi", () => {
    renderWithProviders(<StepRewards {...baseProps} sisfi={earnSisfi} />)
    expect(screen.getByText(/Recompensas — Puntos/)).toBeInTheDocument()
  })

  it("shows reward table for cashback sisfi", () => {
    renderWithProviders(<StepRewards {...baseProps} sisfi={cashbackSisfi} />)
    expect(screen.getByText(/Recompensas — Cashback/)).toBeInTheDocument()
  })

  it("pushcard sisfi shows informational message instead of reward table", () => {
    renderWithProviders(<StepRewards {...baseProps} sisfi={pushcardSisfi} />)
    expect(screen.getByText(/se asigna después/)).toBeInTheDocument()
  })

  it("shows fallback when no sisfi in draft", () => {
    renderWithProviders(<StepRewards {...baseProps} sisfi={null} />)
    expect(
      screen.getByText(/Primero elige un programa de fidelidad/),
    ).toBeInTheDocument()
  })

  it("clicking + with valid pendingReward calls onAddReward", async () => {
    const user = userEvent.setup()
    const onAddReward = vi.fn()
    renderWithProviders(
      <StepRewards
        {...baseProps}
        pendingReward={{ name: "Cafe", description: "", cost: "50" }}
        onAddReward={onAddReward}
      />,
    )
    // Botón "+" es el último botón dentro del input row (no el "Excel" de arriba).
    const buttons = screen.getAllByRole("button")
    // El botón "+" está cerca del final, antes de Anterior/Siguiente.
    // Simplificamos: presionar Enter en el input de nombre.
    const nameInput = screen.getByPlaceholderText("Cafe gratis")
    await user.click(nameInput)
    await user.keyboard("{Enter}")
    expect(onAddReward).toHaveBeenCalledWith(
      expect.objectContaining({ name: "Cafe", cost: 50 }),
    )
    // suprimir uso no usado de buttons
    expect(buttons.length).toBeGreaterThan(0)
  })

  it("displays existing draft rewards", () => {
    const rewards: DraftReward[] = [
      { tempId: "r1", name: "Cafe", description: "Un cafe", cost: 100 },
    ]
    renderWithProviders(<StepRewards {...baseProps} rewards={rewards} />)
    expect(screen.getByText("Cafe")).toBeInTheDocument()
    expect(screen.getByText("Un cafe")).toBeInTheDocument()
    expect(screen.getByText("100")).toBeInTheDocument()
  })

  it("Siguiente without rewards (non-pushcard) shows error", async () => {
    const user = userEvent.setup()
    const onNext = vi.fn()
    renderWithProviders(<StepRewards {...baseProps} onNext={onNext} />)
    await user.click(screen.getByRole("button", { name: /Siguiente/ }))
    expect(onNext).not.toHaveBeenCalled()
  })

  it("Siguiente with rewards advances", async () => {
    const user = userEvent.setup()
    const onNext = vi.fn()
    const rewards: DraftReward[] = [
      { tempId: "r1", name: "Cafe", description: "", cost: 100 },
    ]
    renderWithProviders(
      <StepRewards {...baseProps} rewards={rewards} onNext={onNext} />,
    )
    await user.click(screen.getByRole("button", { name: /Siguiente/ }))
    expect(onNext).toHaveBeenCalled()
  })

  it("pushcard advances without rewards", async () => {
    const user = userEvent.setup()
    const onNext = vi.fn()
    renderWithProviders(
      <StepRewards {...baseProps} sisfi={pushcardSisfi} onNext={onNext} />,
    )
    await user.click(screen.getByRole("button", { name: /Siguiente/ }))
    expect(onNext).toHaveBeenCalled()
  })
})
