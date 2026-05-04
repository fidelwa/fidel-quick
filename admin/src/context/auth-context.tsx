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

export function AuthProvider({ children }: { children: ReactNode }) {
  const [auth, setAuth] = useState<AuthState>(() => {
    const stored = localStorage.getItem("fidel_auth")
    if (stored) {
      const parsed = JSON.parse(stored) as AuthState
      setToken(parsed.token)
      return parsed
    }
    return { token: "", customerId: "", email: "" }
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
