package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

	"api-students-db/app/model"
	"api-students-db/helper"
	"github.com/gofiber/fiber/v2"
)

func TestLoginRateLimiter_OnlyCountsFailed(t *testing.T) {
	app := fiber.New(fiber.Config{
		ProxyHeader: "X-Forwarded-For",
	})
	
	app.Post("/login", LoginRateLimiter(), func(c *fiber.Ctx) error {
		var req struct {
			Status string `json:"status"`
		}
		_ = c.BodyParser(&req)
		if req.Status == "success" {
			return helper.Success(c, fiber.StatusOK, "Login berhasil", nil)
		}
		return helper.Fail(c, fiber.StatusUnauthorized, "Login gagal")
	})

	// 1. 5 failed logins -> 401
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/login", strings.NewReader(`{"status":"fail"}`))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-Forwarded-For", "192.168.1.1")
		
		resp, _ := app.Test(req)
		if resp.StatusCode != fiber.StatusUnauthorized {
			t.Errorf("Expected 401 for failed login %d, got %d", i+1, resp.StatusCode)
		}
	}

	// 2. 6th failed login -> 429
	req6 := httptest.NewRequest("POST", "/login", strings.NewReader(`{"status":"fail"}`))
	req6.Header.Set("Content-Type", "application/json")
	req6.Header.Set("X-Forwarded-For", "192.168.1.1")
	resp6, _ := app.Test(req6)
	if resp6.StatusCode != fiber.StatusTooManyRequests {
		t.Errorf("Expected 429, got %d", resp6.StatusCode)
	}
	if resp6.Header.Get("Retry-After") == "" {
		t.Errorf("Expected Retry-After header")
	}

	// 3. 5 success logins on new IP
	for i := 0; i < 5; i++ {
		r := httptest.NewRequest("POST", "/login", strings.NewReader(`{"status":"success"}`))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("X-Forwarded-For", "10.0.0.1")
		res, _ := app.Test(r)
		if res.StatusCode != fiber.StatusOK {
			t.Fatalf("Expected 200 on success login %d, got %d", i+1, res.StatusCode)
		}
	}

	// 4. Next fail should be 401 (not 429) because success wasn't counted
	r6 := httptest.NewRequest("POST", "/login", strings.NewReader(`{"status":"fail"}`))
	r6.Header.Set("Content-Type", "application/json")
	r6.Header.Set("X-Forwarded-For", "10.0.0.1")
	res6, _ := app.Test(r6)
	if res6.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("Expected 401, got %d. Limiter triggered but shouldn't have!", res6.StatusCode)
	}
}

// === RequirePermission Tests ===

func buildTestPermsMiddleware() *helper.PermissionSet {
	return helper.NewPermissionSet(map[string][]string{
		"admin": {"student:list", "student:read:any", "student:create", "student:update:any", "student:delete"},
		"staff": {"student:list", "student:read:any", "student:create"},
		"user":  {},
	})
}

func TestRequirePermission_NoAuth_Returns401(t *testing.T) {
	perms := buildTestPermsMiddleware()
	app := fiber.New()

	// Tidak ada RequireAuth sebelumnya → auth_user tidak ada di Locals
	app.Get("/test", RequirePermission(perms, "student:list"), func(c *fiber.Ctx) error {
		return helper.Success(c, fiber.StatusOK, "OK", nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", resp.StatusCode)
	}
}

func TestRequirePermission_WithPermission_Returns200(t *testing.T) {
	perms := buildTestPermsMiddleware()
	app := fiber.New()

	// Simulasi RequireAuth: set auth_user di Locals
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_user", model.AuthUser{ID: 1, Username: "admin1", Role: "admin"})
		return c.Next()
	})
	app.Get("/test", RequirePermission(perms, "student:list"), func(c *fiber.Ctx) error {
		return helper.Success(c, fiber.StatusOK, "OK", nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("Expected 200, got %d", resp.StatusCode)
	}
}

func TestRequirePermission_WithoutPermission_Returns403(t *testing.T) {
	perms := buildTestPermsMiddleware()
	app := fiber.New()

	// User role tidak memiliki student:list
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_user", model.AuthUser{ID: 3, Username: "user1", Role: "user"})
		return c.Next()
	})
	app.Get("/test", RequirePermission(perms, "student:list"), func(c *fiber.Ctx) error {
		return helper.Success(c, fiber.StatusOK, "OK", nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403, got %d", resp.StatusCode)
	}
}

func TestRequirePermission_UnknownRole_Returns403(t *testing.T) {
	perms := buildTestPermsMiddleware()
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_user", model.AuthUser{ID: 99, Username: "hacker", Role: "superadmin"})
		return c.Next()
	})
	app.Get("/test", RequirePermission(perms, "student:list"), func(c *fiber.Ctx) error {
		return helper.Success(c, fiber.StatusOK, "OK", nil)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403 for unknown role, got %d", resp.StatusCode)
	}
}

func TestRequirePermission_StaffLacksDelete_Returns403(t *testing.T) {
	perms := buildTestPermsMiddleware()
	app := fiber.New()

	app.Use(func(c *fiber.Ctx) error {
		c.Locals("auth_user", model.AuthUser{ID: 2, Username: "staf", Role: "staff"})
		return c.Next()
	})
	app.Delete("/test", RequirePermission(perms, "student:delete"), func(c *fiber.Ctx) error {
		return helper.Success(c, fiber.StatusOK, "OK", nil)
	})

	req := httptest.NewRequest("DELETE", "/test", nil)
	resp, _ := app.Test(req)
	if resp.StatusCode != fiber.StatusForbidden {
		t.Errorf("Expected 403 for staff without student:delete, got %d", resp.StatusCode)
	}
}

