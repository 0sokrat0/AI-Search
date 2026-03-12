package settings

import (
	"context"
	"maps"
	"strconv"
	"sync"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const collection = "settings"

var defaults = map[string]string{
	"lead_threshold":        "0.70",
	"sender_window_seconds": "60",
	"trader_threshold":      "0.60",
	"merchant_threshold":    "0.60",
	"ps_offer_threshold":    "0.60",
}

type settingDoc struct {
	Key   string `bson:"_id"`
	Value string `bson:"value"`
}

type Store struct {
	col *mongo.Collection

	mu        sync.RWMutex
	cache     map[string]string
	cacheTime time.Time
	cacheTTL  time.Duration
}

func New(db *mongo.Database) *Store {
	return &Store{
		col:      db.Collection(collection),
		cache:    make(map[string]string),
		cacheTTL: 30 * time.Second,
	}
}

func (s *Store) GetAll(ctx context.Context) (map[string]string, error) {
	if err := s.warmUp(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make(map[string]string, len(defaults))
	maps.Copy(out, defaults)
	maps.Copy(out, s.cache)
	return out, nil
}

func (s *Store) SetAll(ctx context.Context, kv map[string]string) error {
	for k, v := range kv {
		filter := bson.M{"_id": k}
		update := bson.M{"$set": bson.M{"value": v}}
		_, err := s.col.UpdateOne(ctx, filter, update, options.Update().SetUpsert(true))
		if err != nil {
			return err
		}
	}
	s.mu.Lock()
	s.cacheTime = time.Time{}
	s.mu.Unlock()
	return nil
}

func (s *Store) GetFloat(ctx context.Context, key string, def float64) float64 {
	_ = s.warmUp(ctx)
	s.mu.RLock()
	v, ok := s.cache[key]
	s.mu.RUnlock()
	if !ok {
		if dv, ok2 := defaults[key]; ok2 {
			v = dv
		} else {
			return def
		}
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func (s *Store) GetString(ctx context.Context, key, def string) string {
	_ = s.warmUp(ctx)
	s.mu.RLock()
	v, ok := s.cache[key]
	s.mu.RUnlock()
	if !ok {
		if dv, ok2 := defaults[key]; ok2 {
			return dv
		}
		return def
	}
	return v
}

func (s *Store) warmUp(ctx context.Context) error {
	s.mu.RLock()
	fresh := time.Since(s.cacheTime) < s.cacheTTL
	s.mu.RUnlock()
	if fresh {
		return nil
	}

	cur, err := s.col.Find(ctx, bson.M{})
	if err != nil {
		return err
	}
	defer cur.Close(ctx)

	var docs []settingDoc
	if err := cur.All(ctx, &docs); err != nil {
		return err
	}

	s.mu.Lock()
	s.cache = make(map[string]string, len(docs))
	for _, d := range docs {
		s.cache[d.Key] = d.Value
	}
	s.cacheTime = time.Now()
	s.mu.Unlock()
	return nil
}
