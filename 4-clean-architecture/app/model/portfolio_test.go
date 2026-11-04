package model_test

import (
	"testing"
	"time"

	"api-students-db/app/model"
	"api-students-db/helper"
)

func TestValidateUpsertPortfolio_Valid(t *testing.T) {
	req := model.UpsertPortfolioRequest{
		Headline: "Full Stack Go Developer",
		Bio:      "Mahasiswa Teknik Informatika yang berfokus pada arsitektur microservices dan Go backend.",
		Projects: []model.ProjectRequest{
			{
				Title:        "E-Commerce Microservices",
				Technologies: []string{"Go", "PostgreSQL", "Docker"},
				Link:         "https://github.com/example/ecommerce",
				Year:         2025,
			},
		},
		Certs: []model.CertificateRequest{
			{
				Name:     "Certified Go Developer",
				Issuer:   "Linux Foundation",
				IssuedAt: time.Now(),
			},
		},
	}

	errs := helper.ValidateStruct(req)
	if errs != nil {
		t.Fatalf("expected nil validation errors, got %v", errs)
	}
}

func TestValidateUpsertPortfolio_HeadlineBioWajib(t *testing.T) {
	req := model.UpsertPortfolioRequest{
		Headline: "",
		Bio:      "",
	}

	errs := helper.ValidateStruct(req)
	if errs == nil {
		t.Fatal("expected validation errors for empty headline and bio, got nil")
	}

	if _, ok := errs["headline"]; !ok {
		t.Fatalf("expected error for headline, got: %v", errs)
	}
	if _, ok := errs["bio"]; !ok {
		t.Fatalf("expected error for bio, got: %v", errs)
	}
}

func TestValidateAddProject_Valid(t *testing.T) {
	req := model.AddProjectRequest{
		Title:        "Distributed Cache Engine",
		Technologies: []string{"Go", "Redis"},
		Link:         "https://github.com/example/cache",
		Year:         2024,
	}

	errs := helper.ValidateStruct(req)
	if errs != nil {
		t.Fatalf("expected nil validation errors, got %v", errs)
	}
}

func TestValidateAddProject_TitleWajib(t *testing.T) {
	req := model.AddProjectRequest{
		Title:        "",
		Technologies: []string{"Go"},
		Year:         2024,
	}

	errs := helper.ValidateStruct(req)
	if errs == nil {
		t.Fatal("expected validation error for empty title, got nil")
	}
	if _, ok := errs["title"]; !ok {
		t.Fatalf("expected error for title, got: %v", errs)
	}
}

func TestValidateAddProject_TechnologiesMinimalSatu(t *testing.T) {
	req := model.AddProjectRequest{
		Title:        "Empty Tech Project",
		Technologies: []string{},
		Year:         2024,
	}

	errs := helper.ValidateStruct(req)
	if errs == nil {
		t.Fatal("expected validation error for empty technologies slice, got nil")
	}
}

func TestValidateAddProject_YearOutOfRange(t *testing.T) {
	// Terlalu lama (< 1900)
	req1 := model.AddProjectRequest{
		Title:        "Old Project",
		Technologies: []string{"Assembly"},
		Year:         1850,
	}
	if errs := helper.ValidateStruct(req1); errs == nil {
		t.Fatal("expected validation error for year < 1900, got nil")
	}

	// Terlalu jauh di masa depan (> 2100)
	req2 := model.AddProjectRequest{
		Title:        "Future Project",
		Technologies: []string{"Quantum"},
		Year:         2150,
	}
	if errs := helper.ValidateStruct(req2); errs == nil {
		t.Fatal("expected validation error for year > 2100, got nil")
	}
}

// TestValidate_NoSQLInjectionRejection menguji penolakan eksplisit terhadap operator MongoDB mentah
func TestValidate_NoSQLInjectionRejection(t *testing.T) {
	// 1. Operator $where di headline
	req1 := model.UpsertPortfolioRequest{
		Headline: "$where: sleep(5000)",
		Bio:      "Normal valid bio for student portfolio testing",
	}
	errs1 := helper.ValidateStruct(req1)
	if errs1 == nil || errs1["headline"] == "" {
		t.Fatalf("expected validation rejection for $where operator in headline, got %v", errs1)
	}

	// 2. Operator {"$gt": ""} di Project Title
	req2 := model.AddProjectRequest{
		Title:        `{"$gt": ""}`,
		Technologies: []string{"Go"},
		Year:         2025,
	}
	errs2 := helper.ValidateStruct(req2)
	if errs2 == nil || errs2["title"] == "" {
		t.Fatalf("expected validation rejection for raw BSON operator in title, got %v", errs2)
	}

	// 3. Operator {"$ne": null} di Technologies
	req3 := model.AddProjectRequest{
		Title:        "Valid Title",
		Technologies: []string{`{"$ne": null}`},
		Year:         2025,
	}
	errs3 := helper.ValidateStruct(req3)
	if errs3 == nil {
		t.Fatal("expected validation rejection for raw BSON operator in technologies")
	}
}
