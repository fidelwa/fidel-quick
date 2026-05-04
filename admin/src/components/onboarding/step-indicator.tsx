import { Check } from "lucide-react"
import { cn } from "@/lib/utils"

const steps = ["Programa", "Recompensas", "Equipo", "Listo"]

interface StepIndicatorProps {
  currentStep: number
  completedSteps: number[]
}

export function StepIndicator({ currentStep, completedSteps }: StepIndicatorProps) {
  return (
    <div className="flex items-center justify-center gap-0">
      {steps.map((label, i) => {
        const step = i + 1
        const isCompleted = completedSteps.includes(step)
        const isCurrent = currentStep === step
        const isLast = i === steps.length - 1

        return (
          <div key={step} className="flex items-center">
            <div className="flex flex-col items-center gap-1.5">
              <div
                className={cn(
                  "flex h-8 w-8 items-center justify-center rounded-full border-2 text-sm font-medium transition-all duration-300 sm:h-9 sm:w-9",
                  isCompleted &&
                    "border-green-600 bg-green-600 text-white",
                  isCurrent &&
                    !isCompleted &&
                    "border-primary bg-primary text-primary-foreground ring-4 ring-primary/20",
                  !isCompleted &&
                    !isCurrent &&
                    "border-muted-foreground/30 text-muted-foreground"
                )}
                style={
                  isCompleted
                    ? { animation: "pulse-success 2s ease-in-out infinite" }
                    : undefined
                }
              >
                {isCompleted ? (
                  <Check
                    className="h-4 w-4"
                    style={{ animation: "check-pop 300ms ease-out" }}
                  />
                ) : (
                  step
                )}
              </div>
              <span
                className={cn(
                  "hidden text-xs font-medium sm:block",
                  isCurrent || isCompleted
                    ? "text-foreground"
                    : "text-muted-foreground"
                )}
              >
                {label}
              </span>
            </div>

            {!isLast && (
              <div className="mx-2 h-0.5 w-8 overflow-hidden rounded-full bg-muted sm:mx-3 sm:w-12">
                <div
                  className={cn(
                    "h-full origin-left rounded-full bg-green-600 transition-transform duration-400",
                    isCompleted ? "scale-x-100" : "scale-x-0"
                  )}
                  style={
                    isCompleted
                      ? { animation: "line-fill 400ms ease-out forwards" }
                      : undefined
                  }
                />
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}
