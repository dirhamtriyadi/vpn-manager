import { useCallback, useEffect, useState } from "react"
import { Link } from "react-router-dom"
import { ArrowLeft, Archive, Trash2, Undo2 } from "lucide-react"
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
import { apiErrorMessage } from "@/lib/api"
import {
  listTrashedInterfaces,
  listTrashedPeers,
  purgeInterface,
  purgePeer,
  restoreInterface,
  restorePeer,
} from "./api"
import { ListControls } from "./ListControls"
import { PaginationControls } from "./PaginationControls"
import type { ListParams, PaginationMeta, Peer, WGInterface } from "./types"

const DEFAULT_META: PaginationMeta = { page: 1, per_page: 10, total: 0, last_page: 1 }
const DEFAULT_TRASH_INTERFACE_PARAMS: Required<Pick<ListParams, "page" | "per_page" | "sort_by" | "sort_order">> & Pick<ListParams, "search"> = { page: 1, per_page: 10, sort_by: "deleted_at", sort_order: "desc", search: "" }
const DEFAULT_TRASH_PEER_PARAMS: Required<Pick<ListParams, "page" | "per_page" | "sort_by" | "sort_order">> & Pick<ListParams, "search"> = { page: 1, per_page: 10, sort_by: "deleted_at", sort_order: "desc", search: "" }
const TRASH_INTERFACE_SORT_OPTIONS = [
  { value: "deleted_at", label: "Deleted" },
  { value: "id", label: "ID" },
  { value: "name", label: "Name" },
  { value: "address", label: "Address" },
  { value: "created_at", label: "Created" },
]
const TRASH_PEER_SORT_OPTIONS = [
  { value: "deleted_at", label: "Deleted" },
  { value: "id", label: "ID" },
  { value: "name", label: "Name" },
  { value: "interface_id", label: "Interface ID" },
  { value: "assigned_ip", label: "Tunnel IP" },
  { value: "created_at", label: "Created" },
]

function formatDeletedAt(value: WGInterface["deleted_at"] | Peer["deleted_at"]): string {
  const raw = typeof value === "string" ? value : value?.Time
  return raw ? new Date(raw).toLocaleString() : "—"
}

export function TrashPage() {
  const [trashedInterfaces, setTrashedInterfaces] = useState<WGInterface[]>([])
  const [trashedPeers, setTrashedPeers] = useState<Peer[]>([])
  const [trashInterfacesMeta, setTrashInterfacesMeta] = useState<PaginationMeta>(DEFAULT_META)
  const [trashPeersMeta, setTrashPeersMeta] = useState<PaginationMeta>(DEFAULT_META)
  const [trashInterfaceParams, setTrashInterfaceParams] = useState(DEFAULT_TRASH_INTERFACE_PARAMS)
  const [trashPeerParams, setTrashPeerParams] = useState(DEFAULT_TRASH_PEER_PARAMS)
  const [banner, setBanner] = useState<{ kind: "error" | "info"; text: string } | null>(null)

  const loadTrash = useCallback(async () => {
    try {
      const [ifaces, peers] = await Promise.all([
        listTrashedInterfaces(trashInterfaceParams),
        listTrashedPeers(trashPeerParams),
      ])
      setTrashedInterfaces(ifaces.data)
      setTrashInterfacesMeta(ifaces.meta)
      setTrashedPeers(peers.data)
      setTrashPeersMeta(peers.meta)
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to load trash") })
    }
  }, [trashInterfaceParams, trashPeerParams])

  useEffect(() => {
    loadTrash()
  }, [loadTrash])

  async function handleRestoreInterface(i: WGInterface) {
    try {
      const msg = await restoreInterface(i.id)
      setBanner({ kind: "info", text: msg ?? "Interface restored" })
      await loadTrash()
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to restore interface") })
    }
  }

  async function handlePurgeInterface(i: WGInterface) {
    if (!confirm(`Permanently delete interface "${i.name}" and all its peers? This cannot be undone.`)) return
    try {
      const msg = await purgeInterface(i.id)
      setBanner({ kind: "info", text: msg ?? "Interface permanently deleted" })
      await loadTrash()
    } catch (e) {
      setBanner({ kind: "error", text: apiErrorMessage(e, "Failed to permanently delete interface") })
    }
  }

  async function handleRestorePeer(p: Peer) {
    try {
      const msg = await restorePeer(p.id)
      setBanner({ kind: "info", text: msg ?? "Peer restored" })
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

      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-xl font-semibold">
            <Archive className="h-5 w-5" />
            Trash
            <Badge variant="muted">{trashInterfacesMeta.total + trashPeersMeta.total}</Badge>
          </h2>
          <p className="text-sm text-muted-foreground">
            Restore deleted items, or permanently delete them so Trash does not pile up.
          </p>
        </div>
        <Button variant="outline" size="sm" asChild>
          <Link to="/">
            <ArrowLeft />
            Back to dashboard
          </Link>
        </Button>
      </div>

      <Card>
        <CardHeader>
          <CardTitle>Deleted interfaces</CardTitle>
          <CardDescription>Interfaces moved to Trash. Restoring also restores their soft-deleted peers.</CardDescription>
        </CardHeader>
        <CardContent>
          <ListControls
            params={trashInterfaceParams}
            sortOptions={TRASH_INTERFACE_SORT_OPTIONS}
            searchPlaceholder="Search deleted interfaces..."
            onChange={(next) => setTrashInterfaceParams((prev) => ({ ...prev, ...next }))}
          />
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
                  <TableCell colSpan={4} className="py-8 text-center text-muted-foreground">
                    No deleted interfaces.
                  </TableCell>
                </TableRow>
              ) : (
                trashedInterfaces.map((i) => (
                  <TableRow key={i.id}>
                    <TableCell className="font-medium">{i.name}</TableCell>
                    <TableCell className="font-mono text-xs">{i.address}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatDeletedAt(i.deleted_at)}</TableCell>
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
          <PaginationControls meta={trashInterfacesMeta} onPageChange={(page) => setTrashInterfaceParams((prev) => ({ ...prev, page }))} />
        </CardContent>
      </Card>

      <Card>
        <CardHeader>
          <CardTitle>Deleted peers</CardTitle>
          <CardDescription>Peers moved to Trash separately from active peer lists.</CardDescription>
        </CardHeader>
        <CardContent>
          <ListControls
            params={trashPeerParams}
            sortOptions={TRASH_PEER_SORT_OPTIONS}
            searchPlaceholder="Search deleted peers..."
            onChange={(next) => setTrashPeerParams((prev) => ({ ...prev, ...next }))}
          />
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
                  <TableCell colSpan={5} className="py-8 text-center text-muted-foreground">
                    No deleted peers.
                  </TableCell>
                </TableRow>
              ) : (
                trashedPeers.map((p) => (
                  <TableRow key={p.id}>
                    <TableCell className="font-medium">{p.name}</TableCell>
                    <TableCell className="font-mono text-xs">{p.interface_id}</TableCell>
                    <TableCell className="font-mono text-xs">{p.assigned_ip}</TableCell>
                    <TableCell className="text-xs text-muted-foreground">{formatDeletedAt(p.deleted_at)}</TableCell>
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
          <PaginationControls meta={trashPeersMeta} onPageChange={(page) => setTrashPeerParams((prev) => ({ ...prev, page }))} />
        </CardContent>
      </Card>
    </div>
  )
}
