import { api, API_BASE_URL } from "@/lib/api"
import type { InterfaceFormValues } from "@/schemas/interface"
import type { PeerFormValues } from "@/schemas/peer"
import type {
  ApiResponse,
  InterfaceStatus,
  Peer,
  WGInterface,
} from "./types"

// ---- interfaces ----

export async function listInterfaces(): Promise<WGInterface[]> {
  const { data } = await api.get<ApiResponse<WGInterface[]>>("/interfaces")
  return data.data ?? []
}

export async function createInterface(
  payload: InterfaceFormValues,
): Promise<{ data: WGInterface; message?: string }> {
  const { data } = await api.post<ApiResponse<WGInterface>>(
    "/interfaces",
    payload,
  )
  return { data: data.data, message: data.message }
}

export async function deleteInterface(id: number): Promise<string | undefined> {
  const { data } = await api.delete<ApiResponse<unknown>>(`/interfaces/${id}`)
  return data.message
}

export async function listTrashedInterfaces(): Promise<WGInterface[]> {
  const { data } = await api.get<ApiResponse<WGInterface[]>>("/interfaces/trash")
  return data.data ?? []
}

export async function restoreInterface(id: number): Promise<string | undefined> {
  const { data } = await api.post<ApiResponse<WGInterface>>(`/interfaces/${id}/restore`)
  return data.message
}

export async function purgeInterface(id: number): Promise<string | undefined> {
  const { data } = await api.delete<ApiResponse<unknown>>(`/interfaces/${id}/purge`)
  return data.message
}

export async function syncInterface(id: number): Promise<string | undefined> {
  const { data } = await api.post<ApiResponse<unknown>>(
    `/interfaces/${id}/sync`,
  )
  return data.message
}

export async function getInterfaceStatus(
  id: number,
): Promise<InterfaceStatus> {
  const { data } = await api.get<ApiResponse<InterfaceStatus>>(
    `/interfaces/${id}/status`,
  )
  return data.data
}

// ---- peers ----

export async function createPeer(
  interfaceId: number,
  payload: PeerFormValues,
): Promise<{ data: Peer; message?: string }> {
  const { data } = await api.post<ApiResponse<Peer>>(
    `/interfaces/${interfaceId}/peers`,
    payload,
  )
  return { data: data.data, message: data.message }
}

export async function updatePeer(
  peerId: number,
  payload: { name: string; client_allowed_ips?: string; persistent_keepalive?: number; enabled?: boolean },
): Promise<Peer> {
  const { data } = await api.put<ApiResponse<Peer>>(`/peers/${peerId}`, payload)
  return data.data
}

export async function deletePeer(peerId: number): Promise<string | undefined> {
  const { data } = await api.delete<ApiResponse<unknown>>(`/peers/${peerId}`)
  return data.message
}

export async function listTrashedPeers(): Promise<Peer[]> {
  const { data } = await api.get<ApiResponse<Peer[]>>("/peers/trash")
  return data.data ?? []
}

export async function restorePeer(peerId: number): Promise<string | undefined> {
  const { data } = await api.post<ApiResponse<Peer>>(`/peers/${peerId}/restore`)
  return data.message
}

export async function purgePeer(peerId: number): Promise<string | undefined> {
  const { data } = await api.delete<ApiResponse<unknown>>(`/peers/${peerId}/purge`)
  return data.message
}

export async function getPeerConfigText(peerId: number): Promise<string> {
  const { data } = await api.get<string>(`/peers/${peerId}/config`, {
    responseType: "text",
  })
  return data
}

// Direct URLs (the backend sets attachment / png headers).
export function peerConfigUrl(peerId: number): string {
  return `${API_BASE_URL}/peers/${peerId}/config`
}

export function peerQrCodeUrl(peerId: number): string {
  return `${API_BASE_URL}/peers/${peerId}/qrcode`
}
