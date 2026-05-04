import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import { StepTransition } from "../step-transition"

describe("StepTransition", () => {
  it("renders children", () => {
    render(
      <StepTransition stepKey={1} direction="forward">
        <p>Step content</p>
      </StepTransition>
    )
    expect(screen.getByText("Step content")).toBeInTheDocument()
  })

  it("applies forward animation", () => {
    const { container } = render(
      <StepTransition stepKey={1} direction="forward">
        <p>Content</p>
      </StepTransition>
    )
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.style.animation).toContain("fade-slide-in-right")
  })

  it("applies backward animation", () => {
    const { container } = render(
      <StepTransition stepKey={1} direction="backward">
        <p>Content</p>
      </StepTransition>
    )
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.style.animation).toContain("fade-slide-in-left")
  })

  it("animation includes 300ms duration", () => {
    const { container } = render(
      <StepTransition stepKey={1} direction="forward">
        <p>Content</p>
      </StepTransition>
    )
    const wrapper = container.firstElementChild as HTMLElement
    expect(wrapper.style.animation).toContain("300ms")
  })
})
