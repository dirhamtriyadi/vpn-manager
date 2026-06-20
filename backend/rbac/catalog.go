// Package rbac defines the canonical permission catalog, the default roles, and
// the seeding/bootstrap logic for the role-based access control system.
package rbac

import "github.com/example/vpn-manager/models"

// Permission name constants. Names follow "<resource>.<action>". The wildcard
// models.PermissionWildcard ("*") grants everything.
const (
	PermUsersView   = "users.view"
	PermUsersCreate = "users.create"
	PermUsersUpdate = "users.update"
	PermUsersDelete = "users.delete"

	PermRolesView   = "roles.view"
	PermRolesManage = "roles.manage"

	PermPermissionsView = "permissions.view"

	PermInterfacesView   = "interfaces.view"
	PermInterfacesCreate = "interfaces.create"
	PermInterfacesUpdate = "interfaces.update"
	PermInterfacesDelete = "interfaces.delete"
	PermInterfacesSync   = "interfaces.sync"

	PermPeersView   = "peers.view"
	PermPeersCreate = "peers.create"
	PermPeersUpdate = "peers.update"
	PermPeersDelete = "peers.delete"

	PermPortForwardsView   = "portforwards.view"
	PermPortForwardsManage = "portforwards.manage"

	PermVPNView   = "vpn.view"
	PermVPNCreate = "vpn.create"
	PermVPNApply  = "vpn.apply"

	PermOpenVPNView   = "openvpn.view"
	PermOpenVPNCreate = "openvpn.create"
	PermOpenVPNApply  = "openvpn.apply"
)

type permissionDef struct {
	Name        string
	Description string
}

// AllPermissions is the canonical catalog. Seeding upserts exactly this set
// (plus the wildcard) so the catalog is the single source of truth.
func AllPermissions() []permissionDef {
	return []permissionDef{
		{models.PermissionWildcard, "Full access (super admin); bypasses ownership scoping"},

		{PermUsersView, "View panel users"},
		{PermUsersCreate, "Create panel users"},
		{PermUsersUpdate, "Update panel users, roles, and direct permissions"},
		{PermUsersDelete, "Delete panel users"},

		{PermRolesView, "View roles"},
		{PermRolesManage, "Create, update, delete roles and their permissions"},

		{PermPermissionsView, "View the permission catalog"},

		{PermInterfacesView, "View WireGuard interfaces"},
		{PermInterfacesCreate, "Create WireGuard interfaces"},
		{PermInterfacesUpdate, "Update WireGuard interfaces"},
		{PermInterfacesDelete, "Delete/trash/restore WireGuard interfaces"},
		{PermInterfacesSync, "Apply WireGuard interfaces to the kernel"},

		{PermPeersView, "View peers"},
		{PermPeersCreate, "Create peers"},
		{PermPeersUpdate, "Update peers"},
		{PermPeersDelete, "Delete/trash/restore peers"},

		{PermPortForwardsView, "View public-IP port forwards"},
		{PermPortForwardsManage, "Create, toggle, and delete public-IP port forwards"},

		{PermVPNView, "View VPN protocols, instances, plans, and status"},
		{PermVPNCreate, "Create VPN instance drafts (L2TP/IPsec, SSTP, PPTP)"},
		{PermVPNApply, "Apply VPN runtime for non-WireGuard protocols"},

		{PermOpenVPNView, "View OpenVPN instances, users, and manifests"},
		{PermOpenVPNCreate, "Create OpenVPN instances, users, and manifests"},
		{PermOpenVPNApply, "Apply OpenVPN runtime"},
	}
}

type roleDef struct {
	Name        string
	Description string
	Permissions []string // permission names; ["*"] means wildcard
}

// SuperAdminRole is the role granted to the bootstrap admin user.
const SuperAdminRole = "super-admin"

// DefaultRoles are seeded on startup. operator can manage every VPN resource but
// not users/roles; viewer is read-only.
func DefaultRoles() []roleDef {
	return []roleDef{
		{SuperAdminRole, "Full access to everything", []string{models.PermissionWildcard}},
		{"operator", "Manage VPNs (interfaces, peers, OpenVPN, other protocols) but not users/roles", []string{
			PermInterfacesView, PermInterfacesCreate, PermInterfacesUpdate, PermInterfacesDelete, PermInterfacesSync,
			PermPeersView, PermPeersCreate, PermPeersUpdate, PermPeersDelete,
			PermPortForwardsView, PermPortForwardsManage,
			PermVPNView, PermVPNCreate, PermVPNApply,
			PermOpenVPNView, PermOpenVPNCreate, PermOpenVPNApply,
		}},
		{"viewer", "Read-only access to VPNs", []string{
			PermInterfacesView, PermPeersView, PermPortForwardsView, PermVPNView, PermOpenVPNView,
		}},
	}
}
