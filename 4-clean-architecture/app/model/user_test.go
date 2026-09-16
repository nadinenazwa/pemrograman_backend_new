package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestUser_PasswordTidakMunculDiJSON(t *testing.T) {
	user := User{
		ID:        1,
		Username:  "nadine",
		Email:     "nadine@example.com",
		Password:  "hashed-password-yang-rahasia",
		Role:      "user",
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	jsonBytes, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("gagal marshal user ke JSON: %v", err)
	}

	jsonStr := string(jsonBytes)

	// Password tidak boleh muncul di JSON
	if strings.Contains(jsonStr, "password") {
		t.Error("field 'password' tidak boleh muncul dalam JSON response")
	}
	if strings.Contains(jsonStr, "hashed-password-yang-rahasia") {
		t.Error("nilai password tidak boleh muncul dalam JSON response")
	}

	// Field lain harus ada
	if !strings.Contains(jsonStr, "username") {
		t.Error("field 'username' harus ada di JSON response")
	}
	if !strings.Contains(jsonStr, "email") {
		t.Error("field 'email' harus ada di JSON response")
	}
	if !strings.Contains(jsonStr, "role") {
		t.Error("field 'role' harus ada di JSON response")
	}
}

func TestRegisterRequest_TidakAdaFieldRole(t *testing.T) {
	// Test ini memastikan RegisterRequest tidak memiliki field Role
	// sehingga mass assignment dicegah.
	jsonInput := `{"username":"nadine","email":"nadine@example.com","password":"Password123","role":"admin"}`

	var req RegisterRequest
	if err := json.Unmarshal([]byte(jsonInput), &req); err != nil {
		t.Fatalf("gagal unmarshal: %v", err)
	}

	// Pastikan tidak ada field role di struct
	// Jadi meskipun client mengirim "role":"admin", tidak akan ter-parse
	jsonOutput, _ := json.Marshal(req)
	if strings.Contains(string(jsonOutput), "admin") {
		t.Error("RegisterRequest seharusnya TIDAK menerima field role dari JSON input")
	}
}
