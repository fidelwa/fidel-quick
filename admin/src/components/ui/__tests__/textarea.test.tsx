import { describe, it, expect } from "vitest"
import { render, screen } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { Textarea } from "../textarea"

describe("Textarea", () => {
  it("renders a textarea element", () => {
    render(<Textarea data-testid="ta" />)
    expect(screen.getByTestId("ta").tagName).toBe("TEXTAREA")
  })

  it("accepts and displays typed text", async () => {
    const user = userEvent.setup()
    render(<Textarea data-testid="ta" />)
    const textarea = screen.getByTestId("ta")
    await user.type(textarea, "Hello world")
    expect(textarea).toHaveValue("Hello world")
  })

  it("applies placeholder", () => {
    render(<Textarea placeholder="Enter text..." />)
    expect(screen.getByPlaceholderText("Enter text...")).toBeInTheDocument()
  })

  it("applies custom className", () => {
    render(<Textarea data-testid="ta" className="custom-class" />)
    expect(screen.getByTestId("ta").className).toContain("custom-class")
  })

  it("can be disabled", () => {
    render(<Textarea data-testid="ta" disabled />)
    expect(screen.getByTestId("ta")).toBeDisabled()
  })

  it("sets rows attribute", () => {
    render(<Textarea data-testid="ta" rows={5} />)
    expect(screen.getByTestId("ta")).toHaveAttribute("rows", "5")
  })

  it("has data-slot attribute", () => {
    render(<Textarea data-testid="ta" />)
    expect(screen.getByTestId("ta")).toHaveAttribute("data-slot", "textarea")
  })
})
