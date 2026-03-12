package telegram

import (
	"context"
	"fmt"

	"github.com/gotd/td/session"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoSessionStorage struct {
	db    *mongo.Database
	phone string
}

func NewMongoSessionStorage(db *mongo.Database, phone string) *MongoSessionStorage {
	return &MongoSessionStorage{
		db:    db,
		phone: phone,
	}
}

func (s *MongoSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	col := s.db.Collection("messenger_accounts")
	var acc AccountConfig
	err := col.FindOne(ctx, bson.M{"phone": s.phone}).Decode(&acc)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, session.ErrNotFound
		}
		return nil, fmt.Errorf("failed to load session: %w", err)
	}

	if len(acc.SessionData) == 0 {
		return nil, session.ErrNotFound
	}

	return acc.SessionData, nil
}

func (s *MongoSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	col := s.db.Collection("messenger_accounts")
	_, err := col.UpdateOne(ctx, bson.M{"phone": s.phone}, bson.M{
		"$set": bson.M{
			"session_data": data,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to store session: %w", err)
	}
	return nil
}
