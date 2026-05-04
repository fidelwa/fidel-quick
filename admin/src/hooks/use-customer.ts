import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getCustomer, updateCustomer } from "@/lib/api-client"
import type { Customer } from "@/types"

export function useCustomer(id: string) {
  return useQuery({
    queryKey: ["customer", id],
    queryFn: () => getCustomer(id),
    enabled: !!id,
  })
}

export function useUpdateCustomer(id: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (data: Partial<Customer>) => updateCustomer(id, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["customer", id] })
    },
  })
}
