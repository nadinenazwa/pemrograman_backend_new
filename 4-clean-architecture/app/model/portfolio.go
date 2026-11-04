package model

import "time"
type StudentPortfolio struct {
	ID        string        `json:"id"`
	StudentID int           `json:"student_id"`
	Headline  string        `json:"headline"`
	Bio       string        `json:"bio"`
	Projects  []Project     `json:"projects"`
	Certs     []Certificate `json:"certificates"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// Project merepresentasikan satu proyek dalam portofolio mahasiswa.
type Project struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Technologies []string `json:"technologies"`
	Link         string   `json:"link,omitempty"`
	Year         int      `json:"year"`
}

// Certificate merepresentasikan satu sertifikat dalam portofolio mahasiswa.
type Certificate struct {
	Name     string    `json:"name"`
	Issuer   string    `json:"issuer"`
	IssuedAt time.Time `json:"issued_at"`
}

// UpsertPortfolioRequest adalah request body untuk PUT /students/:id/portfolio.
// Mengganti seluruh konten portofolio (semantik PUT).
type UpsertPortfolioRequest struct {
	Headline string               `json:"headline" validate:"required,min=2,max=200,nomongoop"`
	Bio      string               `json:"bio" validate:"required,min=2,max=2000,nomongoop"`
	Projects []ProjectRequest     `json:"projects" validate:"omitempty,max=20,dive"`
	Certs    []CertificateRequest `json:"certificates" validate:"omitempty,max=50,dive"`
}

// AddProjectRequest adalah request body untuk POST /students/:id/portfolio/projects.
type AddProjectRequest struct {
	Title        string   `json:"title" validate:"required,min=2,max=200,nomongoop"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=20,dive,min=1,max=100,nomongoop"`
	Link         string   `json:"link" validate:"omitempty,url,max=500"`
	Year         int      `json:"year" validate:"required,min=1900,max=2100"`
}

// ProjectRequest digunakan dalam UpsertPortfolioRequest untuk elemen projects.
type ProjectRequest struct {
	Title        string   `json:"title" validate:"required,min=2,max=200,nomongoop"`
	Technologies []string `json:"technologies" validate:"required,min=1,max=20,dive,min=1,max=100,nomongoop"`
	Link         string   `json:"link" validate:"omitempty,url,max=500"`
	Year         int      `json:"year" validate:"required,min=1900,max=2100"`
}

// CertificateRequest digunakan dalam UpsertPortfolioRequest untuk elemen certificates.
type CertificateRequest struct {
	Name     string    `json:"name" validate:"required,min=2,max=200,nomongoop"`
	Issuer   string    `json:"issuer" validate:"required,min=2,max=200,nomongoop"`
	IssuedAt time.Time `json:"issued_at" validate:"required"`
}

// TechStat adalah hasil aggregation statistik teknologi.
type TechStat struct {
	Technology string `json:"technology"`
	Count      int    `json:"count"`
}
