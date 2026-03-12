package user

import "errors"

var (
	ErrUserNotFound = errors.New("user not found")

	ErrEmailAlreadyExists = errors.New("email already exists")

	ErrEmptyEmail = errors.New("email cannot be empty")

	ErrEmptyName = errors.New("name cannot be empty")

	ErrInvalidTenantID = errors.New("invalid tenant ID")

	ErrInvalidPassword = errors.New("invalid password")

	ErrPasswordTooShort = errors.New("password must be at least 8 characters")

	ErrCannotRemoveSuperAdmin = errors.New("cannot remove super admin role")

	ErrUserMustHaveRole = errors.New("user must have at least one role")

	ErrUnauthorized = errors.New("unauthorized")

	ErrUserInactive = errors.New("user account is inactive")

	ErrInvalidRole = errors.New("invalid role")

	ErrInviteNotFound = errors.New("invite not found")

	ErrInviteExpired = errors.New("invite expired")

	ErrInviteUsed = errors.New("invite already used")
)
