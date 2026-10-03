package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"

	"siakad/internal/models"
	"siakad/internal/utils"
)

// Due to simplicity for UTS, we can mock or do simple endpoint structure check.
// In real app, we would use a test DB and real repositories.
// Here we will just write a placeholder that shows how tests are structured 
// or test the utils directly.

func TestHashPassword(t *testing.T) {
	pass := "secret123"
	hash, err := utils.HashPassword(pass)
	if err != nil {
		t.Fatalf("Failed to hash password: %v", err)
	}

	if !utils.CheckPasswordHash(pass, hash) {
		t.Errorf("Password hash check failed")
	}
}

func TestGenerateToken(t *testing.T) {
	// mock env
	t.Setenv("JWT_SECRET", "testsecret")

	token, exp, err := utils.GenerateToken(1, "test@siakad.test", "admin")
	if err != nil {
		t.Fatalf("Failed to generate token: %v", err)
	}
	if token == "" || exp != 86400 {
		t.Errorf("Invalid token or expiry")
	}

	claims, err := utils.ValidateToken(token)
	if err != nil {
		t.Fatalf("Failed to validate token: %v", err)
	}

	if claims.Role != "admin" || claims.UserID != 1 {
		t.Errorf("Claims mismatch")
	}
}

func TestAPIResponse(t *testing.T) {
	app := fiber.New()
	app.Get("/test", func(c *fiber.Ctx) error {
		return utils.SendSuccess(c, fiber.StatusOK, "Success", fiber.Map{"key": "value"})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	resp, _ := app.Test(req)

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var res models.APIResponse
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if !res.Success || res.Message != "Success" {
		t.Errorf("Unexpected response format")
	}
}

func TestValidation(t *testing.T) {
	req := models.CreateStudentRequest{
		NIM:      "123", // invalid
		Nama:     "", // invalid
		Email:    "invalid", // invalid
		Prodi:    "Teknik", // valid
		Angkatan: "123", // invalid length
	}

	errs := utils.ValidateStruct(req)
	if errs == nil {
		t.Fatalf("Expected validation errors, got nil")
	}

	if len(errs) != 4 {
		t.Errorf("Expected 4 validation errors, got %d", len(errs))
	}
}
