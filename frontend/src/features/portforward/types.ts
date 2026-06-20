export interface PortForward {
  id: number
  interface_id: number
  interface_name: string
  peer_id: number
  peer_name: string
  protocol: string
  public_port: number
  target_port: number
  target_ip: string
  egress?: string
  enabled: boolean
  comment: string
  created_at: string
  updated_at: string
}

export interface CreatePortForwardPayload {
  interface_id: number
  peer_id: number
  protocol: string
  public_port: number
  target_port: number
  comment?: string
}

export interface UpdatePortForwardPayload {
  enabled?: boolean
  target_port?: number
  comment?: string
}
