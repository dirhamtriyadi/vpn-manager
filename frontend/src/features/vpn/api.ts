import { api } from "@/lib/api"
import type { ApiResponse } from "@/features/wireguard/types"
import type {
  ListParams,
  PaginatedResult,
  PaginationMeta,
  VPNInstance,
  VPNInstanceStatus,
  VPNProtocolInfo,
  VPNUser,
} from "./types"

const emptyMeta: PaginationMeta = { page: 1, per_page: 10, total: 0, last_page: 1 }

function paginated<T>(response: ApiResponse<T[]>): PaginatedResult<T> {
  return { data: response.data ?? [], meta: response.meta ?? emptyMeta }
}

export async function listVPNProtocols(): Promise<VPNProtocolInfo[]> {
  const { data } = await api.get<ApiResponse<VPNProtocolInfo[]>>("/vpn/protocols")
  return data.data ?? []
}

export async function listVPNInstances(params: ListParams = {}): Promise<PaginatedResult<VPNInstance>> {
  const { data } = await api.get<ApiResponse<VPNInstance[]>>("/vpn/instances", { params })
  return paginated(data)
}

export async function getVPNInstance(id: number): Promise<VPNInstance> {
  const { data } = await api.get<ApiResponse<VPNInstance>>(`/vpn/instances/${id}`)
  return data.data
}

export async function listVPNInstanceUsers(
  id: number,
  params: ListParams = {},
): Promise<PaginatedResult<VPNUser>> {
  const { data } = await api.get<ApiResponse<VPNUser[]>>(`/vpn/instances/${id}/users`, { params })
  return paginated(data)
}

export async function getVPNInstanceStatus(
  id: number,
  params: ListParams = {},
): Promise<{ data: VPNInstanceStatus; meta: PaginationMeta }> {
  const { data } = await api.get<ApiResponse<VPNInstanceStatus>>(`/vpn/instances/${id}/status`, { params })
  return { data: data.data, meta: data.meta ?? emptyMeta }
}
