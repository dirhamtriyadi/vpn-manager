import { useCallback, useEffect, useState } from "react"
import {
  Plus,
  RefreshCw,
  Trash2,
  QrCode,
  Power,
  Server,
  Wifi,
  WifiOff,
  Archive,
  Undo2,
} from "lucide-react"
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
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { apiErrorMessage } from "@/lib/api"
import type { InterfaceFormValues } from "@/schemas/interface"
import type { PeerFormValues } from "@/schemas/peer"
import { InterfaceForm } from "./InterfaceForm"
import { PeerForm } from "./PeerForm"
import { PeerConfigDialog } from "./PeerConfigDialog"
import { formatBytes, formatHandshake } from "./format"
import {
  createInterface,
  createPeer,
  deleteInterface,
  deletePeer,
  getInterfaceStatus,
  listInterfaces,
  listTrashedInterfaces,
  listTrashedPeers,
  purgeInterface,
  purgePeer,
  restoreInterface,
  restorePeer,
  syncInterface,
  updatePeer,
} from "./api"
import type { InterfaceStatus, Peer, WGInterface } from "./types"

function formatDeletedAt(value: WGInterface["deleted_at"] | Peer["deleted_at"]): string {
  const raw = typeof value === "string" ? value : value?.Time
  return raw ? new Date(raw).toLocaleString() : "—"
}

export function Dashboard() {
  const [interfaces, setInterfaces] = useState<WGInterface[]>([])
  const [trashedInterfaces, setTrashedInterfaces] = useState<WGInterface[]>([])
  const [trashedPeers, setTrashedPeers] = useState<Peer[]>([])
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [status, setStatus] = useState<InterfaceStatus | null>(null)
  const [banner, setBanner] = useState<{ kind: "error" | "info"; text: string } | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [addPeerOpen, setAddPeerOpen] = useState(false)
  const [configPeer, setConfigPeer] = useState<Peer | null>(null)

  const loadInterfaces = useCallback(async () => {
    try {
      const data = await listInterfaces()
      setInterfaces(data)
      setSelectedId((prev) => prev ?? data[0]?.id ?? null)
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to load interfaces. Is the API running?") })
    }
  }, [])

  const loadTrash = useCallback(async () => {
    try {
      const [ifaces, peers] = await Promise.all([listTrashedInterfaces(), listTrashedPeers()])
      setTrashedInterfaces(ifaces)
      setTrashedPeers(peers)
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to load trash") })
    }
  }, [])

  const loadStatus = useCallback(async (id: number) => {
    try {
      const data = await getInterfaceStatus(id)
      setStatus(data)
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to load status") })
    }
  }, [])

  useEffect(() => {
    loadInterfaces()
    loadTrash()
  }, [loadInterfaces, loadTrash])

  // poll status every 5s for the selected interface
  useEffect(() => {
    if (!selectedId) {
      setStatus(null)
      return
    }
    loadStatus(selectedId)
    const t = setInterval(() => loadStatus(selectedId), 5000)
    return () => clearInterval(t)
  }, [selectedId, loadStatus])

  const iface = status?.interface
  const peers = iface?.peers ?? []

  async function handleCreateInterface(values: InterfaceFormValues) {
    setSubmitting(true)
    try {
      const { data, message } = await createInterface(values)
      setCreateOpen(false)
      await loadInterfaces()
      setSelectedId(data.id)
      if (message && message !== "interface created") {
        setBanner({ kind: "info", text: message })
      }
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to create interface") })
    } finally {
      setSubmitting(false)
    }
  }

  async function handleAddPeer(values: PeerFormValues) {
    if (!selectedId) return
    setSubmitting(true)
    try {
      const { message } = await createPeer(selectedId, values)
      setAddPeerOpen(false)
      await loadStatus(selectedId)
      if (message && message !== "peer created") {
        setBanner({ kind: "info", text: message })
      }
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to add peer") })
    } finally {
      setSubmitting(false)
    }
  }

  async function handleTogglePeer(p: Peer) {
    try {
      await updatePeer(p.id, {
        name: p.name,
        client_allowed_ips: p.client_allowed_ips,
        persistent_keepalive: p.persistent_keepalive,
        enabled: !p.enabled,
      })
      if (selectedId) await loadStatus(selectedId)
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to toggle peer") })
    }
  }

  async function handleDeletePeer(p: Peer) {
    if (!confirm(`Move peer "${p.name}" to Trash?`)) return
    try {
      const msg = await deletePeer(p.id)
      setBanner({ kind: "info", text: msg ?? "Peer moved to trash" })
      if (selectedId) await loadStatus(selectedId)
      await loadTrash()
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to delete peer") })
    }
  }

  async function handleSync() {
    if (!selectedId) return
    try {
      const msg = await syncInterface(selectedId)
      setBanner({ kind: "info", text: msg ?? "Applied to kernel" })
      await loadStatus(selectedId)
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Sync failed") })
    }
  }

  async function handleDeleteInterface() {
    if (!selectedId || !iface) return
    if (!confirm(`Move interface "${iface.name}" and all its peers to Trash?`)) return
    try {
      const msg = await deleteInterface(selectedId)
      setBanner({ kind: "info", text: msg ?? "Interface moved to trash" })
      setSelectedId(null)
      setStatus(null)
      await loadInterfaces()
      await loadTrash()
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to delete interface") })
    }
  }

  async function handleRestoreInterface(i: WGInterface) {
    try {
      const msg = await restoreInterface(i.id)
      setBanner({ kind: "info", text: msg ?? "Interface restored" })
      await loadInterfaces()
      await loadTrash()
      setSelectedId(i.id)
      await loadStatus(i.id)
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to restore interface") })
    }
  }

  async function handlePurgeInterface(i: WGInterface) {
    if (!confirm(`Permanently delete interface "${i.name}" and all its peers? This cannot be undone.`)) return
    try {
      const msg = await purgeInterface(i.id)
      setBanner({ kind: "info", text: msg ?? "Interface permanently deleted" })
      await loadInterfaces()
      await loadTrash()
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to permanently delete interface") })
    }
  }

  async function handleRestorePeer(p: Peer) {
    try {
      const msg = await restorePeer(p.id)
      setBanner({ kind: "info", text: msg ?? "Peer restored" })
      if (selectedId) await loadStatus(selectedId)
      await loadTrash()
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to restore peer") })
    }
  }

  async function handlePurgePeer(p: Peer) {
    if (!confirm(`Permanently delete peer "${p.name}"? This cannot be undone.`)) return
    try {
      const msg = await purgePeer(p.id)
      setBanner({ kind: "info", text: msg ?? "Peer permanently deleted" })
      if (selectedId) await loadStatus(selectedId)
      await loadTrash()
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to permanently delete peer") })
    }
  }

  return (
    <div className="space-y-6">
      {banner && (
        <div
          className={
            banner.kind === "error"
              ? "flex items-start justify-between rounded-md border border-destructive/50 bg-destructive/10 p-3 text-sm text-destructive"
              : "flex items-start justify-between rounded-md border border-amber-300 bg-amber-50 p-3 text-sm text-amber-800"
          }
        >
          <span>{banner.text}</span>
          <button onClick={() => setBanner(null)} className="ml-3 text-xs underline">
            dismiss
          </button>
        </div>
      )}

      {/* Interface selector row */}
      <div className="flex flex-wrap items-center gap-3">
        <span className="text-sm font-medium text-muted-foreground">Interface:</span>
        <select
          value={selectedId ?? ""}
          onChange={(e) => setSelectedId(Number(e.target.value) || null)}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm"
        >
          {interfaces.length === 0 && <option value="">No interfaces</option>}
          {interfaces.map((i) => (
            <option key={i.id} value={i.id}>
              {i.name} ({i.address})
            </option>
          ))}
        </select>

        <Dialog open={createOpen} onOpenChange={setCreateOpen}>
          <DialogTrigger asChild>
            <Button variant="outline" size="sm">
              <Plus />
              New interface
            </Button>
          </DialogTrigger>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Create WireGuard interface</DialogTitle>
              <DialogDescription>
                The server keypair is generated automatically.
              </DialogDescription>
            </DialogHeader>
            <InterfaceForm onSubmit={handleCreateInterface} submitting={submitting} />
          </DialogContent>
        </Dialog>
      </div>

      {/* Interface summary */}
      {iface && (
        <Card>
          <CardHeader className="flex flex-row items-start justify-between">
            <div className="space-y-1">
              <CardTitle className="flex items-center gap-2">
                <Server className="h-5 w-5" />
                {iface.name}
                {status?.kernel_up ? (
                  <Badge variant="success">kernel up</Badge>
                ) : (
                  <Badge variant="muted">kernel down</Badge>
                )}
                {!iface.enabled && <Badge variant="destructive">disabled</Badge>}
              </CardTitle>
              <CardDescription>
                {iface.endpoint}:{iface.listen_port} · {iface.address} · DNS {iface.dns || "—"}
              </CardDescription>
              <p className="break-all font-mono text-xs text-muted-foreground">
                pubkey: {iface.public_key}
              </p>
              {status?.kernel_message && (
                <p className="text-xs text-amber-700">{status.kernel_message}</p>
              )}
            </div>
            <div className="flex gap-2">
              <Button variant="outline" size="sm" onClick={handleSync}>
                <Power />
                Apply
              </Button>
              <Button
                variant="outline"
                size="icon"
                onClick={() => selectedId && loadStatus(selectedId)}
                title="Refresh"
              >
                <RefreshCw />
              </Button>
              <Button variant="outline" size="icon" onClick={handleDeleteInterface} title="Delete interface">
                <Trash2 className="text-destructive" />
              </Button>
            </div>
          </CardHeader>

          <CardContent className="space-y-4">
            <div className="flex items-center justify-between">
              <h3 className="text-sm font-semibold">Peers ({peers.length})</h3>
              <Dialog open={addPeerOpen} onOpenChange={setAddPeerOpen}>
                <DialogTrigger asChild>
                  <Button size="sm">
                    <Plus />
                    Add peer
                  </Button>
                </DialogTrigger>
                <DialogContent>
                  <DialogHeader>
                    <DialogTitle>Add peer to {iface.name}</DialogTitle>
                    <DialogDescription>
                      Keys and tunnel IP are assigned automatically.
                    </DialogDescription>
                  </DialogHeader>
                  <PeerForm onSubmit={handleAddPeer} submitting={submitting} />
                </DialogContent>
              </Dialog>
            </div>

            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Status</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Tunnel IP</TableHead>
                  <TableHead>Handshake</TableHead>
                  <TableHead className="text-right">Transfer (↓ / ↑)</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {peers.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="py-8 text-center text-muted-foreground">
                      No peers yet. Add one to generate a client config.
                    </TableCell>
                  </TableRow>
                ) : (
                  peers.map((p) => (
                    <TableRow key={p.id} className={!p.enabled ? "opacity-50" : ""}>
                      <TableCell>
                        {p.online ? (
                          <Badge variant="success" className="gap-1">
                            <Wifi className="h-3 w-3" /> online
                          </Badge>
                        ) : (
                          <Badge variant="muted" className="gap-1">
                            <WifiOff className="h-3 w-3" /> offline
                          </Badge>
                        )}
                      </TableCell>
                      <TableCell className="font-medium">{p.name}</TableCell>
                      <TableCell className="font-mono text-xs">{p.assigned_ip}</TableCell>
                      <TableCell className="text-sm text-muted-foreground">
                        {formatHandshake(p.last_handshake)}
                      </TableCell>
                      <TableCell className="text-right font-mono text-xs">
                        {formatBytes(p.rx_bytes)} / {formatBytes(p.tx_bytes)}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1">
                          <Button
                            variant="ghost"
                            size="icon"
                            title="Show config / QR"
                            onClick={() => setConfigPeer(p)}
                          >
                            <QrCode />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            title={p.enabled ? "Disable" : "Enable"}
                            onClick={() => handleTogglePeer(p)}
                          >
                            <Power className={p.enabled ? "text-green-600" : "text-muted-foreground"} />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            title="Delete"
                            onClick={() => handleDeletePeer(p)}
                          >
                            <Trash2 className="text-destructive" />
                          </Button>
                        </div>
                      </TableCell>
                    </TableRow>
                  ))
                )}
              </TableBody>
            </Table>
          </CardContent>
        </Card>
      )}

      {(trashedInterfaces.length > 0 || trashedPeers.length > 0) && (
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <Archive className="h-5 w-5" />
              Trash
              <Badge variant="muted">{trashedInterfaces.length + trashedPeers.length}</Badge>
            </CardTitle>
            <CardDescription>
              Restore deleted items, or permanently delete them so Trash does not pile up.
            </CardDescription>
          </CardHeader>
          <CardContent className="space-y-6">
            <div className="space-y-2">
              <h3 className="text-sm font-semibold">Interfaces ({trashedInterfaces.length})</h3>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Address</TableHead>
                    <TableHead>Deleted</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {trashedInterfaces.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={4} className="py-4 text-center text-muted-foreground">
                        No deleted interfaces.
                      </TableCell>
                    </TableRow>
                  ) : (
                    trashedInterfaces.map((i) => (
                      <TableRow key={i.id}>
                        <TableCell className="font-medium">{i.name}</TableCell>
                        <TableCell className="font-mono text-xs">{i.address}</TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {formatDeletedAt(i.deleted_at)}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button variant="ghost" size="icon" title="Restore" onClick={() => handleRestoreInterface(i)}>
                              <Undo2 className="text-green-600" />
                            </Button>
                            <Button variant="ghost" size="icon" title="Delete permanently" onClick={() => handlePurgeInterface(i)}>
                              <Trash2 className="text-destructive" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>

            <div className="space-y-2">
              <h3 className="text-sm font-semibold">Peers ({trashedPeers.length})</h3>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>Name</TableHead>
                    <TableHead>Interface ID</TableHead>
                    <TableHead>Tunnel IP</TableHead>
                    <TableHead>Deleted</TableHead>
                    <TableHead className="text-right">Actions</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {trashedPeers.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={5} className="py-4 text-center text-muted-foreground">
                        No deleted peers.
                      </TableCell>
                    </TableRow>
                  ) : (
                    trashedPeers.map((p) => (
                      <TableRow key={p.id}>
                        <TableCell className="font-medium">{p.name}</TableCell>
                        <TableCell className="font-mono text-xs">{p.interface_id}</TableCell>
                        <TableCell className="font-mono text-xs">{p.assigned_ip}</TableCell>
                        <TableCell className="text-xs text-muted-foreground">
                          {formatDeletedAt(p.deleted_at)}
                        </TableCell>
                        <TableCell className="text-right">
                          <div className="flex justify-end gap-1">
                            <Button variant="ghost" size="icon" title="Restore" onClick={() => handleRestorePeer(p)}>
                              <Undo2 className="text-green-600" />
                            </Button>
                            <Button variant="ghost" size="icon" title="Delete permanently" onClick={() => handlePurgePeer(p)}>
                              <Trash2 className="text-destructive" />
                            </Button>
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>
          </CardContent>
        </Card>
      )}

      {!iface && interfaces.length === 0 && (
        <Card>
          <CardContent className="py-12 text-center text-muted-foreground">
            No interfaces yet. Click <strong>New interface</strong> to set up your
            WireGuard concentrator.
          </CardContent>
        </Card>
      )}

      <PeerConfigDialog iface={iface ?? null} peer={configPeer} onClose={() => setConfigPeer(null)} />
    </div>
  )
}
