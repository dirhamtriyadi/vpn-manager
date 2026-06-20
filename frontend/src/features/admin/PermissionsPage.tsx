import { useEffect, useMemo, useState } from "react"
import { KeyRound } from "lucide-react"
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
import { listPermissions } from "./api"
import type { Permission } from "./types"

function groupKey(name: string): string {
  if (name === "*") return "Full access"
  const dot = name.indexOf(".")
  return dot === -1 ? name : name.slice(0, dot)
}

export function PermissionsPage() {
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    listPermissions()
      .then((data) => {
        setPermissions(data)
        setError(null)
      })
      .catch((e) => setError(apiErrorMessage(e, "Failed to load permissions")))
      .finally(() => setLoading(false))
  }, [])

  const groups = useMemo(() => {
    const byKey = new Map<string, Permission[]>()
    for (const p of permissions) {
      const key = groupKey(p.name)
      const arr = byKey.get(key) ?? []
      arr.push(p)
      byKey.set(key, arr)
    }
    return Array.from(byKey.entries())
      .map(([key, items]) => ({ key, items }))
      .sort((a, b) => {
        if (a.key === "Full access") return -1
        if (b.key === "Full access") return 1
        return a.key.localeCompare(b.key)
      })
  }, [permissions])

  return (
    <div className="space-y-6">
      <div>
        <h2 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
          <KeyRound className="h-6 w-6" />
          Permissions
        </h2>
        <p className="text-sm text-muted-foreground">
          The fixed catalog of capabilities. Assign them to roles, or directly to
          users, on the Roles and Users pages.
        </p>
      </div>

      {error && (
        <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      {loading ? (
        <p className="text-sm text-muted-foreground">Loading permissions…</p>
      ) : (
        groups.map((group) => (
          <Card key={group.key}>
            <CardHeader>
              <CardTitle className="flex items-center gap-2 text-base capitalize">
                {group.key}
                <Badge variant="muted">{group.items.length}</Badge>
              </CardTitle>
              <CardDescription>{group.items.length} permission(s)</CardDescription>
            </CardHeader>
            <CardContent>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead className="w-1/3">Name</TableHead>
                    <TableHead>Description</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {group.items.map((p) => (
                    <TableRow key={p.id}>
                      <TableCell className="font-mono text-xs">{p.name}</TableCell>
                      <TableCell className="text-muted-foreground">
                        {p.description || "—"}
                      </TableCell>
                    </TableRow>
                  ))}
                </TableBody>
              </Table>
            </CardContent>
          </Card>
        ))
      )}
    </div>
  )
}
