import { describe, it, expect, vi } from "vitest"
import { render, screen, fireEvent } from "@testing-library/react"
import { MemoryRouter } from "react-router-dom"
import { AuthProvider } from "@/context/auth-context"
import { Sidebar } from "../sidebar"
import type { ReactNode } from "react"

function wrapper({ children }: { children: ReactNode }) {
  return (
    <AuthProvider>
      <MemoryRouter>{children}</MemoryRouter>
    </AuthProvider>
  )
}

describe("Sidebar", () => {
  it("renders the brand name", () => {
    render(<Sidebar />, { wrapper })
    expect(screen.getByText("Fidel Admin")).toBeInTheDocument()
  })

  it("renders all nav items", () => {
    render(<Sidebar />, { wrapper })
    expect(screen.getByText("Inicio")).toBeInTheDocument()
    expect(screen.getByText("Mi Negocio")).toBeInTheDocument()
    expect(screen.getByText("Programas")).toBeInTheDocument()
    expect(screen.getByText("Colaboradores")).toBeInTheDocument()
    expect(screen.getByText("Clientes")).toBeInTheDocument()
    expect(screen.getByText("Feedback")).toBeInTheDocument()
  })

  it("renders logout button", () => {
    render(<Sidebar />, { wrapper })
    expect(screen.getByText("Cerrar sesion")).toBeInTheDocument()
  })

  it("calls onNavigate when a nav link is clicked", () => {
    const onNavigate = vi.fn()
    render(<Sidebar onNavigate={onNavigate} />, { wrapper })
    fireEvent.click(screen.getByText("Programas"))
    expect(onNavigate).toHaveBeenCalled()
  })

  it("calls onNavigate on logout", () => {
    const onNavigate = vi.fn()
    render(<Sidebar onNavigate={onNavigate} />, { wrapper })
    fireEvent.click(screen.getByText("Cerrar sesion"))
    expect(onNavigate).toHaveBeenCalled()
  })

  it("nav links have correct hrefs", () => {
    render(<Sidebar />, { wrapper })
    const programsLink = screen.getByText("Programas").closest("a")
    expect(programsLink).toHaveAttribute("href", "/programas")
    const clientsLink = screen.getByText("Clientes").closest("a")
    expect(clientsLink).toHaveAttribute("href", "/clientes")
  })
})
