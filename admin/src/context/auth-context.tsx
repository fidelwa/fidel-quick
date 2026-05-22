import { createContext, useContext, useState, useEffect, type ReactNode } from "react"
import { setToken } from "@/lib/api-client"

interface AuthState {
  token: string
  customerId: string
  email: string
}

interface AuthContextValue {
  token: string
  customerId: string
  email: string
  isAuthenticated: boolean
  login: (token: string, customerId: string, email: string) => void
  logout: () => void
}

const AuthContext = createContext<AuthContextValue | null>(null)

const EMPTY_AUTH: AuthState = { token: "", customerId: "", email: "" }

// Lectura defensiva del localStorage. Antes hacíamos JSON.parse sin
// try/catch — cualquier corrupción del valor (cadena "undefined", JSON
// trunco, shape cambiado entre deploys) tiraba en el useState initializer
// y se renderizaba pantalla en blanco. Ahora cualquier fallo se trata
// como sesión inexistente y limpia el storage.
function readStoredAuth(): AuthState {
  if (typeof window === "undefined") return EMPTY_AUTH
  try {
    const stored = localStorage.getItem("fidel_auth")
    if (!stored) return EMPTY_AUTH
    const parsed = JSON.parse(stored)
    if (
      parsed &&
      typeof parsed.token === "string" &&
      typeof parsed.customerId === "string" &&
      typeof parsed.email === "string"
    ) {
      return parsed as AuthState
    }
    localStorage.removeItem("fidel_auth")
    return EMPTY_AUTH
  } catch {
    try { localStorage.removeItem("fidel_auth") } catch { /* ignore */ }
    return EMPTY_AUTH
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [auth, setAuth] = useState<AuthState>(() => {
    const initial = readStoredAuth()
    if (initial.token) setToken(initial.token)
    return initial
  })

  useEffect(() => {
    if (auth.token) {
      localStorage.setItem("fidel_auth", JSON.stringify(auth))
      setToken(auth.token)
    } else {
      localStorage.removeItem("fidel_auth")
      setToken("")
    }
  }, [auth])

  const login = (token: string, customerId: string, email: string) => {
    setAuth({ token, customerId, email })
  }

  const logout = () => {
    setAuth({ token: "", customerId: "", email: "" })
  }

  return (
    <AuthContext.Provider
      value={{
        token: auth.token,
        customerId: auth.customerId,
        email: auth.email,
        isAuthenticated: !!auth.token && !!auth.customerId,
        login,
        logout,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) throw new Error("useAuth must be used within AuthProvider")
  return ctx
}
