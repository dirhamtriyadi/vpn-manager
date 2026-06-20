import { useCallback, useEffect, useState } from "react"
import { Globe, Plus, Power, Trash2 } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Badge } from "@/components/ui/badge"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { apiErrorMessage } from "@/lib/api"
import { copyToClipboard } from "@/lib/clipboard"
import { useAuth } from "@/features/auth/AuthContext"
import { ConfirmDialog } from "@/features/admin/ConfirmDialog"
import type { Peer, WGInterface } from "@/features/wireguard/types"
import {
  createPortForward,
  deletePortForward,
  listInterfacesForSelect,
  listPeersForSelect,
  listPortForwards,
  updatePortForward,
} from "./api"
import type { PortForward } from "./types"

const selectClass =
  "flex h-9 w-full rounded-md border border-input bg-background px-3 py-1 text-sm shadow-sm focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:opacity-50"

function routerOSName(peerName: string): string {
  const cleaned = `wg-${peerName}`.toLowerCase().replace(/[^a-z0-9_-]+/g, "-").replace(/^-+|-+$/g, "")
  return cleaned || "wg-vpn"
}

function mikrotikHint(pf: PortForward): string {
  return `/ip/firewall/nat/add chain=dstnat in-interface=${routerOSName(pf.peer_name)} protocol=${pf.protocol} dst-port=${pf.target_port} action=dst-nat to-addresses=LAN_IP to-ports=LAN_PORT`
}

export function PortForwardsPage() {
  const { hasPermission } = useAuth()
  const canManage = hasPermission("portforwards.manage")
  const [forwards, setForwards] = useState<PortForward[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [notice, setNotice] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [toDelete, setToDelete] = useState<PortForward | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    listPortForwards()
      .then((data) => {
        setForwards(data)
        setError(null)
      })
      .catch((e) => setError(apiErrorMessage(e, "Failed to load port forwards")))
      .finally(() => setLoading(false))
  }, [])

  useEffect(load, [load])

  async function toggle(pf: PortForward) {
    try {
      await updatePortForward(pf.id, { enabled: !pf.enabled })
      load()
    } catch (e) {
      setError(apiErrorMessage(e, "Failed to toggle port forward"))
    }
  }

  async function confirmDelete() {
    if (!toDelete) return
    try {
      await deletePortForward(toDelete.id)
      setToDelete(null)
      load()
    } catch (e) {
      setError(apiErrorMessage(e, "Failed to delete port forward"))
      setToDelete(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
            <Globe className="h-6 w-6" />
            Port Forwarding (Public IP)
          </h2>
          <p className="text-sm text-muted-foreground">
            Expose a port on the server's public IP and forward it through the
            tunnel to a peer. The peer (MikroTik) does the final hop to its LAN.
          </p>
        </div>
        {canManage && (
          <Button onClick={() => setCreating(true)}>
            <Plus />
            New port forward
          </Button>
        )}
      </div>

      <div className="rounded-md border border-sky-200 bg-sky-50 p-3 text-sm text-sky-900">
        Flow: <span className="font-mono">internet → server:public_port → tunnel → peer:target_port</span>.
        After creating one, add the matching <span className="font-mono">dstnat</span> rule on the MikroTik
        (shown per row) to reach the actual LAN device.
      </div>

      {error && (
        <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}
      {notice && (
        <div className="rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-900">
          {notice}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">All port forwards</CardTitle>
          <CardDescription>{forwards.length} rule(s)</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading…</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Public port</TableHead>
                  <TableHead>Proto</TableHead>
                  <TableHead>Forwards to</TableHead>
                  <TableHead>Interface</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {forwards.map((pf) => (
                  <TableRow key={pf.id}>
                    <TableCell className="font-mono font-medium">{pf.public_port}</TableCell>
                    <TableCell className="uppercase">{pf.protocol}</TableCell>
                    <TableCell className="font-mono text-xs">
                      {pf.peer_name} ({pf.target_ip}:{pf.target_port})
                    </TableCell>
                    <TableCell>{pf.interface_name}</TableCell>
                    <TableCell>
                      {pf.enabled ? (
                        <Badge variant="success">enabled</Badge>
                      ) : (
                        <Badge variant="muted">disabled</Badge>
                      )}
                    </TableCell>
                    <TableCell className="text-right">
                      {canManage && (
                        <div className="flex justify-end gap-1">
                          <Button
                            variant="outline"
                            size="sm"
                            title="Copy MikroTik dstnat command"
                            onClick={async () => {
                              const ok = await copyToClipboard(mikrotikHint(pf))
                              setNotice(
                                ok
                                  ? "MikroTik command copied — replace LAN_IP / LAN_PORT with your device."
                                  : "Copy failed — select and copy this manually:\n" + mikrotikHint(pf),
                              )
                            }}
                          >
                            MikroTik
                          </Button>
                          <Button variant="outline" size="sm" onClick={() => toggle(pf)} title={pf.enabled ? "Disable" : "Enable"}>
                            <Power />
                          </Button>
                          <Button variant="outline" size="sm" onClick={() => setToDelete(pf)}>
                            <Trash2 />
                          </Button>
                        </div>
                      )}
                    </TableCell>
                  </TableRow>
                ))}
                {forwards.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={6} className="text-center text-muted-foreground">
                      No port forwards yet.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {creating && (
        <CreateDialog
          onClose={() => setCreating(false)}
          onSaved={(warn) => {
            setCreating(false)
            setNotice(warn ?? null)
            load()
          }}
        />
      )}

      <ConfirmDialog
        open={Boolean(toDelete)}
        title="Delete port forward"
        description={`Remove the forward for ${toDelete?.protocol.toUpperCase()} port ${toDelete?.public_port}? The server firewall rules are removed too.`}
        confirmLabel="Delete"
        onConfirm={confirmDelete}
        onCancel={() => setToDelete(null)}
      />
    </div>
  )
}

function CreateDialog({
  onClose,
  onSaved,
}: {
  onClose: () => void
  onSaved: (warning?: string) => void
}) {
  const [interfaces, setInterfaces] = useState<WGInterface[]>([])
  const [peers, setPeers] = useState<Peer[]>([])
  const [interfaceId, setInterfaceId] = useState<number | "">("")
  const [peerId, setPeerId] = useState<number | "">("")
  const [protocol, setProtocol] = useState("tcp")
  const [publicPort, setPublicPort] = useState("")
  const [targetPort, setTargetPort] = useState("")
  const [comment, setComment] = useState("")
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  useEffect(() => {
    listInterfacesForSelect()
      .then(setInterfaces)
      .catch((e) => setFormError(apiErrorMessage(e, "Failed to load interfaces")))
  }, [])

  useEffect(() => {
    setPeerId("")
    setPeers([])
    if (interfaceId === "") return
    listPeersForSelect(Number(interfaceId))
      .then(setPeers)
      .catch((e) => setFormError(apiErrorMessage(e, "Failed to load peers")))
  }, [interfaceId])

  async function submit() {
    setFormError(null)
    const pub = Number(publicPort)
    const tgt = targetPort === "" ? pub : Number(targetPort)
    if (interfaceId === "" || peerId === "") {
      setFormError("Select an interface and a peer.")
      return
    }
    if (!pub || pub < 1 || pub > 65535) {
      setFormError("Public port must be 1–65535.")
      return
    }
    if (!tgt || tgt < 1 || tgt > 65535) {
      setFormError("Target port must be 1–65535.")
      return
    }
    setSubmitting(true)
    try {
      await createPortForward({
        interface_id: Number(interfaceId),
        peer_id: Number(peerId),
        protocol,
        public_port: pub,
        target_port: tgt,
        comment: comment.trim() || undefined,
      })
      onSaved()
    } catch (e) {
      setFormError(apiErrorMessage(e, "Failed to create port forward"))
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open onOpenChange={(o) => !o && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>New port forward</DialogTitle>
          <DialogDescription>
            Forward a public port on the server to a peer's tunnel IP.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="pf-iface">Interface</Label>
              <select
                id="pf-iface"
                className={selectClass}
                value={interfaceId}
                onChange={(e) => setInterfaceId(e.target.value === "" ? "" : Number(e.target.value))}
              >
                <option value="">Select…</option>
                {interfaces.map((i) => (
                  <option key={i.id} value={i.id}>
                    {i.name} ({i.address})
                  </option>
                ))}
              </select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="pf-peer">Peer (target)</Label>
              <select
                id="pf-peer"
                className={selectClass}
                value={peerId}
                disabled={interfaceId === ""}
                onChange={(e) => setPeerId(e.target.value === "" ? "" : Number(e.target.value))}
              >
                <option value="">Select…</option>
                {peers.map((p) => (
                  <option key={p.id} value={p.id}>
                    {p.name} ({p.assigned_ip})
                  </option>
                ))}
              </select>
            </div>
          </div>

          <div className="grid gap-4 sm:grid-cols-3">
            <div className="space-y-1.5">
              <Label htmlFor="pf-proto">Protocol</Label>
              <select id="pf-proto" className={selectClass} value={protocol} onChange={(e) => setProtocol(e.target.value)}>
                <option value="tcp">tcp</option>
                <option value="udp">udp</option>
              </select>
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="pf-public">Public port</Label>
              <Input id="pf-public" type="number" value={publicPort} onChange={(e) => setPublicPort(e.target.value)} placeholder="8080" />
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="pf-target">Target port</Label>
              <Input id="pf-target" type="number" value={targetPort} onChange={(e) => setTargetPort(e.target.value)} placeholder="= public" />
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="pf-comment">Comment (optional)</Label>
            <Input id="pf-comment" value={comment} onChange={(e) => setComment(e.target.value)} placeholder="e.g. CCTV gedung A" />
          </div>

          {formError && <p className="text-sm text-destructive">{formError}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="button" onClick={submit} disabled={submitting}>
              {submitting ? "Creating…" : "Create"}
            </Button>
          </DialogFooter>
        </div>
      </DialogContent>
    </Dialog>
  )
}
