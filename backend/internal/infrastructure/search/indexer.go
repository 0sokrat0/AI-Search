package search

import (
	"context"
	"fmt"
	"strings"

	"github.com/qdrant/go-client/qdrant"
)

type Indexer struct {
	client         *qdrant.Client
	collection     string
	collectionName string
}

func NewIndexer(client *qdrant.Client, collection string) *Indexer {
	return &Indexer{
		client:         client,
		collection:     collection,
		collectionName: collection,
	}
}

func (i *Indexer) CreateCollection(ctx context.Context) error {
	err := i.client.CreateCollection(ctx, &qdrant.CreateCollection{
		CollectionName: i.collectionName,
		VectorsConfig: qdrant.NewVectorsConfig(&qdrant.VectorParams{
			Size:     1024,
			Distance: qdrant.Distance_Cosine,
		}),
	})

	if err != nil && !strings.Contains(err.Error(), "already exists") {
		return err
	}

	return nil
}

func (i *Indexer) mapToValue(m map[string]any) map[string]*qdrant.Value {
	result := make(map[string]*qdrant.Value)
	for k, v := range m {
		result[k] = i.toValue(v)
	}
	return result
}

func (i *Indexer) toValue(v any) *qdrant.Value {
	switch val := v.(type) {
	case string:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: val}}
	case int:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: int64(val)}}
	case int64:
		return &qdrant.Value{Kind: &qdrant.Value_IntegerValue{IntegerValue: val}}
	case float32:
		return &qdrant.Value{Kind: &qdrant.Value_DoubleValue{DoubleValue: float64(val)}}
	default:
		return &qdrant.Value{Kind: &qdrant.Value_StringValue{StringValue: fmt.Sprintf("%v", val)}}
	}
}
