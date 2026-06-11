import axios from "axios"
import type { FieldValues, Path, UseFormSetError } from "react-hook-form"

export const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1"

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { "Content-Type": "application/json" },
})

api.interceptors.response.use(
  (response) => response,
  (error) => Promise.reject(error),
)

export type ApiValidationErrors = Record<string, string[]>

interface ApiErrorItem {
  field?: string
  message: string
}

interface ApiErrorPayload {
  success?: boolean
  message?: string
  errors?: ApiValidationErrors | ApiErrorItem[]
}

function errorPayload(err: unknown): ApiErrorPayload | undefined {
  if (typeof err === "object" && err !== null && "response" in err) {
    return (err as { response?: { data?: ApiErrorPayload } }).response?.data
  }
  return undefined
}

/** Extract a human-readable message from an axios error response. */
export function apiErrorMessage(err: unknown, fallback = "Request failed"): string {
  return errorPayload(err)?.message ?? fallback
}

/** Return Laravel-style validation errors from the backend, if present. */
export function apiValidationErrors(err: unknown): ApiValidationErrors | null {
  const errors = errorPayload(err)?.errors
  if (!errors || Array.isArray(errors)) return null
  return errors
}

/**
 * Apply backend validation errors to React Hook Form fields.
 * Backend errors must use JSON/form field names, e.g. "listen_port".
 */
export function applyServerValidationErrors<T extends FieldValues>(
  setError: UseFormSetError<T>,
  err: unknown,
): boolean {
  const errors = apiValidationErrors(err)
  if (!errors) return false

  Object.entries(errors).forEach(([field, messages]) => {
    setError(field as Path<T>, {
      type: "server",
      message: messages[0] ?? "Invalid value",
    })
  })
  return true
}
