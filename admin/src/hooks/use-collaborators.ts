import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getCollaborators, createCollaborator } from "@/lib/api-client"

export function useCollaborators(customerId: string) {
  return useQuery({
    queryKey: ["collaborators", customerId],
    queryFn: () => getCollaborators(customerId),
    enabled: !!customerId,
  })
}

export function useCreateCollaborator(customerId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: { name: string; phone: string }) =>
      createCollaborator(customerId, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["collaborators", customerId] })
    },
  })
}
