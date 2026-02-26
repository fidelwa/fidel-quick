import { useAuth } from "@/context/auth-context"
import { useCustomer } from "@/hooks/use-customer"
import { usePrograms } from "@/hooks/use-programs"
import { useCashbackPrograms } from "@/hooks/use-cashback-programs"
import { useCollaborators } from "@/hooks/use-collaborators"
import { useClients } from "@/hooks/use-clients"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
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

function formatDate(dateStr: string) {
  return new Date(dateStr).toLocaleDateString("es-MX", {
    day: "2-digit",
    month: "short",
    year: "numeric",
  })
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
          <Skeleton className="h-8 w-80" />
        ) : (
          <h1 className="text-2xl font-bold">
            {customer?.name} Dashboard
          </h1>
        )}
        <p className="text-muted-foreground">
          Panel de administracion de tu programa de fidelidad
        </p>
      </div>

      {/* Summary cards */}
      <div className="grid gap-4 sm:grid-cols-3">
        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Clientes</CardTitle>
            <UserCheck className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{clients?.length ?? 0}</div>
            <p className="text-xs text-muted-foreground">Clientes registrados</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Programas</CardTitle>
            <Trophy className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{totalPrograms}</div>
            <p className="text-xs text-muted-foreground">Programas activos</p>
          </CardContent>
        </Card>

        <Card>
          <CardHeader className="flex flex-row items-center justify-between pb-2">
            <CardTitle className="text-sm font-medium">Colaboradores</CardTitle>
            <Users className="h-4 w-4 text-muted-foreground" />
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-bold">{collaborators?.length ?? 0}</div>
            <p className="text-xs text-muted-foreground">Colaboradores registrados</p>
          </CardContent>
        </Card>
      </div>

      {/* Clients table */}
      <Card>
        <CardHeader>
          <CardTitle>Clientes</CardTitle>
        </CardHeader>
        <CardContent>
          {clients && clients.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nombre</TableHead>
                  <TableHead>Telefono</TableHead>
                  <TableHead>Fecha registro</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {clients.map((client) => (
                  <TableRow key={client.id}>
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
        </CardContent>
      </Card>

      {/* Collaborators table */}
      <Card>
        <CardHeader>
          <CardTitle>Colaboradores</CardTitle>
        </CardHeader>
        <CardContent>
          {collaborators && collaborators.length > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nombre</TableHead>
                  <TableHead>Telefono</TableHead>
                  <TableHead>Hash ID</TableHead>
                  <TableHead>Estado</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {collaborators.map((collab) => (
                  <TableRow key={collab.id}>
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
        </CardContent>
      </Card>

      {/* Programs table */}
      <Card>
        <CardHeader>
          <CardTitle>Programas</CardTitle>
        </CardHeader>
        <CardContent>
          {totalPrograms > 0 ? (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Nombre</TableHead>
                  <TableHead>Tipo</TableHead>
                  <TableHead>Ratio / Rate</TableHead>
                  <TableHead>Estado</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {programs?.map((prog) => (
                  <TableRow key={prog.id}>
                    <TableCell className="font-medium">{prog.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">Earn-Burn</Badge>
                    </TableCell>
                    <TableCell>{prog.points_ratio} pts/$</TableCell>
                    <TableCell>
                      <Badge variant={prog.active ? "default" : "secondary"}>
                        {prog.active ? "Activo" : "Inactivo"}
                      </Badge>
                    </TableCell>
                  </TableRow>
                ))}
                {cashbackPrograms?.map((prog) => (
                  <TableRow key={prog.id}>
                    <TableCell className="font-medium">{prog.name}</TableCell>
                    <TableCell>
                      <Badge variant="outline">Cashback</Badge>
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
        </CardContent>
      </Card>
    </div>
  )
}
