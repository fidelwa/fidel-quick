import { Menu } from "lucide-react"
import { Button } from "@/components/ui/button"
import { useAuth } from "@/context/auth-context"
import { useCustomer } from "@/hooks/use-customer"

export function Header({ onMenuClick }: { onMenuClick: () => void }) {
  const { email, customerId } = useAuth()
  const { data: customer } = useCustomer(customerId)

  return (
    <header className="glass-subtle flex h-14 items-center gap-4 rounded-none px-4 lg:px-6">
      <Button
        variant="ghost"
        size="icon"
        className="lg:hidden"
        onClick={onMenuClick}
      >
        <Menu className="h-5 w-5" />
      </Button>
      <div className="flex-1" />
      <span className="text-sm text-muted-foreground">
        {customer?.name ? `${customer.name} — ${email}` : email}
      </span>
    </header>
  )
}
