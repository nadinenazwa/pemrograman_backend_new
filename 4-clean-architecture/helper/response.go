package helper

import (
	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
)

func Success(c *fiber.Ctx, status int, message string, data any) error {
	return c.Status(status).JSON(model.APIResponse{
		Success: true, Message: message, Data: data,
	})
}

func SuccessList(c *fiber.Ctx, message string, data any, meta *model.Meta) error {
	return c.Status(fiber.StatusOK).JSON(model.APIResponse{
		Success: true, Message: message, Data: data, Meta: meta,
	})
}

func Created(c *fiber.Ctx, message string, data any, location string) error {
	c.Set("Location", location)
	return c.Status(fiber.StatusCreated).JSON(model.APIResponse{
		Success: true, Message: message, Data: data,
	})
}

func NoContent(c *fiber.Ctx) error {
	return c.SendStatus(fiber.StatusNoContent)
}

func Fail(c *fiber.Ctx, status int, message string) error {
	return c.Status(status).JSON(model.APIResponse{
		Success: false, Message: message,
	})
}

func FailValidation(c *fiber.Ctx, message string, errs map[string]string) error {
	return c.Status(fiber.StatusUnprocessableEntity).JSON(model.APIResponse{
		Success: false, Message: message, Errors: errs,
	})
}