import { useEffect } from "react"
import { useForm } from "react-hook-form"
import { z } from "zod/v4"
import { zodResolver } from "@hookform/resolvers/zod"
import { toast } from "sonner"
import { GoogleLogin, type CredentialResponse } from "@react-oauth/google"
import { useAuth } from "@/context/auth-context"
import { useCustomer, useUpdateCustomer } from "@/hooks/use-customer"
import { useMe, useLinkGoogle, useUnlinkGoogle } from "@/hooks/use-me"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { GlassCard, GlassCardContent, GlassCardDescription, GlassCardHeader, GlassCardTitle } from "@/components/ui/glass-card"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form"

const profileSchema = z.object({
  name: z.string().min(1, "El nombre es requerido"),
  slug: z.string().min(1, "El slug es requerido"),
  phone: z.string(),
  address: z.string(),
  description: z.string(),
  welcome_message: z.string(),
  logo_url: z.string(),
})

type ProfileFormValues = z.infer<typeof profileSchema>

export function ProfilePage() {
  const { customerId } = useAuth()
  const { data: customer, isLoading } = useCustomer(customerId)
  const updateCustomer = useUpdateCustomer(customerId)
  const { data: me, isLoading: meLoading } = useMe()
  const linkGoogle = useLinkGoogle()
  const unlinkGoogle = useUnlinkGoogle()

  const form = useForm<ProfileFormValues>({
    resolver: zodResolver(profileSchema),
    defaultValues: {
      name: "",
      slug: "",
      phone: "",
      address: "",
      description: "",
      welcome_message: "",
      logo_url: "",
    },
  })

  useEffect(() => {
    if (customer) {
      form.reset({
        name: customer.name,
        slug: customer.slug,
        phone: customer.phone,
        address: customer.address,
        description: customer.description,
        welcome_message: customer.welcome_message,
        logo_url: customer.logo_url,
      })
    }
  }, [customer, form])

  const onSubmit = (values: ProfileFormValues) => {
    updateCustomer.mutate(values, {
      onSuccess: () => toast.success("Negocio actualizado"),
      onError: (err) => toast.error(err.message),
    })
  }

  const handleGoogleLink = (response: CredentialResponse) => {
    if (!response.credential) {
      toast.error("Google no devolvió credencial")
      return
    }
    linkGoogle.mutate(response.credential, {
      onSuccess: () => toast.success("Cuenta de Google vinculada"),
      onError: (err) => toast.error(err.message || "No se pudo vincular Google"),
    })
  }

  const handleGoogleUnlink = () => {
    unlinkGoogle.mutate(undefined, {
      onSuccess: () => toast.success("Cuenta de Google desvinculada"),
      onError: (err) => toast.error(err.message || "No se pudo desvincular Google"),
    })
  }

  const googleClientId = import.meta.env.VITE_GOOGLE_CLIENT_ID

  if (isLoading) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-96 w-full" />
      </div>
    )
  }

  return (
    <div className="space-y-6">
      <h1 className="text-2xl font-bold">Mi Negocio</h1>

      <GlassCard>
        <GlassCardHeader>
          <GlassCardTitle>Informacion del negocio</GlassCardTitle>
          <GlassCardDescription>Actualiza los datos de tu establecimiento</GlassCardDescription>
        </GlassCardHeader>
        <GlassCardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <div className="grid gap-4 sm:grid-cols-2">
                <FormField
                  control={form.control}
                  name="name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Nombre</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="slug"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Slug</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="phone"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Telefono</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="address"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Direccion</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name="description"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Descripcion</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="welcome_message"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Mensaje de bienvenida</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="logo_url"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>URL del logo</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button type="submit" disabled={updateCustomer.isPending}>
                {updateCustomer.isPending ? "Guardando..." : "Guardar cambios"}
              </Button>
            </form>
          </Form>
        </GlassCardContent>
      </GlassCard>

      <GlassCard>
        <GlassCardHeader>
          <GlassCardTitle>Cuenta de Google</GlassCardTitle>
          <GlassCardDescription>
            Vincula tu cuenta Google para iniciar sesion con un click.
          </GlassCardDescription>
        </GlassCardHeader>
        <GlassCardContent>
          {meLoading ? (
            <Skeleton className="h-12 w-full" />
          ) : me?.google_email ? (
            <div className="flex items-center justify-between gap-4">
              <div>
                <p className="text-sm font-medium">Cuenta vinculada</p>
                <p className="text-sm text-muted-foreground">{me.google_email}</p>
              </div>
              <Button
                variant="outline"
                onClick={handleGoogleUnlink}
                disabled={unlinkGoogle.isPending}
              >
                {unlinkGoogle.isPending ? "Desvinculando..." : "Desvincular"}
              </Button>
            </div>
          ) : googleClientId ? (
            <div className="flex flex-col gap-3">
              <p className="text-sm text-muted-foreground">
                No tienes ninguna cuenta de Google vinculada.
              </p>
              <div className="flex justify-start">
                <GoogleLogin
                  onSuccess={handleGoogleLink}
                  onError={() => toast.error("Error al conectar con Google")}
                  text="continue_with"
                  shape="rectangular"
                  size="large"
                />
              </div>
            </div>
          ) : (
            <p className="text-sm text-muted-foreground">
              Configura <code>VITE_GOOGLE_CLIENT_ID</code> para habilitar Google.
            </p>
          )}
        </GlassCardContent>
      </GlassCard>
    </div>
  )
}
