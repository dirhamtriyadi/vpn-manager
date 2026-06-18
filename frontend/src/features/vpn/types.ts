import type { ListParams, PaginatedResult, PaginationMeta } from "@/features/wireguard/types"

export type VPNProtocol = "wireguard" | "openvpn" | "l2tp_ipsec" | "sstp" | "pptp"

export interface VPNProtocolInfo {
  id: VPNProtocol
  label: string
  status: "available" | "roadmap" | "legacy_roadmap" | string
  description: string
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

export interface ProtocolRoadmap {
  protocol: VPNProtocol
  label: string
  available: boolean
  status: string
  legacy_insecure: boolean
  runtime_strategy: string
  implementation_level: string
  components: string[]
  runtime_execution: string
  firewall_apply: string
  host_verification: string
  enablement_ready: boolean
  enablement_blockers: string[]
  next_steps: string[]
  blocked_message: string
}

export interface ProtocolServicePlan {
  protocol: VPNProtocol
  label: string
  execution_mode: string
  status: string
  components: string[]
  runtime_plan: string[]
  firewall_plan: string[]
  user_plan: string[]
  warnings: string[]
  legacy_insecure: boolean
}

export interface OpenVPNRoadmap {
  available: boolean
  status: string
  runtime_mode: string
  secret_storage_status: string
  manifest_status: string
  lifecycle_status: string
  status_parser_status: string
  firewall_status: string
  user_storage_status: string
  runtime_execution: string
  firewall_apply: string
  host_verification: string
  enablement_ready: boolean
  enablement_blockers: string[]
  next_steps: string[]
  blocked_message: string
}

export interface OpenVPNRuntimeManifestPreviewRequest {
  instance_name: string
  remote_host: string
  listen_port?: number
  protocol?: "udp" | "tcp" | string
  tunnel_cidr: string
  dns?: string
}

export interface OpenVPNRuntimeManifest {
  runtime_mode: string
  files: Record<string, string>
  warnings: string[]
}

export interface OpenVPNLifecyclePlan {
  action: string
  execution_mode: string
  status: string
  project_name: string
  container_name: string
  commands: string[]
  warnings: string[]
}

export interface OpenVPNFirewallPlan {
  status: string
  ownership_key: string
  rules: string[]
  teardown_rules: string[]
  warnings: string[]
}

export interface OpenVPNStatusClientInfo {
  common_name: string
  real_address: string
  virtual_address: string
  bytes_received: number
  bytes_sent: number
  connected_since: string
}

export interface OpenVPNStatusResponse {
  state: string
  clients: OpenVPNStatusClientInfo[]
  raw?: string
}

export interface OpenVPNPersistedRuntimeManifest {
  id: number
  instance_id: number
  runtime_mode: string
  server_conf: string
  compose_yaml: string
  warnings: string[]
  generation_status: string
}

export interface OpenVPNInstanceDraft {
  id: number
  name: string
  remote_host: string
  listen_port: number
  protocol: string
  tunnel_cidr: string
  dns: string
  enabled: boolean
  runtime_mode: string
  secret_storage_status: string
  secret_refs: Record<string, string>
}

export type { ListParams, PaginatedResult, PaginationMeta }
