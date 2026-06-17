import type { ListParams, PaginatedResult, PaginationMeta } from "@/features/wireguard/types"

export type VPNProtocol = "wireguard" | "openvpn" | "l2tp_ipsec" | "sstp" | "pptp"

export interface VPNProtocolInfo {
  id: VPNProtocol
  label: string
  available: boolean
  legacy_insecure: boolean
  runtime_strategy?: string
  config_download: boolean
  qr_code: boolean
  requires_certificates: boolean
}

export interface WireGuardInstanceMeta {
  public_key: string
  dns: string
  mtu: number
  masquerade: boolean
  egress_interface: string
}

export interface WireGuardUserMeta {
  public_key: string
  allowed_ips: string
  client_allowed_ips: string
  persistent_keepalive: number
}

export interface VPNInstance {
  id: number
  protocol: VPNProtocol
  name: string
  listen_port: number
  address: string
  endpoint: string
  enabled: boolean
  status?: string
  legacy_insecure: boolean
  wireguard?: WireGuardInstanceMeta
}

export interface VPNUser {
  id: number
  instance_id: number
  protocol: VPNProtocol
  name: string
  assigned_ip: string
  enabled: boolean
  online?: boolean
  rx_bytes?: number
  tx_bytes?: number
  wireguard?: WireGuardUserMeta
}

export interface VPNInstanceStatus {
  instance: VPNInstance
  users: VPNUser[]
  kernel_up: boolean
  kernel_message?: string
}

export type { ListParams, PaginatedResult, PaginationMeta }
