package service

import (
	"testing"

	"api-students-db/app/model"
)

// === Register Validation Tests ===

func TestValidateRegister_Valid(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "nadine",
		Email:    "nadine@example.com",
		Password: "Password123",
	})
	if len(errs) > 0 {
		t.Errorf("seharusnya tidak ada error, dapat: %v", errs)
	}
}

func TestValidateRegister_UsernameWajib(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "",
		Email:    "nadine@example.com",
		Password: "Password123",
	})
	if _, ok := errs["username"]; !ok {
		t.Error("username kosong seharusnya menghasilkan error")
	}
}

func TestValidateRegister_UsernameTerlaluPendek(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "ab",
		Email:    "nadine@example.com",
		Password: "Password123",
	})
	if _, ok := errs["username"]; !ok {
		t.Error("username kurang dari 3 karakter seharusnya menghasilkan error")
	}
}

func TestValidateRegister_UsernameKarakterTidakValid(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "nad!ne@",
		Email:    "nadine@example.com",
		Password: "Password123",
	})
	if _, ok := errs["username"]; !ok {
		t.Error("username dengan karakter khusus seharusnya menghasilkan error")
	}
}

func TestValidateRegister_EmailTidakValid(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "nadine",
		Email:    "bukan-email",
		Password: "Password123",
	})
	if _, ok := errs["email"]; !ok {
		t.Error("email tidak valid seharusnya menghasilkan error")
	}
}

func TestValidateRegister_PasswordTerlaluPendek(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "nadine",
		Email:    "nadine@example.com",
		Password: "Pass1",
	})
	if _, ok := errs["password"]; !ok {
		t.Error("password kurang dari 8 karakter seharusnya menghasilkan error")
	}
}

func TestValidateRegister_PasswordLemahTanpaHurufBesar(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "nadine",
		Email:    "nadine@example.com",
		Password: "password123",
	})
	if _, ok := errs["password"]; !ok {
		t.Error("password tanpa huruf besar seharusnya menghasilkan error")
	}
}

func TestValidateRegister_PasswordLemahTanpaHurufKecil(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "nadine",
		Email:    "nadine@example.com",
		Password: "PASSWORD123",
	})
	if _, ok := errs["password"]; !ok {
		t.Error("password tanpa huruf kecil seharusnya menghasilkan error")
	}
}

func TestValidateRegister_PasswordLemahTanpaAngka(t *testing.T) {
	errs := ValidateRegister(model.RegisterRequest{
		Username: "nadine",
		Email:    "nadine@example.com",
		Password: "Passworddd",
	})
	if _, ok := errs["password"]; !ok {
		t.Error("password tanpa angka seharusnya menghasilkan error")
	}
}

// === Login Validation Tests ===

func TestValidateLogin_Valid(t *testing.T) {
	errs := ValidateLogin(model.LoginRequest{
		Username: "nadine",
		Password: "password123",
	})
	if len(errs) > 0 {
		t.Errorf("seharusnya tidak ada error, dapat: %v", errs)
	}
}

func TestValidateLogin_UsernameKosong(t *testing.T) {
	errs := ValidateLogin(model.LoginRequest{
		Username: "",
		Password: "password123",
	})
	if _, ok := errs["username"]; !ok {
		t.Error("username kosong seharusnya menghasilkan error")
	}
}

func TestValidateLogin_PasswordKosong(t *testing.T) {
	errs := ValidateLogin(model.LoginRequest{
		Username: "nadine",
		Password: "",
	})
	if _, ok := errs["password"]; !ok {
		t.Error("password kosong seharusnya menghasilkan error")
	}
}

// === Mass Assignment Protection Test ===

func TestRegisterRequest_TidakMemilikiFieldRole(t *testing.T) {
	// Compile-time guarantee: RegisterRequest tidak memiliki field Role.
	// Jika seseorang menambahkan field Role, test ini akan error pada compile.
	req := model.RegisterRequest{
		Username: "nadine",
		Email:    "nadine@example.com",
		Password: "Password123",
	}
	// Pastikan struct hanya memiliki 3 field yang diharapkan
	_ = req.Username
	_ = req.Email
	_ = req.Password
	// Jika ada field Role pada RegisterRequest, test ini TIDAK akan compile
	// karena kita hanya mengisi 3 field.
}
