import { z } from "zod"

const ipRegex = /^(\d{1,3}\.){3}\d{1,3}$/

export const peerSchema = z.object({
  name: z.string().min(1, "Name is required").max(128),
  public_key: z.string().optional().or(z.literal("")),
  assigned_ip: z
    .string()
    .regex(ipRegex, "Must be an IPv4 address")
    .optional()
    .or(z.literal("")),
  client_allowed_ips: z.string().optional().or(z.literal("")),
  persistent_keepalive: z.coerce.number().int().min(0).max(65535).default(25),
  use_preshared_key: z.boolean().default(true),
})

export type PeerFormValues = z.infer<typeof peerSchema>
