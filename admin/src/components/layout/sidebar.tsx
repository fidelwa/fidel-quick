import { NavLink } from "react-router-dom"
import {
  Home,
  Building2,
  Trophy,
  Users,
  UserSearch,
  MessageSquare,
  Stamp,
  LogOut,
} from "lucide-react"
import { cn } from "@/lib/utils"
import { useAuth } from "@/context/auth-context"
import { usePushcardConfig } from "@/hooks/use-pushcard"
import { Separator } from "@/components/ui/separator"

const baseNavItems = [
  { to: "/", label: "Inicio", icon: Home },
  { to: "/perfil", label: "Mi Negocio", icon: Building2 },
  { to: "/programas", label: "Programas", icon: Trophy },
  { to: "/colaboradores", label: "Colaboradores", icon: Users },
  { to: "/clientes", label: "Clientes", icon: UserSearch },
  { to: "/feedback", label: "Feedback", icon: MessageSquare },
]

const pushcardNavItem = {
  to: "/pushcard",
  label: "Tarjeta de sellos",
  icon: Stamp,
}

export function Sidebar({ onNavigate }: { onNavigate?: () => void }) {
  const { logout, customerId } = useAuth()
  const { data: pushcardConfig } = usePushcardConfig(customerId)

  // 'Tarjeta de sellos' solo aparece si el customer tiene pushcard activo.
  // El endpoint devuelve 404 (no config) o un row con active=false; en ambos
  // casos ocultamos la entrada para evitar que el admin navegue a una página
  // de un sisfi que no aplica.
  const showPushcard = pushcardConfig?.active === true
  const navItems = showPushcard
    ? [...baseNavItems.slice(0, 3), pushcardNavItem, ...baseNavItems.slice(3)]
    : baseNavItems

  return (
    <div className="glass-strong flex h-full flex-col rounded-none border-0 text-sidebar-foreground">
      <div className="flex h-14 items-center border-b border-white/40 px-4">
        <span className="text-lg font-semibold tracking-tight">Fidel Admin</span>
      </div>
      <nav className="flex-1 space-y-1 p-3">
        {navItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            end={item.to === "/"}
            onClick={onNavigate}
            className={({ isActive }) =>
              cn(
                "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                isActive
                  ? "bg-sidebar-accent text-sidebar-accent-foreground"
                  : "text-sidebar-foreground/70 hover:bg-sidebar-accent/50 hover:text-sidebar-foreground"
              )
            }
          >
            <item.icon className="h-4 w-4" />
            {item.label}
          </NavLink>
        ))}
      </nav>
      <div className="p-3">
        <Separator className="mb-3" />
        <button
          onClick={() => {
            logout()
            onNavigate?.()
          }}
          className="flex w-full items-center gap-3 rounded-md px-3 py-2 text-sm font-medium text-sidebar-foreground/70 transition-colors hover:bg-sidebar-accent/50 hover:text-sidebar-foreground"
        >
          <LogOut className="h-4 w-4" />
          Cerrar sesion
        </button>
      </div>
    </div>
  )
}
