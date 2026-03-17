package contacts

import (
	"context"
	"time"

	"MRG/internal/domain/contact"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoRepository struct {
	col *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) contact.Repository {
	return &mongoRepository{col: db.Collection("contacts")}
}

func (r *mongoRepository) Save(ctx context.Context, c *contact.Contact) error {
	doc := toDoc(c)
	_, err := r.col.InsertOne(ctx, doc)
	return err
}

func (r *mongoRepository) Update(ctx context.Context, c *contact.Contact) error {
	filter := bson.M{"tenant_id": c.TenantID(), "sender_id": c.SenderID()}
	_, err := r.col.UpdateOne(ctx, filter, bson.M{"$set": toDoc(c)}, options.Update().SetUpsert(true))
	return err
}

func (r *mongoRepository) FindBySenderID(ctx context.Context, tenantID string, senderID int64) (*contact.Contact, error) {
	var doc contactDoc
	filter := bson.M{"tenant_id": tenantID, "sender_id": senderID}
	if err := r.col.FindOne(ctx, filter).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return fromDoc(doc), nil
}

type contactDoc struct {
	TenantID       string    `bson:"tenant_id"`
	SenderID       int64     `bson:"sender_id"`
	SenderName     string    `bson:"sender_name"`
	SenderUsername string    `bson:"sender_username"`
	MerchantID     string    `bson:"merchant_id"`
	IsTeamMember   bool      `bson:"is_team_member"`
	IsSpam         bool      `bson:"is_spam,omitempty"`
	CreatedAt      time.Time `bson:"created_at"`
	UpdatedAt      time.Time `bson:"updated_at"`
}

func toDoc(c *contact.Contact) contactDoc {
	return contactDoc{
		TenantID:       c.TenantID(),
		SenderID:       c.SenderID(),
		SenderName:     c.SenderName(),
		SenderUsername: c.SenderUsername(),
		MerchantID:     c.MerchantID(),
		IsTeamMember:   c.IsTeamMember(),
		IsSpam:         c.IsSpam(),
		CreatedAt:      c.CreatedAt(),
		UpdatedAt:      c.UpdatedAt(),
	}
}

func fromDoc(d contactDoc) *contact.Contact {
	return contact.Restore(
		d.TenantID,
		d.SenderID,
		d.SenderName,
		d.SenderUsername,
		d.MerchantID,
		d.IsTeamMember,
		d.IsSpam,
		d.CreatedAt,
		d.UpdatedAt,
	)
}
