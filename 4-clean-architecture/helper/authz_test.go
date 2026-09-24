package helper

import (
	"testing"
)

// buildTestPerms membuat PermissionSet yang konsisten dengan mapping RBAC Modul 6.
func buildTestPerms() *PermissionSet {
	return NewPermissionSet(map[string][]string{
		"admin": {
			"student:list",
			"student:read:any",
			"student:create",
			"student:update:any",
			"student:delete",
		},
		"staff": {
			"student:list",
			"student:read:any",
			"student:create",
		},
		"user": {}, // tidak ada permission student
	})
}

func TestCan_AdminHasStudentList(t *testing.T) {
	ps := buildTestPerms()
	if !ps.Can("admin", "student:list") {
		t.Error("admin seharusnya memiliki permission student:list")
	}
}

func TestCan_AdminHasAllPermissions(t *testing.T) {
	ps := buildTestPerms()
	perms := []string{
		"student:list", "student:read:any", "student:create",
		"student:update:any", "student:delete",
	}
	for _, p := range perms {
		if !ps.Can("admin", p) {
			t.Errorf("admin seharusnya memiliki permission %s", p)
		}
	}
}

func TestCan_StaffHasStudentCreate(t *testing.T) {
	ps := buildTestPerms()
	if !ps.Can("staff", "student:create") {
		t.Error("staff seharusnya memiliki permission student:create")
	}
}

func TestCan_StaffLacksStudentDelete(t *testing.T) {
	ps := buildTestPerms()
	if ps.Can("staff", "student:delete") {
		t.Error("staff seharusnya TIDAK memiliki permission student:delete")
	}
}

func TestCan_StaffLacksStudentUpdateAny(t *testing.T) {
	ps := buildTestPerms()
	if ps.Can("staff", "student:update:any") {
		t.Error("staff seharusnya TIDAK memiliki permission student:update:any")
	}
}

func TestCan_UserLacksStudentList(t *testing.T) {
	ps := buildTestPerms()
	if ps.Can("user", "student:list") {
		t.Error("user seharusnya TIDAK memiliki permission student:list")
	}
}

func TestCan_UserHasNoPermissions(t *testing.T) {
	ps := buildTestPerms()
	perms := []string{
		"student:list", "student:read:any", "student:create",
		"student:update:any", "student:delete",
	}
	for _, p := range perms {
		if ps.Can("user", p) {
			t.Errorf("user seharusnya TIDAK memiliki permission %s", p)
		}
	}
}

func TestCan_UnknownRoleDenied(t *testing.T) {
	ps := buildTestPerms()
	if ps.Can("hacker", "student:list") {
		t.Error("role tidak dikenal seharusnya ditolak (fail closed)")
	}
}

func TestCan_UnknownPermissionDenied(t *testing.T) {
	ps := buildTestPerms()
	if ps.Can("admin", "nuke:everything") {
		t.Error("permission tidak dikenal seharusnya ditolak (fail closed)")
	}
}

func TestCan_EmptyRoleDenied(t *testing.T) {
	ps := buildTestPerms()
	if ps.Can("", "student:list") {
		t.Error("role kosong seharusnya ditolak")
	}
}

func TestCan_EmptyPermissionDenied(t *testing.T) {
	ps := buildTestPerms()
	if ps.Can("admin", "") {
		t.Error("permission kosong seharusnya ditolak")
	}
}

func TestCan_NilPermissionSetDenied(t *testing.T) {
	var ps *PermissionSet
	if ps.Can("admin", "student:list") {
		t.Error("nil PermissionSet seharusnya menolak semua akses")
	}
}

func TestKnownRoles_ReturnsThreeRoles(t *testing.T) {
	ps := buildTestPerms()
	roles := ps.KnownRoles()
	if len(roles) != 3 {
		t.Fatalf("seharusnya 3 role, dapat %d", len(roles))
	}
	expected := []string{"admin", "staff", "user"}
	for i, r := range expected {
		if roles[i] != r {
			t.Errorf("role[%d] seharusnya %q, dapat %q", i, r, roles[i])
		}
	}
}

func TestPermissionsFor_AdminHasFive(t *testing.T) {
	ps := buildTestPerms()
	perms := ps.PermissionsFor("admin")
	if len(perms) != 5 {
		t.Errorf("admin seharusnya 5 permission, dapat %d", len(perms))
	}
}

func TestPermissionsFor_StaffHasThree(t *testing.T) {
	ps := buildTestPerms()
	perms := ps.PermissionsFor("staff")
	if len(perms) != 3 {
		t.Errorf("staff seharusnya 3 permission, dapat %d", len(perms))
	}
}

func TestPermissionsFor_UserHasZero(t *testing.T) {
	ps := buildTestPerms()
	perms := ps.PermissionsFor("user")
	if len(perms) != 0 {
		t.Errorf("user seharusnya 0 permission, dapat %d", len(perms))
	}
}

func TestPermissionsFor_UnknownRoleReturnsNil(t *testing.T) {
	ps := buildTestPerms()
	perms := ps.PermissionsFor("hacker")
	if perms != nil {
		t.Error("role tidak dikenal seharusnya mengembalikan nil")
	}
}

func TestKnownRoles_NilPermissionSet(t *testing.T) {
	var ps *PermissionSet
	roles := ps.KnownRoles()
	if roles != nil {
		t.Error("nil PermissionSet seharusnya mengembalikan nil")
	}
}
