package tests

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"siakad/internal/handlers"
	"siakad/internal/models"
)

func TestAuthValidation(t *testing.T) {
	app := fiber.New()
	
	// We just want to test the validation part, so we can mock the repo or test it 
	// until it hits the DB. If it hits the DB, it'll crash since DB is nil, but 
	// validation should happen before.
	authHandler := handlers.NewAuthHandler(nil)
	app.Post("/login", authHandler.Login)

	tests := []struct {
		name       string
		body       models.LoginRequest
		wantStatus int
	}{
		{
			name: "Invalid email",
			body: models.LoginRequest{
				Email:    "not-email",
				Password: "password123",
			},
			wantStatus: 422,
		},
		{
			name: "Password too short",
			body: models.LoginRequest{
				Email:    "test@test.com",
				Password: "short",
			},
			wantStatus: 422,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.body)
			req := httptest.NewRequest(http.MethodPost, "/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp, _ := app.Test(req)
			
			if tt.wantStatus == 422 && resp.StatusCode != fiber.StatusUnprocessableEntity {
				t.Errorf("Expected status 422, got %d", resp.StatusCode)
			}
		})
	}
}
