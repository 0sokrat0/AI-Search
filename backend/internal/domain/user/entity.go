package user

import (
	"slices"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type Role string

const (
	RoleSuperAdmin Role = "super_admin"
	RoleEmployee   Role = "employee"
)

type Permission string

const (
	PermissionManageUsers     Permission = "manage_users"
	PermissionManageChats     Permission = "manage_chats"
	PermissionViewLeads       Permission = "view_leads"
	PermissionManageLeads     Permission = "manage_leads"
	PermissionExportData      Permission = "export_data"
	PermissionManageTenant    Permission = "manage_tenant"
	PermissionViewAnalytics   Permission = "view_analytics"
	PermissionManageProposals Permission = "manage_proposals"
	PermissionContactLeads    Permission = "contact_leads"
	PermissionManageTeam      Permission = "manage_team"
	PermissionConfigureAI     Permission = "configure_ai"
)

var rolePermissions = map[Role][]Permission{
	RoleSuperAdmin: {
		PermissionManageUsers,
		PermissionManageChats,
		PermissionViewLeads,
		PermissionManageLeads,
		PermissionExportData,
		PermissionManageTenant,
		PermissionViewAnalytics,
		PermissionManageProposals,
		PermissionContactLeads,
		PermissionManageTeam,
		PermissionConfigureAI,
	},
	RoleEmployee: {
		PermissionManageChats,
		PermissionViewLeads,
		PermissionManageLeads,
		PermissionViewAnalytics,
		PermissionManageProposals,
		PermissionContactLeads,
	},
}

type User struct {
	id               string
	tenantID         string
	email            string
	password         string
	name             string
	roles            []Role
	isActive         bool
	telegramUserID   *int64
	telegramUsername string
	createdAt        time.Time
	updatedAt        time.Time
	lastLogin        *time.Time
}

func New(tenantID, email, name string, roles []Role) *User {
	now := time.Now()
	return &User{
		id:        uuid.New().String(),
		tenantID:  tenantID,
		email:     email,
		name:      name,
		roles:     NormalizeRoles(roles),
		isActive:  true,
		createdAt: now,
		updatedAt: now,
	}
}

func Restore(id, tenantID, email, password, name string, roles []Role, isActive bool, telegramUserID *int64, telegramUsername string, createdAt, updatedAt time.Time, lastLogin *time.Time) *User {
	return &User{
		id:               id,
		tenantID:         tenantID,
		email:            email,
		password:         password,
		name:             name,
		roles:            NormalizeRoles(roles),
		isActive:         isActive,
		telegramUserID:   telegramUserID,
		telegramUsername: telegramUsername,
		createdAt:        createdAt,
		updatedAt:        updatedAt,
		lastLogin:        lastLogin,
	}
}

func (u *User) ID() string               { return u.id }
func (u *User) TenantID() string         { return u.tenantID }
func (u *User) Email() string            { return u.email }
func (u *User) Password() string         { return u.password }
func (u *User) Name() string             { return u.name }
func (u *User) Roles() []Role            { return u.roles }
func (u *User) IsActive() bool           { return u.isActive }
func (u *User) TelegramUserID() *int64   { return u.telegramUserID }
func (u *User) TelegramUsername() string { return u.telegramUsername }
func (u *User) CreatedAt() time.Time     { return u.createdAt }
func (u *User) UpdatedAt() time.Time     { return u.updatedAt }
func (u *User) LastLogin() *time.Time    { return u.lastLogin }

func (u *User) HasTelegramID(id int64) bool {
	return u.telegramUserID != nil && *u.telegramUserID == id
}

func (u *User) Identifier() string {
	if u.telegramUsername != "" {
		return "@" + u.telegramUsername
	}
	if u.email != "" {
		return u.email
	}
	return u.name
}

func (u *User) SetTelegram(id int64, username string) {
	u.telegramUserID = &id
	u.telegramUsername = username
	u.updatedAt = time.Now()
}

func (u *User) HasRole(role Role) bool {
	for _, r := range u.roles {
		if r == role || r == RoleSuperAdmin {
			return true
		}
	}
	return false
}

func (u *User) HasPermission(permission Permission) bool {
	if !u.isActive {
		return false
	}

	for _, role := range u.roles {
		perms, exists := rolePermissions[role]
		if !exists {
			continue
		}
		if slices.Contains(perms, permission) {
			return true
		}
	}
	return false
}

func (u *User) IsSuperAdmin() bool {
	return u.HasRole(RoleSuperAdmin)
}

func (u *User) IsTenantAdmin() bool {
	return false
}

func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.password = string(hashedPassword)
	u.updatedAt = time.Now()
	return nil
}

func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.password), []byte(password))
	return err == nil
}

func (u *User) UpdateProfile(name, email string) error {
	if name == "" {
		return ErrEmptyName
	}
	if email == "" {
		return ErrEmptyEmail
	}
	u.name = name
	u.email = email
	u.updatedAt = time.Now()
	return nil
}

func (u *User) AddRole(role Role) {
	role = NormalizeRole(role)
	if u.HasRole(role) {
		return
	}
	u.roles = append(u.roles, role)
	u.updatedAt = time.Now()
}

func (u *User) RemoveRole(role Role) error {
	role = NormalizeRole(role)
	if role == RoleSuperAdmin && u.HasRole(RoleSuperAdmin) {
		return ErrCannotRemoveSuperAdmin
	}

	var updatedRoles []Role
	for _, r := range u.roles {
		if r != role {
			updatedRoles = append(updatedRoles, r)
		}
	}

	if len(updatedRoles) == 0 {
		return ErrUserMustHaveRole
	}

	u.roles = updatedRoles
	u.updatedAt = time.Now()
	return nil
}

func (u *User) Deactivate() {
	u.isActive = false
	u.updatedAt = time.Now()
}

func (u *User) Activate() {
	u.isActive = true
	u.updatedAt = time.Now()
}

func (u *User) RecordLogin() {
	now := time.Now()
	u.lastLogin = &now
	u.updatedAt = now
}

func (u *User) CanManageUser(target *User) bool {
	if !u.HasPermission(PermissionManageUsers) {
		return false
	}

	if u.IsSuperAdmin() {
		return true
	}

	if u.tenantID != target.tenantID {
		return false
	}

	if target.IsSuperAdmin() {
		return false
	}

	return true
}

func NormalizeRole(role Role) Role {
	switch role {
	case RoleSuperAdmin:
		return RoleSuperAdmin
	case RoleEmployee:
		return RoleEmployee
	case "tenant_admin":
		return RoleSuperAdmin
	case "analyst", "viewer":
		return RoleEmployee
	default:
		return role
	}
}

func NormalizeRoles(roles []Role) []Role {
	if len(roles) == 0 {
		return []Role{RoleEmployee}
	}

	out := make([]Role, 0, len(roles))
	seen := map[Role]struct{}{}
	for _, role := range roles {
		normalized := NormalizeRole(role)
		if normalized != RoleSuperAdmin && normalized != RoleEmployee {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}

	if len(out) == 0 {
		return []Role{RoleEmployee}
	}

	return out
}
