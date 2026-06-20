import { api } from "@/lib/api"
import type { ApiResponse, Peer, WGInterface } from "@/features/wireguard/types"
import type {
  CreatePortForwardPayload,
  PortForward,
  UpdatePortForwardPayload,
} from "./types"

export async function listPortForwards(): Promise<PortForward[]> {
  const { data } = await api.get<ApiResponse<PortForward[]>>("/port-forwards")
  return data.data ?? []
}

export async function createPortForward(payload: CreatePortForwardPayload): Promise<PortForward> {
  const { data } = await api.post<ApiResponse<PortForward>>("/port-forwards", payload)
  return data.data
}

export async function updatePortForward(
  id: number,
  payload: UpdatePortForwardPayload,
): Promise<PortForward> {
  const { data } = await api.put<ApiResponse<PortForward>>(`/port-forwards/${id}`, payload)
  return data.data
}

export async function deletePortForward(id: number): Promise<void> {
  await api.delete(`/port-forwards/${id}`)
}

// Helpers for the create form: list interfaces and a given interface's peers.
export async function listInterfacesForSelect(): Promise<WGInterface[]> {
  const { data } = await api.get<ApiResponse<WGInterface[]>>("/interfaces", {
    params: { per_page: 100 },
  })
  return data.data ?? []
}

export async function listPeersForSelect(interfaceId: number): Promise<Peer[]> {
  const { data } = await api.get<ApiResponse<Peer[]>>(`/interfaces/${interfaceId}/peers`, {
    params: { per_page: 100 },
  })
  return data.data ?? []
}
