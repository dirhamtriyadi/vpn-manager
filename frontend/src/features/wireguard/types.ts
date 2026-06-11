export interface Peer {
  id: number
  interface_id: number
  name: string
  public_key: string
  allowed_ips: string
  assigned_ip: string
  client_allowed_ips: string
  persistent_keepalive: number
  enabled: boolean
  created_at: string
  updated_at: string
  deleted_at?: string | { Time?: string; Valid?: boolean } | null
  // runtime (from /status)
  last_handshake?: string
  rx_bytes: number
  tx_bytes: number
  online: boolean
}

export interface WGInterface {
  id: number
  name: string
  public_key: string
  listen_port: number
  address: string
  dns: string
  mtu: number
  endpoint: string
  enabled: boolean
  peers?: Peer[]
  created_at: string
  updated_at: string
  deleted_at?: string | { Time?: string; Valid?: boolean } | null
}

export interface InterfaceStatus {
  interface: WGInterface
  kernel_up: boolean
  kernel_message?: string
}

export interface PaginationMeta {
  page: number
  per_page: number
  total: number
  last_page: number
  sort_by?: string
  sort_order?: "asc" | "desc"
  search?: string
}

export interface ListParams {
  page?: number
  per_page?: number
  sort_by?: string
  sort_order?: "asc" | "desc"
  search?: string
}

export interface ApiResponse<T> {
  success: boolean
  message: string
  data: T
  meta?: PaginationMeta
}

export interface PaginatedResult<T> {
  data: T[]
  meta: PaginationMeta
}

export type ValidationErrors = Record<string, string[]>

export interface ApiErrorResponse {
  success: false
  message: string
  errors: ValidationErrors | Array<{ field?: string; message: string }>
}
