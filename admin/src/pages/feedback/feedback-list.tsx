import { useAuth } from "@/context/auth-context"
import { useFeedback } from "@/hooks/use-feedback"
import { Skeleton } from "@/components/ui/skeleton"
import { GlassCard, GlassCardContent } from "@/components/ui/glass-card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { formatDateTime } from "@/lib/utils"

export function FeedbackListPage() {
  const { customerId } = useAuth()
  const { data: feedback, isLoading } = useFeedback(customerId)

  return (
    <div className="space-y-6">
      <h1 className="text-3xl font-bold tracking-tight">Feedback</h1>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : !feedback?.length ? (
        <GlassCard>
          <GlassCardContent>
            <p className="py-6 text-center text-sm text-muted-foreground">
              No hay feedback aun.
            </p>
          </GlassCardContent>
        </GlassCard>
      ) : (
        <GlassCard>
          <GlassCardContent>
            <Table>
              <TableHeader>
                <TableRow className="border-b border-white/40 hover:bg-transparent">
                  <TableHead className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Cliente
                  </TableHead>
                  <TableHead className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Mensaje
                  </TableHead>
                  <TableHead className="text-xs font-medium uppercase tracking-wider text-muted-foreground">
                    Fecha
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {feedback.map((f) => (
                  <TableRow key={f.id} className="border-0 hover:bg-white/40">
                    <TableCell className="rounded-l-2xl py-4 font-medium">
                      {f.client_name}
                    </TableCell>
                    <TableCell className="py-4">{f.message}</TableCell>
                    <TableCell className="rounded-r-2xl py-4 text-xs text-muted-foreground">
                      {formatDateTime(f.created_at)}
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          </GlassCardContent>
        </GlassCard>
      )}
    </div>
  )
}
