import axios from "axios"
import type { FieldValues, Path, UseFormSetError } from "react-hook-form"

export const API_BASE_URL =
  import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080/api/v1"

export const api = axios.create({
  baseURL: API_BASE_URL,
  headers: { "Content-Type": "application/json" },
})

const TOKEN_KEY = "vpn_manager_token"

/** Read the stored bearer token, if any. */
export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

/** Persist the bearer token for subsequent requests. */
export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

/** Remove the stored bearer token. */
export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

// Called when the server rejects a token (401) on a protected request so the
// app can drop the session and show the login screen.
let onUnauthorized: (() => void) | null = null
export function setUnauthorizedHandler(handler: (() => void) | null): void {
  onUnauthorized = handler
}

api.interceptors.request.use((config) => {
  const token = getToken()
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

api.interceptors.response.use(
  (response) => response,
  (error) => {
    const status = error?.response?.status
    const url: string = error?.config?.url ?? ""
    // Ignore the login endpoint: a 401 there is "wrong credentials", not an
    // expired session, and the form surfaces that message itself.
    if (status === 401 && !url.includes("/auth/login")) {
      clearToken()
      onUnauthorized?.()
    }
    return Promise.reject(error)
  },
)

export type ApiValidationErrors = Record<string, string[]>

// Backend error envelope:
//   generic:    { success: false, code, message }
//   validation: { success: false, code: "VALIDATION_ERROR", message, errors: { field: [...] } }
// `code` is a stable machine-readable string (e.g. "CONFLICT", "NOT_FOUND") that
// the UI can branch on instead of matching the human-readable message.
interface ApiErrorPayload {
  success?: boolean
  code?: string
  message?: string
  errors?: ApiValidationErrors
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
