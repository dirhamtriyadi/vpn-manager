import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { interfaceSchema, type InterfaceFormValues } from "@/schemas/interface"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { DialogClose, DialogFooter } from "@/components/ui/dialog"

interface Props {
  onSubmit: (values: InterfaceFormValues) => Promise<void> | void
  submitting?: boolean
}

function Field({
  id,
  label,
  hint,
  error,
  children,
}: {
  id: string
  label: string
  hint?: string
  error?: string
  children: React.ReactNode
}) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      {children}
      {hint && !error && (
        <p className="text-xs text-muted-foreground">{hint}</p>
      )}
      {error && <p className="text-xs text-destructive">{error}</p>}
    </div>
  )
}

export function InterfaceForm({ onSubmit, submitting }: Props) {
  const {
    register,
    handleSubmit,
    formState: { errors },
  } = useForm<InterfaceFormValues>({
    resolver: zodResolver(interfaceSchema),
    defaultValues: {
      name: "wg0",
      listen_port: 51820,
      address: "10.8.0.1/24",
      endpoint: "",
      dns: "1.1.1.1",
      mtu: 1420,
      private_key: "",
    },
  })

  return (
    <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
      <div className="grid grid-cols-2 gap-4">
        <Field id="name" label="Name" error={errors.name?.message}>
          <Input id="name" placeholder="wg0" {...register("name")} />
        </Field>
        <Field
          id="listen_port"
          label="Listen port"
          error={errors.listen_port?.message}
        >
          <Input
            id="listen_port"
            type="number"
            {...register("listen_port")}
          />
        </Field>
      </div>

      <Field
        id="address"
        label="Server address (CIDR)"
        hint="The concentrator's tunnel IP and subnet."
        error={errors.address?.message}
      >
        <Input id="address" placeholder="10.8.0.1/24" {...register("address")} />
      </Field>

      <Field
        id="endpoint"
        label="Public endpoint"
        hint="Host clients dial (domain or public IP)."
        error={errors.endpoint?.message}
      >
        <Input
          id="endpoint"
          placeholder="vpn.example.com"
          {...register("endpoint")}
        />
      </Field>

      <div className="grid grid-cols-2 gap-4">
        <Field id="dns" label="Client DNS" error={errors.dns?.message}>
          <Input id="dns" placeholder="1.1.1.1" {...register("dns")} />
        </Field>
        <Field id="mtu" label="MTU" error={errors.mtu?.message}>
          <Input id="mtu" type="number" {...register("mtu")} />
        </Field>
      </div>

      <Field
        id="private_key"
        label="Private key (optional)"
        hint="Leave blank to auto-generate a key pair."
        error={errors.private_key?.message}
      >
        <Input
          id="private_key"
          placeholder="auto-generated if empty"
          {...register("private_key")}
        />
      </Field>

      <DialogFooter>
        <DialogClose asChild>
          <Button type="button" variant="outline">
            Cancel
          </Button>
        </DialogClose>
        <Button type="submit" disabled={submitting}>
          {submitting ? "Creating..." : "Create interface"}
        </Button>
      </DialogFooter>
    </form>
  )
}
