package user

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"time"

	"MRG/internal/domain/user"

	"github.com/google/uuid"
)

type UserUseCase struct {
	userRepo user.Repository
}

func NewUserUseCase(userRepo user.Repository) *UserUseCase {
	return &UserUseCase{
		userRepo: userRepo,
	}
}

type CreateUserRequest struct {
	TenantID string      `json:"tenant_id"`
	Email    string      `json:"email"`
	Name     string      `json:"name"`
	Password string      `json:"password"`
	Roles    []user.Role `json:"roles"`
}

func (uc *UserUseCase) CreateUser(ctx context.Context, req CreateUserRequest) (*user.User, error) {
	req.Roles = user.NormalizeRoles(req.Roles)
	if !isAllowedUserRoleSet(req.Roles) {
		return nil, user.ErrInvalidRole
	}

	if exists, err := uc.userRepo.ExistsByEmail(ctx, req.Email); err != nil {
		return nil, err
	} else if exists {
		return nil, user.ErrEmailAlreadyExists
	}

	newUser := user.New(req.TenantID, req.Email, req.Name, req.Roles)
	if err := newUser.SetPassword(req.Password); err != nil {
		return nil, err
	}

	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		return nil, err
	}

	return newUser, nil
}

type CreateInviteRequest struct {
	TenantID  string    `json:"tenant_id"`
	Role      user.Role `json:"role"`
	CreatedBy string    `json:"created_by"`
}

type CreateInviteResult struct {
	Invite *user.Invite
	Token  string
}

type AcceptInviteRequest struct {
	Name            string `json:"name"`
	Email           string `json:"email"`
	Password        string `json:"password"`
	PasswordConfirm string `json:"password_confirm"`
}

func (uc *UserUseCase) CreateInvite(ctx context.Context, req CreateInviteRequest) (*CreateInviteResult, error) {
	role := user.NormalizeRole(req.Role)
	if role != user.RoleSuperAdmin && role != user.RoleEmployee {
		return nil, user.ErrInvalidRole
	}

	token, tokenHash, err := generateInviteToken()
	if err != nil {
		return nil, err
	}

	invite := user.NewInvite(uuid.NewString(), req.TenantID, role, tokenHash, req.CreatedBy, time.Now().UTC().Add(72*time.Hour))
	if err := uc.userRepo.CreateInvite(ctx, invite); err != nil {
		return nil, err
	}

	return &CreateInviteResult{
		Invite: invite,
		Token:  token,
	}, nil
}

func (uc *UserUseCase) GetInviteByToken(ctx context.Context, token string) (*user.Invite, error) {
	invite, err := uc.userRepo.FindInviteByTokenHash(ctx, hashInviteToken(token))
	if err != nil {
		return nil, err
	}
	if invite.IsUsed() {
		return nil, user.ErrInviteUsed
	}
	if invite.IsExpired(time.Now().UTC()) {
		return nil, user.ErrInviteExpired
	}
	return invite, nil
}

func (uc *UserUseCase) AcceptInvite(ctx context.Context, token string, req AcceptInviteRequest) (*user.User, *user.Invite, error) {
	if req.Password != req.PasswordConfirm {
		return nil, nil, user.ErrInvalidPassword
	}
	if len(req.Password) < 8 {
		return nil, nil, user.ErrPasswordTooShort
	}

	invite, err := uc.GetInviteByToken(ctx, token)
	if err != nil {
		return nil, nil, err
	}

	if exists, err := uc.userRepo.ExistsByEmail(ctx, req.Email); err != nil {
		return nil, nil, err
	} else if exists {
		return nil, nil, user.ErrEmailAlreadyExists
	}

	newUser := user.New(invite.TenantID(), req.Email, req.Name, []user.Role{invite.Role()})
	if err := newUser.SetPassword(req.Password); err != nil {
		return nil, nil, err
	}
	if err := uc.userRepo.Create(ctx, newUser); err != nil {
		return nil, nil, err
	}

	invite.MarkUsed(newUser.ID(), time.Now().UTC())
	if err := uc.userRepo.UpdateInvite(ctx, invite); err != nil {
		return nil, nil, err
	}

	return newUser, invite, nil
}

func (uc *UserUseCase) GetUser(ctx context.Context, userID string) (*user.User, error) {
	return uc.userRepo.FindByID(ctx, userID)
}

type UpdateUserRequest struct {
	Name  string      `json:"name"`
	Email string      `json:"email"`
	Roles []user.Role `json:"roles"`
}

func (uc *UserUseCase) UpdateUser(ctx context.Context, userID string, req UpdateUserRequest) (*user.User, error) {
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if err := u.UpdateProfile(req.Name, req.Email); err != nil {
		return nil, err
	}

	if len(req.Roles) > 0 {
		normalized := user.NormalizeRoles(req.Roles)
		if isAllowedUserRoleSet(normalized) {
			u.SetRoles(normalized)
		}
	}

	if err := uc.userRepo.Update(ctx, u); err != nil {
		return nil, err
	}

	return u, nil
}

func (uc *UserUseCase) DeleteUser(ctx context.Context, userID string) error {
	return uc.userRepo.Delete(ctx, userID)
}

func (uc *UserUseCase) ListUsers(ctx context.Context, tenantID string, limit, offset int) ([]*user.User, error) {
	return uc.userRepo.FindByTenantID(ctx, tenantID, limit, offset)
}

func (uc *UserUseCase) AssignRole(ctx context.Context, userID string, role user.Role) error {
	role = user.NormalizeRole(role)
	if role != user.RoleSuperAdmin && role != user.RoleEmployee {
		return user.ErrInvalidRole
	}
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	u.AddRole(role)

	return uc.userRepo.Update(ctx, u)
}

func (uc *UserUseCase) RevokeRole(ctx context.Context, userID string, role user.Role) error {
	role = user.NormalizeRole(role)
	u, err := uc.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if err := u.RemoveRole(role); err != nil {
		return err
	}

	return uc.userRepo.Update(ctx, u)
}

func isAllowedUserRoleSet(roles []user.Role) bool {
	if len(roles) == 0 {
		return false
	}
	for _, role := range roles {
		if role != user.RoleSuperAdmin && role != user.RoleEmployee {
			return false
		}
	}
	return true
}

func generateInviteToken() (string, string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	return token, hashInviteToken(token), nil
}

func hashInviteToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
