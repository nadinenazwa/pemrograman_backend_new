package model

// WebResponse adalah format respons sukses standar API untuk dokumentasi Swagger.
type WebResponse struct {
	Success   bool   `json:"success" example:"true"`
	Message   string `json:"message" example:"operasi berhasil"`
	Data      any    `json:"data"`
	RequestID string `json:"request_id,omitempty"`
}

// ErrorResponse adalah format respons error standar API untuk dokumentasi Swagger.
type ErrorResponse struct {
	Success   bool              `json:"success" example:"false"`
	Code      string            `json:"code" example:"BAD_REQUEST"`
	Message   string            `json:"message" example:"permintaan tidak valid"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}
