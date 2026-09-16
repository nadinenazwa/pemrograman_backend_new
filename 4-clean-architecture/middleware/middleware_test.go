package middleware

import (
	"net/http/httptest"
	"strings"
	"testing"

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
