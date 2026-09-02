package main

// Meta Paginasi
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// Amplop Respons (Response Envelope)
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

// Entitas Utama Student
type Student struct {
	ID       int     `json:"id"`
	NIM      string  `json:"nim"`
	Name     string  `json:"name"`
	Grade    float64 `json:"grade"`
	IsActive bool    `json:"is_active"`
}

// Struct Request POST & PUT (Semua field wajib dikirim)
type CreateStudentRequest struct {
	NIM      string   `json:"nim"`
	Name     string   `json:"name"`
	Grade    *float64 `json:"grade"`
	IsActive *bool    `json:"is_active"`
}

// Struct Request PATCH (Field opsional)
type UpdateStudentRequest struct {
	NIM      *string  `json:"nim"`
	Name     *string  `json:"name"`
	Grade    *float64 `json:"grade"`
	IsActive *bool    `json:"is_active"`
}

// Storage In-Memory
var studentList = []Student{
	{ID: 1, NIM: "230101001", Name: "Nadine Nazwa Andina", Grade: 88.5, IsActive: true},
	{ID: 2, NIM: "230101002", Name: "Adam Ahmad Bimantoro", Grade: 92.0, IsActive: true},
	{ID: 3, NIM: "230101003", Name: "Lusiana Ramadhan", Grade: 78.0, IsActive: false},
}

var nextID = 4