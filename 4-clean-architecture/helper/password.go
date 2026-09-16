package helper

import (
	"golang.org/x/crypto/bcrypt"
)

const bcryptCost = 12

// HashPassword meng-hash password plaintext menggunakan bcrypt dengan cost 12.
func HashPassword(plain string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// VerifyPassword memverifikasi password plaintext terhadap hash bcrypt.
func VerifyPassword(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}
