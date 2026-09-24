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
// Menggunakan declarative validation tags.
type RegisterRequest struct {
	Username string `json:"username" validate:"required,username,max=30,nospace"`
	Email    string `json:"email" validate:"required,email,max=120"`
	Password string `json:"password" validate:"required,strongpassword,max=72"`
}

// LoginRequest adalah request body untuk login.
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// RefreshRequest adalah request body untuk refresh token.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// LogoutRequest adalah request body untuk logout.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
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
