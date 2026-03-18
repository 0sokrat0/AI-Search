package v1

import (
	"fmt"
	"strings"
	"time"

	"MRG/internal/delivery/http/response"
	"MRG/internal/domain/lead"
	lead_usecase "MRG/internal/usecase/lead"

	"github.com/gofiber/fiber/v2"
)

type LeadHandler struct {
	uc *lead_usecase.UseCase
}

func NewLeadHandler(uc *lead_usecase.UseCase) *LeadHandler {
	return &LeadHandler{uc: uc}
}

type leadDTO struct {
	ID                  string   `json:"id"`
	Name                string   `json:"name"`
	Contact             string   `json:"contact"`
	ChatTitle           string   `json:"chatTitle"`
	Text                string   `json:"text"`
	SourceMessageID     string   `json:"sourceMessageId"`
	Source              string   `json:"source"`
	DetectedBy          string   `json:"detectedBy"`
	SemanticDirection   string   `json:"semanticDirection"`
	SemanticCategory    string   `json:"semanticCategory"`
	MerchantID          string   `json:"merchantId"`
	CompanyID           string   `json:"companyId"`
	Company             string   `json:"company"`
	Status              string   `json:"status"`
	QualificationSource string   `json:"qualificationSource"`
	Priority            string   `json:"priority"`
	Score               float64  `json:"score"`
	Geo                 []string `json:"geo"`
	Products            []string `json:"products"`
	UserFeedback        *bool    `json:"userFeedback"`
	CategoryAssignedAt  string   `json:"categoryAssignedAt"`
	CreatedAt           string   `json:"createdAt"`
	UpdatedAt           string   `json:"updatedAt"`
}

type updateStatusRequest struct {
	Status string `json:"status"`
}

type updateCategoryRequest struct {
	Category string `json:"category"`
}

type setMerchantRequest struct {
	MerchantID string `json:"merchant_id"`
}

type leadBriefSignalDTO struct {
	ID                string  `json:"id"`
	ChatTitle         string  `json:"chatTitle"`
	FromName          string  `json:"fromName"`
	Contact           string  `json:"contact"`
	Text              string  `json:"text"`
	Date              string  `json:"date"`
	Score             float64 `json:"score"`
	IsLead            bool    `json:"isLead"`
	SemanticDirection string  `json:"semanticDirection"`
	SemanticCategory  string  `json:"semanticCategory"`
}

type leadBriefDTO struct {
	Lead         leadDTO              `json:"lead"`
	Signals      []leadBriefSignalDTO `json:"signals"`
	SignalsCount int                  `json:"signalsCount"`
	LastSeenAt   string               `json:"lastSeenAt"`
}

func (h *LeadHandler) GetLeads(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}

	f := lead.ListFilter{
		Limit:         c.QueryInt("limit", 50),
		Offset:        c.QueryInt("offset", 0),
		QualifiedOnly: c.QueryBool("qualified_only", false),
	}
	if s := c.Query("status"); s != "" {
		st := lead.Status(s)
		f.Status = &st
	}
	if m := c.Query("merchant_id"); m != "" {
		f.MerchantID = &m
	}
	if cat := strings.TrimSpace(c.Query("category")); cat != "" {
		if dir, ok := categoryToSemanticDirection(cat); ok {
			f.SemanticDirection = &dir
		}
	}
	if dir := strings.TrimSpace(c.Query("semantic_direction")); dir != "" {
		f.SemanticDirection = &dir
	}

	leads, err := h.uc.List(c.Context(), tenantID, f)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	out := make([]leadDTO, 0, len(leads))
	for _, l := range leads {
		out = append(out, toLeadDTO(l))
	}
	return response.OK(c, out)
}

func (h *LeadHandler) GetLeadBrief(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}
	brief, err := h.uc.GetBrief(c.Context(), tenantID, c.Params("id"))
	if err != nil {
		if err == lead.ErrLeadNotFound {
			return response.ErrorResponse(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
		}
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	leadView := toLeadDTO(brief.Lead)
	signals := make([]leadBriefSignalDTO, 0, len(brief.Signals))
	for _, s := range brief.Signals {
		signals = append(signals, leadBriefSignalDTO{
			ID:                s.ID,
			ChatTitle:         s.ChatTitle,
			FromName:          leadView.Name,
			Contact:           leadView.Contact,
			Text:              s.Text,
			Date:              s.CreatedAt.UTC().Format(time.RFC3339),
			Score:             s.Score,
			IsLead:            s.IsLead,
			SemanticDirection: s.SemanticDirection,
			SemanticCategory:  s.SemanticCategory,
		})
	}

	return response.OK(c, leadBriefDTO{
		Lead:         leadView,
		Signals:      signals,
		SignalsCount: brief.SignalsCount,
		LastSeenAt:   brief.LastSeenAt.UTC().Format(time.RFC3339),
	})
}

func (h *LeadHandler) Approve(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	l, err := h.uc.Approve(c.Context(), tenantID, c.Params("id"))
	return h.respondLead(c, l, err)
}

func (h *LeadHandler) Reject(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	l, err := h.uc.Reject(c.Context(), tenantID, c.Params("id"))
	return h.respondLead(c, l, err)
}

func (h *LeadHandler) UpdateStatus(c *fiber.Ctx) error {
	var req updateStatusRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "cannot parse body")
	}
	to := lead.Status(strings.TrimSpace(req.Status))
	if !lead.IsValidStatus(to) {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_STATUS", "status must be one of: new, contacted, qualified, converted, rejected")
	}
	tenantID := tenantFromCtx(c)
	l, err := h.uc.SetStatus(c.Context(), tenantID, c.Params("id"), to)
	if err == lead.ErrInvalidStatus || err == lead.ErrInvalidStatusTransition {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_STATUS", err.Error())
	}
	return h.respondLead(c, l, err)
}

func (h *LeadHandler) UpdateCategory(c *fiber.Ctx) error {
	var req updateCategoryRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "cannot parse body")
	}
	tenantID := tenantFromCtx(c)
	l, err := h.uc.UpdateCategory(c.Context(), tenantID, c.Params("id"), strings.TrimSpace(req.Category))
	return h.respondLead(c, l, err)
}

func (h *LeadHandler) SetMerchant(c *fiber.Ctx) error {
	var req setMerchantRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "cannot parse body")
	}
	tenantID := tenantFromCtx(c)
	l, err := h.uc.SetMerchant(c.Context(), tenantID, c.Params("id"), req.MerchantID)
	return h.respondLead(c, l, err)
}

func (h *LeadHandler) Delete(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if err := h.uc.Delete(c.Context(), tenantID, c.Params("id")); err != nil {
		if err == lead.ErrLeadNotFound {
			return response.ErrorResponse(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
		}
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return response.OK(c, fiber.Map{"ok": true})
}

func (h *LeadHandler) GetStats(c *fiber.Ctx) error {
	tenantID := tenantFromCtx(c)
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "tenant id required")
	}
	days := c.QueryInt("days", 30)
	if days <= 0 || days > 365 {
		days = 30
	}
	stats, err := h.uc.GetStats(c.Context(), tenantID, days)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return response.OK(c, stats)
}

func (h *LeadHandler) respondLead(c *fiber.Ctx, l *lead.Lead, err error) error {
	if err != nil {
		if err == lead.ErrLeadNotFound {
			return response.ErrorResponse(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
		}
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return response.OK(c, toLeadDTO(l))
}

func toLeadDTO(l *lead.Lead) leadDTO {
	name := strings.TrimSpace(l.SenderName())
	if name == "" {
		name = strings.TrimPrefix(strings.TrimSpace(l.SenderUsername()), "@")
	}
	if name == "" {
		name = fmt.Sprintf("Lead %d", l.SenderID())
	}

	contact := strings.TrimSpace(l.SenderUsername())
	if contact != "" && !strings.HasPrefix(contact, "@") {
		contact = "@" + contact
	}
	if contact == "" {
		contact = name
	}

	var catAssignedAt string
	if l.CategoryAssignedAt() != nil {
		catAssignedAt = l.CategoryAssignedAt().UTC().Format(time.RFC3339)
	}

	return leadDTO{
		ID:                  l.ID(),
		Name:                name,
		Contact:             contact,
		ChatTitle:           l.ChatTitle(),
		Text:                l.Text(),
		SourceMessageID:     l.MessageID(),
		Source:              "signals_inbox",
		DetectedBy:          "vector_sieve",
		SemanticDirection:   l.SemanticDirection(),
		SemanticCategory:    l.SemanticCategory(),
		MerchantID:          l.MerchantID(),
		CompanyID:           l.MerchantID(),
		Company:             l.MerchantID(),
		Status:              string(l.Status()),
		QualificationSource: string(l.QualificationSource()),
		Priority:            string(l.Priority()),
		Score:               l.Score(),
		Geo:                 l.Geo(),
		Products:            l.Products(),
		UserFeedback:        l.UserFeedback(),
		CategoryAssignedAt:  catAssignedAt,
		CreatedAt:           l.CreatedAt().UTC().Format(time.RFC3339),
		UpdatedAt:           l.UpdatedAt().UTC().Format(time.RFC3339),
	}
}

func semanticDirectionToCategory(direction string) string {
	if direction == "" {
		return "leads"
	}
	switch strings.ToLower(strings.TrimSpace(direction)) {
	case "traders", "trader":
		return "traders"
	case "merchant", "merchants", "merch":
		return "merchants"
	case "processing_requests", "processing_request", "processing", "request_processing":
		return "merchants"
	case "ps_offers", "ps_offer", "offer", "offers":
		return "ps_offers"
	case "noise", "spam":
		return "noise"
	default:
		return "leads"
	}
}

func categoryToSemanticDirection(category string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(category)) {
	case "traders", "trader":
		return "traders", true
	case "merchant", "merchants", "merch":
		return "merchant", true
	case "processing_requests", "processing_request", "processing":
		return "merchant", true
	case "ps_offers", "ps_offer", "offers":
		return "ps_offers", true
	case "noise":
		return "noise", true
	case "leads", "lead":
		return "", true
	default:
		return "", false
	}
}
