package helper

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"

	"api-students-db/app/model"
)

func ReqCtx(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

func ParseListQuery(c *fiber.Ctx) model.ListQuery {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	if page < 1 {
		page = 1
	}

	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if limit < 1 {
		limit = 10
	} else if limit > 50 {
		limit = 50
	}

	q := model.ListQuery{
		Page:   page,
		Limit:  limit,
		Sort:   c.Query("sort", "id"),
		Order:  c.Query("order", "asc"),
		Search: c.Query("search", ""),
	}

	if activeStr := c.Query("is_active", ""); activeStr != "" {
		val, err := strconv.ParseBool(activeStr)
		if err == nil {
			q.IsActive = &val
		}
	}
	return q
}