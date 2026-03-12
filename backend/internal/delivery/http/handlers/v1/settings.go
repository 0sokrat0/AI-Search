package v1

import (
	settings_usecase "MRG/internal/usecase/settings"

	"github.com/gofiber/fiber/v2"
)

type SettingsHandler struct {
	uc *settings_usecase.UseCase
}

func NewSettingsHandler(uc *settings_usecase.UseCase) *SettingsHandler {
	return &SettingsHandler{uc: uc}
}

func (h *SettingsHandler) GetSettings(c *fiber.Ctx) error {
	kv, err := h.uc.GetAll(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(kv)
}

func (h *SettingsHandler) UpdateSettings(c *fiber.Ctx) error {
	var patch map[string]string
	if err := c.BodyParser(&patch); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json body")
	}
	if err := h.uc.Update(c.Context(), patch); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	kv, err := h.uc.GetAll(c.Context())
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(kv)
}
