package v1

import (
	"time"

	"MRG/internal/delivery/http/response"
	"MRG/internal/domain/user"
	user_usecase "MRG/internal/usecase/user"

	"github.com/gofiber/fiber/v2"
)

type UserHandler struct {
	userUseCase *user_usecase.UserUseCase
}

func NewUserHandler(userUseCase *user_usecase.UserUseCase) *UserHandler {
	return &UserHandler{
		userUseCase: userUseCase,
	}
}

func (h *UserHandler) CreateUser(c *fiber.Ctx) error {
	var req user_usecase.CreateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request")
	}

	createdUser, err := h.userUseCase.CreateUser(c.Context(), req)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data:    toUserResponse(createdUser),
	})
}

func (h *UserHandler) CreateInvite(c *fiber.Ctx) error {
	authUserID, _ := c.Locals("user_id").(string)

	var req struct {
		TenantID string    `json:"tenant_id"`
		Role     user.Role `json:"role"`
	}
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request")
	}

	result, err := h.userUseCase.CreateInvite(c.Context(), user_usecase.CreateInviteRequest{
		TenantID:  req.TenantID,
		Role:      req.Role,
		CreatedBy: authUserID,
	})
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVITE_CREATE_FAILED", err.Error())
	}

	return c.Status(fiber.StatusCreated).JSON(response.Response{
		Success: true,
		Data: fiber.Map{
			"token":      result.Token,
			"role":       result.Invite.Role(),
			"tenant_id":  result.Invite.TenantID(),
			"expires_at": result.Invite.ExpiresAt(),
		},
	})
}

func (h *UserHandler) GetUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	u, err := h.userUseCase.GetUser(c.Context(), userID)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusNotFound, "NOT_FOUND", err.Error())
	}
	return response.OK(c, toUserResponse(u))
}

func (h *UserHandler) UpdateUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	var req user_usecase.UpdateUserRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request")
	}

	updatedUser, err := h.userUseCase.UpdateUser(c.Context(), userID, req)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	return response.OK(c, toUserResponse(updatedUser))
}

func (h *UserHandler) DeleteUser(c *fiber.Ctx) error {
	userID := c.Params("id")
	if err := h.userUseCase.DeleteUser(c.Context(), userID); err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type RoleRequest struct {
	Role user.Role `json:"role"`
}

func (h *UserHandler) ListUsers(c *fiber.Ctx) error {
	tenantID := c.Query("tenant_id")
	if tenantID == "" {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "tenant_id required")
	}
	limit := c.QueryInt("limit", 100)
	offset := c.QueryInt("offset", 0)
	users, err := h.userUseCase.ListUsers(c.Context(), tenantID, limit, offset)
	if err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}
	out := make([]UserResponse, len(users))
	for i, u := range users {
		out[i] = toUserResponse(u)
	}
	return response.OK(c, out)
}

func (h *UserHandler) AssignRole(c *fiber.Ctx) error {
	userID := c.Params("id")
	var req RoleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request")
	}

	if err := h.userUseCase.AssignRole(c.Context(), userID, req.Role); err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	return response.OK(c, nil)
}

func (h *UserHandler) RevokeRole(c *fiber.Ctx) error {
	userID := c.Params("id")
	var req RoleRequest
	if err := c.BodyParser(&req); err != nil {
		return response.ErrorResponse(c, fiber.StatusBadRequest, "INVALID_REQUEST", "invalid request")
	}

	if err := h.userUseCase.RevokeRole(c.Context(), userID, req.Role); err != nil {
		return response.ErrorResponse(c, fiber.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
	}

	return response.OK(c, nil)
}

type UserResponse struct {
	ID        string      `json:"id"`
	TenantID  string      `json:"tenant_id"`
	Email     string      `json:"email"`
	Name      string      `json:"name"`
	Roles     []user.Role `json:"roles"`
	IsActive  bool        `json:"is_active"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	LastLogin *time.Time  `json:"last_login,omitempty"`
}

func toUserResponse(u *user.User) UserResponse {
	return UserResponse{
		ID:        u.ID(),
		TenantID:  u.TenantID(),
		Email:     u.Email(),
		Name:      u.Name(),
		Roles:     u.Roles(),
		IsActive:  u.IsActive(),
		CreatedAt: u.CreatedAt(),
		UpdatedAt: u.UpdatedAt(),
		LastLogin: u.LastLogin(),
	}
}
