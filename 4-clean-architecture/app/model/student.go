package model

import "time"

// Meta untuk offset-based pagination (users, achievements).
type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

// CursorMeta untuk cursor-based pagination (students).
type CursorMeta struct {
	Limit      int    `json:"limit"`
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// APIResponse adalah format response standar.
type APIResponse struct {
	Success   bool        `json:"success"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Errors    interface{} `json:"errors,omitempty"`
	Meta      *Meta       `json:"meta,omitempty"`
}

// Student merepresentasikan entitas mahasiswa di database.
type Student struct {
	ID        int       `json:"id"`
	NIM       string    `json:"nim"`
	Name      string    `json:"name"`
	Grade     float64   `json:"grade"`
	IsActive  bool      `json:"is_active"`
	OwnerID   int       `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateStudentRequest adalah request body untuk POST /students.
// Menggunakan declarative validation tags.
type CreateStudentRequest struct {
	NIM      string   `json:"nim" validate:"required,validnim"`
	Name     string   `json:"name" validate:"required,min=2,max=100"`
	Grade    *float64 `json:"grade" validate:"required,gte=0,lte=100"`
	IsActive *bool    `json:"is_active" validate:"required"`
}

// UpdateStudentRequest adalah request body untuk PATCH /students/:id.
// Menggunakan pointer + omitnil untuk membedakan "tidak dikirim" vs "dikirim kosong".
type UpdateStudentRequest struct {
	NIM      *string  `json:"nim,omitnil" validate:"omitempty,validnim"`
	Name     *string  `json:"name,omitnil" validate:"omitempty,min=2,max=100"`
	Grade    *float64 `json:"grade,omitnil" validate:"omitempty,gte=0,lte=100"`
	IsActive *bool    `json:"is_active,omitnil"`
}

// IsEmptyPatch mengecek apakah PATCH request tidak mengirim field sama sekali.
// Aturan: minimal satu field harus dikirim. Ini tidak bisa diekspresikan oleh validation tag
// karena tag bekerja per-field, bukan antar-field.
func (r UpdateStudentRequest) IsEmptyPatch() bool {
	return r.NIM == nil && r.Name == nil && r.Grade == nil && r.IsActive == nil
}

// ListQuery menampung parameter query string untuk GetStudents (offset-based, tetap untuk user/achievement).
type ListQuery struct {
	Page     int
	Limit    int
	Sort     string
	Order    string
	Search   string
	IsActive *bool
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}

// CursorQuery menampung parameter untuk cursor-based pagination students.
type CursorQuery struct {
	Limit  int
	Cursor string // base64 encoded cursor
	Search string
}