package users

import "MRG/internal/domain/user"

func toDomain(dbUser *userDB) *user.User {
	return user.Restore(
		dbUser.ID,
		dbUser.TenantID,
		dbUser.Email,
		dbUser.Password,
		dbUser.Name,
		user.NormalizeRoles(dbUser.Roles),
		dbUser.IsActive,
		dbUser.TelegramUserID,
		dbUser.TelegramUsername,
		dbUser.CreatedAt,
		dbUser.UpdatedAt,
		dbUser.LastLogin,
	)
}

func fromDomain(domainUser *user.User) *userDB {
	return &userDB{
		ID:               domainUser.ID(),
		TenantID:         domainUser.TenantID(),
		Email:            domainUser.Email(),
		Password:         domainUser.Password(),
		Name:             domainUser.Name(),
		Roles:            domainUser.Roles(),
		IsActive:         domainUser.IsActive(),
		TelegramUserID:   domainUser.TelegramUserID(),
		TelegramUsername: domainUser.TelegramUsername(),
		CreatedAt:        domainUser.CreatedAt(),
		UpdatedAt:        domainUser.UpdatedAt(),
		LastLogin:        domainUser.LastLogin(),
	}
}

func toInviteDomain(dbInvite *inviteDB) *user.Invite {
	return user.RestoreInvite(
		dbInvite.ID,
		dbInvite.TenantID,
		user.NormalizeRole(dbInvite.Role),
		dbInvite.TokenHash,
		dbInvite.CreatedBy,
		dbInvite.CreatedAt,
		dbInvite.ExpiresAt,
		dbInvite.UsedAt,
		dbInvite.UsedBy,
	)
}

func fromInviteDomain(invite *user.Invite) *inviteDB {
	return &inviteDB{
		ID:        invite.ID(),
		TenantID:  invite.TenantID(),
		Role:      user.NormalizeRole(invite.Role()),
		TokenHash: invite.TokenHash(),
		CreatedBy: invite.CreatedBy(),
		CreatedAt: invite.CreatedAt(),
		ExpiresAt: invite.ExpiresAt(),
		UsedAt:    invite.UsedAt(),
		UsedBy:    invite.UsedBy(),
	}
}
