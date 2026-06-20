import { useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { AlertTriangle, CheckCircle2, Clock, FileKey2, FileText, ServerCog } from "lucide-react"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { apiErrorMessage } from "@/lib/api"
import {
  generateOpenVPNRuntimeManifest,
  getOpenVPNRoadmap,
  listOpenVPNInstanceDrafts,
  planOpenVPNFirewall,
  planOpenVPNLifecycle,
  previewOpenVPNRuntimeManifest,
} from "./api"
import type {
  OpenVPNFirewallPlan,
  OpenVPNInstanceDraft,
  OpenVPNLifecyclePlan,
  OpenVPNPersistedRuntimeManifest,
  OpenVPNRoadmap,
  OpenVPNRuntimeManifest,
} from "./types"

const fallbackRoadmap: OpenVPNRoadmap = {
  available: true,
  status: "available",
  runtime_mode: "container_openvpn",
  secret_storage_status: "encrypted_secret_store",
  manifest_status: "persisted_manifest",
  lifecycle_status: "host_apply",
  status_parser_status: "status_parser",
  firewall_status: "firewall_apply",
  user_storage_status: "encrypted_user_store",
  runtime_execution: "disabled",
  firewall_apply: "disabled",
  host_verification: "disabled",
  enablement_ready: false,
  enablement_blockers: [
    "VPN_EXECUTION_ENABLED must be true before the API writes config and runs container/firewall commands",
  ],
  blocked_message: "OpenVPN is implemented; set VPN_EXECUTION_ENABLED=true so apply can write config and run the container/firewall commands.",
  next_steps: [
    "create an instance with CA/server certificate material and generate its runtime manifest",
    "set VPN_EXECUTION_ENABLED=true, then apply the instance to start the container and firewall rules",
  ],
}

const previewPayload = {
  instance_name: "office",
  remote_host: "vpn.example.com",
  listen_port: 1194,
  protocol: "udp",
  tunnel_cidr: "10.20.0.0/24",
  dns: "1.1.1.1",
}

export function OpenVPNRoadmapPage() {
  const [roadmap, setRoadmap] = useState<OpenVPNRoadmap>(fallbackRoadmap)
  const [manifest, setManifest] = useState<OpenVPNRuntimeManifest | null>(null)
  const [drafts, setDrafts] = useState<OpenVPNInstanceDraft[]>([])
  const [persistedManifest, setPersistedManifest] = useState<OpenVPNPersistedRuntimeManifest | null>(null)
  const [lifecyclePlan, setLifecyclePlan] = useState<OpenVPNLifecyclePlan | null>(null)
  const [firewallPlan, setFirewallPlan] = useState<OpenVPNFirewallPlan | null>(null)
  const [generatingManifestId, setGeneratingManifestId] = useState<number | null>(null)
  const [planningActionId, setPlanningActionId] = useState<number | null>(null)
  const [error, setError] = useState<string | null>(null)
  const [manifestError, setManifestError] = useState<string | null>(null)
  const [draftError, setDraftError] = useState<string | null>(null)

  useEffect(() => {
    getOpenVPNRoadmap()
      .then((data) => {
        setRoadmap(data)
        setError(null)
      })
      .catch((e) => {
        setRoadmap(fallbackRoadmap)
        setError(apiErrorMessage(e, "Failed to load OpenVPN roadmap; showing local fallback."))
      })

    previewOpenVPNRuntimeManifest(previewPayload)
      .then((data) => {
        setManifest(data)
        setManifestError(null)
      })
      .catch((e) => {
        setManifest(null)
        setManifestError(apiErrorMessage(e, "Failed to load OpenVPN runtime manifest preview."))
      })

    listOpenVPNInstanceDrafts({ per_page: 5 })
      .then((result) => {
        setDrafts(result.data)
        setDraftError(null)
      })
      .catch((e) => {
        setDrafts([])
        setDraftError(apiErrorMessage(e, "Failed to load OpenVPN instance drafts."))
      })
  }, [])

  const handleGenerateManifest = async (instanceId: number) => {
    setGeneratingManifestId(instanceId)
    setManifestError(null)
    try {
      const generated = await generateOpenVPNRuntimeManifest(instanceId)
      setPersistedManifest(generated)
    } catch (e) {
      setManifestError(apiErrorMessage(e, "Failed to persist OpenVPN runtime manifest."))
    } finally {
      setGeneratingManifestId(null)
    }
  }

  const handlePlanRuntime = async (instanceId: number) => {
    setPlanningActionId(instanceId)
    setManifestError(null)
    try {
      const [lifecycle, firewall] = await Promise.all([
        planOpenVPNLifecycle(instanceId, "start"),
        planOpenVPNFirewall(instanceId),
      ])
      setLifecyclePlan(lifecycle)
      setFirewallPlan(firewall)
    } catch (e) {
      setManifestError(apiErrorMessage(e, "Failed to generate OpenVPN runtime plans."))
    } finally {
      setPlanningActionId(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="text-2xl font-semibold tracking-tight">OpenVPN</h2>
          <p className="text-sm text-muted-foreground">
            OpenVPN is functional: create an instance with certificate material,
            generate its manifest, then apply (runs the container + firewall when
            VPN_EXECUTION_ENABLED is set).
          </p>
        </div>
        <Button variant="outline" asChild>
          <Link to="/vpn/new">Back to protocols</Link>
        </Button>
      </div>

      {error && (
        <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
          {error}
        </div>
      )}

      <Card>
        <CardHeader>
          <div className="flex items-start justify-between gap-3">
            <div>
              <CardTitle className="flex items-center gap-2">
                <ServerCog className="h-5 w-5" />
                Runtime status
              </CardTitle>
              <CardDescription>
                Instance/user drafts, encrypted certificate storage, persisted
                runtime manifest, and a gated apply that runs docker compose +
                firewall rules.
              </CardDescription>
            </div>
            {roadmap.available ? (
              <Badge>
                <CheckCircle2 className="mr-1 h-3 w-3" />
                Available
              </Badge>
            ) : (
              <Badge variant="secondary">
                <Clock className="mr-1 h-3 w-3" />
                {roadmap.status}
              </Badge>
            )}
          </div>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 md:grid-cols-2">
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              Runtime mode: <span className="font-mono">{roadmap.runtime_mode}</span>
            </div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              Secret storage: <span className="font-mono">{roadmap.secret_storage_status}</span>
            </div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm md:col-span-2">
              Manifest status: <span className="font-mono">{roadmap.manifest_status}</span>
            </div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              Lifecycle: <span className="font-mono">{roadmap.lifecycle_status}</span>
            </div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              Status parser: <span className="font-mono">{roadmap.status_parser_status}</span>
            </div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              Firewall/NAT: <span className="font-mono">{roadmap.firewall_status}</span>
            </div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              User storage: <span className="font-mono">{roadmap.user_storage_status}</span>
            </div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              Runtime execution: <span className="font-mono">{roadmap.runtime_execution}</span>
            </div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              Firewall apply: <span className="font-mono">{roadmap.firewall_apply}</span>
            </div>
            <div className="rounded-md border bg-muted/50 p-3 text-sm md:col-span-2">
              Host verification: <span className="font-mono">{roadmap.host_verification}</span>
            </div>
          </div>
          <div className="flex gap-2 rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
            <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
            <span>{roadmap.blocked_message}</span>
          </div>
          {roadmap.enablement_blockers.length > 0 && (
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              <div className="mb-2 font-medium">Operational enablement blockers</div>
              <ul className="list-disc space-y-1 pl-6 text-muted-foreground">
                {roadmap.enablement_blockers.map((blocker) => (
                  <li key={blocker}>{blocker}</li>
                ))}
              </ul>
            </div>
          )}
          <div>
            <h3 className="mb-2 flex items-center gap-2 text-sm font-semibold">
              <FileKey2 className="h-4 w-4" />
              Required before enabling OpenVPN
            </h3>
            <ul className="list-disc space-y-1 pl-6 text-sm text-muted-foreground">
              {roadmap.next_steps.map((step) => (
                <li key={step}>{step}</li>
              ))}
            </ul>
          </div>
          <Button disabled>Create OpenVPN instance — not enabled yet</Button>
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileKey2 className="h-5 w-5" />
            Saved OpenVPN drafts
          </CardTitle>
          <CardDescription>
            Drafts may store encrypted certificate references, but they still cannot start OpenVPN containers.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-3">
          {draftError && (
            <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
              {draftError}
            </div>
          )}
          {drafts.length === 0 ? (
            <div className="rounded-md border bg-muted/50 p-3 text-sm text-muted-foreground">
              No OpenVPN drafts saved yet. Backend draft creation requires OPENVPN_SECRET_MASTER_KEY and stores only encrypted secret references.
            </div>
          ) : (
            <div className="space-y-2">
              {drafts.map((draft) => (
                <div key={draft.id} className="rounded-md border p-3 text-sm">
                  <div className="flex flex-wrap items-center justify-between gap-2">
                    <div className="font-medium">{draft.name}</div>
                    <div className="flex items-center gap-2">
                      <Badge variant="secondary">disabled draft</Badge>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => void handleGenerateManifest(draft.id)}
                        disabled={generatingManifestId === draft.id}
                      >
                        {generatingManifestId === draft.id ? "Generating..." : "Generate manifest"}
                      </Button>
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => void handlePlanRuntime(draft.id)}
                        disabled={planningActionId === draft.id}
                      >
                        {planningActionId === draft.id ? "Planning..." : "Plan runtime"}
                      </Button>
                    </div>
                  </div>
                  <div className="mt-2 grid gap-2 text-muted-foreground md:grid-cols-3">
                    <span>{draft.protocol}/{draft.listen_port}</span>
                    <span>{draft.tunnel_cidr}</span>
                    <span>{draft.secret_storage_status}</span>
                  </div>
                </div>
              ))}
            </div>
          )}
          {persistedManifest && (
            <div className="rounded-md border bg-muted/50 p-3 text-sm">
              Persisted manifest #{persistedManifest.id} for instance #{persistedManifest.instance_id}: {persistedManifest.generation_status}
            </div>
          )}
          {lifecyclePlan && (
            <div className="space-y-2 rounded-md border bg-muted/50 p-3 text-sm">
              <div className="font-medium">Lifecycle plan: {lifecyclePlan.action} / {lifecyclePlan.execution_mode}</div>
              <pre className="overflow-auto whitespace-pre-wrap text-xs">{lifecyclePlan.commands.join("\n")}</pre>
            </div>
          )}
          {firewallPlan && (
            <div className="space-y-2 rounded-md border bg-muted/50 p-3 text-sm">
              <div className="font-medium">Firewall ownership: {firewallPlan.ownership_key}</div>
              <pre className="overflow-auto whitespace-pre-wrap text-xs">{firewallPlan.rules.join("\n")}</pre>
            </div>
          )}
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle className="flex items-center gap-2">
            <FileText className="h-5 w-5" />
            Runtime manifest preview
          </CardTitle>
          <CardDescription>
            Generated example files for a future container runtime. These are preview-only and are not deployed automatically.
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          {manifestError && (
            <div className="rounded-md border border-amber-200 bg-amber-50 p-3 text-sm text-amber-900">
              {manifestError}
            </div>
          )}
          {manifest && (
            <>
              {manifest.warnings.map((warning) => (
                <div key={warning} className="flex gap-2 rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
                  <AlertTriangle className="mt-0.5 h-4 w-4 shrink-0" />
                  <span>{warning}</span>
                </div>
              ))}
              {Object.entries(manifest.files).map(([filename, content]) => (
                <div key={filename} className="space-y-2">
                  <div className="font-mono text-sm font-semibold">{filename}</div>
                  <pre className="max-h-72 overflow-auto rounded-md border bg-muted p-3 text-xs">
                    {content}
                  </pre>
                </div>
              ))}
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
