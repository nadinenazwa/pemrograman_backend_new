package service

import (
	"regexp"
	"strings"
	"unicode"

	"api-students-db/app/model"
)

var (
	usernameRegex = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
	emailRegex    = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
)

// ValidateRegister memvalidasi request registrasi sesuai aturan modul.
func ValidateRegister(req model.RegisterRequest) map[string]string {
	errs := map[string]string{}

	// Username validasi
	username := strings.TrimSpace(req.Username)
	if username == "" {
		errs["username"] = "Username wajib diisi"
	} else if len(username) < 3 {
		errs["username"] = "Username minimal 3 karakter"
	} else if !usernameRegex.MatchString(username) {
		errs["username"] = "Username hanya boleh mengandung huruf, angka, dan underscore"
	}

	// Email validasi
	email := strings.TrimSpace(req.Email)
	if email == "" {
		errs["email"] = "Email wajib diisi"
	} else if !emailRegex.MatchString(email) {
		errs["email"] = "Format email tidak valid"
	}

	// Password validasi
	if req.Password == "" {
		errs["password"] = "Password wajib diisi"
	} else if len(req.Password) < 8 {
		errs["password"] = "Password minimal 8 karakter"
	} else {
		if msg := validatePasswordStrength(req.Password); msg != "" {
			errs["password"] = msg
		}
	}

	return errs
}

// validatePasswordStrength memeriksa kekuatan password sesuai modul:
// minimal ada huruf besar, huruf kecil, dan angka.
func validatePasswordStrength(password string) string {
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range password {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}

	missing := []string{}
	if !hasUpper {
		missing = append(missing, "huruf besar")
	}
	if !hasLower {
		missing = append(missing, "huruf kecil")
	}
	if !hasDigit {
		missing = append(missing, "angka")
	}

	if len(missing) > 0 {
		return "Password harus mengandung " + strings.Join(missing, ", ")
	}
	return ""
}

// ValidateLogin memvalidasi request login: hanya memastikan kelengkapan field.
func ValidateLogin(req model.LoginRequest) map[string]string {
	errs := map[string]string{}
	if strings.TrimSpace(req.Username) == "" {
		errs["username"] = "Username wajib diisi"
	}
	if req.Password == "" {
		errs["password"] = "Password wajib diisi"
	}
	return errs
}
