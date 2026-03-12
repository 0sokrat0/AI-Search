package response

import "github.com/gofiber/fiber/v2"

type Response struct {
	Success bool   `json:"success"`
	Data    any    `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func OK(c *fiber.Ctx, data any) error {
	return c.JSON(Response{
		Success: true,
		Data:    data,
	})
}

func ErrorResponse(c *fiber.Ctx, statusCode int, code, message string) error {
	return c.Status(statusCode).JSON(Response{
		Success: false,
		Error: &Error{
			Code:    code,
			Message: message,
		},
	})
}
