package messages

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"MRG/internal/domain/message"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoRepository struct {
	col *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) message.Repository {
	return &mongoRepository{col: db.Collection("messages")}
}

type messageDoc struct {
	ID                string     `bson:"_id"`
	TenantID          string     `bson:"tenant_id"`
	ChatID            int64      `bson:"chat_id"`
	ChatTitle         string     `bson:"chat_title"`
	MessageID         int64      `bson:"message_id"`
	SenderID          int64      `bson:"sender_id"`
	SenderName        string     `bson:"sender_name"`
	SenderUsername    string     `bson:"sender_username"`
	ChatPeerType      string     `bson:"chat_peer_type,omitempty"`
	IsScam            bool       `bson:"is_scam"`
	IsFake            bool       `bson:"is_fake"`
	IsPremium         bool       `bson:"is_premium"`
	Text              string     `bson:"text"`
	MediaType         string     `bson:"media_type"`
	CreatedAt         time.Time  `bson:"created_at"`
	IsIgnored         bool       `bson:"is_ignored,omitempty"`
	IsTeamMember      bool       `bson:"is_team_member,omitempty"`
	IsSpamSender      bool       `bson:"is_spam_sender,omitempty"`
	IsDM              bool       `bson:"is_dm,omitempty"`
	IsViewed          bool       `bson:"is_viewed,omitempty"`
	ViewedAt          *time.Time `bson:"viewed_at,omitempty"`
	SimilarityScore   *float64   `bson:"similarity_score,omitempty"`
	ClassifiedAsLead  *bool      `bson:"classified_as_lead,omitempty"`
	SemanticDirection *string    `bson:"semantic_direction,omitempty"`
}

func (r *mongoRepository) Save(ctx context.Context, m *message.Message) error {
	doc := toDoc(m)

	_, err := r.col.UpdateOne(ctx,
		bson.M{"tenant_id": m.TenantID(), "chat_id": m.ChatID(), "message_id": m.MessageID()},
		bson.M{"$set": doc, "$setOnInsert": bson.M{"_id": m.ID(), "created_at": m.CreatedAt()}},
		options.Update().SetUpsert(true),
	)
	return err
}

func (r *mongoRepository) DeleteNoise(ctx context.Context, tenantID string, olderThan time.Duration, ids []string) (int64, error) {
	threshold := time.Now().Add(-olderThan)
	filter := bson.M{
		"tenant_id":  tenantID,
		"created_at": bson.M{"$lt": threshold},
		"$or": bson.A{
			bson.M{"classified_as_lead": false},
			bson.M{"classified_as_lead": nil},
		},
	}
	if len(ids) > 0 {
		filter["_id"] = bson.M{"$in": ids}
	}
	res, err := r.col.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return res.DeletedCount, nil
}

func (r *mongoRepository) FindByID(ctx context.Context, tenantID, id string) (*message.Message, error) {
	var doc messageDoc
	if err := r.col.FindOne(ctx, bson.M{"_id": id, "tenant_id": tenantID}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return fromDoc(doc), nil
}

func (r *mongoRepository) FindByTelegramID(ctx context.Context, tenantID string, chatID, messageID int64) (*message.Message, error) {
	var doc messageDoc
	if err := r.col.FindOne(ctx, bson.M{"tenant_id": tenantID, "chat_id": chatID, "message_id": messageID}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return fromDoc(doc), nil
}

func (r *mongoRepository) FindByChat(ctx context.Context, tenantID string, chatID int64, limit, offset int) ([]*message.Message, error) {
	f := message.ListFilter{ChatID: &chatID, Limit: limit, Offset: offset}
	return r.List(ctx, tenantID, f)
}

func (r *mongoRepository) FindBySender(ctx context.Context, tenantID string, senderID int64, limit, offset int) ([]*message.Message, error) {
	f := message.ListFilter{SenderID: &senderID, Limit: limit, Offset: offset}
	return r.List(ctx, tenantID, f)
}

func (r *mongoRepository) FindUnclassified(ctx context.Context, tenantID string, limit int) ([]*message.Message, error) {
	f := message.ListFilter{Limit: limit}
	return r.List(ctx, tenantID, f)
}

func (r *mongoRepository) List(ctx context.Context, tenantID string, f message.ListFilter) ([]*message.Message, error) {
	page, err := r.ListPage(ctx, tenantID, f)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func (r *mongoRepository) ListPage(ctx context.Context, tenantID string, f message.ListFilter) (*message.ListPage, error) {
	filter := bson.M{"tenant_id": tenantID}
	if f.ChatID != nil {
		filter["chat_id"] = *f.ChatID
	}
	if f.SenderID != nil {
		filter["sender_id"] = *f.SenderID
	}
	if f.FromDate != nil || f.ToDate != nil {
		dateQ := bson.M{}
		if f.FromDate != nil {
			dateQ["$gte"] = *f.FromDate
		}
		if f.ToDate != nil {
			dateQ["$lte"] = *f.ToDate
		}
		filter["created_at"] = dateQ
	}
	if cursor, err := decodeMessageCursor(f.Cursor); err == nil && cursor != nil {
		filter["$or"] = bson.A{
			bson.M{"created_at": bson.M{"$lt": cursor.CreatedAt}},
			bson.M{
				"created_at": cursor.CreatedAt,
				"_id":        bson.M{"$lt": cursor.ID},
			},
		}
	} else if err != nil {
		return nil, err
	}

	limit := int64(50)
	if f.Limit > 0 {
		limit = int64(f.Limit)
	}
	opts := options.Find().
		SetLimit(limit + 1).
		SetSkip(int64(f.Offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}})

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var docs []messageDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*message.Message, len(docs))
	for i, d := range docs {
		out[i] = fromDoc(d)
	}
	nextCursor := ""
	if int64(len(out)) > limit {
		last := out[limit-1]
		out = out[:limit]
		nextCursor, err = encodeMessageCursor(last.ID(), last.CreatedAt())
		if err != nil {
			return nil, err
		}
	}
	return &message.ListPage{Items: out, NextCursor: nextCursor}, nil
}

type messageCursor struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

func encodeMessageCursor(id string, createdAt time.Time) (string, error) {
	payload, err := json.Marshal(messageCursor{ID: id, CreatedAt: createdAt.UTC()})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeMessageCursor(raw string) (*messageCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid message cursor")
	}
	var cursor messageCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, fmt.Errorf("invalid message cursor")
	}
	if cursor.ID == "" || cursor.CreatedAt.IsZero() {
		return nil, fmt.Errorf("invalid message cursor")
	}
	return &cursor, nil
}

func (r *mongoRepository) CountByTenantToday(ctx context.Context, tenantID string) (int64, error) {
	today := time.Now().UTC().Truncate(24 * time.Hour)
	return r.col.CountDocuments(ctx, bson.M{
		"tenant_id":  tenantID,
		"created_at": bson.M{"$gte": today},
	})
}

func (r *mongoRepository) ExistsByTelegramID(ctx context.Context, tenantID string, chatID, messageID int64) (bool, error) {
	n, err := r.col.CountDocuments(ctx, bson.M{
		"tenant_id":  tenantID,
		"chat_id":    chatID,
		"message_id": messageID,
	})
	return n > 0, err
}

func (r *mongoRepository) GetIngestStats(ctx context.Context, tenantID string, days int) (*message.IngestStats, error) {
	if days <= 0 {
		days = 7
	}
	now := time.Now().UTC()
	from := now.AddDate(0, 0, -days)
	dayStart := now.Truncate(24 * time.Hour)
	hourAgo := now.Add(-1 * time.Hour)

	base := bson.M{
		"tenant_id":  tenantID,
		"created_at": bson.M{"$gte": from},
	}

	total, err := r.col.CountDocuments(ctx, base)
	if err != nil {
		return nil, err
	}
	todayCount, err := r.col.CountDocuments(ctx, bson.M{
		"tenant_id":  tenantID,
		"created_at": bson.M{"$gte": dayStart},
	})
	if err != nil {
		return nil, err
	}
	lastHourCount, err := r.col.CountDocuments(ctx, bson.M{
		"tenant_id":  tenantID,
		"created_at": bson.M{"$gte": hourAgo},
	})
	if err != nil {
		return nil, err
	}
	leadCandidates, err := r.col.CountDocuments(ctx, bson.M{
		"tenant_id":          tenantID,
		"created_at":         bson.M{"$gte": from},
		"classified_as_lead": true,
		"is_ignored":         false,
		"similarity_score":   bson.M{"$gte": 0.70},
	})
	if err != nil {
		return nil, err
	}
	teamMessages, err := r.col.CountDocuments(ctx, bson.M{
		"tenant_id":      tenantID,
		"created_at":     bson.M{"$gte": from},
		"is_team_member": true,
	})
	if err != nil {
		return nil, err
	}
	ignoredMessages, err := r.col.CountDocuments(ctx, bson.M{
		"tenant_id":  tenantID,
		"created_at": bson.M{"$gte": from},
		"is_ignored": true,
	})
	if err != nil {
		return nil, err
	}

	uniqPipeline := mongo.Pipeline{
		{{Key: "$match", Value: base}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: nil},
			{Key: "chats", Value: bson.M{"$addToSet": "$chat_id"}},
			{Key: "senders", Value: bson.M{"$addToSet": "$sender_id"}},
		}}},
		{{Key: "$project", Value: bson.D{
			{Key: "unique_chats", Value: bson.M{"$size": "$chats"}},
			{Key: "unique_senders", Value: bson.M{"$size": "$senders"}},
		}}},
	}
	uniqCur, err := r.col.Aggregate(ctx, uniqPipeline)
	if err != nil {
		return nil, err
	}
	defer uniqCur.Close(ctx)
	uniqueChats := int64(0)
	uniqueSenders := int64(0)
	if uniqCur.Next(ctx) {
		var row struct {
			UniqueChats   int64 `bson:"unique_chats"`
			UniqueSenders int64 `bson:"unique_senders"`
		}
		if err := uniqCur.Decode(&row); err == nil {
			uniqueChats = row.UniqueChats
			uniqueSenders = row.UniqueSenders
		}
	}

	var lastDoc struct {
		CreatedAt time.Time `bson:"created_at"`
	}
	lastAt := (*time.Time)(nil)
	lastErr := r.col.FindOne(ctx, bson.M{"tenant_id": tenantID}, options.FindOne().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetProjection(bson.M{"created_at": 1}),
	).Decode(&lastDoc)
	if lastErr == nil {
		t := lastDoc.CreatedAt.UTC()
		lastAt = &t
	} else if lastErr != mongo.ErrNoDocuments {
		return nil, lastErr
	}

	hourlyPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"tenant_id":  tenantID,
			"created_at": bson.M{"$gte": now.Add(-24 * time.Hour)},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.M{
				"$dateToString": bson.M{
					"format": "%Y-%m-%d %H:00",
					"date":   "$created_at",
				},
			}},
			{Key: "count", Value: bson.M{"$sum": 1}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}
	hourlyCur, err := r.col.Aggregate(ctx, hourlyPipeline)
	if err != nil {
		return nil, err
	}
	defer hourlyCur.Close(ctx)
	hourly := make([]message.HourlyBucket, 0, 24)
	for hourlyCur.Next(ctx) {
		var row struct {
			ID    string `bson:"_id"`
			Count int64  `bson:"count"`
		}
		if err := hourlyCur.Decode(&row); err == nil {
			hourly = append(hourly, message.HourlyBucket{
				Hour:  row.ID,
				Count: row.Count,
			})
		}
	}

	topChatsPipeline := mongo.Pipeline{
		{{Key: "$match", Value: base}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "chat_id", Value: "$chat_id"},
				{Key: "chat_title", Value: "$chat_title"},
			}},
			{Key: "count", Value: bson.M{"$sum": 1}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "count", Value: -1}}}},
		{{Key: "$limit", Value: 5}},
	}
	topChatsCur, err := r.col.Aggregate(ctx, topChatsPipeline)
	if err != nil {
		return nil, err
	}
	defer topChatsCur.Close(ctx)
	topChats := make([]message.ChatBucket, 0, 5)
	for topChatsCur.Next(ctx) {
		var row struct {
			ID struct {
				ChatID    int64  `bson:"chat_id"`
				ChatTitle string `bson:"chat_title"`
			} `bson:"_id"`
			Count int64 `bson:"count"`
		}
		if err := topChatsCur.Decode(&row); err == nil {
			topChats = append(topChats, message.ChatBucket{
				ChatID:    row.ID.ChatID,
				ChatTitle: row.ID.ChatTitle,
				Count:     row.Count,
			})
		}
	}

	avgPerHour := 0.0
	if days > 0 {
		avgPerHour = float64(total) / float64(days*24)
	}

	stats := &message.IngestStats{
		Period:          fmt.Sprintf("%dd", days),
		TotalSignals:    total,
		SignalsToday:    todayCount,
		SignalsLastHour: lastHourCount,
		AvgPerHour:      avgPerHour,
		UniqueChats:     uniqueChats,
		UniqueSenders:   uniqueSenders,
		LeadCandidates:  leadCandidates,
		TeamMessages:    teamMessages,
		IgnoredMessages: ignoredMessages,
		LastSignalAt:    lastAt,
		Hourly:          hourly,
		TopChats:        topChats,
	}
	return stats, nil
}

func (r *mongoRepository) SetFlag(ctx context.Context, tenantID, id, field string, value bool) error {
	if field != "is_ignored" && field != "is_team_member" && field != "is_viewed" && field != "is_spam_sender" {
		return errors.New("unknown flag field: " + field)
	}
	set := bson.M{field: value}
	if field == "is_viewed" {
		if value {
			set["viewed_at"] = time.Now().UTC()
		} else {
			set["viewed_at"] = nil
		}
	}
	_, err := r.col.UpdateOne(ctx,
		bson.M{"_id": id, "tenant_id": tenantID},
		bson.M{"$set": set},
	)
	return err
}

func (r *mongoRepository) SetClassification(ctx context.Context, tenantID, id string, isLead bool) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"_id": id, "tenant_id": tenantID},
		bson.M{"$set": bson.M{
			"classified_as_lead": isLead,
			"updated_at":         time.Now().UTC(),
		}},
	)
	return err
}

func (r *mongoRepository) SetSemanticDirection(ctx context.Context, tenantID, id string, direction *string) error {
	set := bson.M{
		"updated_at": time.Now().UTC(),
	}
	if direction == nil {
		set["semantic_direction"] = nil
	} else {
		set["semantic_direction"] = *direction
	}

	_, err := r.col.UpdateOne(ctx,
		bson.M{"_id": id, "tenant_id": tenantID},
		bson.M{"$set": set},
	)
	return err
}

func (r *mongoRepository) CountSenderInChats(ctx context.Context, tenantID string, senderIDs []int64) (map[int64]int, error) {
	if len(senderIDs) == 0 {
		return map[int64]int{}, nil
	}
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{"tenant_id": tenantID, "sender_id": bson.M{"$in": senderIDs}}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$sender_id"},
			{Key: "chats", Value: bson.M{"$addToSet": "$chat_id"}},
		}}},
		{{Key: "$project", Value: bson.M{
			"count": bson.M{"$size": "$chats"},
		}}},
	}
	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	result := make(map[int64]int, len(senderIDs))
	for cur.Next(ctx) {
		var row struct {
			ID    int64 `bson:"_id"`
			Count int   `bson:"count"`
		}
		if err := cur.Decode(&row); err == nil {
			result[row.ID] = row.Count
		}
	}
	return result, nil
}

func (r *mongoRepository) GetChartData(ctx context.Context, tenantID string, from, to time.Time) ([]message.ChartDayBucket, error) {
	pipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"tenant_id":  tenantID,
			"is_ignored": false,
			"created_at": bson.M{"$gte": from, "$lte": to},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.M{
				"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$created_at"},
			}},
			{Key: "total", Value: bson.M{"$sum": 1}},
			{Key: "target", Value: bson.M{"$sum": bson.M{
				"$cond": bson.A{
					bson.M{"$eq": bson.A{"$classified_as_lead", true}},
					1, 0,
				},
			}}},
			{Key: "trader_search", Value: bson.M{"$sum": bson.M{
				"$cond": bson.A{
					bson.M{"$in": bson.A{"$semantic_direction", bson.A{"trader_search", "search_trader", "search_traders"}}},
					1, 0,
				},
			}}},
			{Key: "traders", Value: bson.M{"$sum": bson.M{
				"$cond": bson.A{
					bson.M{"$in": bson.A{"$semantic_direction", bson.A{"trader", "traders"}}},
					1, 0,
				},
			}}},
			{Key: "merchants", Value: bson.M{"$sum": bson.M{
				"$cond": bson.A{
					bson.M{"$in": bson.A{"$semantic_direction", bson.A{"merchant", "merchants", "processing_request", "processing_requests", "request_processing"}}},
					1, 0,
				},
			}}},
			{Key: "ps_offers", Value: bson.M{"$sum": bson.M{
				"$cond": bson.A{
					bson.M{"$in": bson.A{"$semantic_direction", bson.A{"ps_offer", "ps_offers", "offer", "offers"}}},
					1, 0,
				},
			}}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id", Value: 1}}}},
	}

	cur, err := r.col.Aggregate(ctx, pipeline)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	buckets := make([]message.ChartDayBucket, 0, 30)
	for cur.Next(ctx) {
		var row struct {
			ID           string `bson:"_id"`
			Total        int64  `bson:"total"`
			Target       int64  `bson:"target"`
			TraderSearch int64  `bson:"trader_search"`
			Traders      int64  `bson:"traders"`
			Merchants    int64  `bson:"merchants"`
			PSOffers     int64  `bson:"ps_offers"`
		}
		if err := cur.Decode(&row); err == nil {
			buckets = append(buckets, message.ChartDayBucket{
				Day:          row.ID,
				Total:        row.Total,
				Target:       row.Target,
				TraderSearch: row.TraderSearch,
				Traders:      row.Traders,
				Merchants:    row.Merchants,
				PSOffers:     row.PSOffers,
			})
		}
	}
	return buckets, nil
}

func (r *mongoRepository) EnsureIndexes(ctx context.Context) error {
	indices := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "tenant_id", Value: 1}, {Key: "chat_id", Value: 1}, {Key: "message_id", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "created_at", Value: -1}, {Key: "_id", Value: -1}},
		},
		{
			Keys: bson.D{{Key: "tenant_id", Value: 1}, {Key: "sender_id", Value: 1}},
		},
		{
			Keys: bson.D{
				{Key: "tenant_id", Value: 1},
				{Key: "classified_as_lead", Value: 1},
				{Key: "is_ignored", Value: 1},
				{Key: "similarity_score", Value: 1},
			},
		},
	}
	_, err := r.col.Indexes().CreateMany(ctx, indices)
	return err
}

func toDoc(m *message.Message) bson.M {
	return bson.M{
		"tenant_id":          m.TenantID(),
		"chat_id":            m.ChatID(),
		"chat_title":         m.ChatTitle(),
		"message_id":         m.MessageID(),
		"sender_id":          m.SenderID(),
		"sender_name":        m.SenderName(),
		"sender_username":    m.SenderUsername(),
		"chat_peer_type":     m.ChatPeerType(),
		"is_scam":            m.IsScam(),
		"is_fake":            m.IsFake(),
		"is_premium":         m.IsPremium(),
		"text":               m.Text(),
		"media_type":         string(m.MediaType()),
		"is_ignored":         m.IsIgnored(),
		"is_team_member":     m.IsTeamMember(),
		"is_spam_sender":     m.IsSpamSender(),
		"is_dm":              m.IsDM(),
		"is_viewed":          m.IsViewed(),
		"viewed_at":          m.ViewedAt(),
		"similarity_score":   m.SimilarityScore(),
		"classified_as_lead": m.ClassifiedAsLead(),
		"semantic_direction": m.SemanticDirection(),
	}
}

func fromDoc(d messageDoc) *message.Message {
	m := message.Restore(
		d.ID, d.TenantID,
		d.ChatID, d.ChatTitle,
		d.MessageID, d.SenderID,
		d.SenderName, d.SenderUsername,
		d.Text,
		message.MediaType(d.MediaType),
		d.CreatedAt,
		d.IsIgnored, d.IsTeamMember, d.IsSpamSender, d.IsDM, d.IsViewed, d.ViewedAt,
		d.SimilarityScore, d.ClassifiedAsLead, d.SemanticDirection,
		message.Metadata{},
	)
	m.SetChatPeerType(d.ChatPeerType)
	m.SetSenderTrust(d.IsScam, d.IsFake, d.IsPremium)
	return m
}
