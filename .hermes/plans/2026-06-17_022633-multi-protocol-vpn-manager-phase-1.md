# Multi-Protocol VPN Manager Phase 1 Implementation Plan

> **For Hermes:** Use subagent-driven-development skill to implement this plan task-by-task.

**Goal:** Refactor the current WireGuard Panel into a VPN Manager foundation that can later support WireGuard, OpenVPN, L2TP/IPsec, SSTP, and optional legacy PPTP without breaking current WireGuard functionality.

**Architecture:** Keep the existing WireGuard implementation working, but introduce generic VPN domain types, protocol enums, and a driver abstraction. WireGuard remains the first concrete driver; future protocols plug into the same backend/frontend shape instead of being forced into WireGuard-only `interface/peer` concepts.

**Tech Stack:** Go, Gin, GORM, React, TypeScript, existing WireGuard `wgctrl` service, current authenticated API pattern.

---

## Current Context / Assumptions

- Repo: `/workspace/Projects/wireguard`
- Backend currently models:
  - `models.WGInterface` in `backend/models/interface.go`
  - `models.Peer` in `backend/models/peer.go`
  - route groups `/api/v1/interfaces` and `/api/v1/peers` in `backend/routes/routes.go`
- Frontend currently models:
  - `WGInterface` and `Peer` in `frontend/src/features/wireguard/types.ts`
  - WireGuard UI in `frontend/src/features/wireguard/*`
- Existing WireGuard behavior must remain stable while foundation is introduced.
- Phase 1 should not add OpenVPN/L2TP/SSTP/PPTP runtime yet. It only prepares the architecture.
- PPTP should be marked legacy/insecure when it is eventually added.
- Keep protected config/QR downloads through axios; do not expose private-key-bearing config endpoints publicly.

---

## Proposed Approach

Implement this as a compatibility-first refactor:

1. Add generic protocol/domain types without deleting old WireGuard models.
2. Add protocol driver interface and registry.
3. Wrap existing WireGuard behavior as a `wireguard` driver facade.
4. Add read-only/generic VPN instance API aliases first, while keeping `/interfaces` and `/peers` routes working.
5. Update frontend naming/types gradually to present “VPN Manager” while still using WireGuard-specific flows internally.
6. Add tests around protocol enum, registry, and backward-compatible WireGuard mapping.

This avoids a risky big-bang migration.

---

## Phase 1 Tasks

### Task 1: Add protocol enum and generic constants

**Objective:** Introduce a canonical protocol list used by backend and eventually frontend.

**Files:**
- Create: `backend/models/protocol.go`
- Test: `backend/models/protocol_test.go`

**Implementation sketch:**

```go
package models

import "strings"

type VPNProtocol string

const (
    ProtocolWireGuard VPNProtocol = "wireguard"
    ProtocolOpenVPN   VPNProtocol = "openvpn"
    ProtocolL2TPIPsec VPNProtocol = "l2tp_ipsec"
    ProtocolSSTP      VPNProtocol = "sstp"
    ProtocolPPTP      VPNProtocol = "pptp"
)

func (p VPNProtocol) String() string {
    return string(p)
}

func ParseVPNProtocol(value string) (VPNProtocol, bool) {
    switch VPNProtocol(strings.ToLower(strings.TrimSpace(value))) {
    case ProtocolWireGuard:
        return ProtocolWireGuard, true
    case ProtocolOpenVPN:
        return ProtocolOpenVPN, true
    case ProtocolL2TPIPsec:
        return ProtocolL2TPIPsec, true
    case ProtocolSSTP:
        return ProtocolSSTP, true
    case ProtocolPPTP:
        return ProtocolPPTP, true
    default:
        return "", false
    }
}

func (p VPNProtocol) IsLegacyInsecure() bool {
    return p == ProtocolPPTP
}
```

**Tests:**

- Parse valid protocol names.
- Reject unknown protocol names.
- Verify `pptp` is legacy/insecure.

**Verify:**

```bash
cd /workspace/Projects/wireguard/backend
go test ./models
```

---

### Task 2: Add generic VPN instance/user DTOs without DB migration

**Objective:** Define API-facing generic shapes that can map from existing WireGuard records.

**Files:**
- Create: `backend/dto/vpn_dto.go`
- Test: `backend/dto/vpn_dto_test.go` if mapping helpers are placed in DTO package, otherwise test in mapper package.

**Implementation sketch:**

```go
package dto

import "github.com/example/wg-panel/models"

type VPNInstanceResponse struct {
    ID              uint               `json:"id"`
    Protocol        models.VPNProtocol `json:"protocol"`
    Name            string             `json:"name"`
    ListenPort      int                `json:"listen_port"`
    Address         string             `json:"address"`
    Endpoint        string             `json:"endpoint"`
    Enabled         bool               `json:"enabled"`
    Status          string             `json:"status,omitempty"`
    LegacyInsecure  bool               `json:"legacy_insecure"`
    WireGuard       *WireGuardMetadata  `json:"wireguard,omitempty"`
}

type WireGuardMetadata struct {
    PublicKey       string `json:"public_key"`
    DNS             string `json:"dns"`
    MTU             int    `json:"mtu"`
    Masquerade      bool   `json:"masquerade"`
    EgressInterface string `json:"egress_interface"`
}

type VPNUserResponse struct {
    ID         uint               `json:"id"`
    InstanceID uint               `json:"instance_id"`
    Protocol   models.VPNProtocol `json:"protocol"`
    Name       string             `json:"name"`
    AssignedIP string             `json:"assigned_ip"`
    Enabled    bool               `json:"enabled"`
    Online     bool               `json:"online,omitempty"`
    RxBytes    int64              `json:"rx_bytes,omitempty"`
    TxBytes    int64              `json:"tx_bytes,omitempty"`
    WireGuard  *WireGuardUserMeta  `json:"wireguard,omitempty"`
}

type WireGuardUserMeta struct {
    PublicKey           string `json:"public_key"`
    AllowedIPs          string `json:"allowed_ips"`
    ClientAllowedIPs    string `json:"client_allowed_ips"`
    PersistentKeepalive int    `json:"persistent_keepalive"`
}
```

**Notes:**

- This task should not rename DB tables yet.
- It creates a generic API contract for future `/vpn/instances` endpoints.

---

### Task 3: Add protocol driver interface and registry

**Objective:** Create the extension point for WireGuard/OpenVPN/L2TP/SSTP/PPTP drivers.

**Files:**
- Create: `backend/vpn/driver.go`
- Create: `backend/vpn/registry.go`
- Test: `backend/vpn/registry_test.go`

**Implementation sketch:**

```go
package vpn

import "github.com/example/wg-panel/models"

type InstanceStatus struct {
    Protocol models.VPNProtocol `json:"protocol"`
    Up       bool               `json:"up"`
    Message  string             `json:"message,omitempty"`
}

type Driver interface {
    Protocol() models.VPNProtocol
    Status(instanceID uint) (InstanceStatus, error)
    Sync(instanceID uint) error
    GenerateUserConfig(userID uint) ([]byte, string, error)
}
```

Registry:

```go
package vpn

import (
    "fmt"

    "github.com/example/wg-panel/models"
)

type Registry struct {
    drivers map[models.VPNProtocol]Driver
}

func NewRegistry() *Registry {
    return &Registry{drivers: map[models.VPNProtocol]Driver{}}
}

func (r *Registry) Register(driver Driver) error {
    if driver == nil {
        return fmt.Errorf("driver is nil")
    }
    protocol := driver.Protocol()
    if _, exists := r.drivers[protocol]; exists {
        return fmt.Errorf("driver already registered for protocol %s", protocol)
    }
    r.drivers[protocol] = driver
    return nil
}

func (r *Registry) Get(protocol models.VPNProtocol) (Driver, bool) {
    driver, ok := r.drivers[protocol]
    return driver, ok
}
```

**Tests:**

- Register and retrieve driver.
- Reject duplicate protocol driver.
- Reject nil driver.

---

### Task 4: Add WireGuard driver facade

**Objective:** Wrap current WireGuard service/handlers behind the generic driver interface without changing behavior.

**Files:**
- Create: `backend/vpn/wireguard_driver.go`
- Modify only minimally if needed: `backend/wg/service.go`

**Implementation sketch:**

```go
package vpn

import (
    "github.com/example/wg-panel/models"
)

type WireGuardDriver struct{}

func NewWireGuardDriver() *WireGuardDriver {
    return &WireGuardDriver{}
}

func (d *WireGuardDriver) Protocol() models.VPNProtocol {
    return models.ProtocolWireGuard
}

func (d *WireGuardDriver) Status(instanceID uint) (InstanceStatus, error) {
    // Phase 1: call existing interface/kernel status logic if easily reusable.
    // If current status logic is handler-only, return a conservative placeholder
    // and add TODO to extract status service in Phase 2.
    return InstanceStatus{Protocol: models.ProtocolWireGuard, Up: true}, nil
}

func (d *WireGuardDriver) Sync(instanceID uint) error {
    // Phase 1: delegate to existing WireGuard sync service if available.
    return nil
}

func (d *WireGuardDriver) GenerateUserConfig(userID uint) ([]byte, string, error) {
    // Phase 1: delegate to existing peer config generation when extracted.
    // If current logic is handler-only, extraction belongs in a subtask.
    return nil, "", nil
}
```

**Important:**

- Do not duplicate private-key generation or config rendering logic.
- If code exists only in handlers, extract reusable service functions in small steps.
- Keep current `/peers/:peerId/config` and `/peers/:peerId/qrcode` working.

---

### Task 5: Add generic mapping helpers from WireGuard models

**Objective:** Convert current `WGInterface` and `Peer` into generic VPN DTOs.

**Files:**
- Create: `backend/vpn/mappers.go`
- Test: `backend/vpn/mappers_test.go`

**Implementation sketch:**

```go
package vpn

import (
    "github.com/example/wg-panel/dto"
    "github.com/example/wg-panel/models"
)

func MapWGInterfaceToVPNInstance(iface models.WGInterface) dto.VPNInstanceResponse {
    return dto.VPNInstanceResponse{
        ID:             iface.ID,
        Protocol:       models.ProtocolWireGuard,
        Name:           iface.Name,
        ListenPort:     iface.ListenPort,
        Address:        iface.Address,
        Endpoint:       iface.Endpoint,
        Enabled:        iface.Enabled,
        LegacyInsecure: false,
        WireGuard: &dto.WireGuardMetadata{
            PublicKey:       iface.PublicKey,
            DNS:             iface.DNS,
            MTU:             iface.MTU,
            Masquerade:      iface.Masquerade,
            EgressInterface: iface.EgressInterface,
        },
    }
}

func MapPeerToVPNUser(peer models.Peer) dto.VPNUserResponse {
    return dto.VPNUserResponse{
        ID:         peer.ID,
        InstanceID: peer.InterfaceID,
        Protocol:   models.ProtocolWireGuard,
        Name:       peer.Name,
        AssignedIP: peer.AssignedIP,
        Enabled:    peer.Enabled,
        Online:     peer.Online,
        RxBytes:    peer.RxBytes,
        TxBytes:    peer.TxBytes,
        WireGuard: &dto.WireGuardUserMeta{
            PublicKey:           peer.PublicKey,
            AllowedIPs:          peer.AllowedIPs,
            ClientAllowedIPs:    peer.ClientAllowedIPs,
            PersistentKeepalive: peer.PersistentKeepalive,
        },
    }
}
```

---

### Task 6: Add generic VPN instance read endpoints as aliases

**Objective:** Expose generic endpoints without removing existing WireGuard endpoints.

**Files:**
- Create: `backend/handlers/vpn_handler.go`
- Modify: `backend/routes/routes.go`

**New endpoints:**

```text
GET /api/v1/vpn/protocols
GET /api/v1/vpn/instances
GET /api/v1/vpn/instances/:id
GET /api/v1/vpn/instances/:id/users
GET /api/v1/vpn/instances/:id/status
```

**Behavior in Phase 1:**

- `GET /vpn/protocols` returns all known protocols and marks PPTP as `legacy_insecure`.
- `GET /vpn/instances` returns current WireGuard interfaces mapped to generic `VPNInstanceResponse`.
- `GET /vpn/instances/:id/users` returns current peers mapped to generic `VPNUserResponse`.
- Existing `/interfaces` and `/peers` endpoints remain unchanged.

**Example protocol response:**

```json
[
  {"id":"wireguard","label":"WireGuard","available":true,"legacy_insecure":false},
  {"id":"openvpn","label":"OpenVPN","available":false,"legacy_insecure":false},
  {"id":"l2tp_ipsec","label":"L2TP/IPsec","available":false,"legacy_insecure":false},
  {"id":"sstp","label":"SSTP","available":false,"legacy_insecure":false},
  {"id":"pptp","label":"PPTP","available":false,"legacy_insecure":true}
]
```

**Tests:**

- Handler tests if existing test infrastructure supports Gin.
- At minimum, mapper/enum tests plus manual curl verification.

---

### Task 7: Add frontend protocol types and API calls

**Objective:** Introduce frontend generic VPN types and API helpers while leaving the WireGuard UI working.

**Files:**
- Create: `frontend/src/features/vpn/types.ts`
- Create: `frontend/src/features/vpn/api.ts`
- Optional Create: `frontend/src/features/vpn/protocols.ts`

**Implementation sketch:**

```ts
export type VPNProtocol = "wireguard" | "openvpn" | "l2tp_ipsec" | "sstp" | "pptp"

export interface VPNProtocolInfo {
  id: VPNProtocol
  label: string
  available: boolean
  legacy_insecure: boolean
}

export interface VPNInstance {
  id: number
  protocol: VPNProtocol
  name: string
  listen_port: number
  address: string
  endpoint: string
  enabled: boolean
  status?: string
  legacy_insecure: boolean
  wireguard?: {
    public_key: string
    dns: string
    mtu: number
    masquerade: boolean
    egress_interface: string
  }
}

export interface VPNUser {
  id: number
  instance_id: number
  protocol: VPNProtocol
  name: string
  assigned_ip: string
  enabled: boolean
  online?: boolean
  rx_bytes?: number
  tx_bytes?: number
  wireguard?: {
    public_key: string
    allowed_ips: string
    client_allowed_ips: string
    persistent_keepalive: number
  }
}
```

API helpers:

```ts
export async function listVPNProtocols(): Promise<VPNProtocolInfo[]> { ... }
export async function listVPNInstances(params?: ListParams): Promise<PaginatedResult<VPNInstance>> { ... }
export async function listVPNInstanceUsers(id: number, params?: ListParams): Promise<PaginatedResult<VPNUser>> { ... }
```

---

### Task 8: Add UI protocol selector shell

**Objective:** Start presenting the product as VPN Manager without changing the working WireGuard management screens yet.

**Files:**
- Create: `frontend/src/features/vpn/ProtocolSelector.tsx`
- Modify: `frontend/src/App.tsx`
- Optional Modify: `frontend/src/features/wireguard/Dashboard.tsx`

**Behavior:**

- Add a “New VPN” / “Add VPN instance” entry point.
- Show protocol cards:
  - WireGuard: Available
  - OpenVPN: Coming soon
  - L2TP/IPsec: Coming soon
  - SSTP: Coming soon
  - PPTP: Legacy/insecure, Coming soon
- Clicking WireGuard routes to the existing WireGuard interface creation flow.
- Other protocols are disabled or show “coming soon”.

**Do not:**

- Add incomplete forms for OpenVPN/L2TP/SSTP/PPTP yet.
- Break current dashboard create/edit/delete flows.

---

### Task 9: Rename visible branding conservatively

**Objective:** Change user-facing labels from “WireGuard Panel” to “VPN Manager” where safe, but keep protocol-specific wording inside WireGuard screens.

**Files likely to inspect/modify:**
- `frontend/src/App.tsx`
- `frontend/src/features/wireguard/Dashboard.tsx`
- `frontend/src/features/auth/LoginPage.tsx`
- `README.md` if present

**Rules:**

- Product/global label: “VPN Manager”
- WireGuard-specific pages/dialogs: still say “WireGuard” where accurate
- RouterOS script UI remains under WireGuard config because that script currently generates WireGuard client configuration.

---

### Task 10: Verification and cleanup

**Objective:** Verify Phase 1 did not regress current WireGuard behavior.

**Backend verification:**

```bash
cd /workspace/Projects/wireguard/backend
gofmt -w models/protocol.go dto/vpn_dto.go vpn/*.go handlers/vpn_handler.go routes/routes.go
go test ./...
go build ./...
```

**Frontend verification:**

```bash
cd /workspace/Projects/wireguard/frontend
npm run lint
npm run build
```

**Repo verification:**

```bash
cd /workspace/Projects/wireguard
git diff --check
git status --short
```

If `frontend/tsconfig.tsbuildinfo` changes only due to build, restore it before committing.

**Manual API checks:**

```bash
curl -H "Authorization: Bearer <token>" http://localhost:<port>/api/v1/vpn/protocols
curl -H "Authorization: Bearer <token>" http://localhost:<port>/api/v1/vpn/instances
```

Do not paste real tokens into logs or commits.

---

## Files Likely to Change

Backend:

```text
backend/models/protocol.go
backend/models/protocol_test.go
backend/dto/vpn_dto.go
backend/vpn/driver.go
backend/vpn/registry.go
backend/vpn/registry_test.go
backend/vpn/wireguard_driver.go
backend/vpn/mappers.go
backend/vpn/mappers_test.go
backend/handlers/vpn_handler.go
backend/routes/routes.go
```

Frontend:

```text
frontend/src/features/vpn/types.ts
frontend/src/features/vpn/api.ts
frontend/src/features/vpn/protocols.ts
frontend/src/features/vpn/ProtocolSelector.tsx
frontend/src/App.tsx
frontend/src/features/wireguard/Dashboard.tsx
frontend/src/features/auth/LoginPage.tsx
```

Documentation:

```text
README.md
```

---

## Commit Plan

Use small semantic commits:

```bash
git commit -m "feat(vpn): add protocol enum and driver registry"
git commit -m "feat(vpn): expose generic vpn instance aliases"
git commit -m "feat(frontend): add vpn protocol selector shell"
git commit -m "docs: outline multi-protocol vpn manager roadmap"
```

---

## Risks and Tradeoffs

1. **DB migration risk**
   - Avoided in Phase 1 by mapping existing WireGuard tables to generic DTOs.

2. **Naming churn**
   - Avoid deep renames of `WGInterface`/`Peer` in Phase 1. Do that later after generic endpoints are stable.

3. **Driver abstraction too broad**
   - Keep driver interface minimal now: protocol, status, sync, config generation.
   - Add create/update/delete later when non-WireGuard runtimes are actually implemented.

4. **Protocol runtime complexity**
   - OpenVPN, L2TP/IPsec, SSTP, and PPTP require different runtime managers.
   - Do not add fake “support” until the runtime story is designed.

5. **Security**
   - PPTP must remain marked legacy/insecure.
   - Configs containing private keys/certs/passwords must stay behind authenticated endpoints.

---

## Open Questions Before Phase 2

1. Runtime strategy:
   - Host-native system services, or protocol-specific Docker containers?
   - For CasaOS/Docker, protocol containers are probably better.

2. OpenVPN auth mode:
   - Certificate-only?
   - Username/password?
   - Both?

3. L2TP/IPsec implementation:
   - strongSwan + xl2tpd?
   - Or SoftEther as a multi-protocol backend?

4. SSTP implementation:
   - SoftEther?
   - accel-ppp?

5. Migration strategy:
   - Keep old WireGuard tables forever as protocol-specific tables?
   - Or migrate to generic `vpn_instances` and `vpn_users` in Phase 2/3?

---

## Recommended Next Step

Implement Phase 1 first. It gives a safe multi-protocol foundation while preserving all current WireGuard features. After Phase 1 passes tests/builds, plan Phase 2 around runtime container strategy and OpenVPN as the first new protocol.
