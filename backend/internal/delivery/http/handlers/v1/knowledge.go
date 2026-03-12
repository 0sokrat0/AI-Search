package v1

import (
	"strings"

	"MRG/internal/delivery/http/response"
	knowledge_usecase "MRG/internal/usecase/knowledge"

	"github.com/gofiber/fiber/v2"
)

type KnowledgeHandler struct {
	service *knowledge_usecase.ImportService
}

func NewKnowledgeHandler(service *knowledge_usecase.ImportService) *KnowledgeHandler {
	return &KnowledgeHandler{service: service}
}

func (h *KnowledgeHandler) ImportCSV(c *fiber.Ctx) error {
	var body struct {
		FileName string `json:"fileName"`
		Content  string `json:"content"`
	}

	if err := c.BodyParser(&body); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid json body")
	}
	if strings.TrimSpace(body.Content) == "" {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "file content is required")
	}

	result, err := h.service.ImportCSV(c.Context(), body.FileName, body.Content)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "IMPORT_FAILED", err.Error())
	}

	return response.OK(c, result)
}
