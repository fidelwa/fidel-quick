import { Navigate } from "react-router-dom"
import { useAuth } from "@/context/auth-context"
import { useOnboarding } from "@/hooks/use-onboarding"
import { StepIndicator } from "@/components/onboarding/step-indicator"
import { StepTransition } from "@/components/onboarding/step-transition"
import { StepProgram } from "./step-program"
import { StepRewards } from "./step-rewards"
import { StepTeam } from "./step-team"
import { StepAccount } from "./step-account"
import { StepReady } from "./step-ready"

// Flujo nuevo (anonymous wizard):
//   /registro            → recolecta business info, navega a /onboarding
//   /onboarding          → wizard sin auth (state en localStorage)
//     Step 1: programas
//     Step 2: recompensas
//     Step 3: equipo
//     Step 4: cuenta     → aquí se hace POST /onboarding/register o /register/google
//     Step 5: listo      → muestra QR, navega a /
//
// Si el usuario llega ya autenticado, lo mandamos al dashboard — el
// onboarding solo corre para usuarios nuevos. La sesión vieja queda
// inválida tras el deploy de este refactor (los wizard-states viejos
// del server no se respetan).
export function OnboardingLayout() {
  const { isAuthenticated } = useAuth()
  const onboarding = useOnboarding()

  if (isAuthenticated && onboarding.currentStep < 5) {
    // Usuario ya logueado pero no terminó step-ready: probablemente
    // viene de step-account que ya creó la cuenta. Lo dejamos avanzar
    // al step 5 (ready). Si está en steps 1-3 logueado, es un usuario
    // viejo — lo mandamos al dashboard.
    if (onboarding.currentStep <= 3) {
      return <Navigate to="/" replace />
    }
  }

  // Sin auth y sin businessInfo: el usuario llegó a /onboarding sin
  // pasar por /registro. Lo mandamos a /registro.
  if (!isAuthenticated && !onboarding.businessInfo) {
    return <Navigate to="/registro" replace />
  }

  const completedSteps: number[] = []
  if (onboarding.earnBurnDraft || onboarding.cashbackDraft || onboarding.pushcardDraft) {
    completedSteps.push(1)
  }
  const onlyPushcard =
    !onboarding.earnBurnDraft &&
    !onboarding.cashbackDraft &&
    !!onboarding.pushcardDraft
  if (onboarding.rewardDrafts.length > 0 || onboarding.cashbackRewardDrafts.length > 0 || onlyPushcard) {
    completedSteps.push(2)
  }
  if (onboarding.collaboratorDrafts.length > 0) {
    completedSteps.push(3)
  }
  if (isAuthenticated) {
    completedSteps.push(4)
  }

  const renderStep = () => {
    switch (onboarding.currentStep) {
      case 1:
        return (
          <StepProgram
            earnBurnDraft={onboarding.earnBurnDraft}
            cashbackDraft={onboarding.cashbackDraft}
            pushcardDraft={onboarding.pushcardDraft}
            onEarnBurnChange={onboarding.setEarnBurnDraft}
            onCashbackChange={onboarding.setCashbackDraft}
            onPushcardChange={onboarding.setPushcardDraft}
            onNext={onboarding.nextStep}
          />
        )
      case 2:
        return (
          <StepRewards
            earnBurnDraft={onboarding.earnBurnDraft}
            cashbackDraft={onboarding.cashbackDraft}
            pushcardDraft={onboarding.pushcardDraft}
            rewardDrafts={onboarding.rewardDrafts}
            cashbackRewardDrafts={onboarding.cashbackRewardDrafts}
            onRewardsChange={onboarding.setRewardDrafts}
            onCashbackRewardsChange={onboarding.setCashbackRewardDrafts}
            onNext={onboarding.nextStep}
            onPrev={onboarding.prevStep}
          />
        )
      case 3:
        return (
          <StepTeam
            collaboratorDrafts={onboarding.collaboratorDrafts}
            onCollaboratorsChange={onboarding.setCollaboratorDrafts}
            onNext={onboarding.nextStep}
            onPrev={onboarding.prevStep}
          />
        )
      case 4:
        if (!onboarding.businessInfo) {
          return <Navigate to="/registro" replace />
        }
        return (
          <StepAccount
            businessInfo={onboarding.businessInfo}
            earnBurnDraft={onboarding.earnBurnDraft}
            cashbackDraft={onboarding.cashbackDraft}
            pushcardDraft={onboarding.pushcardDraft}
            rewardDrafts={onboarding.rewardDrafts}
            cashbackRewardDrafts={onboarding.cashbackRewardDrafts}
            collaboratorDrafts={onboarding.collaboratorDrafts}
            onSuccess={onboarding.nextStep}
            onPrev={onboarding.prevStep}
          />
        )
      case 5:
        return (
          <StepReady
            earnBurnDraft={onboarding.earnBurnDraft}
            cashbackDraft={onboarding.cashbackDraft}
            pushcardDraft={onboarding.pushcardDraft}
            rewardDrafts={onboarding.rewardDrafts}
            cashbackRewardDrafts={onboarding.cashbackRewardDrafts}
            collaboratorDrafts={onboarding.collaboratorDrafts}
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
