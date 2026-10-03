package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"

	"siakad/internal/utils"
)

func Protected() fiber.Handler {
	return func(c *fiber.Ctx) error {
		authHeader := c.Get("Authorization")
		if authHeader == "" {
			return utils.SendError(c, fiber.StatusUnauthorized, "Token tidak ada/salah/kedaluwarsa", nil)
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			return utils.SendError(c, fiber.StatusUnauthorized, "Token tidak ada/salah/kedaluwarsa", nil)
		}

		tokenStr := parts[1]
		claims, err := utils.ValidateToken(tokenStr)
		if err != nil {
			return utils.SendError(c, fiber.StatusUnauthorized, "Token tidak ada/salah/kedaluwarsa", nil)
		}

		// Store user info in locals
		c.Locals("user_id", claims.UserID)
		c.Locals("role", claims.Role)

		return c.Next()
	}
}

func Role(allowedRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		role, ok := c.Locals("role").(string)
		if !ok {
			return utils.SendError(c, fiber.StatusUnauthorized, "Unauthorized", nil)
		}

		for _, allowedRole := range allowedRoles {
			if role == allowedRole {
				return c.Next()
			}
		}

		return utils.SendError(c, fiber.StatusForbidden, "Forbidden", nil)
	}
}

func AdminOnly() fiber.Handler {
	return Role("admin")
}

func MahasiswaOnly() fiber.Handler {
	return Role("mahasiswa")
}
