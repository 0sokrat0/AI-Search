package v1

import (
	"net/http"

	"MRG/internal/infrastructure/telegram"

	"github.com/gofiber/fiber/v2"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type AccountHandler struct {
	manager *telegram.Manager
}

func NewAccountHandler(manager *telegram.Manager) *AccountHandler {
	return &AccountHandler{manager: manager}
}

func (h *AccountHandler) List(c *fiber.Ctx) error {
	accounts, err := h.manager.ListAccounts(c.Context())
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(accounts)
}

func (h *AccountHandler) Add(c *fiber.Ctx) error {
	var req struct {
		Phone         string `json:"phone"`
		Proxy         string `json:"proxy"`
		ProxyFallback string `json:"proxy_fallback"`
	}
	if err := c.BodyParser(&req); err != nil {
		return err
	}

	acc, err := h.manager.AddAccount(c.Context(), req.Phone, req.Proxy, req.ProxyFallback)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.Status(http.StatusCreated).JSON(acc)
}

func (h *AccountHandler) Authenticate(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	useQR := c.Query("qr") == "true"

	println("Authenticate called for account", id.Hex(), "useQR:", useQR)

	err = h.manager.Authenticate(c.Context(), id, useQR)
	if err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(http.StatusAccepted).JSON(fiber.Map{"status": "auth_started"})
}

func (h *AccountHandler) ProvideCode(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		Code string `json:"code"`
	}
	if err := c.BodyParser(&req); err != nil {
		return err
	}

	h.manager.ProvideCode(id, req.Code)
	return c.SendStatus(http.StatusOK)
}

func (h *AccountHandler) ProvidePassword(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	var req struct {
		Password string `json:"password"`
	}
	if err := c.BodyParser(&req); err != nil {
		return err
	}

	h.manager.ProvidePassword(id, req.Password)
	return c.SendStatus(http.StatusOK)
}

func (h *AccountHandler) Start(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.manager.StartAccountByID(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(http.StatusAccepted)
}

func (h *AccountHandler) Stop(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.manager.StopAccountByID(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(http.StatusAccepted)
}

func (h *AccountHandler) Restart(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.manager.RestartAccountByID(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}

	return c.SendStatus(http.StatusAccepted)
}

func (h *AccountHandler) Delete(c *fiber.Ctx) error {
	id, err := primitive.ObjectIDFromHex(c.Params("id"))
	if err != nil {
		return c.Status(http.StatusBadRequest).JSON(fiber.Map{"error": "invalid id"})
	}

	if err := h.manager.DeleteAccount(c.Context(), id); err != nil {
		return c.Status(http.StatusInternalServerError).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(http.StatusNoContent)
}
