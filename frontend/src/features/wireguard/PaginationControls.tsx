import { Button } from "@/components/ui/button"
import type { PaginationMeta } from "./types"

export function PaginationControls({
  meta,
  onPageChange,
}: {
  meta: PaginationMeta
  onPageChange: (page: number) => void
}) {
  if (meta.total <= meta.per_page && meta.page <= 1) return null

  const start = meta.total === 0 ? 0 : (meta.page - 1) * meta.per_page + 1
  const end = Math.min(meta.page * meta.per_page, meta.total)

  return (
    <div className="flex items-center justify-between gap-3 pt-3 text-xs text-muted-foreground">
      <span>
        Showing {start}-{end} of {meta.total}
      </span>
      <div className="flex items-center gap-2">
        <Button
          variant="outline"
          size="sm"
          disabled={meta.page <= 1}
          onClick={() => onPageChange(Math.max(1, meta.page - 1))}
        >
          Prev
        </Button>
        <span>
          Page {meta.page} / {meta.last_page}
        </span>
        <Button
          variant="outline"
          size="sm"
          disabled={meta.page >= meta.last_page}
          onClick={() => onPageChange(Math.min(meta.last_page, meta.page + 1))}
        >
          Next
        </Button>
      </div>
    </div>
  )
}
