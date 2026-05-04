import { useQuery } from "@tanstack/react-query"
import { getFeedback } from "@/lib/api-client"

export function useFeedback(customerId: string) {
  return useQuery({
    queryKey: ["feedback", customerId],
    queryFn: () => getFeedback(customerId),
    enabled: !!customerId,
  })
}
