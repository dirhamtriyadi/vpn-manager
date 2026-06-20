import { api } from "@/lib/api"
import type { ApiResponse, PaginationMeta } from "@/features/wireguard/types"
import type { Permission, Role, User, UserListParams } from "./types"

const emptyMeta: PaginationMeta = { page: 1, per_page: 10, total: 0, last_page: 1 }

// ---- payloads ----

export interface CreateUserPayload {
  username: string
  name?: string
  password: string
  active?: boolean
  role_ids?: number[]
  permission_ids?: number[]
}

export interface UpdateUserPayload {
  name?: string
  password?: string
  active?: boolean
}

export interface CreateRolePayload {
  name: string
  description?: string
  permission_ids?: number[]
}

export interface UpdateRolePayload {
  name: string
  description?: string
}

// ---- permissions ----

export async function listPermissions(): Promise<Permission[]> {
  const { data } = await api.get<ApiResponse<Permission[]>>("/permissions")
  return data.data ?? []
}

// ---- roles ----

export async function listRoles(): Promise<Role[]> {
  const { data } = await api.get<ApiResponse<Role[]>>("/roles")
  return data.data ?? []
}

export async function createRole(payload: CreateRolePayload): Promise<Role> {
  const { data } = await api.post<ApiResponse<Role>>("/roles", payload)
  return data.data
}

export async function updateRole(id: number, payload: UpdateRolePayload): Promise<Role> {
  const { data } = await api.put<ApiResponse<Role>>(`/roles/${id}`, payload)
  return data.data
}

export async function setRolePermissions(id: number, permissionIds: number[]): Promise<Role> {
  const { data } = await api.put<ApiResponse<Role>>(`/roles/${id}/permissions`, {
    permission_ids: permissionIds,
  })
  return data.data
}

export async function deleteRole(id: number): Promise<void> {
  await api.delete(`/roles/${id}`)
}

// ---- users ----

export async function listUsers(
  params: UserListParams = {},
): Promise<{ data: User[]; meta: PaginationMeta }> {
  const { data } = await api.get<ApiResponse<User[]>>("/users", { params })
  return { data: data.data ?? [], meta: data.meta ?? emptyMeta }
}

export async function createUser(payload: CreateUserPayload): Promise<User> {
  const { data } = await api.post<ApiResponse<User>>("/users", payload)
  return data.data
}

export async function updateUser(id: number, payload: UpdateUserPayload): Promise<User> {
  const { data } = await api.put<ApiResponse<User>>(`/users/${id}`, payload)
  return data.data
}

export async function setUserRoles(id: number, roleIds: number[]): Promise<User> {
  const { data } = await api.put<ApiResponse<User>>(`/users/${id}/roles`, {
    role_ids: roleIds,
  })
  return data.data
}

export async function setUserPermissions(id: number, permissionIds: number[]): Promise<User> {
  const { data } = await api.put<ApiResponse<User>>(`/users/${id}/permissions`, {
    permission_ids: permissionIds,
  })
  return data.data
}

export async function deleteUser(id: number): Promise<void> {
  await api.delete(`/users/${id}`)
}
