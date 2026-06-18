import { api } from "@/lib/api"
import type { ApiResponse } from "@/features/wireguard/types"
import type {
  ListParams,
  PaginatedResult,
  PaginationMeta,
  OpenVPNInstanceDraft,
  OpenVPNLifecyclePlan,
  OpenVPNFirewallPlan,
  OpenVPNPersistedRuntimeManifest,
  OpenVPNRoadmap,
  OpenVPNRuntimeManifest,
  OpenVPNRuntimeManifestPreviewRequest,
  ProtocolRoadmap,
  ProtocolServicePlan,
  VPNInstance,
  VPNInstanceStatus,
  VPNProtocolInfo,
  VPNProtocol,
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

export async function getProtocolRoadmap(protocol: VPNProtocol): Promise<ProtocolRoadmap> {
  const { data } = await api.get<ApiResponse<ProtocolRoadmap>>(`/vpn/roadmaps/${protocol}`)
  return data.data
}

export async function getProtocolServicePlan(protocol: VPNProtocol): Promise<ProtocolServicePlan> {
  const { data } = await api.get<ApiResponse<ProtocolServicePlan>>(`/vpn/service-plans/${protocol}`)
  return data.data
}

export async function getOpenVPNRoadmap(): Promise<OpenVPNRoadmap> {
  const { data } = await api.get<ApiResponse<OpenVPNRoadmap>>("/vpn/openvpn/roadmap")
  return data.data
}

export async function listOpenVPNInstanceDrafts(params: ListParams = {}): Promise<PaginatedResult<OpenVPNInstanceDraft>> {
  const { data } = await api.get<ApiResponse<OpenVPNInstanceDraft[]>>("/vpn/openvpn/instances", { params })
  return paginated(data)
}

export async function getOpenVPNRuntimeManifest(instanceId: number): Promise<OpenVPNPersistedRuntimeManifest> {
  const { data } = await api.get<ApiResponse<OpenVPNPersistedRuntimeManifest>>(`/vpn/openvpn/instances/${instanceId}/runtime-manifest`)
  return data.data
}

export async function generateOpenVPNRuntimeManifest(instanceId: number): Promise<OpenVPNPersistedRuntimeManifest> {
  const { data } = await api.post<ApiResponse<OpenVPNPersistedRuntimeManifest>>(`/vpn/openvpn/instances/${instanceId}/runtime-manifest`)
  return data.data
}

export async function planOpenVPNLifecycle(instanceId: number, action = "start"): Promise<OpenVPNLifecyclePlan> {
  const { data } = await api.post<ApiResponse<OpenVPNLifecyclePlan>>(`/vpn/openvpn/instances/${instanceId}/lifecycle/${action}`)
  return data.data
}

export async function planOpenVPNFirewall(instanceId: number): Promise<OpenVPNFirewallPlan> {
  const { data } = await api.post<ApiResponse<OpenVPNFirewallPlan>>(`/vpn/openvpn/instances/${instanceId}/firewall-plan`)
  return data.data
}

export async function previewOpenVPNRuntimeManifest(
  payload: OpenVPNRuntimeManifestPreviewRequest,
): Promise<OpenVPNRuntimeManifest> {
  const { data } = await api.post<ApiResponse<OpenVPNRuntimeManifest>>("/vpn/openvpn/runtime/preview", payload)
  return data.data
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
