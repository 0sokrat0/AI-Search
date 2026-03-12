package v1

import (
	"MRG/internal/delivery/http/response"
	"MRG/internal/usecase/auth"
	user_usecase "MRG/internal/usecase/user"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	authUseCase *auth.AuthUseCase
	userUseCase *user_usecase.UserUseCase
}

func NewAuthHandler(authUseCase *auth.AuthUseCase, userUseCase *user_usecase.UserUseCase) *AuthHandler {
	return &AuthHandler{
		authUseCase: authUseCase,
		userUseCase: userUseCase,
	}
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req auth.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "Cannot parse request body")
	}

	u, accessToken, refreshToken, err := h.authUseCase.Login(c.Context(), req)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", err.Error())
	}

	return response.OK(c, fiber.Map{
		"user":           toUserResponse(u),
		"token":          accessToken,
		"accessToken":    accessToken,
		"refreshToken":   refreshToken,
		"is_super_admin": u.IsSuperAdmin(),
	})
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refreshToken"`
	}
	if err := c.BodyParser(&body); err != nil || body.RefreshToken == "" {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "refreshToken required")
	}

	accessToken, refreshToken, err := h.authUseCase.Refresh(c.Context(), body.RefreshToken)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusUnauthorized, "UNAUTHORIZED", "invalid or expired refresh token")
	}

	return response.OK(c, fiber.Map{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})
}

func (h *AuthHandler) GetInvite(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invite token required")
	}

	invite, err := h.userUseCase.GetInviteByToken(c.Context(), token)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_INVITE", err.Error())
	}

	return response.OK(c, fiber.Map{
		"role":       invite.Role(),
		"tenant_id":  invite.TenantID(),
		"expires_at": invite.ExpiresAt(),
	})
}

func (h *AuthHandler) AcceptInvite(c *fiber.Ctx) error {
	token := c.Params("token")
	if token == "" {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invite token required")
	}

	var req user_usecase.AcceptInviteRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request")
	}

	createdUser, _, err := h.userUseCase.AcceptInvite(c.Context(), token, req)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVITE_ACCEPT_FAILED", err.Error())
	}

	u, accessToken, refreshToken, err := h.authUseCase.Login(c.Context(), auth.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "LOGIN_FAILED", err.Error())
	}

	if u == nil {
		u = createdUser
	}

	return response.OK(c, fiber.Map{
		"user":         toUserResponse(u),
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	})
}
