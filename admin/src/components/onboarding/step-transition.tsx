import type { ReactNode } from "react"

interface StepTransitionProps {
  stepKey: number
  direction: "forward" | "backward"
  children: ReactNode
}

export function StepTransition({ stepKey, direction, children }: StepTransitionProps) {
  const animation =
    direction === "forward"
      ? "fade-slide-in-right 300ms ease-out"
      : "fade-slide-in-left 300ms ease-out"

  return (
    <div key={stepKey} style={{ animation }}>
      {children}
    </div>
  )
}
