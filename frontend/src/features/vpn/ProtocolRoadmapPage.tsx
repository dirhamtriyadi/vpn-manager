import { useEffect, useMemo, useState } from "react"
import { Link, useParams } from "react-router-dom"
import { AlertTriangle, CheckCircle2, FileText, Shield } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { apiErrorMessage } from "@/lib/api"
import { getProtocolProductionPlan, getProtocolRoadmap, getProtocolServicePlan } from "./api"
import type { ProtocolProductionPlan, ProtocolRoadmap, ProtocolServicePlan, VPNProtocol } from "./types"

const protocolLabels: Record<VPNProtocol, string> = {
  wireguard: "WireGuard",
  openvpn: "OpenVPN",
  l2tp_ipsec: "L2TP/IPsec",
  sstp: "SSTP",
  pptp: "PPTP",
}

function isVPNProtocol(value: string | undefined): value is VPNProtocol {
  return value === "wireguard" || value === "openvpn" || value === "l2tp_ipsec" || value === "sstp" || value === "pptp"
}

function fallbackRoadmap(protocol: VPNProtocol): ProtocolRoadmap {
  const label = protocolLabels[protocol]
  return {
    protocol,
    label,
    available: false,
    status: protocol === "pptp" ? "legacy_roadmap" : "roadmap",
    legacy_insecure: protocol === "pptp",
    runtime_strategy: protocol === "l2tp_ipsec" ? "host_ipsec_ppp" : protocol === "sstp" ? "container_or_host_sstp" : protocol === "pptp" ? "legacy_host_pptpd" : "container_openvpn_preview",
    implementation_level: "service_plan_scaffold",
    components: [],
    runtime_execution: "disabled",
    firewall_apply: "disabled",
    host_verification: "disabled",
    enablement_ready: false,
    enablement_blockers: [
      "VPN_RUNTIME_EXECUTION_ENABLED must be true before service/container commands can run",
      "VPN_FIREWALL_APPLY_ENABLED must be true before firewall/NAT rules can be applied",
      "VPN_HOST_VERIFICATION_PASSED must be true after host-side tests/build and dry-run plan review",
    ],
    next_steps: ["review dry-run service plan on host", "register real driver after verification"],
    blocked_message: `${label} is scaffolded as a service plan but is not available until a real driver is verified.`,
  }
}

export function ProtocolRoadmapPage() {
  const { protocol: protocolParam } = useParams()
  const protocol = isVPNProtocol(protocolParam) ? protocolParam : "l2tp_ipsec"
  const [roadmap, setRoadmap] = useState<ProtocolRoadmap>(() => fallbackRoadmap(protocol))
  const [plan, setPlan] = useState<ProtocolServicePlan | null>(null)
  const [productionPlan, setProductionPlan] = useState<ProtocolProductionPlan | null>(null)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    setRoadmap(fallbackRoadmap(protocol))
    setPlan(null)
    setProductionPlan(null)
    Promise.all([getProtocolRoadmap(protocol), getProtocolServicePlan(protocol), getProtocolProductionPlan(protocol)])
      .then(([nextRoadmap, nextPlan, nextProductionPlan]) => {
        setRoadmap(nextRoadmap)
        setPlan(nextPlan)
        setProductionPlan(nextProductionPlan)
        setError(null)
      })
      .catch((e) => {
        setError(apiErrorMessage(e, "Failed to load protocol roadmap; showing local fallback."))
      })
  }, [protocol])

  const lists = useMemo(() => {
    if (!plan) return []
    return [
      ["Components", plan.components],
      ["Runtime plan", plan.runtime_plan],
      ["Firewall/NAT plan", plan.firewall_plan],
      ["User/credential plan", plan.user_plan],
    ] as const
  }, [plan])

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
            <Shield className="h-6 w-6" />
            {roadmap.label} roadmap
          </h2>
          <p className="text-sm text-muted-foreground">
            Service plan lengkap tanpa menjalankan command host/firewall dari UI.
          </p>
        </div>
        <div className="flex gap-2">
          {protocol === "openvpn" && (
            <Button variant="outline" asChild>
              <Link to="/vpn/openvpn">OpenVPN advanced page</Link>
            </Button>
          )}
          <Button variant="outline" asChild>
            <Link to="/vpn/new">Back to protocols</Link>
          </Button>
        </div>
      </div>

      {error && <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">{error}</div>}

      <Card>
        <CardHeader>
          <CardTitle className="flex flex-wrap items-center gap-2">
            Readiness
            <Badge variant={roadmap.available ? "default" : "secondary"}>
              {roadmap.available ? "Available" : roadmap.status}
            </Badge>
            {roadmap.enablement_ready && (
              <Badge variant="outline">
                <CheckCircle2 className="mr-1 h-3 w-3" />
                Gates ready
              </Badge>
            )}
          </CardTitle>
          <CardDescription>{roadmap.blocked_message}</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="rounded-md border bg-muted/50 p-3 text-sm">Runtime: <span className="font-mono">{roadmap.runtime_strategy}</span></div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">Implementation: <span className="font-mono">{roadmap.implementation_level}</span></div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">Runtime execution: <span className="font-mono">{roadmap.runtime_execution}</span></div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">Firewall apply: <span className="font-mono">{roadmap.firewall_apply}</span></div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm md:col-span-2">Host verification: <span className="font-mono">{roadmap.host_verification}</span></div>
          </div>
          {roadmap.legacy_insecure && (
            <div className="flex gap-2 rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
              <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
              <span>Protocol ini legacy/insecure. Gunakan hanya untuk client lama yang tidak mendukung VPN lebih aman.</span>
            </div>
          )}
          {roadmap.enablement_blockers.length > 0 && (
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              <div className="mb-2 font-medium">Enablement blockers</div>
              <ul className="list-disc space-y-1 pl-6 text-muted-foreground">
                {roadmap.enablement_blockers.map((blocker) => <li key={blocker}>{blocker}</li>)}
              </ul>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Production execution plan</CardTitle>
          <CardDescription>
            Command checklist untuk production. Mode tetap blocked/manual kecuali semua gate host sudah diaktifkan.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {productionPlan ? (
            <>
              <div className="grid gap-3 md:grid-cols-3">
                <div className="rounded-md border bg-muted/50 p-3 text-sm">Ready: <span className="font-mono">{productionPlan.ready ? "yes" : "no"}</span></div>
                <div className="rounded-md border bg-muted/50 p-3 text-sm">Mode: <span className="font-mono">{productionPlan.execution_mode}</span></div>
                <div className="rounded-md border bg-muted/50 p-3 text-sm">Legacy insecure: <span className="font-mono">{productionPlan.legacy_insecure ? "yes" : "no"}</span></div>
              </div>
              {productionPlan.blockers.length > 0 && (
                <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
                  <div className="mb-2 font-medium">Production blockers</div>
                  <ul className="list-disc space-y-1 pl-6">
                    {productionPlan.blockers.map((blocker) => <li key={blocker}>{blocker}</li>)}
                  </ul>
                </div>
              )}
              {[
                ["Config files", productionPlan.config_files],
                ["Runtime commands", productionPlan.runtime_commands],
                ["Firewall commands", productionPlan.firewall_commands],
                ["Status commands", productionPlan.status_commands],
              ].map(([title, items]) => (
                <div key={title as string}>
                  <h3 className="mb-2 text-sm font-semibold">{title}</h3>
                  <ul className="list-disc space-y-1 pl-6 text-sm text-muted-foreground">
                    {(items as string[]).map((item) => <li key={item} className="font-mono text-xs">{item}</li>)}
                  </ul>
                </div>
              ))}
            </>
          ) : (
            <p className="text-sm text-muted-foreground">Loading production plan...</p>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            Dry-run service plan
          </CardTitle>
          <CardDescription>Plan ini untuk review; tidak menginstall daemon, tidak start service, dan tidak apply firewall.</CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {lists.map(([title, items]) => (
            <div key={title}>
              <h3 className="mb-2 text-sm font-semibold">{title}</h3>
              <ul className="list-disc space-y-1 pl-6 text-sm text-muted-foreground">
                {items.map((item) => <li key={item}>{item}</li>)}
              </ul>
            </div>
          ))}
          {plan?.warnings?.length ? (
            <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
              <div className="mb-2 font-medium">Warnings</div>
              <ul className="list-disc space-y-1 pl-6">
                {plan.warnings.map((warning) => <li key={warning}>{warning}</li>)}
              </ul>
            </div>
          ) : null}
        </CardContent>
      </Card>
    </div>
  )
}
