import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import type { ListParams } from "./types"

interface SortOption {
  value: string
  label: string
}

export function ListControls({
  params,
  sortOptions,
  onChange,
  searchPlaceholder = "Search...",
}: {
  params: Required<Pick<ListParams, "per_page" | "sort_by" | "sort_order">> & Pick<ListParams, "search">
  sortOptions: SortOption[]
  onChange: (next: Partial<ListParams>) => void
  searchPlaceholder?: string
}) {
  return (
    <div className="flex flex-wrap items-end gap-3 rounded-md border bg-background/50 p-3">
      <div className="min-w-48 space-y-1">
        <Label className="text-xs">Search</Label>
        <Input
          value={params.search ?? ""}
          placeholder={searchPlaceholder}
          onChange={(e) => onChange({ search: e.target.value, page: 1 })}
        />
      </div>

      <div className="space-y-1">
        <Label className="text-xs">Per page</Label>
        <select
          value={params.per_page}
          onChange={(e) => onChange({ per_page: Number(e.target.value), page: 1 })}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm"
        >
          {[10, 25, 50, 100].map((n) => (
            <option key={n} value={n}>
              {n}
            </option>
          ))}
        </select>
      </div>

      <div className="space-y-1">
        <Label className="text-xs">Sort by</Label>
        <select
          value={params.sort_by}
          onChange={(e) => onChange({ sort_by: e.target.value, page: 1 })}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm"
        >
          {sortOptions.map((option) => (
            <option key={option.value} value={option.value}>
              {option.label}
            </option>
          ))}
        </select>
      </div>

      <div className="space-y-1">
        <Label className="text-xs">Order</Label>
        <select
          value={params.sort_order}
          onChange={(e) => onChange({ sort_order: e.target.value as "asc" | "desc", page: 1 })}
          className="h-9 rounded-md border border-input bg-background px-3 text-sm"
        >
          <option value="asc">ASC</option>
          <option value="desc">DESC</option>
        </select>
      </div>
    </div>
  )
}
