import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { screen } from "@testing-library/react"
import { renderWithProviders } from "@/test/test-utils"
import { StepReady } from "../step-ready"
import type {
  DraftSisfi,
  DraftReward,
  DraftCollaborator,
} from "@/lib/wizard-draft"

const mockFetch = vi.fn()

beforeEach(() => {
  vi.stubGlobal("fetch", mockFetch)
  localStorage.setItem(
    "fidel_auth",
    JSON.stringify({ token: "tok", customerId: "c1", email: "a@b.com" }),
  )
  mockFetch.mockResolvedValue({
    ok: true,
    status: 200,
    json: () =>
      Promise.resolve({
        id: "c1",
        name: "Mi Negocio",
        slug: "mi-negocio",
        onboarding_completed: false,
        active: true,
      }),
  })
})

afterEach(() => {
  vi.restoreAllMocks()
  localStorage.clear()
})

const earnSisfi: DraftSisfi = { type: "earn_burn", name: "Puntos", ratio: 15 }
const cashbackSisfi: DraftSisfi = { type: "cashback", name: "Cashback 5%", rate: 5 }
const pushcardSisfi: DraftSisfi = { type: "pushcard", name: "Sellos", slots: 10 }

const rewards: DraftReward[] = [
  { tempId: "r1", name: "Cafe", description: "", cost: 100 },
  { tempId: "r2", name: "Postre", description: "", cost: 200 },
]

const collaborators: DraftCollaborator[] = [
  { tempId: "t1", name: "Juan", phone: "+525512345678" },
]

const baseProps = {
  sisfi: earnSisfi,
  rewards,
  collaborators,
  onPrev: vi.fn(),
}

describe("StepReady (draft mode)", () => {
  it("renders header", () => {
    renderWithProviders(<StepReady {...baseProps} />)
    expect(screen.getByText("¡Todo listo!")).toBeInTheDocument()
  })

  it("shows the sisfi badge for earn_burn", () => {
    renderWithProviders(<StepReady {...baseProps} sisfi={earnSisfi} />)
    expect(screen.getByText("Puntos")).toBeInTheDocument()
  })

  it("shows the sisfi badge for cashback", () => {
    renderWithProviders(<StepReady {...baseProps} sisfi={cashbackSisfi} />)
    expect(screen.getByText("Cashback 5%")).toBeInTheDocument()
  })

  it("shows the sisfi badge for pushcard", () => {
    renderWithProviders(<StepReady {...baseProps} sisfi={pushcardSisfi} />)
    expect(screen.getByText("Sellos")).toBeInTheDocument()
  })

  it("counts rewards and collaborators in summary", () => {
    renderWithProviders(<StepReady {...baseProps} />)
    expect(screen.getByText("2")).toBeInTheDocument() // rewards
    expect(screen.getByText("1")).toBeInTheDocument() // collaborators
  })

  it("shows the Crear programa button", () => {
    renderWithProviders(<StepReady {...baseProps} />)
    expect(
      screen.getByRole("button", { name: /Crear programa/ }),
    ).toBeInTheDocument()
  })

  it("Anterior button is present", () => {
    renderWithProviders(<StepReady {...baseProps} />)
    expect(screen.getByRole("button", { name: "Anterior" })).toBeInTheDocument()
  })
})
