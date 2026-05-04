import { useAuth } from "@/context/auth-context"
import { useFeedback } from "@/hooks/use-feedback"
import { Skeleton } from "@/components/ui/skeleton"
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
      <h1 className="text-2xl font-bold">Feedback</h1>

      {isLoading ? (
        <div className="space-y-2">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-12 w-full" />
          ))}
        </div>
      ) : !feedback?.length ? (
        <div className="rounded-lg border border-dashed p-8 text-center">
          <p className="text-muted-foreground">No hay feedback aun.</p>
        </div>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>Cliente</TableHead>
              <TableHead>Mensaje</TableHead>
              <TableHead>Fecha</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {feedback.map((f) => (
              <TableRow key={f.id}>
                <TableCell className="font-medium">{f.client_name}</TableCell>
                <TableCell>{f.message}</TableCell>
                <TableCell className="text-xs text-muted-foreground">
                  {formatDateTime(f.created_at)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      )}
    </div>
  )
}
