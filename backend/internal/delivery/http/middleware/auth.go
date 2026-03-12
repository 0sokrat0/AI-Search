package middleware

import (
	"MRG/internal/config"
	"MRG/internal/infrastructure/auth"

	"github.com/gofiber/fiber/v2"
)

const (
	UserIDKey   = "userID"
	TenantIDKey = "tenantID"
)

func AuthRequired(cfg *config.Config) fiber.Handler {
	return func(c *fiber.Ctx) error {
		tokenString := c.Get("Authorization")
		if tokenString != "" {
			const prefix = "Bearer "
			if len(tokenString) < len(prefix) || tokenString[:len(prefix)] != prefix {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or malformed JWT"})
			}
			tokenString = tokenString[len(prefix):]
		} else {
			tokenString = c.Cookies("auth_token")
			if tokenString == "" {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "missing or malformed JWT"})
			}
		}

		claims, err := auth.ValidateJWT(tokenString, cfg)
		if err != nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid or expired JWT"})
		}

		userID, ok := claims["user_id"].(string)
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid JWT payload: user_id"})
		}

		tenantID, ok := claims["tenant_id"].(string)
		if !ok || tenantID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid JWT payload: tenant_id"})
		}

		c.Locals(UserIDKey, userID)
		c.Locals(TenantIDKey, tenantID)
		return c.Next()
	}
}
