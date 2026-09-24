package helper

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"sync"
	"unicode"

	"github.com/go-playground/validator/v10"
)

var (
	validate     *validator.Validate
	validateOnce sync.Once
)

// GetValidator mengembalikan singleton validator instance.
// Dibuat SEKALI untuk seluruh aplikasi, bukan per-request.
func GetValidator() *validator.Validate {
	validateOnce.Do(func() {
		validate = validator.New()

		// Gunakan nama field JSON untuk pesan error
		validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
			name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
			if name == "-" || name == "" {
				return fld.Name
			}
			return name
		})

		// Custom validation: username (huruf, angka, underscore, min 3)
		_ = validate.RegisterValidation("username", validateUsername)

		// Custom validation: strongpassword (min 8, ada huruf besar, kecil, angka)
		_ = validate.RegisterValidation("strongpassword", validateStrongPassword)

		// Custom validation: nospace (tidak boleh ada spasi)
		_ = validate.RegisterValidation("nospace", validateNoSpace)

		// Custom validation domain-specific students: validnim (format NIM harus angka, 7-20 digit)
		_ = validate.RegisterValidation("validnim", validateNIM)
	})
	return validate
}

// validateUsername: minimal 3 karakter, hanya alphanumeric dan underscore
func validateUsername(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if len(val) < 3 {
		return false
	}
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_]+$`, val)
	return matched
}

// validateStrongPassword: min 8 karakter, harus ada huruf besar, kecil, dan angka
func validateStrongPassword(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if len(val) < 8 {
		return false
	}
	var hasUpper, hasLower, hasDigit bool
	for _, ch := range val {
		switch {
		case unicode.IsUpper(ch):
			hasUpper = true
		case unicode.IsLower(ch):
			hasLower = true
		case unicode.IsDigit(ch):
			hasDigit = true
		}
	}
	return hasUpper && hasLower && hasDigit
}

// validateNoSpace: field tidak boleh mengandung spasi
func validateNoSpace(fl validator.FieldLevel) bool {
	return !strings.Contains(fl.Field().String(), " ")
}

// validateNIM: NIM harus terdiri dari digit saja, panjang 7-20 karakter.
// Ini adalah custom validation domain-specific untuk entity students.
func validateNIM(fl validator.FieldLevel) bool {
	val := fl.Field().String()
	if len(val) < 7 || len(val) > 20 {
		return false
	}
	for _, ch := range val {
		if !unicode.IsDigit(ch) {
			return false
		}
	}
	return true
}

// messageFor menerjemahkan tag validation menjadi pesan bermakna dalam Bahasa Indonesia.
func messageFor(fe validator.FieldError) string {
	field := fe.Field()
	tag := fe.Tag()
	param := fe.Param()

	switch tag {
	case "required":
		return fmt.Sprintf("%s wajib diisi", field)
	case "min":
		return fmt.Sprintf("%s minimal %s karakter", field, param)
	case "max":
		return fmt.Sprintf("%s maksimal %s karakter", field, param)
	case "email":
		return fmt.Sprintf("Format %s tidak valid", field)
	case "alphanum":
		return fmt.Sprintf("%s hanya boleh mengandung huruf dan angka", field)
	case "gte":
		return fmt.Sprintf("%s harus >= %s", field, param)
	case "lte":
		return fmt.Sprintf("%s harus <= %s", field, param)
	case "username":
		return fmt.Sprintf("%s minimal 3 karakter, hanya huruf, angka, dan underscore", field)
	case "strongpassword":
		return fmt.Sprintf("%s harus minimal 8 karakter, mengandung huruf besar, huruf kecil, dan angka", field)
	case "nospace":
		return fmt.Sprintf("%s tidak boleh mengandung spasi", field)
	case "validnim":
		return fmt.Sprintf("%s harus berupa 7-20 digit angka (format NIM)", field)
	case "required_without_all":
		return fmt.Sprintf("Minimal satu field harus dikirim")
	case "omitempty":
		return fmt.Sprintf("%s tidak valid", field)
	default:
		return fmt.Sprintf("%s tidak valid (%s)", field, tag)
	}
}

// ValidateStruct memvalidasi struct dan mengembalikan map field→pesan error.
// Mengembalikan nil jika validasi berhasil.
func ValidateStruct(s interface{}) map[string]string {
	v := GetValidator()
	err := v.Struct(s)
	if err == nil {
		return nil
	}

	errs := make(map[string]string)
	for _, fe := range err.(validator.ValidationErrors) {
		errs[fe.Field()] = messageFor(fe)
	}
	return errs
}
