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
    status: "roadmap",
    description: "Needs OpenVPN runtime, certificate authority, server config, and .ovpn generation.",
    available: false,
    legacy_insecure: false,
    runtime_strategy: "container_openvpn_preview",
    config_download: true,
    qr_code: false,
    requires_certificates: true,
  },
  {
    id: "l2tp_ipsec",
    label: "L2TP/IPsec",
    status: "roadmap",
    description: "Needs IPsec/IKE daemon, PPP users, PSK/certificate handling, and firewall/NAT rules.",
    available: false,
    legacy_insecure: false,
    runtime_strategy: "host_ipsec_ppp",
    config_download: false,
    qr_code: false,
    requires_certificates: false,
  },
  {
    id: "sstp",
    label: "SSTP",
    status: "roadmap",
    description: "Needs SSTP daemon, TLS certificate management, users, and service status integration.",
    available: false,
    legacy_insecure: false,
    runtime_strategy: "container_or_host_sstp",
    config_download: false,
    qr_code: false,
    requires_certificates: true,
  },
  {
    id: "pptp",
    label: "PPTP",
    status: "legacy_roadmap",
    description: "Legacy/insecure compatibility protocol; only consider for old clients that cannot use safer VPNs.",
    available: false,
    legacy_insecure: true,
    runtime_strategy: "legacy_host_pptpd",
    config_download: false,
    qr_code: false,
    requires_certificates: false,
  },
]

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
        setError(apiErrorMessage(e, "Failed to load protocol availability; showing local roadmap."))
        setProtocols(fallbackProtocols)
      })
  }, [])

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">New VPN instance</h2>
          <p className="text-sm text-muted-foreground">
            Choose a protocol. WireGuard is available now; every other protocol has a dry-run service plan before host execution is enabled.
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
          <Card key={protocol.id} className={protocol.available ? "border-primary/40" : "opacity-80"}>
            <CardHeader>
              <div className="flex items-start justify-between gap-3">
                <div>
                  <CardTitle className="flex items-center gap-2">
                    <Shield className="h-5 w-5" />
                    {protocol.label}
                  </CardTitle>
                  <CardDescription>{protocol.description}</CardDescription>
                </div>
                {protocol.available ? (
                  <Badge>
                    <CheckCircle2 className="mr-1 h-3 w-3" />
                    Available
                  </Badge>
                ) : (
                  <Badge variant="secondary">
                    <Clock className="mr-1 h-3 w-3" />
                    {protocol.status === "legacy_roadmap" ? "Legacy roadmap" : "Roadmap"}
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
              {protocol.available && (protocol.config_download || protocol.qr_code || protocol.requires_certificates) && (
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
              {protocol.available ? (
                <Button asChild>
                  <Link to="/?create=wireguard">
                    <Plus />
                    Create WireGuard instance
                  </Link>
                </Button>
              ) : protocol.id === "openvpn" ? (
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" asChild>
                    <Link to="/vpn/openvpn">OpenVPN advanced requirements</Link>
                  </Button>
                  <Button variant="outline" asChild>
                    <Link to={`/vpn/${protocol.id}`}>Service plan</Link>
                  </Button>
                </div>
              ) : (
                <Button variant="outline" asChild>
                  <Link to={`/vpn/${protocol.id}`}>View service plan</Link>
                </Button>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
