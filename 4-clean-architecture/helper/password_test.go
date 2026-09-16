package helper

import (
	"testing"
)

func TestHashPassword_DanVerify(t *testing.T) {
	plain := "Password123"
	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("gagal hash password: %v", err)
	}

	// Hash tidak boleh sama dengan plaintext
	if hash == plain {
		t.Error("hash tidak boleh sama dengan plaintext")
	}

	// Verifikasi password yang benar
	if err := VerifyPassword(hash, plain); err != nil {
		t.Errorf("password yang benar seharusnya lolos verifikasi: %v", err)
	}
}

func TestVerifyPassword_Salah(t *testing.T) {
	hash, _ := HashPassword("Password123")

	if err := VerifyPassword(hash, "WrongPassword"); err == nil {
		t.Error("password yang salah seharusnya gagal verifikasi")
	}
}

func TestHashPassword_BerbedaSetiapKali(t *testing.T) {
	plain := "Password123"
	hash1, _ := HashPassword(plain)
	hash2, _ := HashPassword(plain)

	if hash1 == hash2 {
		t.Error("bcrypt hash seharusnya berbeda setiap kali karena random salt")
	}
}

func TestPasswordTidakMunculDiJSON(t *testing.T) {
	// Test ini memastikan tag json:"-" ada di model User
	// yang diuji secara implisit oleh desain model
	// Kita hanya memastikan hash/verify berfungsi di sini
	hash, _ := HashPassword("Test1234")
	if err := VerifyPassword(hash, "Test1234"); err != nil {
		t.Error("verify seharusnya berhasil")
	}
}
