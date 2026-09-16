package helper

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrTokenInvalid = errors.New("token tidak valid")
	ErrTokenExpired = errors.New("token sudah kedaluwarsa")
)

// JWTManager mengelola pembuatan dan validasi JWT access token.
type JWTManager struct {
	secret    []byte
	issuer    string
	accessTTL time.Duration
}

// NewJWTManager membuat JWTManager baru.
// Akan mengembalikan error jika JWT_SECRET kosong atau kurang dari 32 karakter.
func NewJWTManager() (*JWTManager, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET tidak boleh kosong")
	}
	if len(secret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET harus minimal 32 karakter (sekarang: %d)", len(secret))
	}

	issuer := os.Getenv("JWT_ISSUER")
	if issuer == "" {
		issuer = "praktikum-backend"
	}

	ttlMinutes := 15
	if v := os.Getenv("JWT_ACCESS_TTL_MINUTES"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			ttlMinutes = parsed
		}
	}

	return &JWTManager{
		secret:    []byte(secret),
		issuer:    issuer,
		accessTTL: time.Duration(ttlMinutes) * time.Minute,
	}, nil
}

// GenerateAccessToken membuat JWT access token dengan claim yang sesuai modul.
func (m *JWTManager) GenerateAccessToken(userID int, username, role string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      strconv.Itoa(userID),
		"username": username,
		"role":     role,
		"iss":      m.issuer,
		"iat":      now.Unix(),
		"exp":      now.Add(m.accessTTL).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(m.secret)
}

// ParseAccessToken mem-parse dan memvalidasi JWT access token.
// Memeriksa algoritma secara eksplisit (hanya HS256), issuer, dan expiration.
func (m *JWTManager) ParseAccessToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		// Perlindungan algorithm confusion: hanya terima HS256
		if token.Method.Alg() != jwt.SigningMethodHS256.Alg() {
			return nil, fmt.Errorf("algoritma signing tidak diizinkan: %s", token.Method.Alg())
		}
		return m.secret, nil
	},
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(m.issuer),
		jwt.WithExpirationRequired(),
	)

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, ErrTokenInvalid
	}

	if !token.Valid {
		return nil, ErrTokenInvalid
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, ErrTokenInvalid
	}

	return claims, nil
}

// GenerateRefreshToken menghasilkan token random yang aman secara kriptografis.
func GenerateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("gagal generate refresh token: %w", err)
	}
	return hex.EncodeToString(bytes), nil
}

// HashToken meng-hash token menggunakan SHA-256 untuk disimpan di database.
func HashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// RefreshTTL mengembalikan durasi refresh token dari environment variable.
func RefreshTTL() time.Duration {
	days := 7
	if v := os.Getenv("JWT_REFRESH_TTL_DAYS"); v != "" {
		if parsed, err := strconv.Atoi(v); err == nil && parsed > 0 {
			days = parsed
		}
	}
	return time.Duration(days) * 24 * time.Hour
}
