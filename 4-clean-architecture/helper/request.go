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

func RequestContext(c *fiber.Ctx) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.UserContext(), 5*time.Second)
}

func ParamID(c *fiber.Ctx) (int, bool) {
	id, err := strconv.Atoi(c.Params("id"))
	if err != nil || id <= 0 {
		return 0, false
	}
	return id, true
}

type UserContext struct {
	UserID   int
	Username string
	Role     string
}

func CurrentUser(c *fiber.Ctx) (UserContext, bool) {
	u, ok := c.Locals("auth_user").(model.AuthUser)
	if !ok {
		return UserContext{}, false
	}
	return UserContext{
		UserID:   u.ID,
		Username: u.Username,
		Role:     u.Role,
	}, true
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
