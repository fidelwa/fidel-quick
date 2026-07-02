import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getCashbackRewards, createCashbackReward, updateCashbackReward } from "@/lib/api-client"
import type { CashbackReward } from "@/types"

export function useCashbackRewards(programId: string) {
  return useQuery({
    queryKey: ["cashback-rewards", programId],
    queryFn: () => getCashbackRewards(programId),
    enabled: !!programId,
  })
}

export function useCreateCashbackReward(programId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: { name: string; description: string; cost: number }) => {
      const resp = await createCashbackReward(programId, data)
      return { ...data, ...resp } as CashbackReward
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cashback-rewards", programId] })
    },
  })
}

export function useUpdateCashbackReward(programId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ rewardId, ...data }: { rewardId: string } & Partial<Pick<CashbackReward, "name" | "description" | "cost" | "active">>) =>
      updateCashbackReward(programId, rewardId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["cashback-rewards", programId] })
    },
  })
}
