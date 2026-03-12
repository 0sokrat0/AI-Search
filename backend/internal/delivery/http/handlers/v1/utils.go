package v1

import (
	"MRG/internal/delivery/http/middleware"

	"github.com/gofiber/fiber/v2"
)

func tenantFromCtx(c *fiber.Ctx) string {
	t, _ := c.Locals(middleware.TenantIDKey).(string)
	return t
}
