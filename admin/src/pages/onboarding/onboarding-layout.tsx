import { useEffect, useRef } from "react"
import { Navigate } from "react-router-dom"
import { useAuth } from "@/context/auth-context"
import { usePrograms } from "@/hooks/use-programs"
import { useCashbackPrograms } from "@/hooks/use-cashback-programs"
import { useCollaborators } from "@/hooks/use-collaborators"
import { useRewards } from "@/hooks/use-rewards"
import { useCashbackRewards } from "@/hooks/use-cashback-rewards"
import { usePushcardConfig } from "@/hooks/use-pushcard"
import { useOnboarding } from "@/hooks/use-onboarding"
import { useOnboardingStatus, useUpdateOnboardingStep } from "@/hooks/use-onboarding-status"
import { StepIndicator } from "@/components/onboarding/step-indicator"
import { StepTransition } from "@/components/onboarding/step-transition"
import { StepProgram } from "./step-program"
import { StepRewards } from "./step-rewards"
import { StepTeam } from "./step-team"
import { StepReady } from "./step-ready"
import { Skeleton } from "@/components/ui/skeleton"

export function OnboardingLayout() {
  const { isAuthenticated, customerId } = useAuth()
  const { data: onboardingStatus, isLoading: onboardingLoading } = useOnboardingStatus(isAuthenticated)
  const updateStep = useUpdateOnboardingStep()
  const { data: programs } = usePrograms(customerId)
  const { data: cashbackPrograms } = useCashbackPrograms(customerId)
  const { data: existingCollaborators } = useCollaborators(customerId)
  // /pushcard/config devuelve 404 (apperror.NotFound) cuando aún no hay
  // pushcard creada — useQuery lo expone como error y data = undefined.
  // Lo tratamos como "no hay" sin spamear toasts.
  const { data: pushcardFromServer, isLoading: pushcardLoading } = usePushcardConfig(customerId)

  const earnBurnProgramFromServer = programs?.[0] ?? null
  const cashbackProgramFromServer = cashbackPrograms?.[0] ?? null

  const { data: existingRewards } = useRewards(earnBurnProgramFromServer?.id ?? "")
  const { data: existingCbRewards } = useCashbackRewards(cashbackProgramFromServer?.id ?? "")

  const onboarding = useOnboarding()
  const recoveryDone = useRef(false)

  // Reload recovery: sync server data into local state and jump to correct step
  useEffect(() => {
    if (recoveryDone.current) return
    // Wait until base queries have resolved
    if (programs === undefined || cashbackPrograms === undefined) return
    // pushcard responde 404 si no hay config — esperamos isLoading=false
    // (no data === undefined, porque data nunca llega cuando hay 404).
    if (pushcardLoading) return

    // If there are programs, wait for their rewards to resolve too (sequential fetch)
    if (earnBurnProgramFromServer && existingRewards === undefined) return
    if (cashbackProgramFromServer && existingCbRewards === undefined) return

    if (earnBurnProgramFromServer) {
      onboarding.setEarnBurnProgram(earnBurnProgramFromServer)
    }
    if (cashbackProgramFromServer) {
      onboarding.setCashbackProgram(cashbackProgramFromServer)
    }
    if (pushcardFromServer) {
      onboarding.setPushcardConfig(pushcardFromServer)
    }
    if (existingRewards?.length) {
      onboarding.setRewards(existingRewards)
    }
    if (existingCbRewards?.length) {
      onboarding.setCashbackRewards(existingCbRewards)
    }
    if (existingCollaborators?.length) {
      onboarding.setCollaborators(existingCollaborators)
    }

    // Use server step if available, otherwise calculate from data
    if (onboardingStatus?.current_step && onboardingStatus.current_step > 1) {
      onboarding.goToStep(onboardingStatus.current_step)
    } else {
      const hasPrograms =
        (programs?.length ?? 0) > 0 ||
        (cashbackPrograms?.length ?? 0) > 0 ||
        !!pushcardFromServer
      if (hasPrograms) {
        const hasRewards = (existingRewards?.length ?? 0) > 0 || (existingCbRewards?.length ?? 0) > 0
        if (hasRewards) {
          const hasCollaborators = (existingCollaborators?.length ?? 0) > 0
          if (hasCollaborators) {
            onboarding.goToStep(4)
          } else {
            onboarding.goToStep(3)
          }
        } else {
          onboarding.goToStep(2)
        }
      }
    }

    recoveryDone.current = true
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [
    programs,
    cashbackPrograms,
    earnBurnProgramFromServer?.id,
    cashbackProgramFromServer?.id,
    existingRewards,
    existingCbRewards,
    existingCollaborators,
    onboardingStatus,
    pushcardFromServer,
    pushcardLoading,
  ])

  // Persist step changes to server
  const prevStepRef = useRef(onboarding.currentStep)
  useEffect(() => {
    if (onboarding.currentStep !== prevStepRef.current && recoveryDone.current) {
      prevStepRef.current = onboarding.currentStep
      updateStep.mutate(onboarding.currentStep)
    }
  }, [onboarding.currentStep, updateStep])

  if (!isAuthenticated) {
    return <Navigate to="/login" replace />
  }

  if (onboardingLoading) {
    return (
      <div className="flex min-h-screen items-center justify-center">
        <div className="w-full max-w-2xl space-y-6 p-4">
          <Skeleton className="mx-auto h-10 w-64" />
          <Skeleton className="h-64 w-full" />
        </div>
      </div>
    )
  }

  if (onboardingStatus?.completed) {
    return <Navigate to="/" replace />
  }

  const completedSteps: number[] = []
  if (onboarding.earnBurnProgram || onboarding.cashbackProgram || onboarding.pushcardConfig) completedSteps.push(1)
  if (onboarding.rewards.length > 0 || onboarding.cashbackRewards.length > 0) completedSteps.push(2)
  if (onboarding.collaborators.length > 0) completedSteps.push(3)

  const renderStep = () => {
    switch (onboarding.currentStep) {
      case 1:
        return (
          <StepProgram
            earnBurnProgram={onboarding.earnBurnProgram}
            cashbackProgram={onboarding.cashbackProgram}
            pushcardConfig={onboarding.pushcardConfig}
            onEarnBurnCreated={onboarding.setEarnBurnProgram}
            onCashbackCreated={onboarding.setCashbackProgram}
            onPushcardCreated={onboarding.setPushcardConfig}
            onNext={onboarding.nextStep}
          />
        )
      case 2:
        return (
          <StepRewards
            earnBurnProgram={onboarding.earnBurnProgram}
            cashbackProgram={onboarding.cashbackProgram}
            pushcardConfig={onboarding.pushcardConfig}
            rewards={onboarding.rewards}
            cashbackRewards={onboarding.cashbackRewards}
            onRewardsChange={onboarding.setRewards}
            onCashbackRewardsChange={onboarding.setCashbackRewards}
            onNext={onboarding.nextStep}
            onPrev={onboarding.prevStep}
          />
        )
      case 3:
        return (
          <StepTeam
            collaborators={onboarding.collaborators}
            onCollaboratorsChange={onboarding.setCollaborators}
            onNext={onboarding.nextStep}
            onPrev={onboarding.prevStep}
          />
        )
      case 4:
        return (
          <StepReady
            earnBurnProgram={onboarding.earnBurnProgram}
            cashbackProgram={onboarding.cashbackProgram}
            rewards={onboarding.rewards}
            cashbackRewards={onboarding.cashbackRewards}
            collaborators={onboarding.collaborators}
            onPrev={onboarding.prevStep}
          />
        )
      default:
        return null
    }
  }

  return (
    <div className="flex min-h-screen flex-col">
      {/* Header */}
      <div className="border-b px-4 py-4">
        <h1 className="text-center text-lg font-bold">Fidel</h1>
      </div>

      {/* Content */}
      <div className="flex flex-1 flex-col items-center px-4 py-8">
        <div className="w-full max-w-2xl space-y-8">
          <StepIndicator
            currentStep={onboarding.currentStep}
            completedSteps={completedSteps}
          />

          <StepTransition
            stepKey={onboarding.currentStep}
            direction={onboarding.direction}
          >
            {renderStep()}
          </StepTransition>
        </div>
      </div>
    </div>
  )
}
