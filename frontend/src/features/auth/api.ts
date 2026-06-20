import { api } from "@/lib/api"

export interface LoginResult {
  token: string
  token_type: string
  expires_at: string
  username: string
}

/** The authenticated user with role names and the flattened effective
 *  permission set (roles ∪ direct), as returned by GET /auth/me. */
export interface CurrentUser {
  id: number
  username: string
  name: string
  active: boolean
  roles: string[]
  permissions: string[]
}

interface ApiEnvelope<T> {
  success: boolean
  message: string
  data: T
}

export async function login(username: string, password: string): Promise<LoginResult> {
  const { data } = await api.post<ApiEnvelope<LoginResult>>("/auth/login", {
    username,
    password,
  })
  return data.data
}

export async function getMe(): Promise<CurrentUser> {
  const { data } = await api.get<ApiEnvelope<CurrentUser>>("/auth/me")
  return data.data
}
