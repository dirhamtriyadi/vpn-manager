import { z } from "zod"

export const roleSchema = z.object({
  name: z
    .string()
    .min(2, "Name must be at least 2 characters")
    .max(64)
    .regex(/^[a-zA-Z0-9_-]+$/, "Use letters, numbers, - or _ only"),
  description: z.string().max(255).optional().or(z.literal("")),
})

export type RoleFormValues = z.infer<typeof roleSchema>
