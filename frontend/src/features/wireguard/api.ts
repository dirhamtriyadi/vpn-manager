import { api } from "@/lib/api"
import type { InterfaceFormValues } from "@/schemas/interface"
import type { PeerFormValues } from "@/schemas/peer"
import type {
  ApiResponse,
  InterfaceStatus,
  ListParams,
  PaginatedResult,
  PaginationMeta,
  Peer,
  WGInterface,
} from "./types"

const emptyMeta: PaginationMeta = { page: 1, per_page: 10, total: 0, last_page: 1 }

function paginated<T>(response: ApiResponse<T[]>): PaginatedResult<T> {
  return { data: response.data ?? [], meta: response.meta ?? emptyMeta }
}

// ---- interfaces ----

export async function listInterfaces(params: ListParams = {}): Promise<PaginatedResult<WGInterface>> {
  const { data } = await api.get<ApiResponse<WGInterface[]>>("/interfaces", { params })
  return paginated(data)
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

export async function updateInterface(
  id: number,
  payload: InterfaceFormValues,
): Promise<{ data: WGInterface; message?: string }> {
  const { data } = await api.put<ApiResponse<WGInterface>>(
    `/interfaces/${id}`,
    payload,
  )
  return { data: data.data, message: data.message }
}

export async function deleteInterface(id: number): Promise<string | undefined> {
  const { data } = await api.delete<ApiResponse<unknown>>(`/interfaces/${id}`)
  return data.message
}

export async function listTrashedInterfaces(params: ListParams = {}): Promise<PaginatedResult<WGInterface>> {
  const { data } = await api.get<ApiResponse<WGInterface[]>>("/interfaces/trash", { params })
  return paginated(data)
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
  params: ListParams = {},
): Promise<{ data: InterfaceStatus; meta: PaginationMeta }> {
  const { data } = await api.get<ApiResponse<InterfaceStatus>>(
    `/interfaces/${id}/status`,
    { params },
  )
  return { data: data.data, meta: data.meta ?? emptyMeta }
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

export async function listTrashedPeers(params: ListParams = {}): Promise<PaginatedResult<Peer>> {
  const { data } = await api.get<ApiResponse<Peer[]>>("/peers/trash", { params })
  return paginated(data)
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

export async function getPeerQrCodeObjectUrl(peerId: number): Promise<string> {
  const { data } = await api.get<Blob>(`/peers/${peerId}/qrcode`, {
    responseType: "blob",
  })
  return URL.createObjectURL(data)
}

export async function downloadPeerConfigFile(peerId: number, filename: string): Promise<void> {
  const config = await getPeerConfigText(peerId)
  const blob = new Blob([config], { type: "text/plain;charset=utf-8" })
  const url = URL.createObjectURL(blob)
  try {
    const link = document.createElement("a")
    link.href = url
    link.download = filename
    document.body.appendChild(link)
    link.click()
    link.remove()
  } finally {
    URL.revokeObjectURL(url)
  }
}
