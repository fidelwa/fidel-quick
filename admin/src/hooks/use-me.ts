import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getMe, linkGoogle, unlinkGoogle } from "@/lib/api-client"

export function useMe() {
  return useQuery({
    queryKey: ["me"],
    queryFn: getMe,
  })
}

export function useLinkGoogle() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (googleToken: string) => linkGoogle(googleToken),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["me"] })
    },
  })
}

export function useUnlinkGoogle() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => unlinkGoogle(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["me"] })
    },
  })
}
