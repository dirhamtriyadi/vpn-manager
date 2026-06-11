# WireGuard Panel — Frontend (Vite + React + shadcn + RHF + Zod + Axios)

Web UI for the WireGuard management API (`backend/`). Create interfaces, add
peers, watch live handshake/transfer status, and hand clients a QR / `.conf` —
all without the CLI.

## Stack
- **Vite** + **React** + **TypeScript**
- **shadcn/ui** (Button, Input, Label, Card, Table, Dialog, Badge, Textarea)
- **Tailwind CSS** with CSS-variable theming
- **react-hook-form** + **zod** for forms & validation
- **axios** for HTTP

## Structure
```
frontend/src/
├── App.tsx
├── lib/api.ts                 # axios instance + API_BASE_URL + error helper
├── components/ui/             # shadcn components
├── schemas/
│   ├── interface.ts           # zod schema for interface form
│   └── peer.ts                # zod schema for peer form
└── features/wireguard/
    ├── types.ts
    ├── api.ts                 # interface + peer API calls, config/QR URLs
    ├── format.ts              # bytes + handshake humanizers
    ├── InterfaceForm.tsx
    ├── PeerForm.tsx
    ├── PeerConfigDialog.tsx   # QR + config text + copy/download
    └── Dashboard.tsx          # main screen (selector, status, peer table)
```

## Quick start
```bash
cp .env.example .env           # point VITE_API_BASE_URL at the backend
npm install
npm run dev                    # http://localhost:5173
```

## What you can do from the UI
- **New interface** — names it, sets port/subnet/endpoint/DNS; server keypair auto-generated.
- **Add peer** — name only is enough; keys, preshared key and tunnel IP are auto-assigned.
- **Config / QR** — open a peer to scan the QR in the WireGuard app or download its `.conf`.
- **Status** — peer table polls every 5s for online state, last handshake and transfer.
- **Apply** — pushes the current state to the kernel device.
- **Enable/disable & delete** — per peer, with the kernel reconciled automatically.

## Build
```bash
npm run build && npm run preview
```
