import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getCashbackPrograms, createCashbackProgram, updateCashbackProgram } from "@/lib/api-client"
import type { CashbackProgram } from "@/types"

export function useCashbackPrograms(customerId: string) {
  return useQuery({
    queryKey: ["cashback-programs", customerId],
    queryFn: () => getCashbackPrograms(customerId),
    enabled: !!customerId,
  })
}

export function useCreateCashbackProgram() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { customer_id: string; name: string; cashback_rate: number }) =>
      createCashbackProgram(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cashback-programs"] })
    },
  })
}

export function useUpdateCashbackProgram(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (
      data: Partial<
        Pick<
          CashbackProgram,
          | "name"
          | "cashback_rate"
          | "active"
          | "expiry_days"
          | "min_ticket_amount"
          | "max_cashback_per_tx"
          | "max_cashback_per_period"
        >
      >,
    ) => updateCashbackProgram(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cashback-programs"] })
    },
  })
}
