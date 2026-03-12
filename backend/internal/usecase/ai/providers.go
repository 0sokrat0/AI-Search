package ai

import (
	"context"

	"MRG/internal/infrastructure/embeddings"
)

type EmbeddingAdapter struct {
	providerName string
	embedder     embeddings.Embedder
}

func NewEmbeddingAdapter(providerName string, embedder embeddings.Embedder) *EmbeddingAdapter {
	return &EmbeddingAdapter{providerName: providerName, embedder: embedder}
}

func (a *EmbeddingAdapter) Name() string {
	return a.providerName
}

func (a *EmbeddingAdapter) Embed(ctx context.Context, text string, task EmbeddingTask) ([]float32, error) {
	switch task {
	case EmbeddingTaskQuery:
		return a.embedder.EmbedQuery(ctx, text)
	default:
		return a.embedder.EmbedDocument(ctx, text)
	}
}
