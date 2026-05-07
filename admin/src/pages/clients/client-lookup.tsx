import { useState } from "react"
import { useAuth } from "@/context/auth-context"
import { usePrograms } from "@/hooks/use-programs"
import { useCashbackPrograms } from "@/hooks/use-cashback-programs"
import { useClientBalance, useClientTransactions, useCashbackClientBalance, useCashbackClientTransactions } from "@/hooks/use-clients"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { GlassCard, GlassCardContent, GlassCardHeader, GlassCardTitle } from "@/components/ui/glass-card"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Search } from "lucide-react"
import { formatPoints, formatCurrency, formatDateTime } from "@/lib/utils"

export function ClientLookupPage() {
  const { customerId } = useAuth()
  const { data: programs } = usePrograms(customerId)
  const { data: cashbackPrograms } = useCashbackPrograms(customerId)

  const [programType, setProgramType] = useState<"earn_burn" | "cashback">("earn_burn")
  const [selectedProgramId, setSelectedProgramId] = useState("")
  const [clientId, setClientId] = useState("")
  const [searchTriggered, setSearchTriggered] = useState(false)
  const [activeSearch, setActiveSearch] = useState({ programId: "", clientId: "", type: "" as "earn_burn" | "cashback" | "" })

  const earnBalance = useClientBalance(
    activeSearch.type === "earn_burn" ? activeSearch.programId : "",
    activeSearch.type === "earn_burn" ? activeSearch.clientId : ""
  )
  const earnTransactions = useClientTransactions(
    activeSearch.type === "earn_burn" ? activeSearch.programId : "",
    activeSearch.type === "earn_burn" ? activeSearch.clientId : ""
  )
  const cashbackBalance = useCashbackClientBalance(
    activeSearch.type === "cashback" ? activeSearch.programId : "",
    activeSearch.type === "cashback" ? activeSearch.clientId : ""
  )
  const cashbackTransactions = useCashbackClientTransactions(
    activeSearch.type === "cashback" ? activeSearch.programId : "",
    activeSearch.type === "cashback" ? activeSearch.clientId : ""
  )

  const balance = activeSearch.type === "earn_burn" ? earnBalance : cashbackBalance
  const transactions = activeSearch.type === "earn_burn" ? earnTransactions : cashbackTransactions
  const isEarnBurn = activeSearch.type === "earn_burn"

  const handleSearch = () => {
    if (!selectedProgramId || !clientId.trim()) return
    setActiveSearch({ programId: selectedProgramId, clientId: clientId.trim(), type: programType })
    setSearchTriggered(true)
  }

  const currentPrograms = programType === "earn_burn" ? programs : cashbackPrograms

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Buscar Cliente</h1>

      <GlassCard>
        <GlassCardHeader>
          <GlassCardTitle>Consulta de balance y transacciones</GlassCardTitle>
        </GlassCardHeader>
        <GlassCardContent className="space-y-4">
          <Tabs value={programType} onValueChange={(v) => { setProgramType(v as "earn_burn" | "cashback"); setSelectedProgramId("") }}>
            <TabsList>
              <TabsTrigger value="earn_burn">Earn-Burn</TabsTrigger>
              <TabsTrigger value="cashback">Cashback</TabsTrigger>
            </TabsList>
          </Tabs>

          <div className="grid gap-4 sm:grid-cols-3">
            <div className="space-y-2">
              <Label>Programa</Label>
              <Select value={selectedProgramId} onValueChange={setSelectedProgramId}>
                <SelectTrigger>
                  <SelectValue placeholder="Selecciona un programa" />
                </SelectTrigger>
                <SelectContent>
                  {currentPrograms?.map((p) => (
                    <SelectItem key={p.id} value={p.id}>{p.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-2">
              <Label>Client ID</Label>
              <Input
                placeholder="UUID del cliente"
                value={clientId}
                onChange={(e) => setClientId(e.target.value)}
                onKeyDown={(e) => e.key === "Enter" && handleSearch()}
              />
            </div>
            <div className="flex items-end">
              <Button onClick={handleSearch} disabled={!selectedProgramId || !clientId.trim()}>
                <Search className="mr-2 h-4 w-4" />
                Buscar
              </Button>
            </div>
          </div>
        </GlassCardContent>
      </GlassCard>

      {searchTriggered && (
        <>
          <GlassCard>
            <GlassCardHeader>
              <GlassCardTitle>Balance</GlassCardTitle>
            </GlassCardHeader>
            <GlassCardContent>
              {balance.isLoading ? (
                <Skeleton className="h-10 w-32" />
              ) : balance.isError ? (
                <p className="text-destructive">Cliente no encontrado o sin balance</p>
              ) : balance.data ? (
                <p className="text-3xl font-bold">
                  {isEarnBurn
                    ? `${formatPoints(balance.data.balance)} pts`
                    : formatCurrency(balance.data.balance)}
                </p>
              ) : null}
            </GlassCardContent>
          </GlassCard>

          <GlassCard>
            <GlassCardHeader>
              <GlassCardTitle>Transacciones</GlassCardTitle>
            </GlassCardHeader>
            <GlassCardContent>
              {transactions.isLoading ? (
                <div className="space-y-2">
                  {Array.from({ length: 5 }).map((_, i) => (
                    <Skeleton key={i} className="h-10 w-full" />
                  ))}
                </div>
              ) : transactions.isError ? (
                <p className="text-destructive">Error al cargar transacciones</p>
              ) : !transactions.data?.length ? (
                <p className="text-muted-foreground">Sin transacciones</p>
              ) : (
                <Table>
                  <TableHeader>
                    <TableRow>
                      <TableHead>Fecha</TableHead>
                      <TableHead>Tipo</TableHead>
                      <TableHead>Monto</TableHead>
                      <TableHead>Balance</TableHead>
                      <TableHead>Descripcion</TableHead>
                    </TableRow>
                  </TableHeader>
                  <TableBody>
                    {transactions.data.map((t) => (
                      <TableRow key={t.id}>
                        <TableCell className="text-xs">{formatDateTime(t.created_at)}</TableCell>
                        <TableCell>
                          <Badge variant={t.type === "earn" ? "default" : t.type === "burn" ? "destructive" : "secondary"}>
                            {t.type}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          {isEarnBurn
                            ? `${t.type === "burn" ? "-" : "+"}${formatPoints(t.amount)} pts`
                            : `${t.type === "burn" ? "-" : "+"}${formatCurrency(t.amount)}`}
                        </TableCell>
                        <TableCell>
                          {isEarnBurn
                            ? `${formatPoints(t.balance_after)} pts`
                            : formatCurrency(t.balance_after)}
                        </TableCell>
                        <TableCell className="text-muted-foreground text-xs">{t.description || "—"}</TableCell>
                      </TableRow>
                    ))}
                  </TableBody>
                </Table>
              )}
            </GlassCardContent>
          </GlassCard>
        </>
      )}
    </div>
  )
}
