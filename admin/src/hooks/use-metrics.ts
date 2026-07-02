import { useQuery } from "@tanstack/react-query"
import { getCustomerMetrics } from "@/lib/api-client"

export function useCustomerMetrics(customerId: string) {
  return useQuery({
    queryKey: ["customer-metrics", customerId],
    queryFn: () => getCustomerMetrics(customerId),
    enabled: !!customerId,
  })
}
