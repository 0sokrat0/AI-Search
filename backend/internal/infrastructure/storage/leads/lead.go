package leads

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"MRG/internal/domain/lead"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type mongoRepository struct {
	col *mongo.Collection
}

func NewMongoRepository(db *mongo.Database) lead.Repository {
	return &mongoRepository{col: db.Collection("leads")}
}

func (r *mongoRepository) Save(ctx context.Context, l *lead.Lead) error {
	doc := toDoc(l)
	doc["created_at"] = l.CreatedAt()
	_, err := r.col.InsertOne(ctx, doc)
	return err
}

func (r *mongoRepository) Update(ctx context.Context, l *lead.Lead) error {
	filter := bson.M{"_id": l.ID(), "tenant_id": l.TenantID()}
	_, err := r.col.UpdateOne(ctx, filter, bson.M{"$set": toDoc(l)})
	return err
}

func (r *mongoRepository) DeleteByID(ctx context.Context, tenantID, id string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"_id": id, "tenant_id": tenantID})
	return err
}

func (r *mongoRepository) DeleteByMessageID(ctx context.Context, tenantID, messageID string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"tenant_id": tenantID, "message_id": messageID})
	return err
}

func (r *mongoRepository) FindByID(ctx context.Context, tenantID, id string) (*lead.Lead, error) {
	var doc leadDoc
	if err := r.col.FindOne(ctx, bson.M{"_id": id, "tenant_id": tenantID}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return fromDoc(doc), nil
}

func (r *mongoRepository) FindByMessageID(ctx context.Context, tenantID, messageID string) (*lead.Lead, error) {
	var doc leadDoc
	if err := r.col.FindOne(ctx, bson.M{"tenant_id": tenantID, "message_id": messageID}).Decode(&doc); err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil
		}
		return nil, err
	}
	return fromDoc(doc), nil
}

func (r *mongoRepository) FindByMessageIDs(ctx context.Context, tenantID string, messageIDs []string) (map[string]lead.MessageLeadRef, error) {
	if len(messageIDs) == 0 {
		return map[string]lead.MessageLeadRef{}, nil
	}
	type miniDoc struct {
		ID        string  `bson:"_id"`
		MessageID string  `bson:"message_id"`
		Score     float64 `bson:"score"`
	}
	opts := options.Find().SetProjection(bson.M{"_id": 1, "message_id": 1, "score": 1})
	cur, err := r.col.Find(ctx, bson.M{
		"tenant_id":  tenantID,
		"message_id": bson.M{"$in": messageIDs},
	}, opts)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)
	var docs []miniDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	result := make(map[string]lead.MessageLeadRef, len(docs))
	for _, d := range docs {
		result[d.MessageID] = lead.MessageLeadRef{ID: d.ID, Score: d.Score}
	}
	return result, nil
}

func (r *mongoRepository) FindBySender(ctx context.Context, tenantID string, senderID int64, limit, offset int) ([]*lead.Lead, error) {
	opts := options.Find().SetLimit(int64(limit)).SetSkip(int64(offset)).SetSort(bson.D{{Key: "created_at", Value: -1}})
	cur, err := r.col.Find(ctx, bson.M{"tenant_id": tenantID, "sender_id": senderID}, opts)
	if err != nil {
		return nil, err
	}
	defer func(cur *mongo.Cursor, ctx context.Context) {
		err := cur.Close(ctx)
		if err != nil {

		}
	}(cur, ctx)
	return decodeCursor(ctx, cur)
}

func (r *mongoRepository) List(ctx context.Context, tenantID string, f lead.ListFilter) ([]*lead.Lead, error) {
	page, err := r.ListPage(ctx, tenantID, f)
	if err != nil {
		return nil, err
	}
	return page.Items, nil
}

func (r *mongoRepository) ListPage(ctx context.Context, tenantID string, f lead.ListFilter) (*lead.ListPage, error) {
	filter := buildFilter(tenantID, f)
	limit := int64(50)
	if f.Limit > 0 {
		limit = int64(f.Limit)
	}
	if cursor, err := decodeLeadCursor(f.Cursor); err == nil && cursor != nil {
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
	opts := options.Find().
		SetLimit(limit + 1).
		SetSkip(int64(f.Offset)).
		SetSort(bson.D{{Key: "created_at", Value: -1}, {Key: "_id", Value: -1}})

	cur, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	defer func(cur *mongo.Cursor, ctx context.Context) {
		err := cur.Close(ctx)
		if err != nil {

		}
	}(cur, ctx)
	items, err := decodeCursor(ctx, cur)
	if err != nil {
		return nil, err
	}
	nextCursor := ""
	if int64(len(items)) > limit {
		last := items[limit-1]
		items = items[:limit]
		nextCursor, err = encodeLeadCursor(last.ID(), last.CreatedAt())
		if err != nil {
			return nil, err
		}
	}
	return &lead.ListPage{Items: items, NextCursor: nextCursor}, nil
}

type leadCursor struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
}

func encodeLeadCursor(id string, createdAt time.Time) (string, error) {
	payload, err := json.Marshal(leadCursor{ID: id, CreatedAt: createdAt.UTC()})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(payload), nil
}

func decodeLeadCursor(raw string) (*leadCursor, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	payload, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid lead cursor")
	}
	var cursor leadCursor
	if err := json.Unmarshal(payload, &cursor); err != nil {
		return nil, fmt.Errorf("invalid lead cursor")
	}
	if cursor.ID == "" || cursor.CreatedAt.IsZero() {
		return nil, fmt.Errorf("invalid lead cursor")
	}
	return &cursor, nil
}

func (r *mongoRepository) Count(ctx context.Context, tenantID string, f lead.ListFilter) (int64, error) {
	return r.col.CountDocuments(ctx, buildFilter(tenantID, f))
}

var scoreBoundaries = []float64{0.00, 0.50, 0.60, 0.65, 0.70, 0.75, 0.80, 0.85, 0.90, 1.01}

func (r *mongoRepository) GetStats(ctx context.Context, tenantID string, days int) (*lead.Stats, error) {
	from := time.Now().UTC().AddDate(0, 0, -days)
	match := bson.M{
		"tenant_id":  tenantID,
		"created_at": bson.M{"$gte": from},
		"status":     bson.M{"$ne": string(lead.StatusControversial)},
	}

	feedbackPipeline := mongo.Pipeline{
		{{Key: "$match", Value: match}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "feedback", Value: "$user_feedback"},
				{Key: "category", Value: "$semantic_category"},
				{Key: "qualification_source", Value: "$qualification_source"},
			}},
			{Key: "count", Value: bson.M{"$sum": 1}},
			{Key: "avg_score", Value: bson.M{"$avg": "$score"}},
		}}},
	}
	fcur, err := r.col.Aggregate(ctx, feedbackPipeline)
	if err != nil {
		return nil, err
	}
	defer func(fcur *mongo.Cursor, ctx context.Context) {
		err := fcur.Close(ctx)
		if err != nil {

		}
	}(fcur, ctx)

	type feedbackID struct {
		Feedback            *bool  `bson:"feedback"`
		Category            string `bson:"category"`
		QualificationSource string `bson:"qualification_source"`
	}
	type feedbackRow struct {
		ID       feedbackID `bson:"_id"`
		Count    int64      `bson:"count"`
		AvgScore float64    `bson:"avg_score"`
	}
	var frows []feedbackRow
	if err := fcur.All(ctx, &frows); err != nil {
		return nil, err
	}

	stats := &lead.Stats{Period: fmt.Sprintf("%dd", days)}
	var totalScore float64
	var totalCount int64
	for _, row := range frows {
		stats.TotalDetected += row.Count
		totalScore += row.AvgScore * float64(row.Count)
		totalCount += row.Count

		switch row.ID.QualificationSource {
		case string(lead.QualificationSourceAI):
			stats.AIQualified += row.Count
		case string(lead.QualificationSourceManual):
			stats.ManualApproved += row.Count
		}

		if row.ID.Category == "noise" || row.ID.Category == "spam" {
			stats.Rejected += row.Count
			// We can use the average score of noise for AvgScoreRejected
			if stats.AvgScoreRejected == 0 {
				stats.AvgScoreRejected = row.AvgScore
			} else {
				// simple weighted average for rejected
				stats.AvgScoreRejected = (stats.AvgScoreRejected*float64(stats.Rejected-row.Count) + row.AvgScore*float64(row.Count)) / float64(stats.Rejected)
			}
			continue
		}

		if row.ID.Feedback == nil {
			stats.Pending += row.Count
		} else if *row.ID.Feedback {
			stats.Approved += row.Count
			stats.AvgScoreApproved = row.AvgScore
			accumulateCategoryDistribution(&stats.ApprovedByCategory, row.ID.Category, row.Count)
		} else {
			stats.Rejected += row.Count
			stats.AvgScoreRejected = row.AvgScore
			accumulateCategoryDistribution(&stats.RejectedByCategory, row.ID.Category, row.Count)
		}
	}
	if totalCount > 0 {
		stats.AvgScore = totalScore / float64(totalCount)
	}

	bucketPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"tenant_id":  tenantID,
			"created_at": bson.M{"$gte": from},
			"score": bson.M{
				"$type": "number",
			},
		}}},
		{{Key: "$bucket", Value: bson.D{
			{Key: "groupBy", Value: "$score"},
			{Key: "boundaries", Value: scoreBoundaries},
			{Key: "default", Value: "other"},
			{Key: "output", Value: bson.D{
				{Key: "count", Value: bson.M{"$sum": 1}},
				{Key: "approved", Value: bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$user_feedback", true}}, 1, 0}}}},
				{Key: "rejected", Value: bson.M{"$sum": bson.M{"$cond": bson.A{bson.M{"$eq": bson.A{"$user_feedback", false}}, 1, 0}}}},
			}},
		}}},
	}
	bcur, err := r.col.Aggregate(ctx, bucketPipeline)
	if err != nil {
		return nil, err
	}
	defer func(bcur *mongo.Cursor, ctx context.Context) {
		err := bcur.Close(ctx)
		if err != nil {

		}
	}(bcur, ctx)

	type bucketRow struct {
		ID       any   `bson:"_id"`
		Count    int64 `bson:"count"`
		Approved int64 `bson:"approved"`
		Rejected int64 `bson:"rejected"`
	}
	var brows []bucketRow
	if err := bcur.All(ctx, &brows); err != nil {
		return nil, err
	}
	bucketMap := make(map[float64]bucketRow, len(brows))
	for _, b := range brows {
		id, ok := asFloat64(b.ID)
		if !ok {
			continue
		}
		bucketMap[id] = b
	}
	for i := 0; i < len(scoreBoundaries)-1; i++ {
		from := scoreBoundaries[i]
		to := scoreBoundaries[i+1]
		b := bucketMap[from]
		stats.Buckets = append(stats.Buckets, lead.ScoreBucket{
			From: from, To: to,
			Count: b.Count, Approved: b.Approved, Rejected: b.Rejected,
		})
	}

	seriesPipeline := mongo.Pipeline{
		{{Key: "$match", Value: bson.M{
			"tenant_id":         tenantID,
			"created_at":        bson.M{"$gte": from},
			"status":            bson.M{"$ne": string(lead.StatusControversial)},
			"semantic_category": bson.M{"$in": bson.A{"traders", "merchants", "ps_offers"}},
			"qualification_source": bson.M{"$in": bson.A{
				string(lead.QualificationSourceAI),
				string(lead.QualificationSourceManual),
			}},
		}}},
		{{Key: "$group", Value: bson.D{
			{Key: "_id", Value: bson.D{
				{Key: "day", Value: bson.M{
					"$dateToString": bson.M{"format": "%Y-%m-%d", "date": "$created_at"},
				}},
				{Key: "category", Value: "$semantic_category"},
			}},
			{Key: "count", Value: bson.M{"$sum": 1}},
		}}},
		{{Key: "$sort", Value: bson.D{{Key: "_id.day", Value: 1}}}},
	}
	scur, err := r.col.Aggregate(ctx, seriesPipeline)
	if err != nil {
		return nil, err
	}
	defer func(scur *mongo.Cursor, ctx context.Context) {
		err := scur.Close(ctx)
		if err != nil {
		}
	}(scur, ctx)

	type seriesID struct {
		Day      string `bson:"day"`
		Category string `bson:"category"`
	}
	type seriesRow struct {
		ID    seriesID `bson:"_id"`
		Count int64    `bson:"count"`
	}
	var srows []seriesRow
	if err := scur.All(ctx, &srows); err != nil {
		return nil, err
	}

	seriesMap := make(map[string]*lead.CategorySeriesBucket, len(srows))
	for _, row := range srows {
		bucket := seriesMap[row.ID.Day]
		if bucket == nil {
			bucket = &lead.CategorySeriesBucket{Day: row.ID.Day}
			seriesMap[row.ID.Day] = bucket
		}
		switch row.ID.Category {
		case "traders":
			bucket.Traders += row.Count
		case "merchants":
			bucket.Merchants += row.Count
		case "ps_offers":
			bucket.PSOffers += row.Count
		}
	}

	for day := from; !day.After(time.Now().UTC()); day = day.AddDate(0, 0, 1) {
		key := day.Format("2006-01-02")
		if bucket, ok := seriesMap[key]; ok {
			stats.Series = append(stats.Series, *bucket)
			continue
		}
		stats.Series = append(stats.Series, lead.CategorySeriesBucket{Day: key})
	}
	return stats, nil
}

func accumulateCategoryDistribution(target *lead.CategoryDistribution, category string, count int64) {
	switch category {
	case "traders":
		target.Traders += count
	case "merchants":
		target.Merchants += count
	case "ps_offers":
		target.PSOffers += count
	}
}

func asFloat64(v any) (float64, bool) {
	switch x := v.(type) {
	case float64:
		return x, true
	case float32:
		return float64(x), true
	case int:
		return float64(x), true
	case int32:
		return float64(x), true
	case int64:
		return float64(x), true
	case uint:
		return float64(x), true
	case uint32:
		return float64(x), true
	case uint64:
		return float64(x), true
	default:
		return 0, false
	}
}

func buildFilter(tenantID string, f lead.ListFilter) bson.M {
	q := bson.M{"tenant_id": tenantID}
	if f.QualifiedOnly {
		q["qualification_source"] = bson.M{"$in": bson.A{
			string(lead.QualificationSourceAI),
			string(lead.QualificationSourceManual),
		}}
	}
	if f.Status != nil {
		q["status"] = string(*f.Status)
	}
	if f.MerchantID != nil {
		q["merchant_id"] = *f.MerchantID
	}
	if f.SemanticDirection != nil {
		switch strings.ToLower(strings.TrimSpace(*f.SemanticDirection)) {
		case "merchant", "merchants", "merch", "processing_request", "processing_requests", "processing":
			q["semantic_direction"] = bson.M{"$in": bson.A{"merchant", "merchants"}}
		case "trader", "traders":
			q["semantic_direction"] = bson.M{"$in": bson.A{"trader", "traders"}}
		case "ps_offer", "ps_offers", "offer", "offers":
			q["semantic_direction"] = bson.M{"$in": bson.A{"ps_offer", "ps_offers"}}
		default:
			q["semantic_direction"] = *f.SemanticDirection
		}
	}
	if f.ChatID != nil {
		q["chat_id"] = *f.ChatID
	}
	if f.MinScore != nil {
		q["score"] = bson.M{"$gte": *f.MinScore}
	}
	if f.FromDate != nil || f.ToDate != nil {
		dateQ := bson.M{}
		if f.FromDate != nil {
			dateQ["$gte"] = *f.FromDate
		}
		if f.ToDate != nil {
			dateQ["$lte"] = *f.ToDate
		}
		q["created_at"] = dateQ
	}
	if f.Reviewed != nil {
		if *f.Reviewed {
			q["user_feedback"] = bson.M{"$ne": nil}
		} else {
			q["user_feedback"] = nil
		}
	}
	return q
}

func decodeCursor(ctx context.Context, cur *mongo.Cursor) ([]*lead.Lead, error) {
	var docs []leadDoc
	if err := cur.All(ctx, &docs); err != nil {
		return nil, err
	}
	out := make([]*lead.Lead, len(docs))
	for i, d := range docs {
		out[i] = fromDoc(d)
	}
	return out, nil
}

type leadDoc struct {
	ID                  string     `bson:"_id"`
	TenantID            string     `bson:"tenant_id"`
	MessageID           string     `bson:"message_id"`
	ChatID              int64      `bson:"chat_id"`
	ChatTitle           string     `bson:"chat_title"`
	SenderID            int64      `bson:"sender_id"`
	SenderName          string     `bson:"sender_name"`
	SenderUsername      string     `bson:"sender_username"`
	Text                string     `bson:"text"`
	Geo                 []string   `bson:"geo"`
	Products            []string   `bson:"products"`
	SemanticDirection   string     `bson:"semantic_direction,omitempty"`
	SemanticCategory    string     `bson:"semantic_category,omitempty"`
	MerchantID          string     `bson:"merchant_id"`
	Status              string     `bson:"status"`
	QualificationSource string     `bson:"qualification_source,omitempty"`
	Score               float64    `bson:"score"`
	UserFeedback        *bool      `bson:"user_feedback"`
	CategoryAssignedAt  *time.Time `bson:"category_assigned_at,omitempty"`
	CreatedAt           time.Time  `bson:"created_at"`
	UpdatedAt           time.Time  `bson:"updated_at"`
}

func toDoc(l *lead.Lead) bson.M {
	return bson.M{
		"_id":                  l.ID(),
		"tenant_id":            l.TenantID(),
		"message_id":           l.MessageID(),
		"chat_id":              l.ChatID(),
		"chat_title":           l.ChatTitle(),
		"sender_id":            l.SenderID(),
		"sender_name":          l.SenderName(),
		"sender_username":      l.SenderUsername(),
		"text":                 l.Text(),
		"geo":                  l.Geo(),
		"products":             l.Products(),
		"semantic_direction":   l.SemanticDirection(),
		"semantic_category":    l.SemanticCategory(),
		"merchant_id":          l.MerchantID(),
		"status":               string(l.Status()),
		"qualification_source": string(l.QualificationSource()),
		"score":                l.Score(),
		"user_feedback":        l.UserFeedback(),
		"category_assigned_at": l.CategoryAssignedAt(),
		"updated_at":           l.UpdatedAt(),
	}
}

func fromDoc(d leadDoc) *lead.Lead {
	return lead.Restore(
		d.ID, d.TenantID, d.MessageID,
		d.ChatID, d.ChatTitle,
		d.SenderID, d.SenderName, d.SenderUsername,
		d.Text, d.Geo, d.Products,
		d.SemanticDirection,
		d.SemanticCategory,
		d.MerchantID,
		lead.Status(d.Status),
		lead.QualificationSource(d.QualificationSource),
		d.Score,
		d.UserFeedback,
		d.CategoryAssignedAt,
		d.CreatedAt, d.UpdatedAt,
	)
}
