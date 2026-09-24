package service

import (
	"api-students-db/app/model"
	"api-students-db/helper"
)

// CanAccessStudent adalah fungsi otorisasi murni untuk akses data student.
//
// Aturan:
//  1. Jika current.ID == ownerID → true (owner selalu boleh akses data sendiri)
//  2. Jika bukan owner, cek permission anyPermission → true jika punya
//  3. Selain itu → false (fail closed)
//
// Fungsi ini:
//   - Tidak import Fiber
//   - Tidak import repository
//   - Mudah di-unit-test
func CanAccessStudent(
	current model.AuthUser,
	ownerID int,
	perms *helper.PermissionSet,
	anyPermission string,
) bool {
	// 1. Owner selalu boleh mengakses data miliknya sendiri
	if current.ID == ownerID {
		return true
	}

	// 2. Non-owner harus memiliki permission "any"
	if perms.Can(current.Role, anyPermission) {
		return true
	}

	// 3. Deny by default
	return false
}
