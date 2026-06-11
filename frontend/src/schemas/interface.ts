import { z } from "zod"

const cidrRegex =
  /^(\d{1,3}\.){3}\d{1,3}\/\d{1,2}$/

export const interfaceSchema = z.object({
  name: z
    .string()
    .min(2, "Name is required")
    .max(32)
    .regex(/^[a-zA-Z0-9_-]+$/, "Use letters, numbers, - or _ only"),
  listen_port: z.coerce
    .number()
    .int()
    .min(1)
    .max(65535, "Port must be 1–65535"),
  address: z
    .string()
    .regex(cidrRegex, "Must be a CIDR, e.g. 10.8.0.1/24"),
  endpoint: z.string().min(3, "Public endpoint is required").max(255),
  dns: z.string().max(128).optional().or(z.literal("")),
  mtu: z.coerce.number().int().min(576).max(9000).optional(),
  private_key: z.string().optional().or(z.literal("")),
})

export type InterfaceFormValues = z.infer<typeof interfaceSchema>
