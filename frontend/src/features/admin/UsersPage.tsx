import { useCallback, useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Pencil, Plus, Trash2, Users } from "lucide-react"
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
import { apiErrorMessage, applyServerValidationErrors } from "@/lib/api"
import { useAuth } from "@/features/auth/AuthContext"
import { userFormSchema, type UserFormValues } from "@/schemas/user"
import { ConfirmDialog } from "./ConfirmDialog"
import { PermissionPicker } from "./PermissionPicker"
import {
  createUser,
  deleteUser,
  listPermissions,
  listRoles,
  listUsers,
  setUserPermissions,
  setUserRoles,
  updateUser,
} from "./api"
import type { PaginationMeta, Permission, Role, User } from "./types"

const DEFAULT_META: PaginationMeta = { page: 1, per_page: 10, total: 0, last_page: 1 }

export function UsersPage() {
  const { hasPermission, user: currentUser } = useAuth()
  const canCreate = hasPermission("users.create")
  const canUpdate = hasPermission("users.update")
  const canDelete = hasPermission("users.delete")

  const [users, setUsers] = useState<User[]>([])
  const [meta, setMeta] = useState<PaginationMeta>(DEFAULT_META)
  const [roles, setRoles] = useState<Role[]>([])
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [page, setPage] = useState(1)
  const [search, setSearch] = useState("")
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [creating, setCreating] = useState(false)
  const [editing, setEditing] = useState<User | null>(null)
  const [toDelete, setToDelete] = useState<User | null>(null)

  // Roles + permission catalog are needed by the editor; load once.
  useEffect(() => {
    Promise.all([listRoles(), listPermissions()])
      .then(([r, p]) => {
        setRoles(r)
        setPermissions(p)
      })
      .catch((e) => setError(apiErrorMessage(e, "Failed to load roles/permissions")))
  }, [])

  const loadUsers = useCallback(() => {
    setLoading(true)
    listUsers({ page, per_page: 10, search: search || undefined })
      .then((res) => {
        setUsers(res.data)
        setMeta(res.meta)
        setError(null)
      })
      .catch((e) => setError(apiErrorMessage(e, "Failed to load users")))
      .finally(() => setLoading(false))
  }, [page, search])

  useEffect(loadUsers, [loadUsers])

  async function confirmDelete() {
    if (!toDelete) return
    try {
      await deleteUser(toDelete.id)
      setToDelete(null)
      loadUsers()
    } catch (e) {
      setError(apiErrorMessage(e, "Failed to delete user"))
      setToDelete(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
            <Users className="h-6 w-6" />
            Users
          </h2>
          <p className="text-sm text-muted-foreground">
            Panel accounts. Assign roles and optional direct permissions; each user
            only manages the VPNs they own (super admins see everything).
          </p>
        </div>
        {canCreate && (
          <Button onClick={() => setCreating(true)}>
            <Plus />
            New user
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <Card>
        <CardHeader className="flex flex-row items-center justify-between gap-3 space-y-0">
          <div>
            <CardTitle className="text-base">All users</CardTitle>
            <CardDescription>{meta.total} user(s)</CardDescription>
          </div>
          <Input
            placeholder="Search username/name…"
            className="max-w-xs"
            value={search}
            onChange={(e) => {
              setPage(1)
              setSearch(e.target.value)
            }}
          />
        </CardHeader>
        <CardContent className="space-y-4">
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading users…</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Username</TableHead>
                  <TableHead>Name</TableHead>
                  <TableHead>Roles</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {users.map((u) => {
                  const isSelf = currentUser?.id === u.id
                  const isSuper = u.effective_permissions.includes("*")
                  return (
                    <TableRow key={u.id}>
                      <TableCell className="font-medium">
                        <span className="flex items-center gap-2">
                          {u.username}
                          {isSelf && <Badge variant="muted">you</Badge>}
                          {isSuper && <Badge>super admin</Badge>}
                        </span>
                      </TableCell>
                      <TableCell className="text-muted-foreground">{u.name || "—"}</TableCell>
                      <TableCell>
                        <div className="flex flex-wrap gap-1">
                          {u.roles.length === 0 ? (
                            <span className="text-xs text-muted-foreground">none</span>
                          ) : (
                            u.roles.map((r) => (
                              <Badge key={r.id} variant="secondary">
                                {r.name}
                              </Badge>
                            ))
                          )}
                          {u.direct_permissions.length > 0 && (
                            <Badge variant="outline" title="Direct permissions">
                              +{u.direct_permissions.length} direct
                            </Badge>
                          )}
                        </div>
                      </TableCell>
                      <TableCell>
                        {u.active ? (
                          <Badge variant="success">active</Badge>
                        ) : (
                          <Badge variant="muted">disabled</Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        <div className="flex justify-end gap-1">
                          {canUpdate && (
                            <Button variant="outline" size="sm" onClick={() => setEditing(u)}>
                              <Pencil />
                              Edit
                            </Button>
                          )}
                          {canDelete && (
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={isSelf}
                              title={isSelf ? "You cannot delete your own account" : undefined}
                              onClick={() => setToDelete(u)}
                            >
                              <Trash2 />
                            </Button>
                          )}
                        </div>
                      </TableCell>
                    </TableRow>
                  )
                })}
                {users.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={5} className="text-center text-muted-foreground">
                      No users.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}

          {meta.last_page > 1 && (
            <div className="flex items-center justify-between text-sm text-muted-foreground">
              <span>
                Page {meta.page} of {meta.last_page}
              </span>
              <div className="flex gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  disabled={meta.page <= 1}
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                >
                  Previous
                </Button>
                <Button
                  variant="outline"
                  size="sm"
                  disabled={meta.page >= meta.last_page}
                  onClick={() => setPage((p) => p + 1)}
                >
                  Next
                </Button>
              </div>
            </div>
          )}
        </CardContent>
      </Card>

      {(creating || editing) && (
        <UserDialog
          user={editing}
          roles={roles}
          permissions={permissions}
          isSelf={currentUser?.id === editing?.id}
          onClose={() => {
            setCreating(false)
            setEditing(null)
          }}
          onSaved={() => {
            setCreating(false)
            setEditing(null)
            loadUsers()
          }}
        />
      )}

      <ConfirmDialog
        open={Boolean(toDelete)}
        title="Delete user"
        description={`Delete user "${toDelete?.username}"? This cannot be undone.`}
        confirmLabel="Delete"
        onConfirm={confirmDelete}
        onCancel={() => setToDelete(null)}
      />
    </div>
  )
}

function RolePicker({
  roles,
  selected,
  onChange,
}: {
  roles: Role[]
  selected: number[]
  onChange: (ids: number[]) => void
}) {
  const set = new Set(selected)
  function toggle(id: number) {
    const next = new Set(set)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    onChange(Array.from(next))
  }
  if (roles.length === 0) {
    return <p className="text-sm text-muted-foreground">No roles defined.</p>
  }
  return (
    <div className="grid gap-1 rounded-md border p-3 sm:grid-cols-2">
      {roles.map((r) => (
        <label
          key={r.id}
          htmlFor={`role-${r.id}`}
          className="flex items-start gap-2 rounded px-1 py-0.5 hover:bg-muted/50"
          title={r.description}
        >
          <input
            id={`role-${r.id}`}
            type="checkbox"
            className="mt-0.5 h-4 w-4 rounded border-input"
            checked={set.has(r.id)}
            onChange={() => toggle(r.id)}
          />
          <span className="text-sm">{r.name}</span>
        </label>
      ))}
    </div>
  )
}

function UserDialog({
  user,
  roles,
  permissions,
  isSelf,
  onClose,
  onSaved,
}: {
  user: User | null
  roles: Role[]
  permissions: Permission[]
  isSelf: boolean
  onClose: () => void
  onSaved: () => void
}) {
  const isEdit = Boolean(user)
  const [selectedRoles, setSelectedRoles] = useState<number[]>(
    () => user?.roles.map((r) => r.id) ?? [],
  )
  const [selectedPerms, setSelectedPerms] = useState<number[]>(
    () => user?.direct_permissions.map((p) => p.id) ?? [],
  )
  const [showDirect, setShowDirect] = useState(
    () => (user?.direct_permissions.length ?? 0) > 0,
  )
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<UserFormValues>({
    resolver: zodResolver(userFormSchema),
    defaultValues: {
      username: user?.username ?? "",
      name: user?.name ?? "",
      password: "",
      active: user?.active ?? true,
    },
  })

  async function onSubmit(values: UserFormValues) {
    setFormError(null)
    const password = values.password && values.password.length > 0 ? values.password : undefined
    if (!isEdit && !password) {
      setError("password", { type: "manual", message: "Password is required" })
      return
    }
    setSubmitting(true)
    try {
      if (user) {
        await updateUser(user.id, {
          name: values.name || "",
          password,
          active: values.active,
        })
        await setUserRoles(user.id, selectedRoles)
        await setUserPermissions(user.id, selectedPerms)
      } else {
        await createUser({
          username: values.username,
          name: values.name || undefined,
          password: password as string,
          active: values.active ?? true,
          role_ids: selectedRoles,
          permission_ids: selectedPerms,
        })
      }
      onSaved()
    } catch (err) {
      if (!applyServerValidationErrors<UserFormValues>(setError, err)) {
        setFormError(apiErrorMessage(err, "Failed to save user"))
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? `Edit ${user?.username}` : "New user"}</DialogTitle>
          <DialogDescription>
            Assign roles, and optionally grant extra permissions directly.
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="user-username">Username</Label>
              <Input
                id="user-username"
                readOnly={isEdit}
                className={isEdit ? "bg-muted text-muted-foreground" : undefined}
                {...register("username")}
              />
              {errors.username && (
                <p className="text-xs text-destructive">{errors.username.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="user-name">Display name</Label>
              <Input id="user-name" {...register("name")} />
              {errors.name && (
                <p className="text-xs text-destructive">{errors.name.message}</p>
              )}
            </div>
          </div>

          <div className="space-y-1.5">
            <Label htmlFor="user-password">
              {isEdit ? "New password (leave blank to keep)" : "Password"}
            </Label>
            <Input
              id="user-password"
              type="password"
              autoComplete="new-password"
              {...register("password")}
            />
            {errors.password && (
              <p className="text-xs text-destructive">{errors.password.message}</p>
            )}
          </div>

          <label htmlFor="user-active" className="flex items-center gap-2">
            <input
              id="user-active"
              type="checkbox"
              className="h-4 w-4 rounded border-input"
              {...register("active")}
            />
            <span className="text-sm">Active</span>
            {isSelf && (
              <span className="text-xs text-muted-foreground">
                (you cannot disable your own account)
              </span>
            )}
          </label>

          <div className="space-y-1.5">
            <Label>Roles</Label>
            <RolePicker roles={roles} selected={selectedRoles} onChange={setSelectedRoles} />
          </div>

          <div className="space-y-1.5">
            <button
              type="button"
              className="text-sm text-primary hover:underline"
              onClick={() => setShowDirect((v) => !v)}
            >
              {showDirect ? "Hide" : "Add"} direct permissions
              {selectedPerms.length > 0 ? ` (${selectedPerms.length})` : ""}
            </button>
            {showDirect && (
              <>
                <p className="text-xs text-muted-foreground">
                  Granted on top of the user's roles. Use sparingly.
                </p>
                <PermissionPicker
                  permissions={permissions}
                  selected={selectedPerms}
                  onChange={setSelectedPerms}
                />
              </>
            )}
          </div>

          {formError && <p className="text-sm text-destructive">{formError}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? "Saving…" : isEdit ? "Save changes" : "Create user"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
