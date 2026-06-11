import axios from "axios"

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

/** Extract a human-readable message from an axios error response. */
export function apiErrorMessage(err: unknown, fallback = "Request failed"): string {
  if (typeof err === "object" && err !== null && "response" in err) {
    const resp = (err as { response?: { data?: { message?: string } } }).response
    if (resp?.data?.message) return resp.data.message
  }
  return fallback
}
