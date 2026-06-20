import { z } from "zod"

// One schema drives both create and edit. Password is optional here (blank = keep
// current on edit); the create dialog enforces a non-empty password itself. A
// provided password must still be at least 8 characters.
export const userFormSchema = z.object({
  username: z
    .string()
    .min(3, "Username must be at least 3 characters")
    .max(64)
    .regex(/^[a-zA-Z0-9_.-]+$/, "Use letters, numbers, . _ - only"),
  name: z.string().max(128).optional().or(z.literal("")),
  password: z
    .string()
    .min(8, "Password must be at least 8 characters")
    .max(128)
    .optional()
    .or(z.literal("")),
  active: z.boolean().optional(),
})

export type UserFormValues = z.infer<typeof userFormSchema>
