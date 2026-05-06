import { describe, it, expect, vi, beforeEach, afterEach } from "vitest"
import { useState } from "react"
import { screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { renderWithProviders } from "@/test/test-utils"
import { StepProgram } from "../step-program"
import type { DraftSisfi, PendingProgramForm } from "@/lib/wizard-draft"
import { emptyPendingProgramForm } from "@/lib/wizard-draft"

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

// Wrapper que mantiene el form en estado real para que las interacciones
// (clicks, typing) se reflejen en el render.
function Wrapper(props: {
  sisfi?: DraftSisfi | null
  initialForm?: PendingProgramForm
  onSisfiChange?: (sisfi: DraftSisfi | null) => void
  onNext?: () => void
}) {
  const [form, setForm] = useState<PendingProgramForm>(
    props.initialForm ?? emptyPendingProgramForm,
  )
  return (
    <StepProgram
      sisfi={props.sisfi ?? null}
      pendingProgramForm={form}
      onSisfiChange={props.onSisfiChange ?? vi.fn()}
      onPendingProgramFormChange={setForm}
      onNext={props.onNext ?? vi.fn()}
    />
  )
}

describe("StepProgram (draft mode)", () => {
  it("renders title and description", () => {
    renderWithProviders(<Wrapper />)
    expect(screen.getByText("Elige tu programa de fidelidad")).toBeInTheDocument()
    expect(screen.getByText(/Selecciona el tipo de programa/)).toBeInTheDocument()
  })

  it("renders all three program type cards", () => {
    renderWithProviders(<Wrapper />)
    expect(screen.getByText("Puntos")).toBeInTheDocument()
    expect(screen.getByText("Cashback")).toBeInTheDocument()
    expect(screen.getByText("Tarjeta de sellos")).toBeInTheDocument()
  })

  it("shows config fields when Puntos is selected", async () => {
    const user = userEvent.setup()
    renderWithProviders(<Wrapper />)
    await user.click(screen.getByText("Puntos"))
    expect(screen.getByText("Configurar programa de puntos")).toBeInTheDocument()
  })

  it("shows config fields when Cashback is selected", async () => {
    const user = userEvent.setup()
    renderWithProviders(<Wrapper />)
    await user.click(screen.getByText("Cashback"))
    expect(screen.getByText("Configurar programa de cashback")).toBeInTheDocument()
  })

  it("shows config fields when Tarjeta de sellos is selected", async () => {
    const user = userEvent.setup()
    renderWithProviders(<Wrapper />)
    await user.click(screen.getByText("Tarjeta de sellos"))
    expect(screen.getByText("Configurar tarjeta de sellos")).toBeInTheDocument()
  })

  it("Siguiente without selection blocks navigation", async () => {
    const user = userEvent.setup()
    const onNext = vi.fn()
    renderWithProviders(<Wrapper onNext={onNext} />)
    await user.click(screen.getByRole("button", { name: /Siguiente/ }))
    expect(onNext).not.toHaveBeenCalled()
  })

  it("Siguiente with valid Puntos config calls onSisfiChange + onNext", async () => {
    const user = userEvent.setup()
    const onSisfiChange = vi.fn()
    const onNext = vi.fn()
    renderWithProviders(
      <Wrapper onSisfiChange={onSisfiChange} onNext={onNext} />,
    )
    await user.click(screen.getByText("Puntos"))
    const earnNameInput = document.getElementById("earn-name") as HTMLInputElement
    await user.type(earnNameInput, "Mi Programa")
    await user.click(screen.getByRole("button", { name: /Siguiente/ }))
    expect(onSisfiChange).toHaveBeenCalledWith(
      expect.objectContaining({ type: "earn_burn", name: "Mi Programa" }),
    )
    expect(onNext).toHaveBeenCalled()
  })

  it("Siguiente with valid Pushcard config calls onSisfiChange + onNext", async () => {
    const user = userEvent.setup()
    const onSisfiChange = vi.fn()
    const onNext = vi.fn()
    renderWithProviders(
      <Wrapper onSisfiChange={onSisfiChange} onNext={onNext} />,
    )
    await user.click(screen.getByText("Tarjeta de sellos"))
    await user.click(screen.getByRole("button", { name: /Siguiente/ }))
    expect(onSisfiChange).toHaveBeenCalledWith(
      expect.objectContaining({ type: "pushcard", slots: 10 }),
    )
    expect(onNext).toHaveBeenCalled()
  })

  it("pre-fills config from pendingProgramForm", () => {
    const initialForm: PendingProgramForm = {
      ...emptyPendingProgramForm,
      selected: "earn_burn",
      earnName: "Mi Programa",
      earnRatio: "50",
    }
    renderWithProviders(<Wrapper initialForm={initialForm} />)
    const earnNameInput = document.getElementById("earn-name") as HTMLInputElement
    expect(earnNameInput).toHaveValue("Mi Programa")
    expect(earnNameInput).not.toBeDisabled()
  })
})
