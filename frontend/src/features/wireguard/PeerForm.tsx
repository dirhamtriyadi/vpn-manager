import { useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { ChevronDown } from "lucide-react"
import { peerSchema, type PeerFormValues } from "@/schemas/peer"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { DialogClose, DialogFooter } from "@/components/ui/dialog"
import { applyServerValidationErrors } from "@/lib/api"

interface Props {
  onSubmit: (values: PeerFormValues) => Promise<void> | void
  submitting?: boolean
}

export function PeerForm({ onSubmit, submitting }: Props) {
  const [showAdvanced, setShowAdvanced] = useState(false)
  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<PeerFormValues>({
    resolver: zodResolver(peerSchema),
    defaultValues: {
      name: "",
      public_key: "",
      assigned_ip: "",
      client_allowed_ips: "0.0.0.0/0, ::/0",
      persistent_keepalive: 25,
      use_preshared_key: true,
    },
  })

  async function handleValidSubmit(values: PeerFormValues) {
    try {
      await onSubmit(values)
    } catch (err) {
      applyServerValidationErrors<PeerFormValues>(setError, err)
    }
  }

  return (
    <form onSubmit={handleSubmit(handleValidSubmit)} className="space-y-4">
      <div className="space-y-1.5">
        <Label htmlFor="name">Peer name</Label>
        <Input id="name" placeholder="laptop-andi" {...register("name")} />
        {errors.name && (
          <p className="text-xs text-destructive">{errors.name.message}</p>
        )}
      </div>

      <div className="flex items-center gap-2">
        <input
          id="use_preshared_key"
          type="checkbox"
          className="h-4 w-4 rounded border-input"
          {...register("use_preshared_key")}
        />
        <Label htmlFor="use_preshared_key">
          Use preshared key (extra hardening)
        </Label>
      </div>

      <button
        type="button"
        onClick={() => setShowAdvanced((v) => !v)}
        className="flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground"
      >
        <ChevronDown
          className={`h-4 w-4 transition-transform ${showAdvanced ? "rotate-180" : ""}`}
        />
        Advanced (optional)
      </button>

      {showAdvanced && (
        <div className="space-y-4 rounded-md border border-dashed p-3">
          <div className="space-y-1.5">
            <Label htmlFor="public_key">Client public key</Label>
            <Input
              id="public_key"
              placeholder="leave blank to auto-generate"
              {...register("public_key")}
            />
            <p className="text-xs text-muted-foreground">
              If empty, the server generates the key pair and gives you a full
              config + QR.
            </p>
          </div>

          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-1.5">
              <Label htmlFor="assigned_ip">Assigned IP</Label>
              <Input
                id="assigned_ip"
                placeholder="auto (next free)"
                {...register("assigned_ip")}
              />
              {errors.assigned_ip && (
                <p className="text-xs text-destructive">
                  {errors.assigned_ip.message}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="persistent_keepalive">Keepalive (s)</Label>
              <Input
                id="persistent_keepalive"
                type="number"
                {...register("persistent_keepalive")}
              />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="client_allowed_ips">Client AllowedIPs</Label>
            <Input
              id="client_allowed_ips"
              placeholder="0.0.0.0/0, ::/0"
              {...register("client_allowed_ips")}
            />
            <p className="text-xs text-muted-foreground">
              What the client routes through the tunnel.
            </p>
          </div>
        </div>
      )}

      <DialogFooter>
        <DialogClose asChild>
          <Button type="button" variant="outline">
            Cancel
          </Button>
        </DialogClose>
        <Button type="submit" disabled={submitting}>
          {submitting ? "Adding..." : "Add peer"}
        </Button>
      </DialogFooter>
    </form>
  )
}
