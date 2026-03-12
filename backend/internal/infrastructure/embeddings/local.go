package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type LocalEmbedder struct {
	httpClient *http.Client
	apiURL     string
	model      string
}

func NewLocalEmbedder(apiURL, model string) *LocalEmbedder {
	if strings.TrimSpace(apiURL) == "" {
		apiURL = "http://127.0.0.1:1234/v1/embeddings"
	}
	if strings.TrimSpace(model) == "" {
		model = "embeddinggemma"
	}

	apiURL = normalizeEmbeddingURL(apiURL)
	model = normalizeEmbeddingModel(apiURL, model)

	return &LocalEmbedder{
		httpClient: &http.Client{Timeout: 60 * time.Second},
		apiURL:     apiURL,
		model:      model,
	}
}

func (e *LocalEmbedder) Embed(ctx context.Context, text string) ([]float32, error) {
	return e.EmbedDocument(ctx, text)
}

func (e *LocalEmbedder) EmbedDocument(ctx context.Context, text string) ([]float32, error) {
	return e.embed(ctx, text)
}

func (e *LocalEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	return e.embed(ctx, text)
}

func (e *LocalEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	if len(texts) == 0 {
		return nil, nil
	}

	endpoints := candidateEmbeddingEndpoints(e.apiURL)
	models := candidateEmbeddingModels(e.apiURL, e.model)
	var errs []string
	for _, model := range models {
		reqBody := map[string]any{
			"model": model,
			"input": texts,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		for _, endpoint := range endpoints {
			embeddings, reqErr := e.requestEmbeddings(ctx, endpoint, body)
			if reqErr == nil {
				if len(embeddings) != len(texts) {
					return nil, fmt.Errorf("embedding API returned %d embeddings for %d texts", len(embeddings), len(texts))
				}
				return embeddings, nil
			}
			errs = append(errs, fmt.Sprintf("%s model=%s: %v", endpoint, model, reqErr))
			if !shouldTryNextModel(reqErr) {
				break
			}
		}
	}

	return nil, fmt.Errorf("embedding batch request failed for all endpoints: %s", strings.Join(errs, "; "))
}

func (e *LocalEmbedder) embed(ctx context.Context, text string) ([]float32, error) {
	endpoints := candidateEmbeddingEndpoints(e.apiURL)
	models := candidateEmbeddingModels(e.apiURL, e.model)
	var errs []string
	for _, model := range models {
		reqBody := map[string]any{
			"model": model,
			"input": text,
		}

		body, err := json.Marshal(reqBody)
		if err != nil {
			return nil, err
		}

		for _, endpoint := range endpoints {
			embedding, reqErr := e.requestEmbedding(ctx, endpoint, body)
			if reqErr == nil {
				return embedding, nil
			}
			errs = append(errs, fmt.Sprintf("%s model=%s: %v", endpoint, model, reqErr))
			if !shouldTryNextModel(reqErr) {
				break
			}
		}
	}

	return nil, fmt.Errorf("embedding request failed for all endpoints: %s", strings.Join(errs, "; "))
}

func (e *LocalEmbedder) requestEmbedding(ctx context.Context, endpoint string, body []byte) ([]float32, error) {
	embeddings, err := e.requestEmbeddings(ctx, endpoint, body)
	if err != nil {
		return nil, err
	}
	if len(embeddings) == 0 || len(embeddings[0]) == 0 {
		return nil, fmt.Errorf("no embedding returned")
	}
	return embeddings[0], nil
}

func (e *LocalEmbedder) requestEmbeddings(ctx context.Context, endpoint string, body []byte) ([][]float32, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding API error: %d, endpoint: %s, body: %s", resp.StatusCode, endpoint, string(bodyBytes))
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	embeddings, err := parseEmbeddingsResponse(bodyBytes)
	if err != nil {
		return nil, fmt.Errorf("decode embedding response from %s: %w", endpoint, err)
	}

	return embeddings, nil
}

func normalizeEmbeddingURL(raw string) string {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return strings.TrimSpace(raw)
	}

	if u.Path == "" || u.Path == "/" {
		u.Path = "/v1/embeddings"
	}

	return u.String()
}

func normalizeEmbeddingModel(apiURL, model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return model
	}

	if !isOllamaURL(apiURL) {
		return model
	}

	switch model {
	case "text-embedding-qwen3-embedding-0.6b":
		return "qwen3-embedding:0.6b"
	default:
		return model
	}
}

func candidateEmbeddingModels(apiURL, configured string) []string {
	models := make([]string, 0, 5)
	seen := map[string]struct{}{}
	appendModel := func(model string) {
		model = strings.TrimSpace(model)
		if model == "" {
			return
		}
		if _, ok := seen[model]; ok {
			return
		}
		seen[model] = struct{}{}
		models = append(models, model)
	}

	appendModel(normalizeEmbeddingModel(apiURL, configured))
	if !isOllamaURL(apiURL) {
		return models
	}

	appendModel("embeddinggemma")
	appendModel("nomic-embed-text")
	appendModel("mxbai-embed-large")
	appendModel("qwen3-embedding:0.6b")
	return models
}

func shouldTryNextModel(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "model") && strings.Contains(msg, "not found")
}

func parseEmbeddingsResponse(body []byte) ([][]float32, error) {
	var openAIResult struct {
		Data []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &openAIResult); err == nil && len(openAIResult.Data) > 0 {
		embeddings := make([][]float32, 0, len(openAIResult.Data))
		for _, item := range openAIResult.Data {
			if len(item.Embedding) == 0 {
				continue
			}
			embeddings = append(embeddings, item.Embedding)
		}
		if len(embeddings) > 0 {
			return embeddings, nil
		}
	}

	var ollamaResult struct {
		Embeddings [][]float32 `json:"embeddings"`
		Embedding  []float32   `json:"embedding"`
	}
	if err := json.Unmarshal(body, &ollamaResult); err == nil {
		switch {
		case len(ollamaResult.Embeddings) > 0 && len(ollamaResult.Embeddings[0]) > 0:
			return ollamaResult.Embeddings, nil
		case len(ollamaResult.Embedding) > 0:
			return [][]float32{ollamaResult.Embedding}, nil
		}
	}

	return nil, fmt.Errorf("no embeddings returned")
}

func candidateEmbeddingEndpoints(raw string) []string {
	seen := map[string]struct{}{}
	candidates := make([]string, 0, 3)
	appendCandidate := func(endpoint string) {
		if endpoint == "" {
			return
		}
		if _, ok := seen[endpoint]; ok {
			return
		}
		seen[endpoint] = struct{}{}
		candidates = append(candidates, endpoint)
	}

	appendCandidate(raw)
	for _, endpoint := range alternateEmbeddingPaths(raw) {
		appendCandidate(endpoint)
	}

	return candidates
}

func alternateEmbeddingPaths(raw string) []string {
	u, err := url.Parse(raw)
	if err != nil {
		return nil
	}

	switch {
	case strings.HasSuffix(u.Path, "/v1/embeddings"):
		base := strings.TrimSuffix(u.Path, "/v1/embeddings")
		u.Path = base + "/api/embed"
		apiEmbed := u.String()
		u.Path = base + "/api/embeddings"
		return []string{apiEmbed, u.String()}
	case strings.HasSuffix(u.Path, "/api/embed"):
		base := strings.TrimSuffix(u.Path, "/api/embed")
		u.Path = base + "/v1/embeddings"
		return []string{u.String()}
	case strings.HasSuffix(u.Path, "/api/embeddings"):
		base := strings.TrimSuffix(u.Path, "/api/embeddings")
		u.Path = base + "/v1/embeddings"
		return []string{u.String()}
	default:
		if isOllamaURL(raw) {
			base := strings.TrimRight(u.Path, "/")
			u.Path = base + "/api/embed"
			apiEmbed := u.String()
			u.Path = base + "/v1/embeddings"
			return []string{apiEmbed, u.String()}
		}
		return nil
	}
}

func isOllamaURL(raw string) bool {
	u, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}

	host := strings.ToLower(u.Hostname())
	path := strings.ToLower(u.Path)

	return strings.Contains(host, "ollama") ||
		host == "localhost" ||
		host == "127.0.0.1" ||
		strings.Contains(path, "/api/embed") ||
		strings.Contains(path, "/api/embeddings")
}
