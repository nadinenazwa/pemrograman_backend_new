package helper

import (
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"
)

// Error codes stabil yang digunakan di seluruh aplikasi.
const (
	CodeValidationError    = "VALIDATION_ERROR"
	CodeBadRequest         = "BAD_REQUEST"
	CodeUnauthorized       = "UNAUTHORIZED"
	CodeForbidden          = "FORBIDDEN"
	CodeNotFound           = "NOT_FOUND"
	CodeConflict           = "CONFLICT"
	CodeUnsupportedMedia   = "UNSUPPORTED_MEDIA_TYPE"
	CodeNotAcceptable      = "NOT_ACCEPTABLE"
	CodeTooManyRequests    = "TOO_MANY_REQUESTS"
	CodeInternalError      = "INTERNAL_ERROR"
)

// AppError adalah error terstruktur yang digunakan oleh seluruh handler/service.
// Field `cause` hanya untuk logging, TIDAK pernah dikirim ke client.
type AppError struct {
	Status  int               `json:"-"`
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
	cause   error
}

func (e *AppError) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s (cause: %v)", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.cause
}

// ---------- Constructor helpers ----------

func NewAppError(status int, code, message string) *AppError {
	return &AppError{Status: status, Code: code, Message: message}
}

func NewAppErrorWithCause(status int, code, message string, cause error) *AppError {
	return &AppError{Status: status, Code: code, Message: message, cause: cause}
}

func NewValidationError(fields map[string]string) *AppError {
	return &AppError{
		Status:  fiber.StatusUnprocessableEntity,
		Code:    CodeValidationError,
		Message: "Validasi gagal",
		Fields:  fields,
	}
}

func ErrBadRequest(msg string) *AppError {
	return NewAppError(fiber.StatusBadRequest, CodeBadRequest, msg)
}

func ErrUnauthorized(msg string) *AppError {
	return NewAppError(fiber.StatusUnauthorized, CodeUnauthorized, msg)
}

func ErrForbidden(msg string) *AppError {
	return NewAppError(fiber.StatusForbidden, CodeForbidden, msg)
}

func ErrNotFoundMsg(msg string) *AppError {
	return NewAppError(fiber.StatusNotFound, CodeNotFound, msg)
}

func ErrConflict(msg string) *AppError {
	return NewAppError(fiber.StatusConflict, CodeConflict, msg)
}

func ErrUnsupportedMedia(msg string) *AppError {
	return NewAppError(fiber.StatusUnsupportedMediaType, CodeUnsupportedMedia, msg)
}

func ErrNotAcceptable(msg string) *AppError {
	return NewAppError(fiber.StatusNotAcceptable, CodeNotAcceptable, msg)
}

func ErrTooManyRequests(msg string) *AppError {
	return NewAppError(fiber.StatusTooManyRequests, CodeTooManyRequests, msg)
}

func ErrInternal(msg string, cause error) *AppError {
	return NewAppErrorWithCause(fiber.StatusInternalServerError, CodeInternalError, msg, cause)
}

// ---------- Centralized Error Handler ----------

// ErrorResponse adalah format JSON response error yang dikirim ke client.
type ErrorResponse struct {
	Success   bool              `json:"success"`
	Code      string            `json:"code"`
	Message   string            `json:"message"`
	Fields    map[string]string `json:"fields,omitempty"`
	RequestID string            `json:"request_id,omitempty"`
}

// ErrorHandler adalah centralized error handler untuk Fiber.
// Dipasang sebagai fiber.Config.ErrorHandler.
func ErrorHandler(c *fiber.Ctx, err error) error {
	requestID, _ := c.Locals("requestid").(string)

	// Cek apakah error adalah AppError
	if appErr, ok := err.(*AppError); ok {
		// Log cause jika ada (hanya internal)
		if appErr.cause != nil {
			slog.Error("request_failed",
				slog.String("request_id", requestID),
				slog.String("code", appErr.Code),
				slog.String("cause", appErr.cause.Error()),
			)
		}

		return c.Status(appErr.Status).JSON(ErrorResponse{
			Success:   false,
			Code:      appErr.Code,
			Message:   appErr.Message,
			Fields:    appErr.Fields,
			RequestID: requestID,
		})
	}

	// Cek apakah error adalah *fiber.Error (dari middleware Fiber)
	if fiberErr, ok := err.(*fiber.Error); ok {
		code := CodeInternalError
		switch fiberErr.Code {
		case fiber.StatusBadRequest:
			code = CodeBadRequest
		case fiber.StatusUnauthorized:
			code = CodeUnauthorized
		case fiber.StatusForbidden:
			code = CodeForbidden
		case fiber.StatusNotFound:
			code = CodeNotFound
		case fiber.StatusTooManyRequests:
			code = CodeTooManyRequests
		}

		return c.Status(fiberErr.Code).JSON(ErrorResponse{
			Success:   false,
			Code:      code,
			Message:   fiberErr.Message,
			RequestID: requestID,
		})
	}

	// Error tidak dikenal → 500
	slog.Error("request_failed",
		slog.String("request_id", requestID),
		slog.String("error", err.Error()),
	)

	return c.Status(fiber.StatusInternalServerError).JSON(ErrorResponse{
		Success:   false,
		Code:      CodeInternalError,
		Message:   "Terjadi kesalahan pada server",
		RequestID: requestID,
	})
}
