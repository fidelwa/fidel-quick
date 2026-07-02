import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import {
  getPushcardConfig,
  upsertPushcardConfig,
  getPushcardCards,
} from "@/lib/api-client"

export function usePushcardConfig(customerId: string) {
  return useQuery({
    queryKey: ["pushcard-config", customerId],
    queryFn: () => getPushcardConfig(customerId),
    enabled: !!customerId,
    retry: false, // 404 if no config yet
  })
}

export function useUpsertPushcardConfig(customerSisfiID: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: {
      card_slots: number
      reward_on_complete?: string
      card_expiry_days?: number | null
    }) => upsertPushcardConfig(customerSisfiID, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["pushcard-config"] })
    },
  })
}

export function usePushcardCards(
  customerSisfiID: string,
  status?: "open" | "completed" | "redeemed" | "cancelled"
) {
  return useQuery({
    queryKey: ["pushcard-cards", customerSisfiID, status],
    queryFn: () => getPushcardCards(customerSisfiID, status),
    enabled: !!customerSisfiID,
  })
}
