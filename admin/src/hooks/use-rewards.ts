import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getRewards, createReward, updateReward } from "@/lib/api-client"
import type { Reward } from "@/types"

export function useRewards(programId: string) {
  return useQuery({
    queryKey: ["rewards", programId],
    queryFn: () => getRewards(programId),
    enabled: !!programId,
  })
}

export function useCreateReward(programId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async (data: { name: string; description: string; points_cost: number }) => {
      const resp = await createReward(programId, data)
      return { ...data, ...resp } as Reward
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["rewards", programId] })
    },
  })
}

export function useUpdateReward(programId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ rewardId, ...data }: { rewardId: string } & Partial<Pick<Reward, "name" | "description" | "points_cost" | "active">>) =>
      updateReward(programId, rewardId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["rewards", programId] })
    },
  })
}
