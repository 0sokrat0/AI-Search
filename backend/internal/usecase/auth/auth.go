package auth

import (
	"context"

	"MRG/internal/config"
	"MRG/internal/domain/user"
	"MRG/internal/infrastructure/auth"
)

type AuthUseCase struct {
	userRepo user.Repository
	cfg      *config.Config
}

func NewAuthUseCase(userRepo user.Repository, cfg *config.Config) *AuthUseCase {
	return &AuthUseCase{
		userRepo: userRepo,
		cfg:      cfg,
	}
}

type LoginRequest struct {
	Email    string
	Password string
}

func (uc *AuthUseCase) Login(ctx context.Context, req LoginRequest) (*user.User, string, string, error) {
	u, err := uc.userRepo.FindByEmail(ctx, req.Email)
	if err != nil {
		return nil, "", "", err
	}

	if !u.IsActive() {
		return nil, "", "", user.ErrUserInactive
	}

	if !u.CheckPassword(req.Password) {
		return nil, "", "", user.ErrInvalidPassword
	}

	u.RecordLogin()
	if err := uc.userRepo.Update(ctx, u); err != nil {
		return nil, "", "", err
	}

	accessToken, err := auth.GenerateJWT(u.ID(), u.TenantID(), uc.cfg)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := auth.GenerateRefreshJWT(u.ID(), u.TenantID(), uc.cfg)
	if err != nil {
		return nil, "", "", err
	}

	return u, accessToken, refreshToken, nil
}

func (uc *AuthUseCase) Refresh(ctx context.Context, refreshTokenStr string) (string, string, error) {
	claims, err := auth.ValidateJWT(refreshTokenStr, uc.cfg)
	if err != nil {
		return "", "", user.ErrUnauthorized
	}

	if t, ok := claims["type"].(string); ok && t != "refresh" {
		return "", "", user.ErrUnauthorized
	}

	userID, _ := claims["user_id"].(string)
	tenantID, _ := claims["tenant_id"].(string)
	if userID == "" {
		return "", "", user.ErrUnauthorized
	}

	accessToken, err := auth.GenerateJWT(userID, tenantID, uc.cfg)
	if err != nil {
		return "", "", err
	}

	newRefresh, err := auth.GenerateRefreshJWT(userID, tenantID, uc.cfg)
	if err != nil {
		return "", "", err
	}

	return accessToken, newRefresh, nil
}
