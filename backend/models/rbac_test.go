package models

import "testing"

func TestEffectivePermissionsUnionsRolesAndDirect(t *testing.T) {
	u := User{
		Roles: []Role{
			{Name: "operator", Permissions: []Permission{{Name: "interfaces.view"}, {Name: "interfaces.create"}}},
		},
		Permissions: []Permission{{Name: "openvpn.apply"}, {Name: "interfaces.view"}},
	}
	eff := u.EffectivePermissions()
	for _, want := range []string{"interfaces.view", "interfaces.create", "openvpn.apply"} {
		if !eff[want] {
			t.Fatalf("expected effective permission %q, got %v", want, eff)
		}
	}
	if len(eff) != 3 {
		t.Fatalf("expected 3 unique permissions (dedup interfaces.view), got %d: %v", len(eff), eff)
	}
	if !u.HasPermission("openvpn.apply") {
		t.Fatal("expected direct permission to satisfy HasPermission")
	}
	if u.HasPermission("users.delete") {
		t.Fatal("did not expect an ungranted permission")
	}
	if u.IsSuperAdmin() {
		t.Fatal("operator user must not be a super admin")
	}
}

func TestWildcardGrantsEverything(t *testing.T) {
	u := User{Roles: []Role{{Name: "super-admin", Permissions: []Permission{{Name: PermissionWildcard}}}}}
	if !u.IsSuperAdmin() {
		t.Fatal("wildcard role must make IsSuperAdmin true")
	}
	if !u.HasPermission("anything.at.all") {
		t.Fatal("wildcard must satisfy any permission check")
	}
}

func TestWildcardAsDirectPermission(t *testing.T) {
	u := User{Permissions: []Permission{{Name: PermissionWildcard}}}
	if !u.IsSuperAdmin() || !u.HasPermission("users.delete") {
		t.Fatal("wildcard granted directly must also grant everything")
	}
}
