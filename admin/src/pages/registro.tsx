import { useState, useRef, useEffect } from "react"
import { useNavigate, Navigate, Link } from "react-router-dom"
import { useForm } from "react-hook-form"
import { z } from "zod"
import { zodResolver } from "@hookform/resolvers/zod"
import { useAuth } from "@/context/auth-context"
import { onboardingRegister, setToken } from "@/lib/api-client"
import { useSlugCheck } from "@/hooks/use-slug-check"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Textarea } from "@/components/ui/textarea"
import { Label } from "@/components/ui/label"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { toast } from "sonner"
import { Loader2, Check, X } from "lucide-react"

const registroSchema = z
  .object({
    name: z.string().min(1, "El nombre es requerido"),
    slug: z
      .string()
      .min(3, "Minimo 3 caracteres")
      .max(50, "Maximo 50 caracteres")
      .regex(/^[a-z0-9-]+$/, "Solo letras minusculas, numeros y guiones"),
    phone: z.string().min(10, "Minimo 10 digitos"),
    description: z.string().optional(),
    admin_email: z.string().email("Email invalido"),
    admin_password: z.string().min(8, "Minimo 8 caracteres"),
    confirm_password: z.string(),
  })
  .refine((data) => data.admin_password === data.confirm_password, {
    message: "Las contraseñas no coinciden",
    path: ["confirm_password"],
  })

type RegistroForm = z.infer<typeof registroSchema>

function nameToSlug(name: string) {
  return name
    .toLowerCase()
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .replace(/[^a-z0-9\s-]/g, "")
    .replace(/\s+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")
}

export function RegistroPage() {
  const { isAuthenticated, login } = useAuth()
  const navigate = useNavigate()
  const [loading, setLoading] = useState(false)
  const slugManuallyEdited = useRef(false)

  const form = useForm<RegistroForm>({
    resolver: zodResolver(registroSchema),
    defaultValues: {
      name: "",
      slug: "",
      phone: "",
      description: "",
      admin_email: "",
      admin_password: "",
      confirm_password: "",
    },
  })

  const slugValue = form.watch("slug")
  const nameValue = form.watch("name")
  const { isAvailable, isChecking } = useSlugCheck(slugValue)

  useEffect(() => {
    if (!slugManuallyEdited.current && nameValue) {
      form.setValue("slug", nameToSlug(nameValue))
    }
  }, [nameValue, form])

  if (isAuthenticated) {
    return <Navigate to="/" replace />
  }

  const onSubmit = async (data: RegistroForm) => {
    if (isAvailable === false) {
      toast.error("El slug no esta disponible")
      return
    }

    setLoading(true)
    try {
      const res = await onboardingRegister({
        name: data.name,
        slug: data.slug,
        phone: data.phone,
        description: data.description,
        admin_email: data.admin_email,
        admin_password: data.admin_password,
      })
      setToken(res.token)
      login(res.token, res.admin.customer_id, res.admin.email)
      navigate("/onboarding")
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Error al registrar")
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="flex min-h-screen items-center justify-center p-4">
      <Card className="w-full max-w-lg">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Crea tu programa de fidelidad</CardTitle>
          <CardDescription>
            Registra tu negocio y configura todo en minutos
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="name">Nombre del negocio</Label>
              <Input
                id="name"
                placeholder="Mi Restaurante"
                {...form.register("name")}
              />
              {form.formState.errors.name && (
                <p className="text-sm text-destructive">{form.formState.errors.name.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="slug">Slug (URL)</Label>
              <div className="relative">
                <Input
                  id="slug"
                  placeholder="mi-restaurante"
                  {...form.register("slug", {
                    onChange: () => {
                      slugManuallyEdited.current = true
                    },
                  })}
                />
                <div className="absolute right-2.5 top-1/2 -translate-y-1/2">
                  {isChecking && (
                    <Loader2 className="h-4 w-4 animate-spin text-muted-foreground" />
                  )}
                  {!isChecking && isAvailable === true && (
                    <Check className="h-4 w-4 text-green-600" />
                  )}
                  {!isChecking && isAvailable === false && (
                    <X className="h-4 w-4 text-destructive" />
                  )}
                </div>
              </div>
              {slugValue.length >= 3 && (
                <p className="text-sm text-muted-foreground">
                  Tu URL sera: fidel.app/unirse/<span className="font-semibold text-foreground">{slugValue}</span>
                </p>
              )}
              {form.formState.errors.slug && (
                <p className="text-sm text-destructive">{form.formState.errors.slug.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="phone">Telefono del negocio</Label>
              <Input
                id="phone"
                placeholder="+525512345678"
                {...form.register("phone")}
              />
              {form.formState.errors.phone && (
                <p className="text-sm text-destructive">{form.formState.errors.phone.message}</p>
              )}
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Descripcion (opcional)</Label>
              <Textarea
                id="description"
                placeholder="Describe brevemente tu negocio..."
                rows={3}
                {...form.register("description")}
              />
            </div>

            <div className="space-y-2">
              <Label htmlFor="admin_email">Email del administrador</Label>
              <Input
                id="admin_email"
                type="email"
                placeholder="admin@minegocio.com"
                {...form.register("admin_email")}
              />
              {form.formState.errors.admin_email && (
                <p className="text-sm text-destructive">{form.formState.errors.admin_email.message}</p>
              )}
            </div>

            <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
              <div className="space-y-2">
                <Label htmlFor="admin_password">Password</Label>
                <Input
                  id="admin_password"
                  type="password"
                  placeholder="Minimo 8 caracteres"
                  {...form.register("admin_password")}
                />
                {form.formState.errors.admin_password && (
                  <p className="text-sm text-destructive">{form.formState.errors.admin_password.message}</p>
                )}
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm_password">Confirmar password</Label>
                <Input
                  id="confirm_password"
                  type="password"
                  placeholder="Repite tu password"
                  {...form.register("confirm_password")}
                />
                {form.formState.errors.confirm_password && (
                  <p className="text-sm text-destructive">{form.formState.errors.confirm_password.message}</p>
                )}
              </div>
            </div>

            <Button type="submit" className="w-full" disabled={loading}>
              {loading ? (
                <>
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                  Registrando...
                </>
              ) : (
                "Crear cuenta"
              )}
            </Button>
          </form>

          <p className="mt-4 text-center text-sm text-muted-foreground">
            ¿Ya tienes cuenta?{" "}
            <Link to="/login" className="text-primary underline-offset-4 hover:underline">
              Inicia sesion
            </Link>
          </p>
        </CardContent>
      </Card>
    </div>
  )
}
