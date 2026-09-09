package service

import "api-students-db/app/model"

// 1. Validasi untuk POST (Semua wajib diisi)
func ValidateCreate(req model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}
	if len(req.NIM) == 0 {
		errs["nim"] = "NIM wajib diisi"
	}
	if len(req.Name) == 0 {
		errs["name"] = "Nama wajib diisi"
	}
	if req.Grade == nil {
		errs["grade"] = "Grade wajib diisi"
	} else if *req.Grade < 0 || *req.Grade > 100 {
		errs["grade"] = "Grade harus bernilai antara 0 sampai 100"
	}
	if req.IsActive == nil {
		errs["is_active"] = "IsActive wajib diisi"
	}
	return errs
}

// 2. Validasi untuk PUT (Seluruh isi diganti, wajib ada)
func ValidateReplace(req model.CreateStudentRequest) map[string]string {
	errs := map[string]string{}
	if len(req.NIM) == 0 {
		errs["nim"] = "NIM wajib diisi"
	}
	if len(req.Name) == 0 {
		errs["name"] = "Nama wajib diisi"
	}
	if req.Grade == nil {
		errs["grade"] = "Grade wajib diisi"
	}
	if req.IsActive == nil {
		errs["is_active"] = "IsActive wajib diisi"
	}
	return errs
}

// ValidatePatch mengecek rentang grade SEBELUM data lama diambil dari
func ValidatePatch(req model.UpdateStudentRequest) map[string]string {
	errs := map[string]string{}
	if req.Grade != nil && (*req.Grade < 0 || *req.Grade > 100) {
		errs["grade"] = "Grade harus 0-100"
	}
	return errs
}

// 3. Penerapan perubahan untuk PATCH (Mengubah sebagian)
func ApplyPatch(current model.Student, req model.UpdateStudentRequest) model.Student {
	if req.NIM != nil {
		current.NIM = *req.NIM
	}
	if req.Name != nil {
		current.Name = *req.Name
	}
	if req.Grade != nil {
		current.Grade = *req.Grade
	}
	if req.IsActive != nil {
		current.IsActive = *req.IsActive
	}
	return current
}

func CountTotalPages(total, limit int) int {
	if limit <= 0 {
		return 0
	}
	return (total + limit - 1) / limit
}