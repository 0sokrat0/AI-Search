package embeddings

import "context"

type Embedder interface {
	Embed(ctx context.Context, text string) ([]float32, error)
	EmbedDocument(ctx context.Context, text string) ([]float32, error)
	EmbedQuery(ctx context.Context, text string) ([]float32, error)
}

type BatchEmbedder interface {
	EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error)
}
