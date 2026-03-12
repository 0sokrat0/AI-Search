package users

import (
	"context"
	"time"

	"MRG/internal/domain/user"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoRepository struct {
	collection       *mongo.Collection
	inviteCollection *mongo.Collection
}

type userDB struct {
	ID               string      `bson:"id"`
	TenantID         string      `bson:"tenant_id"`
	Email            string      `bson:"email"`
	Password         string      `bson:"password"`
	Name             string      `bson:"name"`
	Roles            []user.Role `bson:"roles"`
	IsActive         bool        `bson:"is_active"`
	TelegramUserID   *int64      `bson:"telegram_user_id,omitempty"`
	TelegramUsername string      `bson:"telegram_username,omitempty"`
	CreatedAt        time.Time   `bson:"created_at"`
	UpdatedAt        time.Time   `bson:"updated_at"`
	LastLogin        *time.Time  `bson:"last_login,omitempty"`
}

type inviteDB struct {
	ID        string     `bson:"id"`
	TenantID  string     `bson:"tenant_id"`
	Role      user.Role  `bson:"role"`
	TokenHash string     `bson:"token_hash"`
	CreatedBy string     `bson:"created_by"`
	CreatedAt time.Time  `bson:"created_at"`
	ExpiresAt time.Time  `bson:"expires_at"`
	UsedAt    *time.Time `bson:"used_at,omitempty"`
	UsedBy    string     `bson:"used_by,omitempty"`
}

func NewMongoRepository(db *mongo.Database) user.Repository {
	return &mongoRepository{
		collection:       db.Collection("users"),
		inviteCollection: db.Collection("user_invites"),
	}
}

func (r *mongoRepository) Create(ctx context.Context, u *user.User) error {
	dbUser := fromDomain(u)
	_, err := r.collection.InsertOne(ctx, dbUser)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return user.ErrEmailAlreadyExists
		}
		return err
	}
	return nil
}

func (r *mongoRepository) Update(ctx context.Context, u *user.User) error {
	dbUser := fromDomain(u)
	filter := bson.M{"id": dbUser.ID}
	update := bson.M{"$set": dbUser}
	_, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	return nil
}

func (r *mongoRepository) Delete(ctx context.Context, id string) error {
	filter := bson.M{"id": id}
	_, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	return nil
}

func (r *mongoRepository) FindByID(ctx context.Context, id string) (*user.User, error) {
	var dbUser userDB
	filter := bson.M{"id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&dbUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toDomain(&dbUser), nil
}

func (r *mongoRepository) FindByEmail(ctx context.Context, email string) (*user.User, error) {
	var dbUser userDB
	filter := bson.M{"email": email}
	err := r.collection.FindOne(ctx, filter).Decode(&dbUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, user.ErrUserNotFound
		}
		return nil, err
	}
	return toDomain(&dbUser), nil
}

func (r *mongoRepository) FindByTelegramID(ctx context.Context, tenantID string, telegramUserID int64) (*user.User, error) {
	var dbUser userDB
	filter := bson.M{"tenant_id": tenantID, "telegram_user_id": telegramUserID}
	err := r.collection.FindOne(ctx, filter).Decode(&dbUser)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return toDomain(&dbUser), nil
}

func (r *mongoRepository) IsTeamMember(ctx context.Context, tenantID string, telegramUserID int64) (bool, error) {
	u, err := r.FindByTelegramID(ctx, tenantID, telegramUserID)
	if err != nil || u == nil {
		return false, err
	}
	return u.IsActive(), nil
}

func (r *mongoRepository) FindByTenantID(ctx context.Context, tenantID string, limit, offset int) ([]*user.User, error) {
	var users []*user.User
	filter := bson.M{"tenant_id": tenantID}
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset))
	cursor, err := r.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	for cursor.Next(ctx) {
		var dbUser userDB
		if err := cursor.Decode(&dbUser); err != nil {
			return nil, err
		}
		users = append(users, toDomain(&dbUser))
	}

	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func (r *mongoRepository) CountByTenantID(ctx context.Context, tenantID string) (int64, error) {
	filter := bson.M{"tenant_id": tenantID}
	return r.collection.CountDocuments(ctx, filter)
}

func (r *mongoRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	count, err := r.collection.CountDocuments(ctx, bson.M{"email": email})
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (r *mongoRepository) CreateInvite(ctx context.Context, invite *user.Invite) error {
	_, err := r.inviteCollection.InsertOne(ctx, fromInviteDomain(invite))
	return err
}

func (r *mongoRepository) FindInviteByTokenHash(ctx context.Context, tokenHash string) (*user.Invite, error) {
	var dbInvite inviteDB
	err := r.inviteCollection.FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&dbInvite)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, user.ErrInviteNotFound
		}
		return nil, err
	}
	return toInviteDomain(&dbInvite), nil
}

func (r *mongoRepository) UpdateInvite(ctx context.Context, invite *user.Invite) error {
	dbInvite := fromInviteDomain(invite)
	_, err := r.inviteCollection.UpdateOne(ctx, bson.M{"id": dbInvite.ID}, bson.M{"$set": dbInvite})
	return err
}
