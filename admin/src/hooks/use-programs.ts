import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getPrograms, createProgram, updateProgram } from "@/lib/api-client"
import type { Program } from "@/types"

export function usePrograms(customerId: string) {
  return useQuery({
    queryKey: ["programs", customerId],
    queryFn: () => getPrograms(customerId),
    enabled: !!customerId,
  })
}

export function useCreateProgram() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { customer_id: string; type: string; name: string; points_ratio: number }) =>
      createProgram(data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["programs"] })
    },
  })
}

export function useUpdateProgram(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Pick<Program, "name" | "points_ratio" | "active">>) =>
      updateProgram(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["programs"] })
    },
  })
}
