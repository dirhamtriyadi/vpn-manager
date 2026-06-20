import { useMemo } from "react"
import type { Permission } from "./types"

interface Props {
  permissions: Permission[]
  selected: number[]
  onChange: (ids: number[]) => void
  disabled?: boolean
}

interface Group {
  key: string
  label: string
  items: Permission[]
}

function groupName(name: string): string {
  if (name === "*") return "Full access"
  const dot = name.indexOf(".")
  return dot === -1 ? name : name.slice(0, dot)
}

/** Grouped checkbox list for assigning permissions. Permissions are bucketed by
 *  the resource prefix before the first dot; the wildcard "*" gets its own
 *  "Full access" group. */
export function PermissionPicker({ permissions, selected, onChange, disabled }: Props) {
  const groups = useMemo<Group[]>(() => {
    const byKey = new Map<string, Permission[]>()
    for (const p of permissions) {
      const key = groupName(p.name)
      const arr = byKey.get(key) ?? []
      arr.push(p)
      byKey.set(key, arr)
    }
    return Array.from(byKey.entries())
      .map(([key, items]) => ({ key, label: key, items }))
      .sort((a, b) => {
        if (a.key === "Full access") return -1
        if (b.key === "Full access") return 1
        return a.key.localeCompare(b.key)
      })
  }, [permissions])

  const selectedSet = useMemo(() => new Set(selected), [selected])

  function toggle(id: number) {
    if (disabled) return
    const next = new Set(selectedSet)
    if (next.has(id)) next.delete(id)
    else next.add(id)
    onChange(Array.from(next))
  }

  function toggleGroup(items: Permission[], allSelected: boolean) {
    if (disabled) return
    const next = new Set(selectedSet)
    for (const p of items) {
      if (allSelected) next.delete(p.id)
      else next.add(p.id)
    }
    onChange(Array.from(next))
  }

  if (permissions.length === 0) {
    return <p className="text-sm text-muted-foreground">No permissions available.</p>
  }

  return (
    <div className="max-h-72 space-y-3 overflow-auto rounded-md border p-3">
      {groups.map((group) => {
        const allSelected = group.items.every((p) => selectedSet.has(p.id))
        return (
          <div key={group.key} className="space-y-1.5">
            <div className="flex items-center justify-between">
              <span className="text-xs font-semibold uppercase tracking-wide text-muted-foreground">
                {group.label}
              </span>
              <button
                type="button"
                className="text-xs text-primary hover:underline disabled:opacity-50"
                onClick={() => toggleGroup(group.items, allSelected)}
                disabled={disabled}
              >
                {allSelected ? "Clear" : "Select all"}
              </button>
            </div>
            <div className="grid gap-1 sm:grid-cols-2">
              {group.items.map((p) => (
                <label
                  key={p.id}
                  htmlFor={`perm-${p.id}`}
                  className="flex items-start gap-2 rounded px-1 py-0.5 hover:bg-muted/50"
                  title={p.description}
                >
                  <input
                    id={`perm-${p.id}`}
                    type="checkbox"
                    className="mt-0.5 h-4 w-4 rounded border-input"
                    checked={selectedSet.has(p.id)}
                    onChange={() => toggle(p.id)}
                    disabled={disabled}
                  />
                  <span className="font-mono text-xs">{p.name}</span>
                </label>
              ))}
            </div>
          </div>
        )
      })}
    </div>
  )
}
