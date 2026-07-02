import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getFeatureFlags, updateFeatureFlag, getMe } from "@/lib/api-client"
import type { FeatureFlagUpdate } from "@/types"

// useFeatureFlag returns whether a given flag is enabled for the current
// customer, based on the flags resolved server-side and returned by /auth/me.
// While the session is loading it returns `false` (safe default — gate hidden).
//
//   const showBeta = useFeatureFlag("admin.beta_dashboard")
//   if (showBeta) { ... }
export function useFeatureFlag(key: string): boolean {
  const { data } = useQuery({
    queryKey: ["me"],
    queryFn: getMe,
  })
  return data?.flags?.[key] ?? false
}

// useFeatureFlags exposes the whole resolved flag map for the current customer.
export function useFeatureFlags(): Record<string, boolean> {
  const { data } = useQuery({
    queryKey: ["me"],
    queryFn: getMe,
  })
  return data?.flags ?? {}
}

// --- Admin management (list + toggle) ---

// useFeatureFlagAdminList lists every flag definition (admin view).
export function useFeatureFlagAdminList() {
  return useQuery({
    queryKey: ["feature-flags"],
    queryFn: getFeatureFlags,
  })
}

// useUpdateFeatureFlag toggles/updates a flag and refreshes both the admin list
// and the current session's resolved flags.
export function useUpdateFeatureFlag() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: ({ key, data }: { key: string; data: FeatureFlagUpdate }) =>
      updateFeatureFlag(key, data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["feature-flags"] })
      queryClient.invalidateQueries({ queryKey: ["me"] })
    },
  })
}
