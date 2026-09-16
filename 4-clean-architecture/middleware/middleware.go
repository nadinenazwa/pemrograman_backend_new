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
func RequestLogger(logger *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		requestID, _ := c.Locals("requestid").(string)
		logger.Info("http_request",
			slog.String("request_id", requestID),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
			slog.Int("status", c.Response().StatusCode()),
			slog.Duration("duration", time.Since(start)),
			slog.String("ip", c.IP()),
		)
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
			return helper.Fail(c, fiber.StatusUnsupportedMediaType, "Content-Type harus application/json")
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
			return helper.Fail(c, fiber.StatusUnauthorized, "Token autentikasi diperlukan")
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// Parse dan validasi token
		claims, err := jwtManager.ParseAccessToken(tokenStr)
		if err != nil {
			c.Set("WWW-Authenticate", `Bearer realm="api"`)
			if err == helper.ErrTokenExpired {
				return helper.Fail(c, fiber.StatusUnauthorized, "Token sudah kedaluwarsa, silakan refresh")
			}
			return helper.Fail(c, fiber.StatusUnauthorized, "Token tidak valid")
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
			return helper.Fail(c, fiber.StatusTooManyRequests, "Terlalu banyak percobaan login, coba lagi nanti")
		},
		SkipSuccessfulRequests: true,
	})
}