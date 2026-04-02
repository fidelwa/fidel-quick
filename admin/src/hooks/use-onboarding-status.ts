import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query"
import { getOnboarding, updateOnboardingStep, completeOnboarding } from "@/lib/api-client"

export function useOnboardingStatus(enabled: boolean) {
  return useQuery({
    queryKey: ["onboarding"],
    queryFn: getOnboarding,
    enabled,
  })
}

export function useUpdateOnboardingStep() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: (step: number) => updateOnboardingStep(step),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["onboarding"] })
    },
  })
}

export function useCompleteOnboarding() {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: () => completeOnboarding(),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["onboarding"] })
    },
  })
}
