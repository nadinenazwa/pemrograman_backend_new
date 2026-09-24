package middleware

import (
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"api-students-db/app/model"
	"api-students-db/helper"
)

func Register(app *fiber.App, logger *slog.Logger, allowedOrigins string) {
	app.Use(requestid.New())
	app.Use(recover.New())
	app.Use(helmet.New())

	// CORS menggunakan ALLOWED_ORIGINS dari environment
	app.Use(cors.New(cors.Config{
		AllowOrigins: allowedOrigins,
		AllowMethods: "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders: "Origin,Content-Type,Accept,Authorization",
	}))

	app.Use(RequestLogger(logger))
}

// RequestLogger mencatat SETIAP request (apa pun hasilnya) sebagai satu
// log entry. Menggunakan status response AKTUAL yang diberikan ke client.
// 4xx → WARN (msg=request_rejected), 5xx → ERROR (msg=request_failed).
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		requestID, _ := c.Locals("requestid").(string)

		// Gunakan status response AKTUAL yang diberikan ke client.
		// Karena ErrorHandler dijalankan setelah middleware, kita infer status dari error.
		status := c.Response().StatusCode()
		if err != nil {
			if appErr, ok := err.(*helper.AppError); ok {
				status = appErr.Status
			} else if fiberErr, ok := err.(*fiber.Error); ok {
				status = fiberErr.Code
			} else {
				status = fiber.StatusInternalServerError
			}
		}

		// Field dasar yang selalu di-log
		fields := []any{
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", status),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		}

		// Jika user sudah terautentikasi, tambahkan user_id dan role
		// TIDAK log password, access token, refresh token, atau JWT_SECRET
		if authUser, ok := c.Locals("auth_user").(model.AuthUser); ok {
			fields = append(fields,
				slog.Int("user_id", authUser.ID),
				slog.String("role", authUser.Role),
			)
		}

		// Log level berdasarkan status code aktual
		switch {
		case status >= 500:
			logger.Error("request_failed", fields...)
		case status >= 400:
			logger.Warn("request_rejected", fields...)
		default:
			logger.Info("http_request", fields...)
		}

		return err
	}
}

var methodsWithBody = map[string]bool{
	fiber.MethodPost: true, fiber.MethodPut: true, fiber.MethodPatch: true,
}

// RequireJSON menggantikan checkContentTypeJSON yang dulu dipanggil manual
// di CreateStudent, ReplaceStudent, UpdateStudentPartial.
func RequireJSON(c *fiber.Ctx) error {
	if methodsWithBody[c.Method()] {
		contentType := c.Get("Content-Type")
		if !strings.HasPrefix(contentType, fiber.MIMEApplicationJSON) {
			return helper.ErrUnsupportedMedia("Content-Type harus application/json")
		}
	}
	return c.Next()
}

// RequireAuth adalah middleware untuk memvalidasi JWT access token.
// Menyimpan AuthUser ke Fiber Locals jika token valid.
func RequireAuth(jwtManager *helper.JWTManager) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")

		// Header tidak ada atau format salah
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)
			return helper.ErrUnauthorized("Token autentikasi diperlukan")
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse dan validasi token
		claims, err := jwtManager.ParseAccessToken(tokenStr)
		if err != nil {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)
			if err == helper.ErrTokenExpired {
				return helper.ErrUnauthorized("Token sudah kedaluwarsa, silakan refresh")
			}
			return helper.ErrUnauthorized("Token tidak valid")
		}

		// Ambil data user dari claims
		sub, _ := claims["sub"].(string)
		userID, _ := strconv.Atoi(sub)
		username, _ := claims["username"].(string)
		role, _ := claims["role"].(string)

		// Simpan AuthUser ke Locals
		c.Locals("auth_user", model.AuthUser{
			ID:       userID,
			Username: username,
			Role:     role,
		})

		return c.Next()
	}
}

// LoginRateLimiter membuat rate limiter khusus untuk endpoint login.
// Mencegah brute force dengan membatasi percobaan login.
func LoginRateLimiter() fiber.Handler {
	return limiter.New(limiter.Config{
		Max:        5,
		Expiration: 15 * time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			retryAfter := 900 // 15 menit dalam detik
			c.Set("Retry-After", strconv.Itoa(retryAfter))
			return helper.ErrTooManyRequests("Terlalu banyak percobaan login, coba lagi nanti")
		},
		SkipSuccessfulRequests: true,
	})
}


func RequirePermission(perms *helper.PermissionSet, permission string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		authUser, ok := c.Locals("auth_user").(model.AuthUser)
		if !ok {
			return helper.ErrUnauthorized("Token autentikasi diperlukan")
		}

		if !perms.Can(authUser.Role, permission) {
			return helper.ErrForbidden("Anda tidak memiliki izin untuk aksi ini")
		}

		return c.Next()
	}
}