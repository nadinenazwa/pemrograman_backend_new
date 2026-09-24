package service

import (
	"testing"

	"api-students-db/app/model"
	"api-students-db/helper"
)

// buildAuthzPerms membuat PermissionSet yang konsisten dengan mapping RBAC Modul 6.
func buildAuthzPerms() *helper.PermissionSet {
	return helper.NewPermissionSet(map[string][]string{
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
		"user": {},
	})
}

// === Owner Tests (menggunakan ID eksplisit) ===

func TestCanAccessStudent_OwnerUserCanRead(t *testing.T) {
	perms := buildAuthzPerms()
	// User ID 5 adalah owner student dengan OwnerID 5
	user := model.AuthUser{ID: 5, Username: "pemilik", Role: "user"}
	if !CanAccessStudent(user, 5, perms, "student:read:any") {
		t.Error("owner (user role, ID=5) seharusnya boleh mengakses data sendiri (ownerID=5)")
	}
}

func TestCanAccessStudent_OwnerUserCanUpdate(t *testing.T) {
	perms := buildAuthzPerms()
	// User ID 5 adalah owner → boleh update data sendiri meskipun role user
	user := model.AuthUser{ID: 5, Username: "pemilik", Role: "user"}
	if !CanAccessStudent(user, 5, perms, "student:update:any") {
		t.Error("owner (user role, ID=5) seharusnya boleh mengupdate data sendiri (ownerID=5)")
	}
}

func TestCanAccessStudent_OwnerStaffCanRead(t *testing.T) {
	perms := buildAuthzPerms()
	// Staff ID 3 juga owner → boleh
	staff := model.AuthUser{ID: 3, Username: "staf", Role: "staff"}
	if !CanAccessStudent(staff, 3, perms, "student:read:any") {
		t.Error("owner (staff role, ID=3) seharusnya boleh mengakses data sendiri (ownerID=3)")
	}
}

// === Non-Owner Tests (menggunakan ID eksplisit) ===

func TestCanAccessStudent_NonOwnerUserDeniedRead(t *testing.T) {
	perms := buildAuthzPerms()
	// User ID 7 bukan owner student yang ownerID=5, dan role user tidak punya permission
	user := model.AuthUser{ID: 7, Username: "bukan_pemilik", Role: "user"}
	if CanAccessStudent(user, 5, perms, "student:read:any") {
		t.Error("non-owner (user role, ID=7) seharusnya TIDAK boleh mengakses data orang lain (ownerID=5)")
	}
}

func TestCanAccessStudent_NonOwnerUserDeniedUpdate(t *testing.T) {
	perms := buildAuthzPerms()
	user := model.AuthUser{ID: 7, Username: "bukan_pemilik", Role: "user"}
	if CanAccessStudent(user, 5, perms, "student:update:any") {
		t.Error("non-owner (user role, ID=7) seharusnya TIDAK boleh mengupdate data orang lain (ownerID=5)")
	}
}

func TestCanAccessStudent_NonOwnerAdminCanRead(t *testing.T) {
	perms := buildAuthzPerms()
	// Admin ID 7 bukan owner student ownerID=5, tapi punya student:read:any
	admin := model.AuthUser{ID: 7, Username: "admin1", Role: "admin"}
	if !CanAccessStudent(admin, 5, perms, "student:read:any") {
		t.Error("non-owner admin (ID=7) dengan student:read:any seharusnya boleh mengakses data orang lain (ownerID=5)")
	}
}

func TestCanAccessStudent_NonOwnerAdminCanUpdate(t *testing.T) {
	perms := buildAuthzPerms()
	admin := model.AuthUser{ID: 7, Username: "admin1", Role: "admin"}
	if !CanAccessStudent(admin, 5, perms, "student:update:any") {
		t.Error("non-owner admin (ID=7) dengan student:update:any seharusnya boleh mengupdate data orang lain (ownerID=5)")
	}
}

func TestCanAccessStudent_NonOwnerStaffCanRead(t *testing.T) {
	perms := buildAuthzPerms()
	// Staff ID 7 bukan owner, punya student:read:any
	staff := model.AuthUser{ID: 7, Username: "staf1", Role: "staff"}
	if !CanAccessStudent(staff, 5, perms, "student:read:any") {
		t.Error("non-owner staff (ID=7) dengan student:read:any seharusnya boleh mengakses data orang lain (ownerID=5)")
	}
}

func TestCanAccessStudent_NonOwnerStaffDeniedUpdate(t *testing.T) {
	perms := buildAuthzPerms()
	// Staff ID 7 bukan owner, TIDAK punya student:update:any
	staff := model.AuthUser{ID: 7, Username: "staf1", Role: "staff"}
	if CanAccessStudent(staff, 5, perms, "student:update:any") {
		t.Error("non-owner staff (ID=7) TANPA student:update:any seharusnya TIDAK boleh mengupdate data orang lain (ownerID=5)")
	}
}

// === Edge Cases ===

func TestCanAccessStudent_UnknownRoleDenied(t *testing.T) {
	perms := buildAuthzPerms()
	hacker := model.AuthUser{ID: 99, Username: "hacker", Role: "superadmin"}
	if CanAccessStudent(hacker, 5, perms, "student:read:any") {
		t.Error("role tidak dikenal seharusnya ditolak (fail closed)")
	}
}

func TestCanAccessStudent_NilPermissionSetOwnerAllowed(t *testing.T) {
	// Bahkan tanpa PermissionSet, owner tetap boleh akses data sendiri
	var perms *helper.PermissionSet
	user := model.AuthUser{ID: 5, Username: "pemilik", Role: "user"}
	if !CanAccessStudent(user, 5, perms, "student:read:any") {
		t.Error("owner seharusnya tetap boleh mengakses data sendiri meskipun PermissionSet nil")
	}
}

func TestCanAccessStudent_NilPermissionSetNonOwnerDenied(t *testing.T) {
	var perms *helper.PermissionSet
	user := model.AuthUser{ID: 7, Username: "bukan", Role: "admin"}
	if CanAccessStudent(user, 5, perms, "student:read:any") {
		t.Error("non-owner seharusnya ditolak jika PermissionSet nil")
	}
}
