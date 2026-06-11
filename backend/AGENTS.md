# AGENTS.md - Backend

## Stack and structure

- Language/framework: Go, Gin HTTP router, GORM models, wgctrl/netlink WireGuard operations.
- Keep request DTOs in `dto/*_dto.go` with `json` and `validate` tags.
- Keep response envelope types/helpers in `dto/common.go`.
- Keep HTTP handlers in `handlers/` and WireGuard kernel logic in `wg/`.
- Do not move WireGuard netlink logic into handlers.

## Standard JSON response format

All JSON API endpoints must use the standard response helpers from `dto/common.go`.
Do not hand-roll `c.JSON(...)` response envelopes in handlers unless returning non-JSON content such as `.conf` text or PNG QR code.

Success single/list response:

```json
{
  "success": true,
  "message": "data fetched successfully",
  "data": {}
}
```

Success pagination response:

```json
{
  "success": true,
  "message": "data fetched successfully",
  "data": [],
  "meta": {
    "page": 1,
    "per_page": 10,
    "total": 42,
    "last_page": 5
  }
}
```

List endpoints must be paginated with query params `page`, `per_page`, `sort_by`, `sort_order`, and optional `search`.
Default `per_page` is 10 and the backend caps it at 100. `sort_order` must be `asc` or `desc`; `sort_by` must be checked against a per-endpoint whitelist to avoid SQL injection. Current paginated endpoints include active interfaces, interface peers/status, trashed interfaces, and trashed peers.

Generic error response:

```json
{
  "success": false,
  "message": "failed to fetch data",
  "errors": [
    { "message": "failed to fetch data" }
  ]
}
```

Validation error response must be Laravel-style so frontend forms can map errors directly by field name:

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

Use helpers:

```go
dto.OK(c, "interfaces fetched successfully", ifaces)
dto.Created(c, "peer created", peer)
dto.NoData(c, http.StatusOK, "peer moved to trash")
dto.Error(c, http.StatusInternalServerError, "failed to fetch peers")
dto.ValidationError(c, errs)
```

## Validation rules

- Use `middleware.Validate(req)` after `ShouldBindJSON`.
- Validator output must use JSON tag field names, e.g. `listen_port`, not Go struct names like `ListenPort`.
- Validation errors must be `map[string][]string` compatible with React Hook Form `setError`.
- Add validation tags to DTO fields, not to GORM model fields.

## WireGuard behavior

- Peer create/update/delete must use incremental sync (`ReplacePeers: false`) so unrelated peers do not disconnect.
- Full sync/reconcile may use `ReplacePeers: true` only for interface create/update/manual sync/recovery.
- Delete to Trash must remove the peer/device from the kernel but keep DB records recoverable.
- Restore must reapply the restored peer/interface to the kernel.

## Verification

Run when Go toolchain is available:

```bash
go test ./...
go build ./...
```
