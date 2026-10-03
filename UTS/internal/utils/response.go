package utils

import (
	"errors"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"

	"siakad/internal/models"
)

var validate = validator.New()

func ValidateStruct(s interface{}) map[string][]string {
	err := validate.Struct(s)
	if err != nil {
		errs := make(map[string][]string)
		for _, err := range err.(validator.ValidationErrors) {
			field := err.Field() // or use json tag if possible, but basic is ok
			errs[field] = append(errs[field], err.Error())
		}
		return errs
	}
	return nil
}

// Custom error handling
func SendError(c *fiber.Ctx, status int, message string, errs interface{}) error {
	return c.Status(status).JSON(models.APIResponse{
		Success: false,
		Message: message,
		Errors:  errs,
	})
}

func SendSuccess(c *fiber.Ctx, status int, message string, data interface{}) error {
	return c.Status(status).JSON(models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
	})
}

func SendSuccessWithMeta(c *fiber.Ctx, status int, message string, data interface{}, meta models.Meta) error {
	return c.Status(status).JSON(models.APIResponse{
		Success: true,
		Message: message,
		Data:    data,
		Meta:    &meta,
	})
}

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	message := "Internal Server Error"

	var e *fiber.Error
	if errors.As(err, &e) {
		code = e.Code
		message = e.Message
	}

	return c.Status(code).JSON(models.APIResponse{
		Success: false,
		Message: message,
	})
}
