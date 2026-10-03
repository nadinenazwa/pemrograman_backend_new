package models

import (
	"time"
)

type User struct {
	ID       int    `json:"id"`
	Email    string `json:"email"`
	Password string `json:"-"` // never return password
	Role     string `json:"role"`
}

type Student struct {
	ID          int        `json:"id"`
	UserID      int        `json:"user_id"`
	NIM         string     `json:"nim"`
	Nama        string     `json:"nama"`
	Prodi       string     `json:"prodi"`
	Angkatan    string     `json:"angkatan"`
	IPKTerakhir float64    `json:"ipk_terakhir"`
	DeletedAt   *time.Time `json:"deleted_at,omitempty"`
}

type Course struct {
	ID       int    `json:"id"`
	KodeMK   string `json:"kode_mk"`
	NamaMK   string `json:"nama_mk"`
	SKS      int    `json:"sks"`
	Semester string `json:"semester"`
	Kuota    int    `json:"kuota"`
}

type Enrollment struct {
	ID            int       `json:"id"`
	StudentID     int       `json:"student_id"`
	CourseID      int       `json:"course_id"`
	TahunAkademik string    `json:"tahun_akademik"`
	CreatedAt     time.Time `json:"created_at"`
}

// Request/Response types

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type LoginResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"` // in seconds
	User        struct {
		ID    int    `json:"id"`
		Email string `json:"email"`
		Role  string `json:"role"`
	} `json:"user"`
}

type MeStudentData struct {
	NIM      string `json:"nim"`
	Nama     string `json:"nama"`
	Prodi    string `json:"prodi"`
	Angkatan string `json:"angkatan"`
}

type MeResponse struct {
	User    User           `json:"user"`
	Student *MeStudentData `json:"student,omitempty"`
}

type CreateStudentRequest struct {
	NIM         string   `json:"nim" validate:"required,len=12"`
	Nama        string   `json:"nama" validate:"required"`
	Email       string   `json:"email" validate:"required,email"`
	Prodi       string   `json:"prodi" validate:"required"`
	Angkatan    string   `json:"angkatan" validate:"required,len=4"`
	IPKTerakhir *float64 `json:"ipk_terakhir,omitempty" validate:"omitempty,min=0.00,max=4.00"`
}

type UpdateStudentRequest struct {
	Nama        string   `json:"nama" validate:"required"`
	Prodi       string   `json:"prodi" validate:"required"`
	Angkatan    string   `json:"angkatan" validate:"required,len=4"`
	IPKTerakhir *float64 `json:"ipk_terakhir,omitempty" validate:"omitempty,min=0.00,max=4.00"`
}

type CourseWithQuota struct {
	Course
	Terisi    int `json:"terisi"`
	SisaKuota int `json:"sisa_kuota"`
}

type StudentDetailResponse struct {
	Student
	Courses  []Course `json:"courses"`
	TotalSKS int      `json:"total_sks"`
	BatasSKS int      `json:"batas_sks"`
}

type CreateEnrollmentRequest struct {
	CourseID      int    `json:"course_id" validate:"required"`
	TahunAkademik string `json:"tahun_akademik" validate:"required"` // e.g. "2026/2027-Ganjil"
}

// Common JSON Responses
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Meta struct {
	CurrentPage int `json:"current_page"`
	PerPage     int `json:"per_page"`
	Total       int `json:"total"`
	LastPage    int `json:"last_page"`
}
