package model

import "time"

// User merepresentasikan entitas pengguna di database.
type User struct {
	ID        int       `json:"id"`
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Tidak pernah muncul di JSON response
	Role      string    `json:"role"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

// RegisterRequest adalah request body untuk registrasi.
// Tidak memiliki field Role untuk mencegah mass assignment.
type RegisterRequest struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest adalah request body untuk login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// RefreshRequest adalah request body untuk refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// LogoutRequest adalah request body untuk logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token"`
}

// TokenPair berisi access token dan refresh token.
type TokenPair struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

// AuthUser berisi informasi user yang terautentikasi, disimpan di Fiber Locals.
type AuthUser struct {
	ID       int    `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}
