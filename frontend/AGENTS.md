# AGENTS.md - Frontend

## Stack and structure

- Framework: React + TypeScript + Vite.
- Forms use React Hook Form + Zod schemas from `src/schemas/`.
- API functions live in `src/features/wireguard/api.ts`.
- Shared Axios instance and error helpers live in `src/lib/api.ts`.
- WireGuard UI components live in `src/features/wireguard/`.

## API response contract

All JSON API calls should expect the backend standard envelope:

```ts
interface ApiResponse<T> {
  success: boolean
  message: string
  data: T
  meta?: PaginationMeta
}
```

Paginated responses include `meta`:

```ts
interface PaginationMeta {
  page: number
  per_page: number
  total: number
  last_page: number
}
```

Frontend list displays must request paginated backend data with `page`, `per_page`, `sort_by`, `sort_order`, and optional `search`; keep the returned `meta`; and render search, per-page, sort, ASC/DESC, and Prev/Next controls when appropriate. Do not silently fetch every row for tables/lists.

Use React Router for distinct pages. The dashboard (`/`) should stay focused on active interfaces/peers. Trash management must live on its own route (`/trash`) with explicit navigation buttons/links; do not stack Trash tables below the main dashboard.

Generic errors look like:

```json
{
  "success": false,
  "message": "failed to fetch data",
  "errors": [{ "message": "failed to fetch data" }]
}
```

Validation errors are Laravel-style:

```json
{
  "success": false,
  "message": "validation failed",
  "errors": {
    "name": ["The name field is required."],
    "listen_port": ["The listen port must be at least 1."]
  }
}
```

## Form error handling

For React Hook Form forms, always apply backend validation errors to fields with:

```ts
import { applyServerValidationErrors } from "@/lib/api"
```

Pattern:

```ts
async function handleValidSubmit(values: FormValues) {
  try {
    await onSubmit(values)
  } catch (err) {
    applyServerValidationErrors<FormValues>(setError, err)
  }
}
```

Parent submit handlers that catch API errors must rethrow the error after setting banners/toasts, otherwise the child form cannot map server validation errors to fields.

## UI behavior

- Delete means move to Trash, not permanent delete.
- Permanent delete must use explicit purge action and browser confirmation.
- Restore/purge actions should reload active lists and Trash lists.
- Peer add/update/delete should not cause unrelated peers to disconnect; if this appears in UI behavior, inspect backend incremental peer sync.

## Verification

Run before finishing frontend changes:

```bash
npm run lint
npm run build
```
