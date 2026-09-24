package model

type Meta struct {
	Page       int `json:"page"`
	Limit      int `json:"limit"`
	Total      int `json:"total"`
	TotalPages int `json:"total_pages"`
}

type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
	Meta    *Meta       `json:"meta,omitempty"`
}

type Student struct {
	ID       int     `json:"id"`
	NIM      string  `json:"nim"`
	Name     string  `json:"name"`
	Grade    float64 `json:"grade"`
	IsActive bool    `json:"is_active"`
	OwnerID  int     `json:"owner_id"`
}

type CreateStudentRequest struct {
	NIM      string   `json:"nim"`
	Name     string   `json:"name"`
	Grade    *float64 `json:"grade"`
	IsActive *bool    `json:"is_active"`
}

type UpdateStudentRequest struct {
	NIM      *string  `json:"nim"`
	Name     *string  `json:"name"`
	Grade    *float64 `json:"grade"`
	IsActive *bool    `json:"is_active"`
}

// ListQuery menampung parameter query string untuk GetStudents.
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