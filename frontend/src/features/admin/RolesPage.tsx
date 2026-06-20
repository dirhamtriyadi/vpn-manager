import { useCallback, useEffect, useState } from "react"
import { useForm } from "react-hook-form"
import { zodResolver } from "@hookform/resolvers/zod"
import { Pencil, Plus, Shield, Trash2 } from "lucide-react"
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
import { roleSchema, type RoleFormValues } from "@/schemas/role"
import { ConfirmDialog } from "./ConfirmDialog"
import { PermissionPicker } from "./PermissionPicker"
import {
  createRole,
  deleteRole,
  listPermissions,
  listRoles,
  setRolePermissions,
  updateRole,
} from "./api"
import type { Permission, Role } from "./types"

const SUPER_ADMIN_ROLE = "super-admin"

export function RolesPage() {
  const { hasPermission } = useAuth()
  const canManage = hasPermission("roles.manage")
  const [roles, setRoles] = useState<Role[]>([])
  const [permissions, setPermissions] = useState<Permission[]>([])
  const [error, setError] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [editing, setEditing] = useState<Role | null>(null)
  const [creating, setCreating] = useState(false)
  const [toDelete, setToDelete] = useState<Role | null>(null)

  const load = useCallback(() => {
    setLoading(true)
    Promise.all([listRoles(), listPermissions()])
      .then(([r, p]) => {
        setRoles(r)
        setPermissions(p)
        setError(null)
      })
      .catch((e) => setError(apiErrorMessage(e, "Failed to load roles")))
      .finally(() => setLoading(false))
  }, [])

  useEffect(load, [load])

  async function confirmDelete() {
    if (!toDelete) return
    try {
      await deleteRole(toDelete.id)
      setToDelete(null)
      load()
    } catch (e) {
      setError(apiErrorMessage(e, "Failed to delete role"))
      setToDelete(null)
    }
  }

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="flex items-center gap-2 text-2xl font-semibold tracking-tight">
            <Shield className="h-6 w-6" />
            Roles
          </h2>
          <p className="text-sm text-muted-foreground">
            Bundle permissions into roles, then assign roles to users.
          </p>
        </div>
        {canManage && (
          <Button onClick={() => setCreating(true)}>
            <Plus />
            New role
          </Button>
        )}
      </div>

      {error && (
        <div className="rounded-md border border-destructive/30 bg-destructive/10 p-3 text-sm text-destructive">
          {error}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">All roles</CardTitle>
          <CardDescription>{roles.length} role(s)</CardDescription>
        </CardHeader>
        <CardContent>
          {loading ? (
            <p className="text-sm text-muted-foreground">Loading roles…</p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Name</TableHead>
                  <TableHead>Description</TableHead>
                  <TableHead>Permissions</TableHead>
                  <TableHead className="text-right">Actions</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {roles.map((role) => {
                  const isSuper = role.name === SUPER_ADMIN_ROLE
                  return (
                    <TableRow key={role.id}>
                      <TableCell className="font-medium">
                        <span className="flex items-center gap-2">
                          {role.name}
                          {isSuper && <Badge variant="muted">protected</Badge>}
                        </span>
                      </TableCell>
                      <TableCell className="text-muted-foreground">
                        {role.description || "—"}
                      </TableCell>
                      <TableCell>
                        {role.permissions.some((p) => p.name === "*") ? (
                          <Badge>all (*)</Badge>
                        ) : (
                          <Badge variant="muted">{role.permissions.length}</Badge>
                        )}
                      </TableCell>
                      <TableCell className="text-right">
                        {canManage && (
                          <div className="flex justify-end gap-1">
                            <Button
                              variant="outline"
                              size="sm"
                              onClick={() => setEditing(role)}
                            >
                              <Pencil />
                              Edit
                            </Button>
                            <Button
                              variant="outline"
                              size="sm"
                              disabled={isSuper}
                              title={isSuper ? "The super-admin role is protected" : undefined}
                              onClick={() => setToDelete(role)}
                            >
                              <Trash2 />
                            </Button>
                          </div>
                        )}
                      </TableCell>
                    </TableRow>
                  )
                })}
                {roles.length === 0 && (
                  <TableRow>
                    <TableCell colSpan={4} className="text-center text-muted-foreground">
                      No roles.
                    </TableCell>
                  </TableRow>
                )}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      {(creating || editing) && (
        <RoleDialog
          role={editing}
          permissions={permissions}
          onClose={() => {
            setCreating(false)
            setEditing(null)
          }}
          onSaved={() => {
            setCreating(false)
            setEditing(null)
            load()
          }}
        />
      )}

      <ConfirmDialog
        open={Boolean(toDelete)}
        title="Delete role"
        description={`Delete role "${toDelete?.name}"? Users keep their other roles and direct permissions.`}
        confirmLabel="Delete"
        onConfirm={confirmDelete}
        onCancel={() => setToDelete(null)}
      />
    </div>
  )
}

function RoleDialog({
  role,
  permissions,
  onClose,
  onSaved,
}: {
  role: Role | null
  permissions: Permission[]
  onClose: () => void
  onSaved: () => void
}) {
  const isEdit = Boolean(role)
  const isSuper = role?.name === SUPER_ADMIN_ROLE
  const [selected, setSelected] = useState<number[]>(
    () => role?.permissions.map((p) => p.id) ?? [],
  )
  const [formError, setFormError] = useState<string | null>(null)
  const [submitting, setSubmitting] = useState(false)

  const {
    register,
    handleSubmit,
    setError,
    formState: { errors },
  } = useForm<RoleFormValues>({
    resolver: zodResolver(roleSchema),
    defaultValues: { name: role?.name ?? "", description: role?.description ?? "" },
  })

  async function onSubmit(values: RoleFormValues) {
    setFormError(null)
    setSubmitting(true)
    try {
      if (role) {
        if (!isSuper) {
          await updateRole(role.id, values)
          await setRolePermissions(role.id, selected)
        }
      } else {
        await createRole({ ...values, permission_ids: selected })
      }
      onSaved()
    } catch (err) {
      if (!applyServerValidationErrors<RoleFormValues>(setError, err)) {
        setFormError(apiErrorMessage(err, "Failed to save role"))
      }
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <Dialog open onOpenChange={(open) => !open && onClose()}>
      <DialogContent className="max-w-2xl">
        <DialogHeader>
          <DialogTitle>{isEdit ? "Edit role" : "New role"}</DialogTitle>
          <DialogDescription>
            {isSuper
              ? "The super-admin role is protected and cannot be changed."
              : "Pick the permissions this role grants."}
          </DialogDescription>
        </DialogHeader>
        <form onSubmit={handleSubmit(onSubmit)} className="space-y-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="space-y-1.5">
              <Label htmlFor="role-name">Name</Label>
              <Input
                id="role-name"
                readOnly={isSuper}
                className={isSuper ? "bg-muted text-muted-foreground" : undefined}
                {...register("name")}
              />
              {errors.name && (
                <p className="text-xs text-destructive">{errors.name.message}</p>
              )}
            </div>
            <div className="space-y-1.5">
              <Label htmlFor="role-description">Description</Label>
              <Input id="role-description" readOnly={isSuper} {...register("description")} />
              {errors.description && (
                <p className="text-xs text-destructive">{errors.description.message}</p>
              )}
            </div>
          </div>

          <div className="space-y-1.5">
            <Label>Permissions</Label>
            <PermissionPicker
              permissions={permissions}
              selected={selected}
              onChange={setSelected}
              disabled={isSuper}
            />
          </div>

          {formError && <p className="text-sm text-destructive">{formError}</p>}

          <DialogFooter>
            <Button type="button" variant="outline" onClick={onClose}>
              Cancel
            </Button>
            <Button type="submit" disabled={submitting || isSuper}>
              {submitting ? "Saving…" : isEdit ? "Save changes" : "Create role"}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
