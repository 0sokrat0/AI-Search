package ai

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

type EmbeddingTask string

const (
	EmbeddingTaskDocument EmbeddingTask = "document"
	EmbeddingTaskQuery    EmbeddingTask = "query"
)

type RAGDocument struct {
	ID       string
	Source   string
	Snippet  string
	Score    float64
	Metadata map[string]string
}

type EmbeddingProvider interface {
	Name() string
	Embed(ctx context.Context, text string, task EmbeddingTask) ([]float32, error)
}

type RAGProvider interface {
	Name() string
	Retrieve(ctx context.Context, query string, topK int) ([]RAGDocument, error)
}

type LLMProvider interface {
	Name() string
	GenerateJSON(ctx context.Context, systemPrompt, userPrompt, schema string) (string, error)
}

type Orchestrator struct {
	embeddings []EmbeddingProvider
	rag        []RAGProvider
	llm        []LLMProvider
}

func NewOrchestrator(embeddings []EmbeddingProvider, rag []RAGProvider, llm []LLMProvider) *Orchestrator {
	return &Orchestrator{embeddings: embeddings, rag: rag, llm: llm}
}

type FallbackError struct {
	Operation string
	Attempts  []string
}

func (e *FallbackError) Error() string {
	return fmt.Sprintf("%s failed after %d attempts: %s", e.Operation, len(e.Attempts), strings.Join(e.Attempts, "; "))
}

func (o *Orchestrator) Embed(ctx context.Context, text string, task EmbeddingTask) ([]float32, string, error) {
	if len(o.embeddings) == 0 {
		return nil, "", errors.New("no embedding providers configured")
	}

	attempts := make([]string, 0, len(o.embeddings))
	for _, provider := range o.embeddings {
		vec, err := provider.Embed(ctx, text, task)
		if err == nil && len(vec) > 0 {
			return vec, provider.Name(), nil
		}
		if err != nil {
			attempts = append(attempts, provider.Name()+": "+err.Error())
		} else {
			attempts = append(attempts, provider.Name()+": empty embedding")
		}
	}

	return nil, "", &FallbackError{Operation: "embed", Attempts: attempts}
}

func (o *Orchestrator) Retrieve(ctx context.Context, query string, topK int) ([]RAGDocument, string, error) {
	if len(o.rag) == 0 {
		return nil, "", errors.New("no rag providers configured")
	}

	attempts := make([]string, 0, len(o.rag))
	for _, provider := range o.rag {
		docs, err := provider.Retrieve(ctx, query, topK)
		if err == nil && len(docs) > 0 {
			return docs, provider.Name(), nil
		}
		if err != nil {
			attempts = append(attempts, provider.Name()+": "+err.Error())
		} else {
			attempts = append(attempts, provider.Name()+": empty result")
		}
	}

	return nil, "", &FallbackError{Operation: "retrieve", Attempts: attempts}
}

func (o *Orchestrator) GenerateJSON(ctx context.Context, systemPrompt, userPrompt, schema string) (string, string, error) {
	if len(o.llm) == 0 {
		return "", "", errors.New("no llm providers configured")
	}

	attempts := make([]string, 0, len(o.llm))
	for _, provider := range o.llm {
		out, err := provider.GenerateJSON(ctx, systemPrompt, userPrompt, schema)
		if err == nil && strings.TrimSpace(out) != "" {
			return out, provider.Name(), nil
		}
		if err != nil {
			attempts = append(attempts, provider.Name()+": "+err.Error())
		} else {
			attempts = append(attempts, provider.Name()+": empty response")
		}
	}

	return "", "", &FallbackError{Operation: "generate_json", Attempts: attempts}
}
