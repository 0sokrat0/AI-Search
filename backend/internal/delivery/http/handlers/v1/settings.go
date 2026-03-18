package v1

import (
	"fmt"
	"strconv"
	"time"

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

type cleanupNoiseRequest struct {
	OlderThanHours any      `json:"older_than_hours"`
	MessageIDs     []string `json:"message_ids"`
}

func parseCleanupHours(raw any) (int, error) {
	switch v := raw.(type) {
	case string:
		hours, err := strconv.Atoi(v)
		if err != nil {
			return 0, err
		}
		return hours, nil
	case float64:
		return int(v), nil
	case int:
		return v, nil
	case int64:
		return int(v), nil
	default:
		return 0, fmt.Errorf("invalid older_than_hours")
	}
}

func (h *SettingsHandler) CleanupNoise(c *fiber.Ctx) error {
	var req cleanupNoiseRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid json body")
	}

	hours, err := parseCleanupHours(req.OlderThanHours)
	if err != nil || hours <= 0 || hours > 24*365 {
		return fiber.NewError(fiber.StatusBadRequest, "older_than_hours must be between 1 and 8760")
	}

	deleted, err := h.uc.CleanupNoise(c.Context(), tenantFromCtx(c), time.Duration(hours)*time.Hour, req.MessageIDs)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}

	return c.JSON(fiber.Map{
		"deleted":     deleted,
		"hours":       hours,
		"message_ids": len(req.MessageIDs),
	})
}
