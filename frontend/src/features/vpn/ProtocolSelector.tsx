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
    available: false,
    legacy_insecure: false,
    config_download: false,
    qr_code: false,
    requires_certificates: true,
  },
  {
    id: "l2tp_ipsec",
    label: "L2TP/IPsec",
    available: false,
    legacy_insecure: false,
    config_download: false,
    qr_code: false,
    requires_certificates: false,
  },
  {
    id: "sstp",
    label: "SSTP",
    available: false,
    legacy_insecure: false,
    config_download: false,
    qr_code: false,
    requires_certificates: true,
  },
  {
    id: "pptp",
    label: "PPTP",
    available: false,
    legacy_insecure: true,
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
            Choose a protocol. WireGuard is available now; the others are staged for the multi-protocol roadmap.
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
                  <CardDescription>
                    {protocol.available
                      ? "Available now through the existing WireGuard flow."
                      : "Coming soon as a protocol driver."}
                  </CardDescription>
                </div>
                {protocol.available ? (
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
              ) : (
                <Button disabled variant="outline">
                  Coming soon
                </Button>
              )}
            </CardContent>
          </Card>
        ))}
      </div>
    </div>
  )
}
