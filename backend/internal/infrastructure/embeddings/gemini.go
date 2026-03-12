package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const geminiAPIBaseURL = "https://generativelanguage.googleapis.com/v1beta/models"

type GeminiEmbedder struct {
	apiKey     string
	modelName  string
	httpClient *http.Client
}

func NewGeminiEmbedder(apiKey, modelName string) *GeminiEmbedder {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		modelName = "gemini-embedding-001"
	}
	modelName = strings.TrimPrefix(modelName, "models/")

	return &GeminiEmbedder{
		apiKey:     apiKey,
		modelName:  modelName,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

func (e *GeminiEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return e.EmbedDocument(ctx, text)
}

func (e *GeminiEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return e.embed(ctx, text, "RETRIEVAL_DOCUMENT")
}

func (e *GeminiEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.embed(ctx, text, "RETRIEVAL_QUERY")
}

func (e *GeminiEmbedder) embed(ctx context.Context, text string, taskType string) ([]float32, error) {
	reqBody := map[string]any{
		"content": map[string]any{
			"parts": []map[string]string{
				{"text": text},
			},
		},
		"taskType": taskType,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	url := fmt.Sprintf("%s/%s:embedContent?key=%s", geminiAPIBaseURL, e.modelName, e.apiKey)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to make http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var errResp map[string]any
		if json.NewDecoder(resp.Body).Decode(&errResp) == nil {
			return nil, fmt.Errorf("gemini api error: %s, response: %v", resp.Status, errResp)
		}
		return nil, fmt.Errorf("gemini api error: %s", resp.Status)
	}

	var result struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response body: %w", err)
	}

	return result.Embedding.Values, nil
}
