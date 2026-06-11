# AGENTS.md - WireGuard Panel

## Scope

This repository contains a Go/Gin/GORM backend in `backend/` and a React/Vite/TypeScript frontend in `frontend/`.

All AI agents working in this repository must follow the backend and frontend AGENTS.md files before editing code.

## Project conventions

- Prefer implementation-first changes, then verify with available commands.
- Do not print or commit private keys, preshared keys, tokens, or passwords.
- WireGuard CRUD behavior matters: normal peer create/update/delete must be incremental and must not reset unrelated peers.
- Full WireGuard reconcile is acceptable only for interface create/update/manual sync/recovery.
- Soft delete/Trash behavior:
  - normal delete moves records to Trash;
  - restore reactivates and reapplies to kernel;
  - purge permanently deletes;
  - purge must be explicit and confirmed in frontend.
- List/table endpoints must use standard pagination (`page`, `per_page`), sorting (`sort_by`, `sort_order`), and optional `search`, then return `meta` with `page`, `per_page`, `total`, `last_page`, `sort_by`, `sort_order`, and `search`.
- Frontend routes must keep the main dashboard and Trash management separate: `/` for active interfaces/peers, `/trash` for restore/permanent delete flows.

## Verification

Backend when Go is available:

```bash
cd backend
go test ./...
go build ./...
```

Frontend:

```bash
cd frontend
npm run lint
npm run build
```
