package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"math"
	"net/http"
	"strings"
	"time"
)

type Embedder interface {
	EmbedDocument(ctx context.Context, post ScrapedPost, insight *Insight) ([]float32, error)
	EmbedQuery(ctx context.Context, query string) ([]float32, error)
	Name() string
}

type MockEmbedder struct {
	dimensions int
}

type GeminiEmbedder struct {
	apiKey     string
	model      string
	dimensions int
	client     *http.Client
}

func NewEmbedder(cfg Config) Embedder {
	switch cfg.EmbeddingProvider {
	case "gemini":
		if cfg.GeminiAPIKey != "" {
			return NewGeminiEmbedder(cfg.GeminiAPIKey, cfg.GeminiEmbeddingModel, cfg.EmbeddingDimensions)
		}
	}

	return NewMockEmbedder(cfg.EmbeddingDimensions)
}

func NewMockEmbedder(dimensions int) *MockEmbedder {
	if dimensions < 32 {
		dimensions = 32
	}
	return &MockEmbedder{dimensions: dimensions}
}

func NewGeminiEmbedder(apiKey, model string, dimensions int) *GeminiEmbedder {
	if dimensions < 32 {
		dimensions = 256
	}

	return &GeminiEmbedder{
		apiKey:     apiKey,
		model:      defaultString(model, "gemini-embedding-001"),
		dimensions: dimensions,
		client:     &http.Client{Timeout: 45 * time.Second},
	}
}

func (m *MockEmbedder) Name() string {
	return fmt.Sprintf("mock-hash:%d", m.dimensions)
}

func (g *GeminiEmbedder) Name() string {
	return fmt.Sprintf("gemini-embedding:%s", g.model)
}

func (m *MockEmbedder) EmbedDocument(_ context.Context, post ScrapedPost, insight *Insight) ([]float32, error) {
	return m.embed(buildEmbeddingText(post, insight)), nil
}

func (m *MockEmbedder) EmbedQuery(_ context.Context, query string) ([]float32, error) {
	return m.embed(query), nil
}

func (m *MockEmbedder) embed(text string) []float32 {
	vector := make([]float32, m.dimensions)
	tokens := embeddingTokens(text)
	if len(tokens) == 0 {
		return vector
	}

	for _, token := range tokens {
		index, sign := hashToken(token, m.dimensions)
		vector[index] += sign
	}

	return normalizeVector(vector)
}

func (g *GeminiEmbedder) EmbedDocument(ctx context.Context, post ScrapedPost, insight *Insight) ([]float32, error) {
	return g.embed(ctx, buildEmbeddingText(post, insight), "RETRIEVAL_DOCUMENT", post.Title)
}

func (g *GeminiEmbedder) EmbedQuery(ctx context.Context, query string) ([]float32, error) {
	return g.embed(ctx, query, "RETRIEVAL_QUERY", "")
}

func (g *GeminiEmbedder) embed(ctx context.Context, text, taskType, title string) ([]float32, error) {
	text = normalizeText(text)
	if text == "" {
		return nil, fmt.Errorf("empty text for embedding")
	}

	requestBody := map[string]any{
		"content": map[string]any{
			"parts": []map[string]string{
				{"text": text},
			},
		},
		"taskType":             taskType,
		"outputDimensionality": g.dimensions,
	}
	if title = normalizeText(title); title != "" && taskType == "RETRIEVAL_DOCUMENT" {
		requestBody["title"] = title
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:embedContent?key=%s", g.model, g.apiKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := g.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&apiErr)
		return nil, fmt.Errorf("gemini embedding request failed: status=%d body=%v", resp.StatusCode, apiErr)
	}

	var payload struct {
		Embedding struct {
			Values []float32 `json:"values"`
		} `json:"embedding"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	if len(payload.Embedding.Values) == 0 {
		return nil, fmt.Errorf("empty embedding response")
	}
	return normalizeVector(payload.Embedding.Values), nil
}

func buildEmbeddingText(post ScrapedPost, insight *Insight) string {
	parts := []string{
		post.Platform,
		post.Channel,
		post.Title,
		insight.CorePainPoint,
		insight.CurrentWorkaround,
		insight.SaaSFeasibility,
		post.Content,
		strings.Join(post.MatchedKeywords, " "),
	}
	return normalizeText(strings.Join(parts, "\n"))
}

func embeddingTokens(text string) []string {
	text = strings.ToLower(normalizeText(text))
	text = clusterNonWordPattern.ReplaceAllString(text, " ")
	parts := strings.Fields(text)
	var tokens []string
	for _, part := range parts {
		if _, exists := clusterStopWords[part]; exists {
			continue
		}
		if utfLen(part) <= 1 {
			continue
		}
		tokens = append(tokens, part)
	}

	if containsHan(text) {
		tokens = append(tokens, chineseClusterTokens(text)...)
	}

	return uniqueStrings(tokens)
}

func hashToken(token string, dimensions int) (int, float32) {
	hasher := fnv.New64a()
	_, _ = hasher.Write([]byte(token))
	value := hasher.Sum64()
	index := int(value % uint64(dimensions))
	sign := float32(1)
	if (value>>1)%2 == 0 {
		sign = -1
	}
	return index, sign
}

func normalizeVector(values []float32) []float32 {
	var sum float64
	for _, value := range values {
		sum += float64(value * value)
	}
	if sum == 0 {
		return values
	}

	norm := float32(math.Sqrt(sum))
	normalized := make([]float32, len(values))
	for index, value := range values {
		normalized[index] = value / norm
	}
	return normalized
}
