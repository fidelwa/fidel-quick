import { describe, it, expect } from "vitest"
import { cn, formatDate, formatDateTime, formatCurrency, formatPoints } from "../utils"

describe("cn", () => {
  it("merges class names", () => {
    expect(cn("px-2", "py-1")).toBe("px-2 py-1")
  })

  it("handles conditional classes", () => {
    expect(cn("base", false && "hidden", "extra")).toBe("base extra")
  })

  it("resolves tailwind conflicts (last wins)", () => {
    expect(cn("px-2", "px-4")).toBe("px-4")
  })

  it("handles undefined and null", () => {
    expect(cn("base", undefined, null, "end")).toBe("base end")
  })

  it("returns empty string for no args", () => {
    expect(cn()).toBe("")
  })
})

describe("formatDate", () => {
  it("formats date in es-MX locale with year", () => {
    const result = formatDate("2024-06-15T12:00:00Z")
    expect(result).toContain("2024")
    expect(result).toMatch(/jun/i)
  })

  it("handles ISO date strings", () => {
    const result = formatDate("2024-01-15T12:00:00Z")
    expect(result).toContain("2024")
  })
})

describe("formatDateTime", () => {
  it("includes time component", () => {
    const result = formatDateTime("2024-06-15T14:30:00Z")
    expect(result).toContain("2024")
    expect(result).toContain("15")
  })
})

describe("formatCurrency", () => {
  it("formats as MXN currency", () => {
    const result = formatCurrency(1500)
    expect(result).toContain("1,500")
    expect(result).toContain("$")
  })

  it("handles zero", () => {
    const result = formatCurrency(0)
    expect(result).toContain("0")
    expect(result).toContain("$")
  })

  it("handles decimals", () => {
    const result = formatCurrency(99.99)
    expect(result).toContain("99.99")
  })

  it("handles negative amounts", () => {
    const result = formatCurrency(-50)
    expect(result).toContain("50")
  })
})

describe("formatPoints", () => {
  it("formats with es-MX separators", () => {
    expect(formatPoints(1000)).toContain("1,000")
  })

  it("formats zero", () => {
    expect(formatPoints(0)).toBe("0")
  })

  it("formats large numbers", () => {
    expect(formatPoints(1000000)).toContain("1,000,000")
  })
})
