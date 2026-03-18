package v1

import (
	"errors"
	"strings"

	"MRG/internal/delivery/http/response"
	signal_usecase "MRG/internal/usecase/signal"
	"time"
	"github.com/gofiber/fiber/v2"
)

type SignalHandler struct {
	app signal_usecase.ServiceAPI
}

func NewSignalHandler(app signal_usecase.ServiceAPI) *SignalHandler {
	return &SignalHandler{app: app}
}

type bindContactRequest struct {
	MerchantID string `json:"merchant_id"`
}

func (h *SignalHandler) BindContact(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}

	var req bindContactRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "cannot parse body")
	}

	err := h.app.BindContact(c.Context(), signal_usecase.BindContactInput{
		TenantID:   tenantID,
		MessageID:  c.Params("id"),
		MerchantID: req.MerchantID,
	})
	if err != nil {
		if errors.Is(err, signal_usecase.ErrSignalNotFound) {
			return response.ErrorResponse(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
		}
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	return response.OK(c, fiber.Map{"ok": true})
}

func (h *SignalHandler) GetInbox(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}

	limit := c.QueryInt("limit", 50)
	offset := c.QueryInt("offset", 0)
	if limit <= 0 || limit > 50000 {
		limit = 50
	}

	var fromDate, toDate *time.Time
	if from := c.Query("from"); from != "" {
		if t, err := time.Parse(time.RFC3339, from); err == nil {
			fromDate = &t
		}
	}
	if to := c.Query("to"); to != "" {
		if t, err := time.Parse(time.RFC3339, to); err == nil {
			toDate = &t
		}
	}

	out, err := h.app.GetInbox(c.Context(), signal_usecase.InboxQuery{
		TenantID:     tenantID,
		Limit:        limit,
		Offset:       offset,
		Tab:          normalizeInboxTab(c.Query("tab")),
		Category:     normalizeInboxCategory(c.Query("category")),
		ShowArchived: c.QueryBool("show_archived", false),
		FromDate:     fromDate,
		ToDate:       toDate,
	})
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	return response.OK(c, out)
}

func (h *SignalHandler) GetChart(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}

	now := time.Now().UTC()
	from := now.AddDate(0, 0, -30)
	to := now

	if f := c.Query("from"); f != "" {
		if t, err := time.Parse(time.RFC3339, f); err == nil {
			from = t
		}
	}
	if t := c.Query("to"); t != "" {
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			to = parsed
		}
	}

	buckets, err := h.app.GetChart(c.Context(), tenantID, from, to)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return response.OK(c, buckets)
}

func (h *SignalHandler) GetStats(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}
	days := c.QueryInt("days", 7)
	if days <= 0 || days > 365 {
		days = 7
	}

	stats, err := h.app.GetStats(c.Context(), tenantID, days)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return response.OK(c, stats)
}

type signalFeedbackRequest struct {
	IsLead        bool     `json:"is_lead"`
	Category      string   `json:"category"`
	SemanticFlags []string `json:"semantic_flags"`
}

func (h *SignalHandler) FeedbackSignal(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}

	var req signalFeedbackRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "cannot parse body")
	}

	out, err := h.app.FeedbackSignal(c.Context(), signal_usecase.FeedbackInput{
		TenantID:      tenantID,
		MessageID:     c.Params("id"),
		IsLead:        req.IsLead,
		Category:      req.Category,
		SemanticFlags: req.SemanticFlags,
	})
	if err != nil {
		switch {
		case errors.Is(err, signal_usecase.ErrSieveUnavailable):
			return response.ErrorResponse(c, fiber.StatusServiceUnavailable, "SIEVE_UNAVAILABLE", err.Error())
		case errors.Is(err, signal_usecase.ErrSignalNotFound):
			return response.ErrorResponse(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
		case errors.Is(err, signal_usecase.ErrEmptySignalText):
			return response.ErrorResponse(c, fiber.StatusBadRequest, "EMPTY_TEXT", err.Error())
		default:
			return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		}
	}

	return response.OK(c, out)
}

type signalFlagRequest struct {
	Field string `json:"field"`
	Value bool   `json:"value"`
}

func (h *SignalHandler) FlagSignal(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}

	var req signalFlagRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "cannot parse body")
	}

	err := h.app.FlagSignal(c.Context(), signal_usecase.FlagInput{
		TenantID:  tenantID,
		MessageID: c.Params("id"),
		Field:     req.Field,
		Value:     req.Value,
	})
	if err != nil {
		if errors.Is(err, signal_usecase.ErrInvalidFlagField) {
			return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_FIELD", err.Error())
		}
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	return response.OK(c, fiber.Map{"ok": true, "field": req.Field, "value": req.Value})
}

func (h *SignalHandler) GetSenderHistory(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}

	limit := c.QueryInt("limit", 20)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	out, err := h.app.GetSenderHistory(c.Context(), tenantID, c.Params("senderID"), limit)
	if err != nil {
		if errors.Is(err, signal_usecase.ErrInvalidSenderID) {
			return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_SENDER", err.Error())
		}
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	return response.OK(c, out)
}

func normalizeInboxTab(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "lead", "leads":
		return "lead"
	default:
		return "all"
	}
}

func normalizeInboxCategory(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "trader", "traders":
		return "traders"
	case "merchant", "merchants", "merch":
		return "merchants"
	case "processing_request", "processing_requests", "request":
		return "merchants"
	case "ps_offer", "ps_offers", "offer":
		return "ps_offers"
	case "noise", "spam":
		return "noise"
	default:
		return "all"
	}
}
