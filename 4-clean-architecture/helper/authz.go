package helper

import "sort"

// PermissionSet menyimpan mapping role -> permission di memory.
// Dibangun sekali saat startup dari database, tidak di-query per request.
//
// Prinsip FAIL CLOSED:
//   - Role tidak dikenal → deny
//   - Permission tidak dikenal → deny
//   - Mapping tidak ditemukan → deny
type PermissionSet struct {
	roles map[string]map[string]bool // role → {permission → true}
}

// NewPermissionSet membuat PermissionSet dari map role ke daftar permission.
// Contoh input:
//
//	map[string][]string{
//	    "admin": {"student:list", "student:create"},
//	    "staff": {"student:list"},
//	    "user":  {},
//	}
func NewPermissionSet(rolePerms map[string][]string) *PermissionSet {
	ps := &PermissionSet{
		roles: make(map[string]map[string]bool, len(rolePerms)),
	}
	for role, perms := range rolePerms {
		m := make(map[string]bool, len(perms))
		for _, p := range perms {
			m[p] = true
		}
		ps.roles[role] = m
	}
	return ps
}

// Can mengecek apakah role memiliki permission tertentu.
// Mengembalikan false jika role atau permission tidak dikenal (fail closed).
func (ps *PermissionSet) Can(role, permission string) bool {
	if ps == nil {
		return false
	}
	perms, ok := ps.roles[role]
	if !ok {
		return false
	}
	return perms[permission]
}

// KnownRoles mengembalikan daftar role yang diketahui, terurut.
func (ps *PermissionSet) KnownRoles() []string {
	if ps == nil {
		return nil
	}
	roles := make([]string, 0, len(ps.roles))
	for r := range ps.roles {
		roles = append(roles, r)
	}
	sort.Strings(roles)
	return roles
}

// PermissionsFor mengembalikan daftar permission untuk role tertentu, terurut.
// Mengembalikan nil jika role tidak dikenal.
func (ps *PermissionSet) PermissionsFor(role string) []string {
	if ps == nil {
		return nil
	}
	perms, ok := ps.roles[role]
	if !ok {
		return nil
	}
	result := make([]string, 0, len(perms))
	for p := range perms {
		result = append(result, p)
	}
	sort.Strings(result)
	return result
}
