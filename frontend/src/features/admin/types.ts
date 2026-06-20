import type { PaginationMeta } from "@/features/wireguard/types"

export interface Permission {
  id: number
  name: string
  description?: string
}

export interface RoleBrief {
  id: number
  name: string
  description?: string
}

export interface Role {
  id: number
  name: string
  description: string
  permissions: Permission[]
  created_at: string
  updated_at: string
}

export interface User {
  id: number
  username: string
  name: string
  active: boolean
  roles: RoleBrief[]
  direct_permissions: Permission[]
  effective_permissions: string[]
  created_at: string
  updated_at: string
}

export interface UserListParams {
  page?: number
  per_page?: number
  search?: string
  sort_by?: string
  sort_order?: "asc" | "desc"
}

export type { PaginationMeta }
