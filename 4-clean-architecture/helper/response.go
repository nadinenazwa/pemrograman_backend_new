package helper

import (
	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
)

func Success(c *fiber.Ctx, status int, message string, data any) error {
	requestID, _ := c.Locals("requestid").(string)
	return c.Status(status).JSON(fiber.Map{
		"success":    true,
		"message":    message,
		"data":       data,
		"request_id": requestID,
	})
}

func SuccessList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	requestID, _ := c.Locals("requestid").(string)
	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"success":    true,
		"message":    message,
		"data":       data,
		"meta":       meta,
		"request_id": requestID,
	})
}

func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	requestID, _ := c.Locals("requestid").(string)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success":    true,
		"message":    message,
		"data":       data,
		"request_id": requestID,
	})
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

// Fail tetap dipertahankan untuk kompatibilitas (auth, achievement, health check).
// Endpoint students TIDAK boleh menggunakan Fail lagi.
func Fail(c *fiber.Ctx, status int, message string) error {
	requestID, _ := c.Locals("requestid").(string)
	code := CodeInternalError
	switch status {
	case fiber.StatusBadRequest:
		code = CodeBadRequest
	case fiber.StatusUnauthorized:
		code = CodeUnauthorized
	case fiber.StatusForbidden:
		code = CodeForbidden
	case fiber.StatusNotFound:
		code = CodeNotFound
	case fiber.StatusConflict:
		code = CodeConflict
	case fiber.StatusUnsupportedMediaType:
		code = CodeUnsupportedMedia
	case fiber.StatusNotAcceptable:
		code = CodeNotAcceptable
	case fiber.StatusTooManyRequests:
		code = CodeTooManyRequests
	case fiber.StatusUnprocessableEntity:
		code = CodeValidationError
	}
	return c.Status(status).JSON(ErrorResponse{
		Success:   false,
		Code:      code,
		Message:   message,
		RequestID: requestID,
	})
}

func FailValidation(c *fiber.Ctx, message string, errs map[string]string) error {
	requestID, _ := c.Locals("requestid").(string)
	return c.Status(fiber.StatusUnprocessableEntity).JSON(ErrorResponse{
		Success:   false,
		Code:      CodeValidationError,
		Message:   message,
		Fields:    errs,
		RequestID: requestID,
	})
}