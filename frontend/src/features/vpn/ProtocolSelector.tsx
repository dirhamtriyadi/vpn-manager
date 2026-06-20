import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { AlertTriangle, CheckCircle2, Clock, Plus, Shield } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import { apiErrorMessage } from "@/lib/api"
import { listVPNProtocols } from "./api"
import type { VPNProtocolInfo } from "./types"

const fallbackProtocols: VPNProtocolInfo[] = [
  {
    id: "wireguard",
    label: "WireGuard",
    status: "available",
    description: "Fast kernel-backed VPN using the existing WireGuard interface and peer workflow.",
    available: true,
    legacy_insecure: false,
    runtime_strategy: "host_kernel_netlink",
    config_download: true,
    qr_code: true,
    requires_certificates: false,
  },
  {
    id: "openvpn",
    label: "OpenVPN",
    status: "available",
    description: "Containerized OpenVPN with certificate-authority secret storage, server config + .ovpn generation, and gated runtime apply.",
    available: true,
    legacy_insecure: false,
    runtime_strategy: "container_openvpn",
    config_download: true,
    qr_code: false,
    requires_certificates: true,
  },
  {
    id: "l2tp_ipsec",
    label: "L2TP/IPsec",
    status: "available",
    description: "Host IPsec/IKE (strongSwan) + xl2tpd with PPP users, PSK handling, firewall/NAT rules, and gated runtime apply.",
    available: true,
    legacy_insecure: false,
    runtime_strategy: "host_ipsec_ppp",
    config_download: false,
    qr_code: false,
    requires_certificates: false,
  },
  {
    id: "sstp",
    label: "SSTP",
    status: "available",
    description: "Host SSTP daemon with TLS certificate material, PPP users, service status integration, and gated runtime apply.",
    available: true,
    legacy_insecure: false,
    runtime_strategy: "host_sstp",
    config_download: false,
    qr_code: false,
    requires_certificates: true,
  },
  {
    id: "pptp",
    label: "PPTP",
    status: "legacy_available",
    description: "Legacy/insecure compatibility protocol (pptpd); functional but enable only for old clients that cannot use safer VPNs.",
    available: true,
    legacy_insecure: true,
    runtime_strategy: "host_pptpd",
    config_download: false,
    qr_code: false,
    requires_certificates: false,
  },
]

// All protocols are implemented/functional; non-WireGuard protocols still need
// VPN_EXECUTION_ENABLED + host daemons to actually apply. The /vpn/protocols
// `available` flag is registry-based (only WireGuard), so we treat the `status`
// field as the source of truth for "functional".
function isFunctional(p: VPNProtocolInfo): boolean {
  return p.available || p.status === "available" || p.status === "legacy_available"
}

export function ProtocolSelector() {
  const [protocols, setProtocols] = useState<VPNProtocolInfo[]>(fallbackProtocols)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    listVPNProtocols()
      .then((items) => {
        setProtocols(items.length > 0 ? items : fallbackProtocols)
        setError(null)
      })
      .catch((e) => {
        setError(apiErrorMessage(e, "Failed to load protocol availability; showing local fallback."))
        setProtocols(fallbackProtocols)
      })
  }, [])

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">New VPN instance</h2>
          <p className="text-sm text-muted-foreground">
            Choose a protocol. WireGuard applies instantly; OpenVPN, L2TP/IPsec,
            SSTP, and PPTP write host config and run provisioning on apply when
            VPN_EXECUTION_ENABLED is set.
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link to="/">Back to dashboard</Link>
        </Button>
      </div>

      {error && (
        <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          {error}
        </div>
      )}

      <div className="grid gap-4 md:grid-cols-2 xl:grid-cols-3">
        {protocols.map((protocol) => (
          <Card key={protocol.id} className={isFunctional(protocol) ? "border-primary/40" : "opacity-80"}>
            <CardHeader>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Shield className="h-5 w-5" />
                    {protocol.label}
                  </CardTitle>
                  <CardDescription>{protocol.description}</CardDescription>
                </div>
                {protocol.legacy_insecure ? (
                  <Badge variant="secondary">
                    <Clock className="mr-1 h-3 w-3" />
                    Legacy
                  </Badge>
                ) : isFunctional(protocol) ? (
                  <Badge>
                    <CheckCircle2 className="mr-1 h-3 w-3" />
                    Available
                  </Badge>
                ) : (
                  <Badge variant="secondary">
                    <Clock className="mr-1 h-3 w-3" />
                    Roadmap
                  </Badge>
                )}
              </div>
            </CardHeader>
            <CardContent className="space-y-3">
              {protocol.runtime_strategy && (
                <div className="rounded-md border bg-muted/50 p-2 text-xs text-muted-foreground">
                  Runtime: <span className="font-mono">{protocol.runtime_strategy}</span>
                </div>
              )}
              {(protocol.config_download || protocol.qr_code || protocol.requires_certificates) && (
                <div className="flex flex-wrap gap-2 text-xs">
                  {protocol.config_download && <Badge variant="outline">Config download</Badge>}
                  {protocol.qr_code && <Badge variant="outline">QR code</Badge>}
                  {protocol.requires_certificates && <Badge variant="outline">Certificates</Badge>}
                </div>
              )}
              {protocol.legacy_insecure && (
                <div className="flex gap-2 rounded-md border border-destructive/30 bg-destructive/10 p-2 text-xs text-destructive">
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                  <span>PPTP is legacy/insecure and should only be used for old client compatibility.</span>
                </div>
              )}
              {protocol.id === "wireguard" ? (
                <Button asChild>
                  <Link to="/?create=wireguard">
                    <Plus />
                    Create WireGuard instance
                  </Link>
                </Button>
              ) : protocol.id === "openvpn" ? (
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" asChild>
                    <Link to="/vpn/openvpn">OpenVPN advanced</Link>
                  </Button>
                  <Button variant="outline" asChild>
                    <Link to={`/vpn/${protocol.id}`}>Configure & apply</Link>
                  </Button>
                </div>
              ) : (
                <Button variant="outline" asChild>
                  <Link to={`/vpn/${protocol.id}`}>Configure & apply</Link>
                </Button>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
