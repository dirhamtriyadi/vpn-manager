import { useCallback, useEffect, useState } from "react"
import { Link } from "react-router-dom"
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
  syncInterface,
  updatePeer,
} from "./api"
import { ListControls } from "./ListControls"
import { PaginationControls } from "./PaginationControls"
import type { InterfaceStatus, ListParams, PaginationMeta, Peer, WGInterface } from "./types"


const DEFAULT_META: PaginationMeta = { page: 1, per_page: 10, total: 0, last_page: 1 }
const DEFAULT_INTERFACE_PARAMS: Required<Pick<ListParams, "page" | "per_page" | "sort_by" | "sort_order">> & Pick<ListParams, "search"> = { page: 1, per_page: 10, sort_by: "id", sort_order: "asc", search: "" }
const DEFAULT_PEER_PARAMS: Required<Pick<ListParams, "page" | "per_page" | "sort_by" | "sort_order">> & Pick<ListParams, "search"> = { page: 1, per_page: 10, sort_by: "id", sort_order: "asc", search: "" }
const INTERFACE_SORT_OPTIONS = [
  { value: "id", label: "ID" },
  { value: "name", label: "Name" },
  { value: "listen_port", label: "Listen port" },
  { value: "address", label: "Address" },
  { value: "endpoint", label: "Endpoint" },
  { value: "enabled", label: "Enabled" },
  { value: "created_at", label: "Created" },
]
const PEER_SORT_OPTIONS = [
  { value: "id", label: "ID" },
  { value: "name", label: "Name" },
  { value: "assigned_ip", label: "Tunnel IP" },
  { value: "enabled", label: "Enabled" },
  { value: "created_at", label: "Created" },
]

export function Dashboard() {
  const [interfaces, setInterfaces] = useState<WGInterface[]>([])
  const [interfacesMeta, setInterfacesMeta] = useState<PaginationMeta>(DEFAULT_META)
  const [peersMeta, setPeersMeta] = useState<PaginationMeta>(DEFAULT_META)
  const [interfaceParams, setInterfaceParams] = useState(DEFAULT_INTERFACE_PARAMS)
  const [peerParams, setPeerParams] = useState(DEFAULT_PEER_PARAMS)
  const [selectedId, setSelectedId] = useState<number | null>(null)
  const [status, setStatus] = useState<InterfaceStatus | null>(null)
  const [banner, setBanner] = useState<{ kind: "error" | "info"; text: string } | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const [createOpen, setCreateOpen] = useState(false)
  const [addPeerOpen, setAddPeerOpen] = useState(false)
  const [configPeer, setConfigPeer] = useState<Peer | null>(null)

  const loadInterfaces = useCallback(async () => {
    try {
      const result = await listInterfaces(interfaceParams)
      setInterfaces(result.data)
      setInterfacesMeta(result.meta)
      setSelectedId((prev) => {
        if (prev && result.data.some((i) => i.id === prev)) return prev
        return result.data[0]?.id ?? null
      })
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to load interfaces. Is the API running?") })
    }
  }, [interfaceParams])

  const loadStatus = useCallback(async (id: number) => {
    try {
      const result = await getInterfaceStatus(id, peerParams)
      setStatus(result.data)
      setPeersMeta(result.meta)
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to load status") })
    }
  }, [peerParams])

  useEffect(() => {
    loadInterfaces()
  }, [loadInterfaces])

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

  useEffect(() => {
    setPeerParams((prev) => ({ ...prev, page: 1 }))
  }, [selectedId])

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
      throw e
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
      throw e
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
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to delete interface") })
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
        <PaginationControls meta={interfacesMeta} onPageChange={(page) => setInterfaceParams((prev) => ({ ...prev, page }))} />

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

        <Button variant="outline" size="sm" asChild>
          <Link to="/trash">
            <Archive />
            Trash
          </Link>
        </Button>
      </div>

      <ListControls
        params={interfaceParams}
        sortOptions={INTERFACE_SORT_OPTIONS}
        searchPlaceholder="Search interfaces..."
        onChange={(next) => setInterfaceParams((prev) => ({ ...prev, ...next }))}
      />

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
              <h3 className="text-sm font-semibold">Peers ({peersMeta.total})</h3>
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

            <ListControls
              params={peerParams}
              sortOptions={PEER_SORT_OPTIONS}
              searchPlaceholder="Search peers..."
              onChange={(next) => setPeerParams((prev) => ({ ...prev, ...next }))}
            />

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
            <PaginationControls meta={peersMeta} onPageChange={(page) => setPeerParams((prev) => ({ ...prev, page }))} />
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
