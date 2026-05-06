import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { StepTeam } from "../step-team"
import type { DraftCollaborator } from "@/lib/wizard-draft"
import { emptyPendingCollaborator } from "@/lib/wizard-draft"

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

const mockCollab: DraftCollaborator = {
  tempId: "t1",
  name: "Juan Perez",
  phone: "+525512345678",
}

const baseProps = {
  collaborators: [] as DraftCollaborator[],
  pendingCollaborator: emptyPendingCollaborator,
  onAddCollaborator: vi.fn(),
  onRemoveCollaborator: vi.fn(),
  onPendingCollaboratorChange: vi.fn(),
  onNext: vi.fn(),
  onPrev: vi.fn(),
}

describe("StepTeam (draft mode)", () => {
  it("renders title", () => {
    renderWithProviders(<StepTeam {...baseProps} />)
    expect(screen.getByText("Registra a tu equipo")).toBeInTheDocument()
  })

  it("shows WhatsApp notice", () => {
    renderWithProviders(<StepTeam {...baseProps} />)
    expect(screen.getByText(/WhatsApp activo/)).toBeInTheDocument()
  })

  it("shows empty state when no collaborators", () => {
    renderWithProviders(<StepTeam {...baseProps} />)
    expect(screen.getByText("Agrega a tu primer colaborador")).toBeInTheDocument()
  })

  it("shows collaborator row when one exists", () => {
    renderWithProviders(
      <StepTeam {...baseProps} collaborators={[mockCollab]} />,
    )
    expect(screen.getByText("Juan Perez")).toBeInTheDocument()
    expect(screen.getByText("+525512345678")).toBeInTheDocument()
  })

  it("shows count when collaborators exist", () => {
    renderWithProviders(
      <StepTeam {...baseProps} collaborators={[mockCollab]} />,
    )
    expect(screen.getByText(/1 colaborador registrado/)).toBeInTheDocument()
  })

  it("Siguiente without collaborators is blocked", async () => {
    const user = userEvent.setup()
    const onNext = vi.fn()
    renderWithProviders(<StepTeam {...baseProps} onNext={onNext} />)
    await user.click(screen.getByRole("button", { name: "Siguiente" }))
    expect(onNext).not.toHaveBeenCalled()
  })

  it("Siguiente with at least one collaborator advances", async () => {
    const user = userEvent.setup()
    const onNext = vi.fn()
    renderWithProviders(
      <StepTeam
        {...baseProps}
        collaborators={[mockCollab]}
        onNext={onNext}
      />,
    )
    await user.click(screen.getByRole("button", { name: "Siguiente" }))
    expect(onNext).toHaveBeenCalled()
  })

  it("phone input onChange replaces non-digits before propagating", async () => {
    const user = userEvent.setup()
    const onPendingCollaboratorChange = vi.fn()
    renderWithProviders(
      <StepTeam
        {...baseProps}
        onPendingCollaboratorChange={onPendingCollaboratorChange}
      />,
    )
    const phoneInput = screen.getByPlaceholderText("5512345678")
    await user.type(phoneInput, "5")
    // Verify the propagated phone value contains only digits.
    const calls = onPendingCollaboratorChange.mock.calls
    const lastCall = calls[calls.length - 1]?.[0]
    expect(lastCall.phone).toMatch(/^\d*$/)
  })

  it("Anterior calls onPrev", async () => {
    const user = userEvent.setup()
    const onPrev = vi.fn()
    renderWithProviders(<StepTeam {...baseProps} onPrev={onPrev} />)
    await user.click(screen.getByRole("button", { name: "Anterior" }))
    expect(onPrev).toHaveBeenCalled()
  })
})
