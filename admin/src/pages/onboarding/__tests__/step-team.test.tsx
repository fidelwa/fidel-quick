import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { StepTeam } from "../step-team"
import type { Collaborator } from "@/types"

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

const mockCollaborator: Collaborator = {
  id: "col1", customer_id: "c1", name: "Juan Perez",
  phone: "+525512345678", hash_id: "abc", active: true,
}

const defaultProps = {
  collaborators: [] as Collaborator[],
  onCollaboratorsChange: vi.fn(),
  onNext: vi.fn(),
  onPrev: vi.fn(),
}

describe("StepTeam", () => {
  it("renders title", () => {
    renderWithProviders(<StepTeam {...defaultProps} />)
    expect(screen.getByText("Registra a tu equipo")).toBeInTheDocument()
  })

  it("shows WhatsApp notice", () => {
    renderWithProviders(<StepTeam {...defaultProps} />)
    expect(screen.getByText(/WhatsApp activo/)).toBeInTheDocument()
  })

  it("shows empty state when no collaborators", () => {
    renderWithProviders(<StepTeam {...defaultProps} />)
    expect(screen.getByText("Agrega a tu primer colaborador")).toBeInTheDocument()
  })

  it("shows collaborator in table when collaborators exist", () => {
    renderWithProviders(
      <StepTeam {...defaultProps} collaborators={[mockCollaborator]} />
    )
    expect(screen.getByText("Juan Perez")).toBeInTheDocument()
    expect(screen.getByText("+525512345678")).toBeInTheDocument()
  })

  it("shows multiple collaborators", () => {
    const collab2: Collaborator = {
      id: "col2", customer_id: "c1", name: "Maria Lopez",
      phone: "+525598765432", hash_id: "def", active: true,
    }
    renderWithProviders(
      <StepTeam {...defaultProps} collaborators={[mockCollaborator, collab2]} />
    )
    expect(screen.getByText("Juan Perez")).toBeInTheDocument()
    expect(screen.getByText("Maria Lopez")).toBeInTheDocument()
  })

  it("shows collaborator count", () => {
    renderWithProviders(
      <StepTeam {...defaultProps} collaborators={[mockCollaborator]} />
    )
    expect(screen.getByText(/1 colaborador registrado/)).toBeInTheDocument()
  })

  it("has country code selector", () => {
    renderWithProviders(<StepTeam {...defaultProps} />)
    const select = screen.getByDisplayValue("MX (+52)")
    expect(select).toBeInTheDocument()
  })

  it("has input fields for name and phone", () => {
    renderWithProviders(<StepTeam {...defaultProps} />)
    expect(screen.getByPlaceholderText("Juan Perez")).toBeInTheDocument()
    expect(screen.getByPlaceholderText("5512345678")).toBeInTheDocument()
  })

  it("has navigation buttons", () => {
    renderWithProviders(<StepTeam {...defaultProps} />)
    expect(screen.getByRole("button", { name: "Anterior" })).toBeInTheDocument()
    expect(screen.getByRole("button", { name: "Siguiente" })).toBeInTheDocument()
  })

  it("has add button", () => {
    renderWithProviders(<StepTeam {...defaultProps} />)
    const buttons = screen.getAllByRole("button")
    expect(buttons.length).toBeGreaterThanOrEqual(3) // prev + next + add
  })

  it("can type in name and phone fields", async () => {
    const user = userEvent.setup()
    renderWithProviders(<StepTeam {...defaultProps} />)

    const nameInput = screen.getByPlaceholderText("Juan Perez")
    const phoneInput = screen.getByPlaceholderText("5512345678")

    await user.type(nameInput, "Ana Garcia")
    await user.type(phoneInput, "5511112222")

    expect(nameInput).toHaveValue("Ana Garcia")
    expect(phoneInput).toHaveValue("5511112222")
  })

  it("strips non-digit characters from phone input", async () => {
    const user = userEvent.setup()
    renderWithProviders(<StepTeam {...defaultProps} />)

    const phoneInput = screen.getByPlaceholderText("5512345678")
    await user.type(phoneInput, "55-1234-5678")

    expect(phoneInput).toHaveValue("5512345678")
  })
})
