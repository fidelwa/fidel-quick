import * as React from "react"

import { cn } from "@/lib/utils"

/**
 * GlassCard — superficie vidriada del sistema Glassmorphism Aurora.
 *
 * Usa este componente (en lugar del shadcn Card por defecto) para superficies
 * que deban respetar el lenguaje visual del panel: fondo translúcido, blur,
 * borde sutil y sombra suave sobre el gradiente aurora.
 *
 * Variantes:
 * - default → glass (rgba 0.65, blur 20)
 * - strong  → glass-strong (rgba 0.80, blur 24) — overlays, sidebar
 * - subtle  → glass-subtle (rgba 0.40, blur 12) — chips, separadores
 */
type GlassCardProps = React.ComponentProps<"div"> & {
  variant?: "default" | "strong" | "subtle"
}

function GlassCard({ className, variant = "default", ...props }: GlassCardProps) {
  const variantClass =
    variant === "strong" ? "glass-strong" : variant === "subtle" ? "glass-subtle" : "glass"
  return (
    <div
      data-slot="glass-card"
      className={cn(
        variantClass,
        "flex flex-col gap-6 rounded-2xl py-6 text-foreground",
        className,
      )}
      {...props}
    />
  )
}

function GlassCardHeader({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="glass-card-header"
      className={cn(
        "flex items-start justify-between gap-2 px-6",
        className,
      )}
      {...props}
    />
  )
}

function GlassCardTitle({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="glass-card-title"
      className={cn("text-base leading-none font-semibold tracking-tight", className)}
      {...props}
    />
  )
}

function GlassCardDescription({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div
      data-slot="glass-card-description"
      className={cn("text-muted-foreground text-sm", className)}
      {...props}
    />
  )
}

function GlassCardContent({ className, ...props }: React.ComponentProps<"div">) {
  return (
    <div data-slot="glass-card-content" className={cn("px-6", className)} {...props} />
  )
}

export {
  GlassCard,
  GlassCardHeader,
  GlassCardTitle,
  GlassCardDescription,
  GlassCardContent,
}
