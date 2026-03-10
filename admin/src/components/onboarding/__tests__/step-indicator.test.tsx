import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { StepIndicator } from "../step-indicator"

describe("StepIndicator", () => {
  it("renders all 4 step labels", () => {
    render(<StepIndicator currentStep={1} completedSteps={[]} />)
    expect(screen.getByText("Programa")).toBeInTheDocument()
    expect(screen.getByText("Recompensas")).toBeInTheDocument()
    expect(screen.getByText("Equipo")).toBeInTheDocument()
    expect(screen.getByText("Listo")).toBeInTheDocument()
  })

  it("shows step numbers for non-completed steps", () => {
    render(<StepIndicator currentStep={1} completedSteps={[]} />)
    expect(screen.getByText("1")).toBeInTheDocument()
    expect(screen.getByText("2")).toBeInTheDocument()
    expect(screen.getByText("3")).toBeInTheDocument()
    expect(screen.getByText("4")).toBeInTheDocument()
  })

  it("shows check icon for completed steps", () => {
    const { container } = render(
      <StepIndicator currentStep={3} completedSteps={[1, 2]} />
    )
    // Completed steps should not show their number
    expect(screen.queryByText("1")).not.toBeInTheDocument()
    expect(screen.queryByText("2")).not.toBeInTheDocument()
    // Current and future steps show numbers
    expect(screen.getByText("3")).toBeInTheDocument()
    expect(screen.getByText("4")).toBeInTheDocument()
    // Check icons rendered (SVG)
    const svgs = container.querySelectorAll("svg")
    expect(svgs.length).toBeGreaterThanOrEqual(2)
  })

  it("highlights current step", () => {
    render(
      <StepIndicator currentStep={2} completedSteps={[1]} />
    )
    // Step 2 circle (direct parent of text "2") should have ring classes
    const step2Text = screen.getByText("2")
    const circle = step2Text.closest("[class*='ring-4']")
    expect(circle).not.toBeNull()
  })

  it("renders with all steps completed", () => {
    render(<StepIndicator currentStep={4} completedSteps={[1, 2, 3]} />)
    // Steps 1-3 should be checks, step 4 should show "4"
    expect(screen.queryByText("1")).not.toBeInTheDocument()
    expect(screen.queryByText("2")).not.toBeInTheDocument()
    expect(screen.queryByText("3")).not.toBeInTheDocument()
    expect(screen.getByText("4")).toBeInTheDocument()
  })

  it("renders with empty completedSteps at step 1", () => {
    render(<StepIndicator currentStep={1} completedSteps={[]} />)
    const step1Text = screen.getByText("1")
    const circle = step1Text.closest("[class*='ring-4']")
    expect(circle).not.toBeNull()
  })
})
