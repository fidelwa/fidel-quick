import { useAuth } from "@/context/auth-context"
import { useCustomer } from "@/hooks/use-customer"
import { usePrograms } from "@/hooks/use-programs"
import { useCashbackPrograms } from "@/hooks/use-cashback-programs"
import { useCollaborators } from "@/hooks/use-collaborators"
import { useClients } from "@/hooks/use-clients"
import {
  GlassCard,
  GlassCardContent,
  GlassCardHeader,
  GlassCardTitle,
} from "@/components/ui/glass-card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Trophy, Users, UserCheck } from "lucide-react"
import type { LucideIcon } from "lucide-react"

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString("es-MX", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  })
}

type KpiCardProps = {
  label: string
  value: number
  caption: string
  icon: LucideIcon
  /** Tailwind class for the icon halo background, eg "bg-aurora-violet/30". */
  accent: string
}

function KpiCard({ label, value, caption, icon: Icon, accent }: KpiCardProps) {
  return (
    <GlassCard className="gap-4">
      <GlassCardHeader className="items-center">
        <div>
          <GlassCardTitle className="text-sm font-medium text-muted-foreground">
            {label}
          </GlassCardTitle>
        </div>
        <div
          className={`flex h-9 w-9 items-center justify-center rounded-full ${accent}`}
        >
          <Icon className="h-4 w-4" />
        </div>
      </GlassCardHeader>
      <GlassCardContent>
        <div className="text-3xl font-bold tracking-tight">{value}</div>
        <p className="mt-1 text-xs text-muted-foreground">{caption}</p>
      </GlassCardContent>
    </GlassCard>
  )
}

export function DashboardPage() {
  const { customerId } = useAuth()
  const { data: customer, isLoading: loadingCustomer } = useCustomer(customerId)
  const { data: programs } = usePrograms(customerId)
  const { data: cashbackPrograms } = useCashbackPrograms(customerId)
  const { data: collaborators } = useCollaborators(customerId)
  const { data: clients } = useClients(customerId)

  const totalPrograms = (programs?.length ?? 0) + (cashbackPrograms?.length ?? 0)

  return (
    <div className="space-y-6">
      {/* Title */}
      <div>
        {loadingCustomer ? (
          <Skeleton className="h-9 w-80" />
        ) : (
          <h1 className="text-3xl font-bold tracking-tight text-foreground">
            {customer?.name} Dashboard
          </h1>
        )}
        <p className="mt-1 text-sm text-muted-foreground">
          Panel de administracion de tu programa de fidelidad
        </p>
      </div>

      {/* KPI cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <KpiCard
          label="Clientes"
          value={clients?.length ?? 0}
          caption="Clientes registrados"
          icon={UserCheck}
          accent="bg-[#B49DD9]/30 text-[#5b3d8a]"
        />
        <KpiCard
          label="Programas"
          value={totalPrograms}
          caption="Programas activos"
          icon={Trophy}
          accent="bg-[#E0B0CC]/40 text-[#8a3d6a]"
        />
        <KpiCard
          label="Colaboradores"
          value={collaborators?.length ?? 0}
          caption="Colaboradores registrados"
          icon={Users}
          accent="bg-[#A8CDE0]/40 text-[#2c5d75]"
        />
      </div>

      {/* Clients table */}
      <GlassCard>
        <GlassCardHeader>
          <GlassCardTitle className="text-lg">Clientes</GlassCardTitle>
        </GlassCardHeader>
        <GlassCardContent>
          {clients && clients.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow className="border-white/30 hover:bg-transparent">
                  <TableHead>Nombre</TableHead>
                  <TableHead>Telefono</TableHead>
                  <TableHead>Fecha registro</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {clients.map((client) => (
                  <TableRow key={client.id} className="border-white/20 hover:bg-white/30">
                    <TableCell className="font-medium">{client.name || "—"}</TableCell>
                    <TableCell>{client.phone}</TableCell>
                    <TableCell>{formatDate(client.created_at)}</TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground">Sin clientes registrados</p>
          )}
        </GlassCardContent>
      </GlassCard>

      {/* Collaborators table */}
      <GlassCard>
        <GlassCardHeader>
          <GlassCardTitle className="text-lg">Colaboradores</GlassCardTitle>
        </GlassCardHeader>
        <GlassCardContent>
          {collaborators && collaborators.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow className="border-white/30 hover:bg-transparent">
                  <TableHead>Nombre</TableHead>
                  <TableHead>Telefono</TableHead>
                  <TableHead>Hash ID</TableHead>
                  <TableHead>Estado</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {collaborators.map((collab) => (
                  <TableRow key={collab.id} className="border-white/20 hover:bg-white/30">
                    <TableCell className="font-medium">{collab.name}</TableCell>
                    <TableCell>{collab.phone}</TableCell>
                    <TableCell className="font-mono text-xs">{collab.hash_id}</TableCell>
                    <TableCell>
                      <Badge variant={collab.active ? "default" : "secondary"}>
                        {collab.active ? "Activo" : "Inactivo"}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground">Sin colaboradores registrados</p>
          )}
        </GlassCardContent>
      </GlassCard>

      {/* Programs table */}
      <GlassCard>
        <GlassCardHeader>
          <GlassCardTitle className="text-lg">Programas</GlassCardTitle>
        </GlassCardHeader>
        <GlassCardContent>
          {totalPrograms > 0 ? (
            <Table>
              <TableHeader>
                <TableRow className="border-white/30 hover:bg-transparent">
                  <TableHead>Nombre</TableHead>
                  <TableHead>Tipo</TableHead>
                  <TableHead>Ratio / Rate</TableHead>
                  <TableHead>Estado</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {programs?.map((prog) => (
                  <TableRow key={prog.id} className="border-white/20 hover:bg-white/30">
                    <TableCell className="font-medium">{prog.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="border-white/55 bg-white/40">
                        Earn-Burn
                      </Badge>
                    </TableCell>
                    <TableCell>1 pt / ${prog.points_ratio}</TableCell>
                    <TableCell>
                      <Badge variant={prog.active ? "default" : "secondary"}>
                        {prog.active ? "Activo" : "Inactivo"}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
                {cashbackPrograms?.map((prog) => (
                  <TableRow key={prog.id} className="border-white/20 hover:bg-white/30">
                    <TableCell className="font-medium">{prog.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline" className="border-white/55 bg-white/40">
                        Cashback
                      </Badge>
                    </TableCell>
                    <TableCell>{prog.cashback_rate}%</TableCell>
                    <TableCell>
                      <Badge variant={prog.active ? "default" : "secondary"}>
                        {prog.active ? "Activo" : "Inactivo"}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          ) : (
            <p className="text-sm text-muted-foreground">Sin programas registrados</p>
          )}
        </GlassCardContent>
      </GlassCard>
    </div>
  )
}
