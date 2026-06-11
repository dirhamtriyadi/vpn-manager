import { useEffect, useMemo, useState } from "react"
import { AlertTriangle, Check, Copy, Download, Router, Trash2 } from "lucide-react"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Button } from "@/components/ui/button"
import { Textarea } from "@/components/ui/textarea"
import {
  getPeerConfigText,
  peerConfigUrl,
  peerQrCodeUrl,
} from "./api"
import type { Peer, WGInterface } from "./types"

interface Props {
  iface: WGInterface | null
  peer: Peer | null
  onClose: () => void
}

function configValue(config: string, key: string): string {
  const line = config
    .split("\n")
    .find((l) => l.trim().toLowerCase().startsWith(`${key.toLowerCase()} =`))
  return line?.split("=").slice(1).join("=").trim() ?? ""
}

function routerOSName(value: string): string {
  const cleaned = value.toLowerCase().replace(/[^a-z0-9_-]+/g, "-").replace(/^-+|-+$/g, "")
  return cleaned || "wg-vpn"
}

function quoteRouterOS(value: string): string {
  return value.replace(/\\/g, "\\\\").replace(/"/g, "\\\"")
}

function normalizeRouterOSAddressList(value: string): string {
  return value
    .split(",")
    .map((item) => item.trim())
    .filter(Boolean)
    .join(",")
}

function routerOSIPv4Routes(value: string): string[] {
  return normalizeRouterOSAddressList(value)
    .split(",")
    .filter((item) => /^\d{1,3}(\.\d{1,3}){3}\/\d{1,2}$/.test(item))
}

function buildRouterOSScript(iface: WGInterface | null, peer: Peer, config: string): string {
  const name = routerOSName(`wg-${peer.name}`)
  const privateKey = configValue(config, "PrivateKey")
  const presharedKey = configValue(config, "PresharedKey")
  const serverPublicKey = iface?.public_key || configValue(config, "PublicKey")
  const endpoint = iface?.endpoint || "CHANGE_ME_ENDPOINT"
  const endpointPort = iface?.listen_port || 51820
  const mtu = iface?.mtu || 1420
  const address = peer.assigned_ip ? `${peer.assigned_ip}/32` : configValue(config, "Address")
  const allowedAddress = normalizeRouterOSAddressList(
    peer.client_allowed_ips || configValue(config, "AllowedIPs") || "0.0.0.0/0",
  )
  const ipv4Routes = routerOSIPv4Routes(allowedAddress)
  const keepalive = peer.persistent_keepalive || 25

  const lines = [
    `# RouterOS 7 WireGuard client script for peer: ${peer.name}`,
    `# Paste this into RouterOS Terminal. Review endpoint/allowed-address before running.`,
  ]

  if (!privateKey || privateKey.includes("<your-private-key>")) {
    lines.push(
      `# WARNING: this peer was created with a public key supplied by the client,`,
      `# so the panel does not have the RouterOS private key.`,
      `# Replace CHANGE_ME_PRIVATE_KEY or generate the key directly on RouterOS.`,
    )
  }

  lines.push(
    `/interface/wireguard/remove [find name="${quoteRouterOS(name)}"]`,
    `/interface/wireguard/add name="${quoteRouterOS(name)}" mtu=${mtu} private-key="${quoteRouterOS(privateKey || "CHANGE_ME_PRIVATE_KEY")}" comment="wg-panel ${quoteRouterOS(name)}"`,
    `/ip/address/remove [find interface="${quoteRouterOS(name)}"]`,
    `/ip/address/add address=${address} interface="${quoteRouterOS(name)}" comment="wg-panel ${quoteRouterOS(name)}"`,
    `/interface/wireguard/peers/remove [find interface="${quoteRouterOS(name)}"]`,
  )

  let peerCmd = `/interface/wireguard/peers/add interface="${quoteRouterOS(name)}" public-key="${quoteRouterOS(serverPublicKey)}" endpoint-address=${endpoint} endpoint-port=${endpointPort} allowed-address=${allowedAddress} persistent-keepalive=${keepalive}s comment="wg-panel ${quoteRouterOS(name)}"`
  if (presharedKey) {
    peerCmd += ` preshared-key="${quoteRouterOS(presharedKey)}"`
  }
  lines.push(peerCmd)
  if (ipv4Routes.length > 0) {
    lines.push(`/ip/route/remove [find comment="wg-panel ${quoteRouterOS(name)}"]`)
    for (const route of ipv4Routes) {
      lines.push(`/ip/route/add dst-address=${route} gateway="${quoteRouterOS(name)}" comment="wg-panel ${quoteRouterOS(name)}"`)
    }
  }
  lines.push(`/interface/wireguard/print detail where name="${quoteRouterOS(name)}"`)
  lines.push(`/interface/wireguard/peers/print detail where interface="${quoteRouterOS(name)}"`)
  lines.push(`/ping ${iface?.address?.split("/")[0] || "10.8.0.1"} count=5`)

  return lines.join("\n")
}

function buildRouterOSTeardownScript(peer: Peer): string {
  const name = routerOSName(`wg-${peer.name}`)
  return [
    `# RouterOS 7 teardown script for peer: ${peer.name}`,
    `# Use this ONLY when you want to remove the VPN client from this RouterOS router.`,
    `# It deletes the WireGuard interface, its IP address, routes, and wg-panel firewall/NAT rules.`,
    `# It does NOT delete the peer from WG Panel. Use the panel Delete/Trash button for that.`,
    `/ip/address/remove [find interface="${quoteRouterOS(name)}"]`,
    `/interface/wireguard/peers/remove [find interface="${quoteRouterOS(name)}"]`,
    `/ip/route/remove [find comment="wg-panel ${quoteRouterOS(name)}"]`,
    `/ip/firewall/nat/remove [find comment="wg-panel ${quoteRouterOS(name)}"]`,
    `/ip/firewall/filter/remove [find comment="wg-panel ${quoteRouterOS(name)}"]`,
    `/interface/wireguard/remove [find name="${quoteRouterOS(name)}"]`,
    `/interface/wireguard/print`,
  ].join("\n")
}

export function PeerConfigDialog({ iface, peer, onClose }: Props) {
  const [config, setConfig] = useState("")
  const [loading, setLoading] = useState(false)
  const [copied, setCopied] = useState(false)
  const [copiedScript, setCopiedScript] = useState(false)
  const [copiedTeardown, setCopiedTeardown] = useState(false)
  const [mode, setMode] = useState<"conf" | "routeros">("conf")
  const [routerOSMode, setRouterOSMode] = useState<"install" | "teardown">("install")

  useEffect(() => {
    if (!peer) return
    setLoading(true)
    setConfig("")
    setMode("conf")
    setRouterOSMode("install")
    getPeerConfigText(peer.id)
      .then(setConfig)
      .catch(() => setConfig("# Failed to load config"))
      .finally(() => setLoading(false))
  }, [peer])

  const routerOSScript = useMemo(() => {
    if (!peer) return ""
    return buildRouterOSScript(iface, peer, config)
  }, [iface, peer, config])

  const routerOSTeardownScript = useMemo(() => {
    if (!peer) return ""
    return buildRouterOSTeardownScript(peer)
  }, [peer])

  async function copyConfig() {
    await navigator.clipboard.writeText(config)
    setCopied(true)
    setTimeout(() => setCopied(false), 1500)
  }

  async function copyRouterOSScript() {
    await navigator.clipboard.writeText(routerOSScript)
    setCopiedScript(true)
    setTimeout(() => setCopiedScript(false), 1500)
  }

  async function copyRouterOSTeardownScript() {
    await navigator.clipboard.writeText(routerOSTeardownScript)
    setCopiedTeardown(true)
    setTimeout(() => setCopiedTeardown(false), 1500)
  }

  return (
    <Dialog open={!!peer} onOpenChange={(o) => !o && onClose()}>
      <DialogContent className="max-w-3xl">
        <DialogHeader>
          <DialogTitle>Client config — {peer?.name}</DialogTitle>
          <DialogDescription>
            Use QR/.conf for standard WireGuard clients, or open RouterOS for install and teardown scripts.
          </DialogDescription>
        </DialogHeader>

        {peer && (
          <div className="space-y-4">
            <div className="flex gap-2">
              <Button
                type="button"
                variant={mode === "conf" ? "default" : "outline"}
                size="sm"
                onClick={() => setMode("conf")}
              >
                WireGuard .conf / QR
              </Button>
              <Button
                type="button"
                variant={mode === "routeros" ? "default" : "outline"}
                size="sm"
                onClick={() => setMode("routeros")}
              >
                <Router />
                RouterOS
              </Button>
            </div>

            {mode === "conf" ? (
              <div className="grid gap-4 sm:grid-cols-[200px_1fr]">
                <div className="flex items-start justify-center">
                  <img
                    src={peerQrCodeUrl(peer.id)}
                    alt="WireGuard config QR"
                    className="h-[200px] w-[200px] rounded-md border bg-white p-2"
                  />
                </div>
                <div className="space-y-2">
                  <Textarea
                    readOnly
                    value={loading ? "Loading..." : config}
                    className="h-[200px] font-mono text-xs"
                  />
                  <div className="flex gap-2">
                    <Button
                      type="button"
                      variant="outline"
                      size="sm"
                      onClick={copyConfig}
                    >
                      {copied ? <Check /> : <Copy />}
                      {copied ? "Copied" : "Copy"}
                    </Button>
                    <Button asChild size="sm">
                      <a href={peerConfigUrl(peer.id)} download={`${peer.name}.conf`}>
                        <Download />
                        Download .conf
                      </a>
                    </Button>
                  </div>
                </div>
              </div>
            ) : (
              <div className="space-y-3">
                <div className="flex flex-wrap gap-2 rounded-md border bg-muted/30 p-2">
                  <Button
                    type="button"
                    variant={routerOSMode === "install" ? "default" : "outline"}
                    size="sm"
                    onClick={() => setRouterOSMode("install")}
                  >
                    <Router />
                    Install script
                  </Button>
                  <Button
                    type="button"
                    variant={routerOSMode === "teardown" ? "default" : "outline"}
                    size="sm"
                    onClick={() => setRouterOSMode("teardown")}
                  >
                    <Trash2 />
                    Teardown script
                  </Button>
                </div>

                {routerOSMode === "install" ? (
                  <>
                    <div className="rounded-md border border-sky-200 bg-sky-50 p-3 text-sm text-sky-900">
                      <div className="font-medium">RouterOS install script</div>
                      <p className="mt-1 text-xs">
                        Copy this when you want a RouterOS router to connect as a WireGuard client. Paste it into RouterOS Terminal after reviewing the endpoint and allowed-address.
                      </p>
                    </div>
                    <Textarea
                      readOnly
                      value={loading ? "Loading..." : routerOSScript}
                      className="h-[320px] font-mono text-xs"
                    />
                    <div className="flex flex-wrap items-center gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={copyRouterOSScript}
                      >
                        {copiedScript ? <Check /> : <Copy />}
                        {copiedScript ? "Copied" : "Copy RouterOS script"}
                      </Button>
                      <p className="text-xs text-muted-foreground">
                        Paste into RouterOS Terminal. For first test, keep allowed-address limited to the VPN subnet.
                      </p>
                    </div>
                  </>
                ) : (
                  <>
                    <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
                      <div className="flex items-center gap-2 font-medium">
                        <AlertTriangle className="h-4 w-4" />
                        RouterOS teardown script
                      </div>
                      <p className="mt-1 text-xs">
                        This script removes the WireGuard interface, IP address, routes, firewall, and NAT rules from the RouterOS router only. It does not delete the peer from WG Panel.
                      </p>
                    </div>
                    <Textarea
                      readOnly
                      value={routerOSTeardownScript}
                      className="h-[240px] font-mono text-xs"
                    />
                    <div className="flex flex-wrap items-center gap-2">
                      <Button
                        type="button"
                        variant="outline"
                        size="sm"
                        onClick={copyRouterOSTeardownScript}
                      >
                        {copiedTeardown ? <Check /> : <Copy />}
                        {copiedTeardown ? "Copied" : "Copy teardown script"}
                      </Button>
                      <p className="text-xs text-muted-foreground">
                        Paste into RouterOS Terminal only when you want to remove this VPN interface from the router.
                      </p>
                    </div>
                  </>
                )}
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
