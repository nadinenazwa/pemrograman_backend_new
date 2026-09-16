package helper

import (
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func setupJWTEnv(t *testing.T) {
	t.Helper()
	os.Setenv("JWT_SECRET", "test-secret-key-that-is-at-least-32-characters-long")
	os.Setenv("JWT_ISSUER", "praktikum-backend")
	os.Setenv("JWT_ACCESS_TTL_MINUTES", "15")
}

func cleanupJWTEnv(t *testing.T) {
	t.Helper()
	os.Unsetenv("JWT_SECRET")
	os.Unsetenv("JWT_ISSUER")
	os.Unsetenv("JWT_ACCESS_TTL_MINUTES")
}

func TestNewJWTManager_SecretKosong(t *testing.T) {
	os.Unsetenv("JWT_SECRET")
	_, err := NewJWTManager()
	if err == nil {
		t.Error("seharusnya error jika JWT_SECRET kosong")
	}
}

func TestNewJWTManager_SecretTerlaluPendek(t *testing.T) {
	os.Setenv("JWT_SECRET", "short")
	defer os.Unsetenv("JWT_SECRET")
	_, err := NewJWTManager()
	if err == nil {
		t.Error("seharusnya error jika JWT_SECRET kurang dari 32 karakter")
	}
}

func TestJWT_GenerateDanParse(t *testing.T) {
	setupJWTEnv(t)
	defer cleanupJWTEnv(t)

	mgr, err := NewJWTManager()
	if err != nil {
		t.Fatalf("gagal membuat JWTManager: %v", err)
	}

	tokenStr, err := mgr.GenerateAccessToken(1, "nadine", "user")
	if err != nil {
		t.Fatalf("gagal generate token: %v", err)
	}

	claims, err := mgr.ParseAccessToken(tokenStr)
	if err != nil {
		t.Fatalf("gagal parse token: %v", err)
	}

	if claims["username"] != "nadine" {
		t.Errorf("username seharusnya 'nadine', dapat '%v'", claims["username"])
	}
	if claims["role"] != "user" {
		t.Errorf("role seharusnya 'user', dapat '%v'", claims["role"])
	}
	if claims["sub"] != "1" {
		t.Errorf("sub seharusnya '1', dapat '%v'", claims["sub"])
	}
	if claims["iss"] != "praktikum-backend" {
		t.Errorf("issuer seharusnya 'praktikum-backend', dapat '%v'", claims["iss"])
	}
}

func TestJWT_TokenExpired(t *testing.T) {
	setupJWTEnv(t)
	defer cleanupJWTEnv(t)

	// Buat token yang sudah expired
	secret := []byte(os.Getenv("JWT_SECRET"))
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      "1",
		"username": "nadine",
		"role":     "user",
		"iss":      "praktikum-backend",
		"iat":      now.Add(-2 * time.Hour).Unix(),
		"exp":      now.Add(-1 * time.Hour).Unix(), // expired 1 jam lalu
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(secret)

	mgr, _ := NewJWTManager()
	_, err := mgr.ParseAccessToken(tokenStr)
	if err != ErrTokenExpired {
		t.Errorf("seharusnya ErrTokenExpired, dapat: %v", err)
	}
}

func TestJWT_TokenSignatureInvalid(t *testing.T) {
	setupJWTEnv(t)
	defer cleanupJWTEnv(t)

	mgr, _ := NewJWTManager()
	tokenStr, _ := mgr.GenerateAccessToken(1, "nadine", "user")

	// Ubah satu karakter pada token untuk membuat signature invalid
	modified := tokenStr[:len(tokenStr)-1] + "X"

	_, err := mgr.ParseAccessToken(modified)
	if err == nil {
		t.Error("seharusnya error untuk token dengan signature yang dimodifikasi")
	}
}

func TestJWT_AlgNone_Ditolak(t *testing.T) {
	setupJWTEnv(t)
	defer cleanupJWTEnv(t)

	// Buat token dengan alg: none
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      "1",
		"username": "nadine",
		"role":     "admin",
		"iss":      "praktikum-backend",
		"iat":      now.Unix(),
		"exp":      now.Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, _ := token.SignedString(jwt.UnsafeAllowNoneSignatureType)

	mgr, _ := NewJWTManager()
	_, err := mgr.ParseAccessToken(tokenStr)
	if err == nil {
		t.Error("token dengan alg:none seharusnya ditolak")
	}
}

func TestJWT_AlgSalah_Ditolak(t *testing.T) {
	setupJWTEnv(t)
	defer cleanupJWTEnv(t)

	// Buat token string palsu dengan header yang salah
	// Manipulasi header JWT untuk menggunakan algoritma HS384
	mgr, _ := NewJWTManager()
	tokenStr, _ := mgr.GenerateAccessToken(1, "nadine", "user")

	// Ubah header dari HS256 ke HS384
	parts := strings.SplitN(tokenStr, ".", 3)
	if len(parts) != 3 {
		t.Fatal("token format tidak valid")
	}

	// Buat token baru dengan HS384
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      "1",
		"username": "nadine",
		"role":     "user",
		"iss":      "praktikum-backend",
		"iat":      now.Unix(),
		"exp":      now.Add(1 * time.Hour).Unix(),
	}
	wrongToken := jwt.NewWithClaims(jwt.SigningMethodHS384, claims)
	wrongSecret := []byte(os.Getenv("JWT_SECRET"))
	wrongTokenStr, _ := wrongToken.SignedString(wrongSecret)

	_, err := mgr.ParseAccessToken(wrongTokenStr)
	if err == nil {
		t.Error("token dengan algoritma HS384 seharusnya ditolak")
	}
}

func TestJWT_IssuerSalah_Ditolak(t *testing.T) {
	setupJWTEnv(t)
	defer cleanupJWTEnv(t)

	secret := []byte(os.Getenv("JWT_SECRET"))
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      "1",
		"username": "nadine",
		"role":     "user",
		"iss":      "issuer-yang-salah",
		"iat":      now.Unix(),
		"exp":      now.Add(1 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, _ := token.SignedString(secret)

	mgr, _ := NewJWTManager()
	_, err := mgr.ParseAccessToken(tokenStr)
	if err == nil {
		t.Error("token dengan issuer yang salah seharusnya ditolak")
	}
}

func TestHashToken(t *testing.T) {
	token := "test-refresh-token-123"
	hash1 := HashToken(token)
	hash2 := HashToken(token)

	if hash1 != hash2 {
		t.Error("hash dari token yang sama seharusnya konsisten")
	}

	if hash1 == token {
		t.Error("hash tidak boleh sama dengan token asli")
	}
}

func TestGenerateRefreshToken(t *testing.T) {
	token1, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("gagal generate refresh token: %v", err)
	}

	token2, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("gagal generate refresh token: %v", err)
	}

	if token1 == token2 {
		t.Error("dua refresh token seharusnya berbeda")
	}

	if len(token1) != 64 { // hex encoded 32 bytes = 64 chars
		t.Errorf("panjang refresh token seharusnya 64 karakter, dapat %d", len(token1))
	}
}
