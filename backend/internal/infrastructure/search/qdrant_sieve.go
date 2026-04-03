package search

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"MRG/internal/infrastructure/embeddings"
	"MRG/internal/infrastructure/textnorm"

	"github.com/google/uuid"
	"github.com/qdrant/go-client/qdrant"
)

const (
	leadReferenceCollection       = "lead_reference_points"
	discussionReferenceCollection = "market_discussion_points"

	payloadTextKey              = "text"
	payloadIsLeadKey            = "is_lead"
	payloadSemanticDirectionKey = "semantic_direction"
	payloadSemanticFlagsKey     = "semantic_flags"

	directionMerchant     = "merchant"
	directionTraderSearch = "trader_search"
	directionTrader       = "trader"
	directionPSOffer      = "ps_offer"
	directionNoise        = "noise"

	categoryMerchants    = "merchants"
	categoryTraderSearch = "trader_search"
	categoryTraders      = "traders"
	categoryPSOffers     = "ps_offers"
	categoryNoise        = "noise"
)

var orderedLeadCategories = []string{categoryTraderSearch, categoryTraders, categoryMerchants, categoryPSOffers}

type QdrantSieve struct {
	embedder            embeddings.Embedder
	client              *qdrant.Client
	thresholdFn         func() float32
	categoryThresholdFn func(category string) float32
	windowSizeFn        func() time.Duration
	messageCache        map[string][]timedText
	collectionMu        sync.Mutex
	leadCollectionReady bool
	mu                  sync.Mutex
}

type timedText struct {
	text string
	at   time.Time
}

type detectionMeta struct {
	isLead            bool
	score             float32
	semanticDirection string
	semanticCategory  string
}

type embeddedReference struct {
	text      string
	isLead    bool
	direction string
	vector    []float32
}

func NewQdrantSieve(embedder embeddings.Embedder, client *qdrant.Client, thresholdFn func() float32, windowSizeFn func() time.Duration) *QdrantSieve {
	return &QdrantSieve{
		embedder:     embedder,
		client:       client,
		thresholdFn:  thresholdFn,
		windowSizeFn: windowSizeFn,
		messageCache: make(map[string][]timedText),
	}
}

func (s *QdrantSieve) WithCategoryThresholds(fn func(category string) float32) *QdrantSieve {
	s.categoryThresholdFn = fn
	return s
}

const maxContextRunes = 512

const queryTopK = uint64(12)

func (s *QdrantSieve) DetectLead(ctx context.Context, text string, senderKey string) (bool, float32, error) {
	meta, err := s.detectLeadMeta(ctx, s.combineSenderWindow(text, senderKey))
	if err != nil {
		return false, 0, err
	}
	return meta.isLead, meta.score, nil
}

func (s *QdrantSieve) DetectLeadSingle(ctx context.Context, text string) (bool, float32, error) {
	meta, err := s.detectLeadMeta(ctx, text)
	if err != nil {
		return false, 0, err
	}
	return meta.isLead, meta.score, nil
}

func (s *QdrantSieve) DetectLeadWithMeta(ctx context.Context, text string, senderKey string) (bool, float32, string, error) {
	meta, err := s.detectLeadMeta(ctx, s.combineSenderWindow(text, senderKey))
	if err != nil {
		return false, 0, "", err
	}
	return meta.isLead, meta.score, meta.semanticDirection, nil
}

func (s *QdrantSieve) DetectLeadSingleWithMeta(ctx context.Context, text string) (bool, float32, string, error) {
	meta, err := s.detectLeadMeta(ctx, text)
	if err != nil {
		return false, 0, "", err
	}
	return meta.isLead, meta.score, meta.semanticDirection, nil
}

func (s *QdrantSieve) detectLeadMeta(ctx context.Context, text string) (detectionMeta, error) {
	combined := normalizeContext(text)
	if combined == "" {
		return detectionMeta{}, nil
	}

	lower := strings.ToLower(combined)
	// Task 2: Exclude RUB (ruble inflows)
	if strings.Contains(lower, "rub") || strings.Contains(lower, "руб") || strings.Contains(lower, "₽") {
		return detectionMeta{
			isLead:            false,
			score:             0.1,
			semanticDirection: directionNoise,
			semanticCategory:  categoryNoise,
		}, nil
	}

	vec, err := s.embedder.Embed(ctx, combined)
	if err != nil {
		return detectionMeta{}, fmt.Errorf("embed: %w", err)
	}

	results, err := s.queryHybridSimilar(ctx, vec, combined, queryTopK, true)
	if err != nil || len(results) == 0 {
		return detectionMeta{}, nil
	}

	bestCategory, bestDirection, bestScore := bestCategoryMatch(results)

	// Force/Boost Merchant detection (traffic, price/service requests)
	isMerchantHint := strings.Contains(lower, "трафик") || strings.Contains(lower, "traffic") ||
		strings.Contains(lower, "прайс") || strings.Contains(lower, "price") ||
		strings.Contains(lower, "подключить") || strings.Contains(lower, "интеграция") ||
		strings.Contains(lower, "платежка") || strings.Contains(lower, "эквайринг")

	// Force/Boost Trader detection
	isTraderSearchHint := strings.Contains(lower, "ищу трейдера") ||
		strings.Contains(lower, "ищем трейдера") ||
		strings.Contains(lower, "нужен трейдер") ||
		strings.Contains(lower, "looking for trader") ||
		strings.Contains(lower, "search trader")
	isTraderHint := strings.Contains(lower, "трейдер") || strings.Contains(lower, "trader")

	if isMerchantHint && (bestCategory == "" || bestCategory == categoryNoise) {
		bestCategory = categoryMerchants
		bestDirection = directionMerchant
		bestScore = 0.9
	} else if isTraderSearchHint && (bestCategory == "" || bestCategory == categoryNoise) {
		bestCategory = categoryTraderSearch
		bestDirection = directionTraderSearch
		bestScore = 0.9
	} else if isTraderHint && (bestCategory == "" || bestCategory == categoryNoise) {
		bestCategory = categoryTraders
		bestDirection = directionTrader
		bestScore = 0.9
	}

	if bestCategory == "" {
		return detectionMeta{}, nil
	}

	threshold := s.thresholdForCategory(bestCategory)
	isLead := bestScore >= threshold
	return detectionMeta{
		isLead:            isLead,
		score:             bestScore,
		semanticDirection: bestDirection,
		semanticCategory:  bestCategory,
	}, nil
}

func (s *QdrantSieve) DetectDiscussion(ctx context.Context, text string, threshold float64) (bool, float32, error) {
	combined := normalizeContext(text)
	if combined == "" {
		return false, 0, nil
	}
	vec, err := s.embedder.Embed(ctx, combined)
	if err != nil {
		return false, 0, fmt.Errorf("embed: %w", err)
	}

	results, err := s.querySimilar(ctx, discussionReferenceCollection, vec, 5, false)
	if err != nil {
		return false, 0, err
	}
	if len(results) == 0 {
		return false, 0, nil
	}

	bestScore := results[0].GetScore()
	return float64(bestScore) >= threshold, bestScore, nil
}

func extractDirection(payload map[string]*qdrant.Value) string {
	for _, key := range []string{payloadSemanticDirectionKey, "direction", "category", "segment", "topic"} {
		if v, ok := payload[key]; ok {
			if s := strings.TrimSpace(v.GetStringValue()); s != "" {
				return s
			}
		}
	}
	return ""
}

func normalizeContext(text string) string {
	combined := textnorm.ForEmbedding(text)
	if combined == "" {
		return ""
	}
	runes := []rune(combined)
	if len(runes) > maxContextRunes {
		return string(runes[len(runes)-maxContextRunes:])
	}
	return combined
}

func (s *QdrantSieve) combineSenderWindow(text, senderKey string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	windowSize := s.windowSizeFn()
	now := time.Now()
	s.messageCache[senderKey] = append(s.messageCache[senderKey], timedText{text: text, at: now})

	cutoff := now.Add(-windowSize)
	fresh := s.messageCache[senderKey][:0]
	for _, msg := range s.messageCache[senderKey] {
		if msg.at.After(cutoff) {
			fresh = append(fresh, msg)
		}
	}
	s.messageCache[senderKey] = fresh

	var combined strings.Builder
	for _, msg := range fresh {
		if combined.Len() > 0 {
			combined.WriteByte(' ')
		}
		combined.WriteString(msg.text)
	}
	return combined.String()
}

func (s *QdrantSieve) AddReferencePoint(ctx context.Context, text string, isLead bool) error {
	return s.AddReferencePointWithMeta(ctx, text, isLead, "")
}

func (s *QdrantSieve) AddReferencePointWithMeta(ctx context.Context, text string, isLead bool, direction string) error {
	return s.AddReferencePointWithMetaFlags(ctx, text, isLead, direction, nil)
}

func (s *QdrantSieve) AddReferencePointWithMetaFlags(ctx context.Context, text string, isLead bool, direction string, flags []string) error {
	ref, err := s.embedSingleReference(ctx, text, isLead, direction)
	if err != nil {
		return err
	}

	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: leadReferenceCollection,
		Wait:           boolPtr(true),
		Points:         []*qdrant.PointStruct{buildLeadPoint(ref, flags)},
	})
	return err
}

func (s *QdrantSieve) AddReferencePointsBatch(ctx context.Context, texts []string, isLead []bool) error {
	directions := make([]string, len(texts))
	return s.AddReferencePointsBatchWithMeta(ctx, texts, isLead, directions)
}

func (s *QdrantSieve) AddReferencePointsBatchWithMeta(ctx context.Context, texts []string, isLead []bool, directions []string) error {
	if len(texts) == 0 {
		return nil
	}
	if len(texts) != len(isLead) {
		return fmt.Errorf("texts and isLead length mismatch")
	}
	if len(directions) != 0 && len(texts) != len(directions) {
		return fmt.Errorf("texts and directions length mismatch")
	}

	references := make([]embeddedReference, 0, len(texts))
	for i, text := range texts {
		direction := ""
		if len(directions) != 0 {
			direction = directions[i]
		}
		ref, err := normalizeReference(text, isLead[i], direction)
		if err != nil {
			continue
		}
		references = append(references, ref)
	}
	if len(references) == 0 {
		return fmt.Errorf("texts cannot be empty")
	}

	if err := s.embedReferences(ctx, references); err != nil {
		return err
	}

	points := make([]*qdrant.PointStruct, 0, len(references))
	for _, ref := range references {
		points = append(points, buildLeadPoint(ref, nil))
	}

	_, err := s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: leadReferenceCollection,
		Wait:           boolPtr(true),
		Points:         points,
	})
	return err
}

func (s *QdrantSieve) thresholdForCategory(category string) float32 {
	if s.categoryThresholdFn != nil {
		return s.categoryThresholdFn(category)
	}
	if s.thresholdFn != nil {
		return s.thresholdFn()
	}
	return 0.60
}

func (s *QdrantSieve) AddDiscussionPoint(ctx context.Context, text string) error {
	text = textnorm.ForEmbedding(text)
	if text == "" {
		return fmt.Errorf("text cannot be empty")
	}
	vec, err := s.embedder.EmbedDocument(ctx, text)
	if err != nil {
		return fmt.Errorf("embed: %w", err)
	}
	if err := s.ensureCollectionWithName(ctx, discussionReferenceCollection, uint64(len(vec))); err != nil {
		return err
	}
	_, err = s.client.Upsert(ctx, &qdrant.UpsertPoints{
		CollectionName: discussionReferenceCollection,
		Wait:           boolPtr(true),
		Points: []*qdrant.PointStruct{
			{
				Id: &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: uuid.New().String()}},
				Vectors: &qdrant.Vectors{VectorsOptions: &qdrant.Vectors_Vector{Vector: &qdrant.Vector{
					Vector: &qdrant.Vector_Dense{Dense: &qdrant.DenseVector{Data: vec}},
				}}},
				Payload: map[string]*qdrant.Value{
					payloadTextKey: {Kind: &qdrant.Value_StringValue{StringValue: text}},
				},
			},
		},
	})
	return err
}

func (s *QdrantSieve) querySimilar(ctx context.Context, collection string, vector []float32, limit uint64, withPayload bool) ([]*qdrant.ScoredPoint, error) {
	exists, err := s.client.CollectionExists(ctx, collection)
	if err != nil || !exists {
		return nil, err
	}

	query := &qdrant.QueryPoints{
		CollectionName: collection,
		Query: &qdrant.Query{
			Variant: &qdrant.Query_Nearest{
				Nearest: &qdrant.VectorInput{
					Variant: &qdrant.VectorInput_Dense{
						Dense: &qdrant.DenseVector{Data: vector},
					},
				},
			},
		},
		Limit: &limit,
	}
	if withPayload {
		query.WithPayload = &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
		}
	}

	results, err := s.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("qdrant query: %w", err)
	}
	return results, nil
}

func (s *QdrantSieve) queryHybridSimilar(ctx context.Context, vector []float32, text string, limit uint64, withPayload bool) ([]*qdrant.ScoredPoint, error) {
	if err := s.ensureCollection(ctx, uint64(len(vector))); err != nil {
		return nil, err
	}

	indices, values := buildSparseKeywordVector(text)
	if len(indices) == 0 || len(values) == 0 {
		return s.querySimilar(ctx, leadReferenceCollection, vector, limit, withPayload)
	}

	query := &qdrant.QueryPoints{
		CollectionName: leadReferenceCollection,
		Prefetch: []*qdrant.PrefetchQuery{
			{
				Query: qdrant.NewQuerySparse(indices, values),
				Using: strPtr(sparseKeywordVectorName),
				Limit: &limit,
			},
			{
				Query: qdrant.NewQueryDense(vector),
				Limit: &limit,
			},
		},
		Query: qdrant.NewQueryFusion(qdrant.Fusion_RRF),
		Limit: &limit,
	}
	if withPayload {
		query.WithPayload = &qdrant.WithPayloadSelector{
			SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
		}
	}

	results, err := s.client.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("qdrant hybrid query: %w", err)
	}
	return results, nil
}

func bestCategoryMatch(results []*qdrant.ScoredPoint) (string, string, float32) {
	bestScores := make(map[string]float32, len(orderedLeadCategories))
	bestDirections := make(map[string]string, len(orderedLeadCategories))
	for _, result := range results {
		category, direction := detectCategoryFromPayload(result.GetPayload())
		if category == "" {
			continue
		}
		if score := result.GetScore(); score > bestScores[category] {
			bestScores[category] = score
			bestDirections[category] = direction
		}
	}

	bestCategory := ""
	var bestScore float32
	bestDirection := ""
	for _, category := range orderedLeadCategories {
		if score := bestScores[category]; score > bestScore {
			bestCategory = category
			bestScore = score
			bestDirection = bestDirections[category]
		}
	}
	return bestCategory, bestDirection, bestScore
}

func normalizeReference(text string, isLead bool, direction string) (embeddedReference, error) {
	normalizedText := textnorm.ForEmbedding(text)
	if normalizedText == "" {
		return embeddedReference{}, fmt.Errorf("text cannot be empty")
	}
	return embeddedReference{
		text:      normalizedText,
		isLead:    isLead,
		direction: strings.TrimSpace(direction),
	}, nil
}

func (s *QdrantSieve) embedSingleReference(ctx context.Context, text string, isLead bool, direction string) (embeddedReference, error) {
	ref, err := normalizeReference(text, isLead, direction)
	if err != nil {
		return embeddedReference{}, err
	}
	vector, err := s.embedder.EmbedDocument(ctx, ref.text)
	if err != nil {
		return embeddedReference{}, fmt.Errorf("embed: %w", err)
	}
	if len(vector) == 0 {
		return embeddedReference{}, fmt.Errorf("empty embedding returned")
	}
	ref.vector = vector
	if err := s.ensureCollection(ctx, uint64(len(vector))); err != nil {
		return embeddedReference{}, err
	}
	return ref, nil
}

func (s *QdrantSieve) embedReferences(ctx context.Context, refs []embeddedReference) error {
	if len(refs) == 0 {
		return nil
	}

	texts := make([]string, 0, len(refs))
	for _, ref := range refs {
		texts = append(texts, ref.text)
	}

	var vectors [][]float32
	if batchEmbedder, ok := s.embedder.(embeddings.BatchEmbedder); ok {
		embedded, err := batchEmbedder.EmbedDocuments(ctx, texts)
		if err != nil {
			return fmt.Errorf("embed batch: %w", err)
		}
		vectors = embedded
	} else {
		vectors = make([][]float32, 0, len(texts))
		for _, text := range texts {
			vec, err := s.embedder.EmbedDocument(ctx, text)
			if err != nil {
				return fmt.Errorf("embed: %w", err)
			}
			vectors = append(vectors, vec)
		}
	}

	if len(vectors) != len(refs) {
		return fmt.Errorf("embedding count mismatch: got %d want %d", len(vectors), len(refs))
	}
	if len(vectors[0]) == 0 {
		return fmt.Errorf("empty embedding returned")
	}
	if err := s.ensureCollection(ctx, uint64(len(vectors[0]))); err != nil {
		return err
	}

	for i := range refs {
		if len(vectors[i]) == 0 {
			return fmt.Errorf("empty embedding returned at index %d", i)
		}
		refs[i].vector = vectors[i]
	}
	return nil
}

func buildLeadPoint(ref embeddedReference, flags []string) *qdrant.PointStruct {
	indices, values := buildSparseKeywordVector(ref.text)
	vectors := map[string]*qdrant.Vector{
		"": qdrant.NewVectorDense(ref.vector),
	}
	if len(indices) > 0 && len(values) > 0 {
		vectors[sparseKeywordVectorName] = qdrant.NewVectorSparse(indices, values)
	}
	return &qdrant.PointStruct{
		Id:      &qdrant.PointId{PointIdOptions: &qdrant.PointId_Uuid{Uuid: referencePointID(ref.text, ref.isLead, ref.direction)}},
		Vectors: qdrant.NewVectorsMap(vectors),
		Payload: buildLeadPayload(ref.text, ref.isLead, ref.direction, flags),
	}
}

func boolPtr(v bool) *bool {
	return &v
}

func strPtr(v string) *string {
	return &v
}

func referencePointID(text string, isLead bool, direction string) string {
	key := fmt.Sprintf("%s|%t|%s", text, isLead, strings.TrimSpace(direction))
	return uuid.NewSHA1(uuid.NameSpaceOID, []byte(key)).String()
}

func (s *QdrantSieve) ensureCollection(ctx context.Context, vectorSize uint64) error {
	s.collectionMu.Lock()
	defer s.collectionMu.Unlock()

	if s.leadCollectionReady {
		return nil
	}
	exists, err := s.client.CollectionExists(ctx, leadReferenceCollection)
	if err != nil {
		return err
	}
	if !exists {
		if err := s.client.CreateCollection(ctx, &qdrant.CreateCollection{
			CollectionName: leadReferenceCollection,
			VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
				Size:     vectorSize,
				Distance: qdrant.Distance_Cosine,
			}),
			SparseVectorsConfig: qdrant.NewSparseVectorsConfig(map[string]*qdrant.SparseVectorParams{
				sparseKeywordVectorName: {},
			}),
		}); err != nil {
			return err
		}
		s.leadCollectionReady = true
		return nil
	}

	info, err := s.client.GetCollectionInfo(ctx, leadReferenceCollection)
	if err != nil {
		return err
	}
	sparseConfig := info.GetConfig().GetParams().GetSparseVectorsConfig()
	if sparseConfig == nil || sparseConfig.GetMap()[sparseKeywordVectorName] == nil {
		if err := s.client.UpdateCollection(ctx, &qdrant.UpdateCollection{
			CollectionName: leadReferenceCollection,
			SparseVectorsConfig: qdrant.NewSparseVectorsConfig(map[string]*qdrant.SparseVectorParams{
				sparseKeywordVectorName: {},
			}),
		}); err != nil {
			return err
		}
		if err := s.backfillSparseVectors(ctx, leadReferenceCollection); err != nil {
			return err
		}
	}

	s.leadCollectionReady = true
	return nil
}

func (s *QdrantSieve) ensureCollectionWithName(ctx context.Context, name string, vectorSize uint64) error {
	exists, err := s.client.CollectionExists(ctx, name)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return s.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: name,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     vectorSize,
			Distance: qdrant.Distance_Cosine,
		}),
	})
}

func (s *QdrantSieve) backfillSparseVectors(ctx context.Context, collection string) error {
	offset := (*qdrant.PointId)(nil)
	limit := uint32(256)
	wait := true

	for {
		points, nextOffset, err := s.client.ScrollAndOffset(ctx, &qdrant.ScrollPoints{
			CollectionName: collection,
			Limit:          &limit,
			Offset:         offset,
			WithPayload: &qdrant.WithPayloadSelector{
				SelectorOptions: &qdrant.WithPayloadSelector_Enable{Enable: true},
			},
		})
		if err != nil {
			return err
		}
		if len(points) == 0 {
			return nil
		}

		updates := make([]*qdrant.PointVectors, 0, len(points))
		for _, point := range points {
			text := strings.TrimSpace(point.GetPayload()[payloadTextKey].GetStringValue())
			if text == "" {
				continue
			}
			indices, values := buildSparseKeywordVector(text)
			if len(indices) == 0 || len(values) == 0 {
				continue
			}
			updates = append(updates, &qdrant.PointVectors{
				Id: point.GetId(),
				Vectors: qdrant.NewVectorsMap(map[string]*qdrant.Vector{
					sparseKeywordVectorName: qdrant.NewVectorSparse(indices, values),
				}),
			})
		}
		if len(updates) > 0 {
			if _, err := s.client.UpdateVectors(ctx, &qdrant.UpdatePointVectors{
				CollectionName: collection,
				Wait:           &wait,
				Points:         updates,
			}); err != nil {
				return err
			}
		}
		if nextOffset == nil {
			return nil
		}
		offset = nextOffset
	}
}

func buildLeadPayload(text string, isLead bool, direction string, flags []string) map[string]*qdrant.Value {
	payload := map[string]*qdrant.Value{
		payloadTextKey:   {Kind: &qdrant.Value_StringValue{StringValue: text}},
		payloadIsLeadKey: {Kind: &qdrant.Value_BoolValue{BoolValue: isLead}},
	}
	if direction != "" {
		payload[payloadSemanticDirectionKey] = &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: direction}}
	}
	if len(flags) == 0 {
		return payload
	}

	values := make([]*qdrant.Value, 0, len(flags))
	seen := make(map[string]struct{}, len(flags))
	for _, f := range flags {
		v := strings.ToLower(strings.TrimSpace(f))
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		values = append(values, &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: v}})
	}
	if len(values) > 0 {
		payload[payloadSemanticFlagsKey] = &qdrant.Value{Kind: &qdrant.Value_ListValue{ListValue: &qdrant.ListValue{Values: values}}}
	}

	return payload
}

func detectCategoryFromPayload(payload map[string]*qdrant.Value) (string, string) {
	direction := normalizeDirection(extractDirection(payload))
	if direction != "" {
		if category := directionToCategory(direction); category != "" {
			return category, direction
		}
	}

	if v, ok := payload[payloadIsLeadKey]; ok && !v.GetBoolValue() {
		return categoryNoise, directionNoise
	}
	return "", ""
}

func normalizeDirection(direction string) string {
	normalized := strings.ToLower(strings.TrimSpace(direction))
	switch normalized {
	case "", "lead":
		return ""
	case "merchant", "merchants", "merch":
		return directionMerchant
	case "trader_search", "search_trader", "search_traders", "looking_for_trader":
		return directionTraderSearch
	case "trader", "traders":
		return directionTrader
	case "processing_request", "processing_requests", "request_processing":
		return directionMerchant
	case "ps_offer", "ps_offers", "offer", "provider":
		return directionPSOffer
	case "noise", "spam", "negative":
		return directionNoise
	default:
		return normalized
	}
}

func directionToCategory(direction string) string {
	switch normalizeDirection(direction) {
	case directionMerchant:
		return categoryMerchants
	case directionTraderSearch:
		return categoryTraderSearch
	case directionTrader:
		return categoryTraders
	case directionPSOffer:
		return categoryPSOffers
	case directionNoise:
		return categoryNoise
	default:
		return ""
	}
}
