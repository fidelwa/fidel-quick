import { useEffect, useRef } from "react"
import { Navigate } from "react-router-dom"
import { useAuth } from "@/context/auth-context"
import { useOnboarding } from "@/hooks/use-onboarding"
import {
  useOnboardingStatus,
  useUpdateOnboardingStep,
} from "@/hooks/use-onboarding-status"
import { StepIndicator } from "@/components/onboarding/step-indicator"
import { StepTransition } from "@/components/onboarding/step-transition"
import { StepProgram } from "./step-program"
import { StepRewards } from "./step-rewards"
import { StepTeam } from "./step-team"
import { StepReady } from "./step-ready"
import { Skeleton } from "@/components/ui/skeleton"

export function OnboardingLayout() {
  const { isAuthenticated, customerId } = useAuth()
  const { data: onboardingStatus, isLoading: onboardingLoading } =
    useOnboardingStatus(isAuthenticated)
  const updateStep = useUpdateOnboardingStep()
  const onboarding = useOnboarding(customerId)

  // Persistir el paso actual en backend (sólo para retomar dispositivo
  // distinto — el draft local guarda el resto de los datos).
  const prevStepRef = useRef(onboarding.currentStep)
  useEffect(() => {
    if (onboarding.currentStep !== prevStepRef.current) {
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

  // Un paso se marca como completado solo si el usuario ya lo dejó atrás.
  // Si retrocede con "Anterior", el check del paso al que vuelve se quita.
  const completedSteps: number[] = []
  for (let i = 1; i < onboarding.currentStep; i++) {
    completedSteps.push(i)
  }

  const renderStep = () => {
    switch (onboarding.currentStep) {
      case 1:
        return (
          <StepProgram
            sisfi={onboarding.sisfi}
            pendingProgramForm={onboarding.pendingProgramForm}
            onSisfiChange={onboarding.setSisfi}
            onPendingProgramFormChange={onboarding.setPendingProgramForm}
            onNext={onboarding.nextStep}
          />
        )
      case 2:
        return (
          <StepRewards
            sisfi={onboarding.sisfi}
            rewards={onboarding.rewards}
            pendingReward={onboarding.pendingReward}
            onAddReward={onboarding.addReward}
            onRemoveReward={onboarding.removeReward}
            onSetRewards={onboarding.setRewards}
            onPendingRewardChange={onboarding.setPendingReward}
            onNext={onboarding.nextStep}
            onPrev={onboarding.prevStep}
          />
        )
      case 3:
        return (
          <StepTeam
            collaborators={onboarding.collaborators}
            pendingCollaborator={onboarding.pendingCollaborator}
            onAddCollaborator={onboarding.addCollaborator}
            onRemoveCollaborator={onboarding.removeCollaborator}
            onPendingCollaboratorChange={onboarding.setPendingCollaborator}
            onNext={onboarding.nextStep}
            onPrev={onboarding.prevStep}
          />
        )
      case 4:
        return (
          <StepReady
            sisfi={onboarding.sisfi}
            rewards={onboarding.rewards}
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
      <div className="border-b px-4 py-4">
        <h1 className="text-center text-lg font-bold">Fidel</h1>
      </div>

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
