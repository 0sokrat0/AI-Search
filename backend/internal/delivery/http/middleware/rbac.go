package middleware

import (
	"errors"

	"MRG/internal/domain/user"

	"github.com/gofiber/fiber/v2"
)

func PermissionRequired(p user.Permission, userRepo user.Repository) fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID, ok := c.Locals("userID").(string)
		if !ok || userID == "" {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: user ID not found in token"})
		}

		u, err := userRepo.FindByID(c.Context(), userID)
		if err != nil {
			if errors.Is(err, user.ErrUserNotFound) {
				return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "Unauthorized: user not found"})
			}
			return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "Could not retrieve user"})
		}

		if !u.HasPermission(p) {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{"error": "Forbidden: you don't have permission to perform this action"})
		}

		return c.Next()
	}
}
