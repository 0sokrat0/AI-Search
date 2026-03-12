package ai

import (
	"context"
	"errors"
	"testing"
)

type fakeEmbeddingProvider struct {
	name string
	vec  []float32
	err  error
}

func (p fakeEmbeddingProvider) Name() string { return p.name }
func (p fakeEmbeddingProvider) Embed(context.Context, string, EmbeddingTask) ([]float32, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.vec, nil
}

type fakeRAGProvider struct {
	name string
	docs []RAGDocument
	err  error
}

func (p fakeRAGProvider) Name() string { return p.name }
func (p fakeRAGProvider) Retrieve(context.Context, string, int) ([]RAGDocument, error) {
	if p.err != nil {
		return nil, p.err
	}
	return p.docs, nil
}

type fakeLLMProvider struct {
	name string
	out  string
	err  error
}

func (p fakeLLMProvider) Name() string { return p.name }
func (p fakeLLMProvider) GenerateJSON(context.Context, string, string, string) (string, error) {
	if p.err != nil {
		return "", p.err
	}
	return p.out, nil
}

func TestEmbedFallback(t *testing.T) {
	o := NewOrchestrator(
		[]EmbeddingProvider{
			fakeEmbeddingProvider{name: "primary", err: errors.New("timeout")},
			fakeEmbeddingProvider{name: "secondary", vec: []float32{0.1, 0.2}},
		},
		nil,
		nil,
	)

	vec, provider, err := o.Embed(context.Background(), "hello", EmbeddingTaskQuery)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if provider != "secondary" {
		t.Fatalf("expected secondary provider, got %s", provider)
	}
	if len(vec) != 2 {
		t.Fatalf("unexpected vec len: %d", len(vec))
	}
}

func TestRetrieveFallbackError(t *testing.T) {
	o := NewOrchestrator(
		nil,
		[]RAGProvider{
			fakeRAGProvider{name: "qdrant-main", err: errors.New("unavailable")},
			fakeRAGProvider{name: "qdrant-backup", docs: nil},
		},
		nil,
	)

	_, _, err := o.Retrieve(context.Background(), "query", 5)
	if err == nil {
		t.Fatal("expected error")
	}
	var fbErr *FallbackError
	if !errors.As(err, &fbErr) {
		t.Fatalf("expected FallbackError, got %T", err)
	}
	if fbErr.Operation != "retrieve" {
		t.Fatalf("expected retrieve operation, got %s", fbErr.Operation)
	}
}

func TestGenerateJSONFallback(t *testing.T) {
	o := NewOrchestrator(
		nil,
		nil,
		[]LLMProvider{
			fakeLLMProvider{name: "gemini", err: errors.New("429")},
			fakeLLMProvider{name: "backup", out: `{"ok":true}`},
		},
	)

	out, provider, err := o.GenerateJSON(context.Background(), "sys", "user", "schema")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if provider != "backup" {
		t.Fatalf("expected backup provider, got %s", provider)
	}
	if out != `{"ok":true}` {
		t.Fatalf("unexpected output: %s", out)
	}
}
